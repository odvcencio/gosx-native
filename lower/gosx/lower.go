package gosx

import (
	"fmt"
	"html"
	"strconv"
	"strings"
	"unicode"

	gosxlang "m31labs.dev/gosx"
	"m31labs.dev/gosx-native/target"
	gosxir "m31labs.dev/gosx/ir"
	islandprogram "m31labs.dev/gosx/island/program"
	"m31labs.dev/gosx/nir"
	gotreesitter "github.com/odvcencio/gotreesitter"
)

type structInfo struct {
	Fields     []nir.PropField
	FieldNames map[string]string
	FieldTypes map[string]string
}

type componentSignature struct {
	PropsName string
	PropsType string
}

type lowerer struct {
	prog       *gosxir.Program
	structs    map[string]structInfo
	signatures map[string]componentSignature

	propsName string
	propRefs  map[string]string
	propTypes map[string]string
	signals   map[string]string
	handlers  map[string]gosxir.HandlerInfo
	scope     *gosxir.ExprScope
}

// LowerSource compiles GoSX source through gosx's existing front end and
// translates the resulting component IR into gosx-native NIR.
func LowerSource(src []byte) (*nir.Module, error) {
	prog, err := gosxlang.Compile(src)
	if err != nil {
		return nil, err
	}

	tree, lang, err := gosxlang.Parse(src)
	if err != nil {
		return nil, err
	}
	defer tree.Release()

	l := &lowerer{
		prog:       prog,
		structs:    collectStructs(tree.RootNode(), src, lang),
		signatures: collectComponentSignatures(tree.RootNode(), src, lang),
	}

	mod := &nir.Module{Version: 1, SourceLanguage: "go"}
	for _, comp := range prog.Components {
		lowered, err := l.lowerComponent(comp)
		if err != nil {
			return nil, err
		}
		if lowered != nil {
			mod.Components = append(mod.Components, lowered)
		}
	}
	return mod, nil
}

func (l *lowerer) lowerComponent(comp gosxir.Component) (*nir.Component, error) {
	sig := l.signatures[comp.Name]
	if sig.PropsName == "" {
		sig.PropsName = "props"
	}
	if sig.PropsType == "" {
		sig.PropsType = comp.PropsType
	}

	info := l.structs[sig.PropsType]
	l.propsName = sig.PropsName
	l.propRefs = make(map[string]string)
	l.propTypes = make(map[string]string)
	for raw, canonical := range info.FieldNames {
		l.propRefs[sig.PropsName+"."+raw] = sig.PropsName + "." + canonical
	}
	for raw, typ := range info.FieldTypes {
		canonical := info.FieldNames[raw]
		if canonical == "" {
			continue
		}
		l.propTypes[sig.PropsName+"."+canonical] = typ
	}

	c := &nir.Component{
		Name:  comp.Name,
		Props: &nir.Props{Fields: info.Fields},
		Span:  irSpan(comp.Span),
	}

	l.signals = make(map[string]string)
	l.handlers = handlersByName(comp.Scope)
	l.scope = l.exprScope(comp.Scope, sig.PropsName)

	if comp.Scope != nil {
		for _, sig := range comp.Scope.Signals {
			decl, err := l.lowerSignal(sig)
			if err != nil {
				return nil, fmt.Errorf("signal %s: %w", sig.Name, err)
			}
			c.Signals = append(c.Signals, decl)
			l.signals[decl.Name] = decl.Type
		}
		for _, computed := range comp.Scope.Computeds {
			decl, err := l.lowerComputed(computed)
			if err != nil {
				return nil, fmt.Errorf("computed %s: %w", computed.Name, err)
			}
			c.Computeds = append(c.Computeds, decl)
			l.signals[decl.Name] = decl.Type
		}
	}

	body, err := l.lowerView(comp.Root)
	if err != nil {
		return nil, fmt.Errorf("component %s body: %w", comp.Name, err)
	}
	c.Body = body
	return c, nil
}

func (l *lowerer) lowerSignal(sig gosxir.SignalInfo) (*nir.SignalDecl, error) {
	init, err := l.lowerRxExpr(sig.InitExpr)
	if err != nil {
		return nil, err
	}
	typ := nativeTypeFromHint(sig.TypeHint)
	if typ == "" {
		typ = l.inferExprType(init)
	}
	if typ == "" {
		return nil, fmt.Errorf("cannot infer native type from %q", sig.InitExpr)
	}
	return &nir.SignalDecl{
		Name: sig.Name,
		Type: typ,
		Init: init,
	}, nil
}

func (l *lowerer) lowerComputed(computed gosxir.ComputedInfo) (*nir.ComputedDecl, error) {
	body, err := l.lowerRxExpr(computed.BodyExpr)
	if err != nil {
		return nil, err
	}
	typ := l.inferExprType(body)
	if typ == "" {
		return nil, fmt.Errorf("cannot infer native type from %q", computed.BodyExpr)
	}
	return &nir.ComputedDecl{
		Name: computed.Name,
		Type: typ,
		Body: body,
	}, nil
}

func (l *lowerer) lowerView(id gosxir.NodeID) (nir.View, error) {
	if int(id) >= len(l.prog.Nodes) {
		return nil, fmt.Errorf("node %d out of range", id)
	}
	node := l.prog.NodeAt(id)
	switch node.Kind {
	case gosxir.NodeComponent:
		if target.IsScene3DTag(node.Tag) {
			return l.lowerScene3DView(node)
		}
		if isConditionalComponent(node.Tag) {
			return l.lowerConditionalView(node)
		}
		if isLoopComponent(node.Tag) {
			return l.lowerLoopView(node)
		}
		return l.lowerComponentRefView(node)
	case gosxir.NodeElement:
		return l.lowerElementView(node)
	case gosxir.NodeText, gosxir.NodeRawHTML:
		value := strings.TrimSpace(node.Text)
		if value == "" {
			return nil, nil
		}
		return &nir.Text{Value: value, Span: irSpan(node.Span)}, nil
	case gosxir.NodeExpr:
		expr, err := l.lowerRxExpr(node.Text)
		if err != nil {
			return nil, err
		}
		return &nir.ExprHole{Expr: *expr, Span: irSpan(node.Span)}, nil
	case gosxir.NodeFragment:
		element := &nir.Element{Tag: "view", Span: irSpan(node.Span)}
		for _, childID := range node.Children {
			child, err := l.lowerView(childID)
			if err != nil {
				return nil, err
			}
			if child != nil {
				element.Children = append(element.Children, child)
			}
		}
		return element, nil
	default:
		return nil, fmt.Errorf("unsupported GoSX node kind %d", node.Kind)
	}
}

func (l *lowerer) lowerViewWithScope(id gosxir.NodeID, scope *gosxir.ExprScope) (nir.View, error) {
	prev := l.scope
	l.scope = scope
	defer func() {
		l.scope = prev
	}()
	return l.lowerView(id)
}

func (l *lowerer) lowerElementView(node *gosxir.Node) (nir.View, error) {
	element := &nir.Element{
		Tag:  nativeTagForElement(node.Tag, node.Attrs),
		Span: irSpan(node.Span),
	}
	for _, attr := range node.Attrs {
		handler, ok, err := l.lowerHandlerAttr(attr)
		if err != nil {
			return nil, err
		}
		if ok {
			element.Handlers = append(element.Handlers, handler)
			continue
		}
		loweredAttr, ok, err := l.lowerElementAttr(attr)
		if err != nil {
			return nil, err
		}
		if ok {
			element.Attrs = append(element.Attrs, loweredAttr)
		}
	}
	for _, childID := range node.Children {
		child, err := l.lowerView(childID)
		if err != nil {
			return nil, err
		}
		if child != nil {
			element.Children = append(element.Children, child)
		}
	}
	return element, nil
}

func (l *lowerer) lowerComponentRefView(node *gosxir.Node) (nir.View, error) {
	if len(node.Children) > 0 {
		return nil, fmt.Errorf("component %s children require slot support", node.Tag)
	}
	ref := &nir.ComponentRef{
		Name: node.Tag,
		Span: irSpan(node.Span),
	}
	for _, attr := range node.Attrs {
		prop, ok, err := l.lowerComponentProp(attr)
		if err != nil {
			return nil, err
		}
		if ok {
			ref.Props = append(ref.Props, prop)
		}
	}
	return ref, nil
}

func (l *lowerer) lowerScene3DView(node *gosxir.Node) (nir.View, error) {
	element := &nir.Element{
		Tag:  scene3DNativeTag(node.Tag),
		Span: irSpan(node.Span),
	}
	for _, attr := range node.Attrs {
		loweredAttrs, ok, err := l.lowerScene3DAttr(node, attr)
		if err != nil {
			return nil, err
		}
		if ok {
			element.Attrs = append(element.Attrs, loweredAttrs...)
		}
	}
	if element.Tag != "scene3d" {
		for _, childID := range node.Children {
			child, err := l.lowerView(childID)
			if err != nil {
				return nil, err
			}
			if child != nil {
				element.Children = append(element.Children, child)
			}
		}
		return element, nil
	}
	element.Scene3D = &nir.Scene3DPayload{}
	for _, childID := range node.Children {
		item, ok, err := l.lowerScene3DItem(childID)
		if err != nil {
			return nil, err
		}
		if ok {
			element.Scene3D.Items = append(element.Scene3D.Items, item)
		}
	}
	return element, nil
}

func (l *lowerer) lowerScene3DItem(id gosxir.NodeID) (nir.Scene3DItem, bool, error) {
	if int(id) >= len(l.prog.Nodes) {
		return nir.Scene3DItem{}, false, fmt.Errorf("node %d out of range", id)
	}
	node := l.prog.NodeAt(id)
	if node.Kind == gosxir.NodeText || node.Kind == gosxir.NodeRawHTML {
		if strings.TrimSpace(node.Text) == "" {
			return nir.Scene3DItem{}, false, nil
		}
		return nir.Scene3DItem{}, false, fmt.Errorf("Scene3D text children are not supported by native lowering yet")
	}
	if node.Kind != gosxir.NodeComponent || !target.IsScene3DTag(node.Tag) {
		return nir.Scene3DItem{}, false, fmt.Errorf("Scene3D child <%s> is not a supported scene tag", node.Tag)
	}
	item := nir.Scene3DItem{
		Tag:  scene3DNativeTag(node.Tag),
		Span: irSpan(node.Span),
	}
	for _, attr := range node.Attrs {
		loweredAttrs, ok, err := l.lowerScene3DAttr(node, attr)
		if err != nil {
			return nir.Scene3DItem{}, false, err
		}
		if ok {
			item.Attrs = append(item.Attrs, loweredAttrs...)
		}
	}
	if len(node.Children) > 0 {
		if !scene3DHTMLTag(node.Tag) {
			return nir.Scene3DItem{}, false, fmt.Errorf("Scene3D item <%s> children are not supported by native lowering yet", node.Tag)
		}
		if !scene3DHasHTMLAttr(item.Attrs) {
			markup, err := l.lowerScene3DHTMLChildren(node.Children)
			if err != nil {
				return nir.Scene3DItem{}, false, err
			}
			if strings.TrimSpace(markup) != "" {
				item.Attrs = append(item.Attrs, nir.Attr{
					Name:  "html",
					Value: nir.RxExpr{Kind: "literal", Literal: &nir.Literal{Type: "string", Value: strings.TrimSpace(markup)}},
					Span:  irSpan(node.Span),
				})
			}
		}
	}
	if scene3DHTMLTag(node.Tag) && !scene3DHasHTMLAttr(item.Attrs) {
		return nir.Scene3DItem{}, false, fmt.Errorf("Scene3D <Html> requires literal html, markup, content, or static children")
	}
	return item, true, nil
}

func (l *lowerer) lowerScene3DHTMLChildren(children []gosxir.NodeID) (string, error) {
	var sb strings.Builder
	for _, childID := range children {
		markup, err := l.lowerScene3DHTMLNode(childID)
		if err != nil {
			return "", err
		}
		sb.WriteString(markup)
	}
	return sb.String(), nil
}

func (l *lowerer) lowerScene3DHTMLNode(id gosxir.NodeID) (string, error) {
	if int(id) >= len(l.prog.Nodes) {
		return "", fmt.Errorf("node %d out of range", id)
	}
	node := l.prog.NodeAt(id)
	switch node.Kind {
	case gosxir.NodeText:
		text := strings.TrimSpace(node.Text)
		if text == "" {
			return "", nil
		}
		return html.EscapeString(text), nil
	case gosxir.NodeRawHTML:
		return strings.TrimSpace(node.Text), nil
	case gosxir.NodeElement:
		var sb strings.Builder
		sb.WriteByte('<')
		sb.WriteString(node.Tag)
		for _, attr := range node.Attrs {
			markup, ok, err := lowerScene3DHTMLAttr(attr)
			if err != nil {
				return "", err
			}
			if ok {
				sb.WriteByte(' ')
				sb.WriteString(markup)
			}
		}
		sb.WriteByte('>')
		for _, childID := range node.Children {
			child, err := l.lowerScene3DHTMLNode(childID)
			if err != nil {
				return "", err
			}
			sb.WriteString(child)
		}
		sb.WriteString("</")
		sb.WriteString(node.Tag)
		sb.WriteByte('>')
		return sb.String(), nil
	case gosxir.NodeFragment:
		return l.lowerScene3DHTMLChildren(node.Children)
	case gosxir.NodeExpr:
		return "", fmt.Errorf("Scene3D <Html> children must be static for native lowering; use a literal html attribute for pre-rendered markup")
	default:
		return "", fmt.Errorf("Scene3D <Html> child kind %d is not supported by native lowering yet", node.Kind)
	}
}

func lowerScene3DHTMLAttr(attr gosxir.Attr) (string, bool, error) {
	if attr.IsEvent {
		return "", false, fmt.Errorf("Scene3D <Html> child attribute %q cannot be an event handler", attr.Name)
	}
	switch attr.Kind {
	case gosxir.AttrStatic:
		return attr.Name + "=" + strconv.Quote(html.EscapeString(attr.Value)), true, nil
	case gosxir.AttrBool:
		return attr.Name, true, nil
	case gosxir.AttrExpr, gosxir.AttrSpread:
		return "", false, fmt.Errorf("Scene3D <Html> child attributes must be static for native lowering")
	default:
		return "", false, nil
	}
}

func scene3DHasHTMLAttr(attrs []nir.Attr) bool {
	for _, attr := range attrs {
		switch attr.Name {
		case "html", "markup", "content":
			return true
		}
	}
	return false
}

func (l *lowerer) lowerScene3DAttr(node *gosxir.Node, attr gosxir.Attr) ([]nir.Attr, bool, error) {
	if attr.IsEvent {
		return nil, false, fmt.Errorf("Scene3D attribute %q cannot be an event handler", attr.Name)
	}
	switch attr.Kind {
	case gosxir.AttrStatic:
		return []nir.Attr{{
			Name:  attr.Name,
			Value: nir.RxExpr{Kind: "literal", Literal: &nir.Literal{Type: "string", Value: attr.Value}},
			Span:  irSpan(node.Span),
		}}, true, nil
	case gosxir.AttrBool:
		return []nir.Attr{{
			Name:  attr.Name,
			Value: nir.RxExpr{Kind: "literal", Literal: &nir.Literal{Type: "bool", Value: "true"}},
			Span:  irSpan(node.Span),
		}}, true, nil
	case gosxir.AttrExpr:
		expr, err := l.lowerRxExpr(attr.Expr)
		if err != nil {
			return nil, false, err
		}
		return []nir.Attr{{Name: attr.Name, Value: *expr, Span: irSpan(node.Span)}}, true, nil
	case gosxir.AttrSpread:
		attrs, err := l.lowerScene3DSpreadAttrs(node, attr)
		if err != nil {
			return nil, false, err
		}
		return attrs, len(attrs) > 0, nil
	default:
		return nil, false, nil
	}
}

type scene3DSpreadAttrSpec struct {
	Name     string
	Type     string
	Fallback string
}

func (l *lowerer) lowerScene3DSpreadAttrs(node *gosxir.Node, attr gosxir.Attr) ([]nir.Attr, error) {
	expr, err := l.lowerRxExpr(attr.Expr)
	if err != nil {
		return nil, err
	}
	if expr.Kind != "ref" || expr.Ref == "" {
		return nil, fmt.Errorf("Scene3D spread props must be a props map reference")
	}
	if typ := l.inferExprType(expr); typ != "Map<String, Any>" {
		return nil, fmt.Errorf("Scene3D spread props must be map[string]any, got %s", typ)
	}
	specs := scene3DSpreadAttrSpecs(node.Tag)
	if len(specs) == 0 {
		return nil, fmt.Errorf("Scene3D spread props are not supported for <%s>", node.Tag)
	}
	out := make([]nir.Attr, 0, len(specs))
	for _, spec := range specs {
		out = append(out, nir.Attr{
			Name:  spec.Name,
			Value: scene3DSpreadAttrExpr(*expr, spec),
			Span:  irSpan(node.Span),
		})
	}
	return out, nil
}

func scene3DSpreadAttrExpr(source nir.RxExpr, spec scene3DSpreadAttrSpec) nir.RxExpr {
	return nir.RxExpr{
		Kind: "call",
		Call: &nir.Call{
			Callee: "gsxScene3DSpread" + upperFirst(spec.Type),
			Args: []nir.RxExpr{
				source,
				{Kind: "literal", Literal: &nir.Literal{Type: "string", Value: spec.Name}},
				{Kind: "literal", Literal: &nir.Literal{Type: spec.Type, Value: spec.Fallback}},
			},
		},
	}
}

func scene3DSpreadAttrSpecs(tag string) []scene3DSpreadAttrSpec {
	commonNode := []scene3DSpreadAttrSpec{
		{Name: "id", Type: "string", Fallback: ""},
		{Name: "kind", Type: "string", Fallback: ""},
		{Name: "color", Type: "string", Fallback: "#8de1ff"},
		{Name: "x", Type: "float", Fallback: "0.0"},
		{Name: "y", Type: "float", Fallback: "0.0"},
		{Name: "z", Type: "float", Fallback: "0.0"},
		{Name: "width", Type: "float", Fallback: "1.0"},
		{Name: "height", Type: "float", Fallback: "1.0"},
		{Name: "depth", Type: "float", Fallback: "1.0"},
		{Name: "count", Type: "int", Fallback: "0"},
		{Name: "size", Type: "float", Fallback: "0.0"},
	}
	switch strings.ToLower(strings.TrimSpace(tag)) {
	case "scene3d":
		return []scene3DSpreadAttrSpec{
			{Name: "width", Type: "float", Fallback: "640.0"},
			{Name: "height", Type: "float", Fallback: "360.0"},
			{Name: "background", Type: "string", Fallback: "#101820"},
		}
	case "camera":
		return []scene3DSpreadAttrSpec{
			{Name: "kind", Type: "string", Fallback: "perspective"},
			{Name: "x", Type: "float", Fallback: "0.0"},
			{Name: "y", Type: "float", Fallback: "0.0"},
			{Name: "z", Type: "float", Fallback: "0.0"},
			{Name: "fov", Type: "float", Fallback: "0.0"},
			{Name: "near", Type: "float", Fallback: "0.0"},
			{Name: "far", Type: "float", Fallback: "0.0"},
		}
	case "environment":
		return []scene3DSpreadAttrSpec{
			{Name: "background", Type: "string", Fallback: ""},
			{Name: "ambientColor", Type: "string", Fallback: ""},
			{Name: "ambientIntensity", Type: "float", Fallback: "0.0"},
		}
	case "mesh", "model", "points", "instancedmesh", "computeparticles":
		return commonNode
	case "directionallight", "pointlight", "ambientlight", "spotlight", "hemispherelight":
		return []scene3DSpreadAttrSpec{
			{Name: "id", Type: "string", Fallback: ""},
			{Name: "color", Type: "string", Fallback: "#ffffff"},
			{Name: "intensity", Type: "float", Fallback: "1.0"},
			{Name: "x", Type: "float", Fallback: "0.0"},
			{Name: "y", Type: "float", Fallback: "0.0"},
			{Name: "z", Type: "float", Fallback: "0.0"},
		}
	case "html":
		return []scene3DSpreadAttrSpec{
			{Name: "id", Type: "string", Fallback: ""},
			{Name: "html", Type: "string", Fallback: ""},
			{Name: "className", Type: "string", Fallback: ""},
			{Name: "x", Type: "float", Fallback: "0.0"},
			{Name: "y", Type: "float", Fallback: "0.0"},
			{Name: "z", Type: "float", Fallback: "0.0"},
			{Name: "width", Type: "float", Fallback: "1.8"},
			{Name: "height", Type: "float", Fallback: "0.72"},
			{Name: "opacity", Type: "float", Fallback: "1.0"},
			{Name: "offsetX", Type: "float", Fallback: "0.0"},
			{Name: "offsetY", Type: "float", Fallback: "0.0"},
			{Name: "pointerEvents", Type: "string", Fallback: "none"},
			{Name: "occlude", Type: "bool", Fallback: "false"},
		}
	case "postfx.bloom":
		return []scene3DSpreadAttrSpec{
			{Name: "threshold", Type: "float", Fallback: "0.0"},
			{Name: "intensity", Type: "float", Fallback: "0.0"},
			{Name: "radius", Type: "float", Fallback: "0.0"},
		}
	case "postfx.vignette":
		return []scene3DSpreadAttrSpec{
			{Name: "intensity", Type: "float", Fallback: "0.0"},
			{Name: "radius", Type: "float", Fallback: "0.0"},
		}
	case "postfx.colorgrading":
		return []scene3DSpreadAttrSpec{
			{Name: "saturation", Type: "float", Fallback: "0.0"},
			{Name: "contrast", Type: "float", Fallback: "0.0"},
			{Name: "exposure", Type: "float", Fallback: "0.0"},
		}
	case "postfx.tonemap":
		return []scene3DSpreadAttrSpec{
			{Name: "mode", Type: "string", Fallback: ""},
			{Name: "exposure", Type: "float", Fallback: "0.0"},
		}
	default:
		return nil
	}
}

func (l *lowerer) lowerComponentProp(attr gosxir.Attr) (nir.Attr, bool, error) {
	if attr.IsEvent {
		return nir.Attr{}, false, fmt.Errorf("component event prop %q requires callback prop support", attr.Name)
	}
	switch attr.Kind {
	case gosxir.AttrStatic:
		return nir.Attr{
			Name:  attr.Name,
			Value: nir.RxExpr{Kind: "literal", Literal: &nir.Literal{Type: "string", Value: attr.Value}},
		}, true, nil
	case gosxir.AttrBool:
		return nir.Attr{
			Name:  attr.Name,
			Value: nir.RxExpr{Kind: "literal", Literal: &nir.Literal{Type: "bool", Value: "true"}},
		}, true, nil
	case gosxir.AttrExpr:
		expr, err := l.lowerRxExpr(attr.Expr)
		if err != nil {
			return nir.Attr{}, false, err
		}
		return nir.Attr{Name: attr.Name, Value: *expr}, true, nil
	case gosxir.AttrSpread:
		return nir.Attr{}, false, fmt.Errorf("component prop spread is not supported")
	default:
		return nir.Attr{}, false, nil
	}
}

func (l *lowerer) lowerConditionalView(node *gosxir.Node) (nir.View, error) {
	conditionSource, ok := viewAttrSource(node.Attrs, false, "when", "if", "cond", "test")
	if !ok || strings.TrimSpace(conditionSource) == "" {
		return nil, fmt.Errorf("%s requires a when/if/cond/test attribute", node.Tag)
	}
	condition, err := l.lowerRxExpr(conditionSource)
	if err != nil {
		return nil, fmt.Errorf("condition %q: %w", conditionSource, err)
	}
	conditional := &nir.Conditional{
		Condition: *condition,
		Span:      irSpan(node.Span),
	}
	for _, childID := range node.Children {
		child, err := l.lowerView(childID)
		if err != nil {
			return nil, err
		}
		if child != nil {
			conditional.Then = append(conditional.Then, child)
		}
	}
	if fallbackSource, ok := viewAttrSource(node.Attrs, true, "fallback", "else"); ok && strings.TrimSpace(fallbackSource) != "" {
		fallback, err := l.lowerRxExpr(fallbackSource)
		if err != nil {
			return nil, fmt.Errorf("fallback %q: %w", fallbackSource, err)
		}
		conditional.Else = append(conditional.Else, &nir.ExprHole{Expr: *fallback, Span: irSpan(node.Span)})
	}
	return conditional, nil
}

func (l *lowerer) lowerLoopView(node *gosxir.Node) (nir.View, error) {
	itemsSource, ok := viewAttrSource(node.Attrs, false, "of", "each", "items")
	if !ok || strings.TrimSpace(itemsSource) == "" {
		return nil, fmt.Errorf("%s requires an of/each/items attribute", node.Tag)
	}
	items, err := l.lowerRxExpr(itemsSource)
	if err != nil {
		return nil, fmt.Errorf("items %q: %w", itemsSource, err)
	}

	itemName := staticAttrValue(node.Attrs, "as", "item")
	if itemName == "" {
		itemName = "item"
	}
	indexName := staticAttrValue(node.Attrs, "index")
	loop := &nir.Loop{
		Items:     *items,
		ItemName:  itemName,
		IndexName: indexName,
		Span:      irSpan(node.Span),
	}

	childScope := cloneExprScope(l.scope)
	childScope.Props[itemName] = true
	childScope.Props["_item"] = true
	if indexName != "" {
		childScope.Props[indexName] = true
	}
	childScope.Props["_index"] = true
	for _, childID := range node.Children {
		child, err := l.lowerViewWithScope(childID, childScope)
		if err != nil {
			return nil, err
		}
		if child != nil {
			loop.Body = append(loop.Body, child)
		}
	}
	return loop, nil
}

func viewAttrSource(attrs []gosxir.Attr, quoteStatic bool, names ...string) (string, bool) {
	for _, attr := range attrs {
		if !stringIn(attr.Name, names) {
			continue
		}
		switch attr.Kind {
		case gosxir.AttrExpr:
			return attr.Expr, true
		case gosxir.AttrStatic:
			if !quoteStatic {
				return attr.Value, true
			}
			return strconv.Quote(attr.Value), true
		case gosxir.AttrBool:
			return "true", true
		default:
			return "", false
		}
	}
	return "", false
}

func staticAttrValue(attrs []gosxir.Attr, names ...string) string {
	for _, attr := range attrs {
		if attr.Kind == gosxir.AttrStatic && stringIn(attr.Name, names) {
			return attr.Value
		}
	}
	return ""
}

func cloneExprScope(scope *gosxir.ExprScope) *gosxir.ExprScope {
	if scope == nil {
		return &gosxir.ExprScope{
			Signals:       make(map[string]bool),
			SignalAliases: make(map[string]string),
			Props:         make(map[string]bool),
			Handlers:      make(map[string]bool),
			EventFields:   make(map[string]bool),
		}
	}
	next := &gosxir.ExprScope{
		Signals:       make(map[string]bool, len(scope.Signals)),
		SignalAliases: make(map[string]string, len(scope.SignalAliases)),
		Props:         make(map[string]bool, len(scope.Props)),
		Handlers:      make(map[string]bool, len(scope.Handlers)),
		EventFields:   make(map[string]bool, len(scope.EventFields)),
	}
	for key, value := range scope.Signals {
		next.Signals[key] = value
	}
	for key, value := range scope.SignalAliases {
		next.SignalAliases[key] = value
	}
	for key, value := range scope.Props {
		next.Props[key] = value
	}
	for key, value := range scope.Handlers {
		next.Handlers[key] = value
	}
	for key, value := range scope.EventFields {
		next.EventFields[key] = value
	}
	return next
}

func stringIn(value string, choices []string) bool {
	for _, choice := range choices {
		if value == choice {
			return true
		}
	}
	return false
}

func (l *lowerer) lowerHandlerAttr(attr gosxir.Attr) (nir.Handler, bool, error) {
	if attr.IsEvent {
		body, err := l.lowerHandlerBody(attr.Expr)
		if err != nil {
			return nir.Handler{}, false, err
		}
		return nir.Handler{Event: nativeEvent(attr.Name), Body: body}, true, nil
	}
	if attr.Kind == gosxir.AttrStatic && strings.HasPrefix(attr.Name, "data-on-") {
		body, err := l.lowerInlineHandler(attr.Value)
		if err != nil {
			return nir.Handler{}, false, err
		}
		return nir.Handler{Event: nativeEvent(attr.Name), Body: body}, true, nil
	}
	return nir.Handler{}, false, nil
}

func (l *lowerer) lowerElementAttr(attr gosxir.Attr) (nir.Attr, bool, error) {
	if !nativeAttr(attr.Name) {
		return nir.Attr{}, false, nil
	}
	switch attr.Kind {
	case gosxir.AttrStatic:
		return nir.Attr{
			Name:  attr.Name,
			Value: nir.RxExpr{Kind: "literal", Literal: &nir.Literal{Type: "string", Value: attr.Value}},
		}, true, nil
	case gosxir.AttrBool:
		return nir.Attr{
			Name:  attr.Name,
			Value: nir.RxExpr{Kind: "literal", Literal: &nir.Literal{Type: "bool", Value: "true"}},
		}, true, nil
	case gosxir.AttrExpr:
		expr, err := l.lowerRxExpr(attr.Expr)
		if err != nil {
			return nir.Attr{}, false, err
		}
		return nir.Attr{Name: attr.Name, Value: *expr}, true, nil
	default:
		return nir.Attr{}, false, nil
	}
}

func (l *lowerer) lowerHandlerBody(expr string) (nir.RxBlock, error) {
	if handler, ok := l.handlers[expr]; ok {
		var block nir.RxBlock
		for _, stmtSource := range handler.Statements {
			stmt, err := l.lowerRxStmt(stmtSource)
			if err != nil {
				return nir.RxBlock{}, fmt.Errorf("handler %s statement %q: %w", handler.Name, stmtSource, err)
			}
			block.Stmts = append(block.Stmts, stmt)
		}
		return block, nil
	}
	return l.lowerInlineHandler(expr)
}

func (l *lowerer) lowerInlineHandler(source string) (nir.RxBlock, error) {
	stmt, err := l.lowerRxStmt(source)
	if err != nil {
		return nir.RxBlock{}, err
	}
	return nir.RxBlock{Stmts: []nir.RxStmt{stmt}}, nil
}

func (l *lowerer) lowerRxStmt(source string) (nir.RxStmt, error) {
	exprs, rootID, err := gosxir.ParseExpr(source, l.scope)
	if err != nil {
		return nir.RxStmt{}, err
	}
	root := exprs[rootID]
	if root.Op == islandprogram.OpSignalSet {
		if len(root.Operands) != 1 {
			return nir.RxStmt{}, fmt.Errorf("signal set expects one operand")
		}
		value, err := l.rxExprFromProgram(exprs, root.Operands[0])
		if err != nil {
			return nir.RxStmt{}, err
		}
		return nir.RxStmt{Kind: "signal_set", Target: root.Value, Value: value}, nil
	}
	expr, err := l.rxExprFromProgram(exprs, rootID)
	if err != nil {
		return nir.RxStmt{}, err
	}
	return nir.RxStmt{Kind: "expr", Expr: expr}, nil
}

func (l *lowerer) lowerRxExpr(source string) (*nir.RxExpr, error) {
	exprs, rootID, err := gosxir.ParseExpr(source, l.scope)
	if err != nil {
		return nil, err
	}
	return l.rxExprFromProgram(exprs, rootID)
}

func (l *lowerer) rxExprFromProgram(exprs []islandprogram.Expr, id islandprogram.ExprID) (*nir.RxExpr, error) {
	if int(id) >= len(exprs) {
		return nil, fmt.Errorf("expr %d out of range", id)
	}
	expr := exprs[id]
	if ref, ok := l.refFromProgram(exprs, id); ok {
		return &nir.RxExpr{Kind: "ref", Ref: ref}, nil
	}
	switch expr.Op {
	case islandprogram.OpEventGet:
		return &nir.RxExpr{Kind: "ref", Ref: "event." + expr.Value}, nil
	case islandprogram.OpLitString:
		return &nir.RxExpr{Kind: "literal", Literal: &nir.Literal{Type: "string", Value: expr.Value}}, nil
	case islandprogram.OpLitInt:
		return &nir.RxExpr{Kind: "literal", Literal: &nir.Literal{Type: "int", Value: expr.Value}}, nil
	case islandprogram.OpLitFloat:
		return &nir.RxExpr{Kind: "literal", Literal: &nir.Literal{Type: "float", Value: expr.Value}}, nil
	case islandprogram.OpLitBool:
		return &nir.RxExpr{Kind: "literal", Literal: &nir.Literal{Type: "bool", Value: expr.Value}}, nil
	case islandprogram.OpAdd, islandprogram.OpSub, islandprogram.OpMul, islandprogram.OpDiv, islandprogram.OpMod,
		islandprogram.OpEq, islandprogram.OpNeq, islandprogram.OpLt, islandprogram.OpGt, islandprogram.OpLte, islandprogram.OpGte,
		islandprogram.OpAnd, islandprogram.OpOr, islandprogram.OpConcat:
		if len(expr.Operands) != 2 {
			return nil, fmt.Errorf("binary op %d expects two operands", expr.Op)
		}
		left, err := l.rxExprFromProgram(exprs, expr.Operands[0])
		if err != nil {
			return nil, err
		}
		right, err := l.rxExprFromProgram(exprs, expr.Operands[1])
		if err != nil {
			return nil, err
		}
		return &nir.RxExpr{
			Kind:  "binop",
			BinOp: &nir.BinOp{Op: binOpText(expr.Op), Left: *left, Right: *right},
		}, nil
	case islandprogram.OpCond:
		if len(expr.Operands) != 3 {
			return nil, fmt.Errorf("conditional expression expects three operands")
		}
		condition, err := l.rxExprFromProgram(exprs, expr.Operands[0])
		if err != nil {
			return nil, err
		}
		thenExpr, err := l.rxExprFromProgram(exprs, expr.Operands[1])
		if err != nil {
			return nil, err
		}
		elseExpr, err := l.rxExprFromProgram(exprs, expr.Operands[2])
		if err != nil {
			return nil, err
		}
		return &nir.RxExpr{
			Kind: "cond",
			Cond: &nir.Cond{Condition: *condition, Then: *thenExpr, Else: *elseExpr},
		}, nil
	case islandprogram.OpCall:
		call := &nir.Call{Callee: expr.Value}
		for _, operand := range expr.Operands {
			arg, err := l.rxExprFromProgram(exprs, operand)
			if err != nil {
				return nil, err
			}
			call.Args = append(call.Args, *arg)
		}
		return &nir.RxExpr{Kind: "call", Call: call}, nil
	case islandprogram.OpLen, islandprogram.OpToUpper, islandprogram.OpToLower, islandprogram.OpTrim,
		islandprogram.OpSplit, islandprogram.OpJoin, islandprogram.OpReplace, islandprogram.OpSubstring,
		islandprogram.OpStartsWith, islandprogram.OpEndsWith, islandprogram.OpContains,
		islandprogram.OpToString, islandprogram.OpToInt, islandprogram.OpToFloat:
		return l.callExprFromProgram(exprs, expr)
	default:
		return nil, fmt.Errorf("unsupported GoSX expression opcode %d", expr.Op)
	}
}

func (l *lowerer) callExprFromProgram(exprs []islandprogram.Expr, expr islandprogram.Expr) (*nir.RxExpr, error) {
	call := &nir.Call{Callee: portableCallName(expr.Op)}
	if call.Callee == "" {
		return nil, fmt.Errorf("unsupported GoSX call opcode %d", expr.Op)
	}
	for _, operand := range expr.Operands {
		arg, err := l.rxExprFromProgram(exprs, operand)
		if err != nil {
			return nil, err
		}
		call.Args = append(call.Args, *arg)
	}
	if expr.Value != "" || ((expr.Op == islandprogram.OpSplit || expr.Op == islandprogram.OpJoin) && len(expr.Operands) == 1) {
		call.Args = append(call.Args, nir.RxExpr{
			Kind:    "literal",
			Literal: &nir.Literal{Type: "string", Value: expr.Value},
		})
	}
	return &nir.RxExpr{Kind: "call", Call: call}, nil
}

func (l *lowerer) refFromProgram(exprs []islandprogram.Expr, id islandprogram.ExprID) (string, bool) {
	if int(id) >= len(exprs) {
		return "", false
	}
	expr := exprs[id]
	switch expr.Op {
	case islandprogram.OpPropGet:
		return l.canonicalRef(expr.Value), true
	case islandprogram.OpSignalGet:
		return expr.Value, true
	case islandprogram.OpIndex:
		if len(expr.Operands) != 2 {
			return "", false
		}
		base, ok := l.refFromProgram(exprs, expr.Operands[0])
		if !ok {
			return "", false
		}
		field := exprs[expr.Operands[1]]
		if field.Op != islandprogram.OpLitString || field.Value == "" {
			return "", false
		}
		return l.canonicalRef(base + "." + field.Value), true
	default:
		return "", false
	}
}

func (l *lowerer) inferExprType(expr *nir.RxExpr) string {
	if expr == nil {
		return ""
	}
	if expr.Kind == "literal" && expr.Literal != nil {
		return nativeTypeFromHint(expr.Literal.Type)
	}
	if expr.Kind == "ref" {
		if typ := l.propTypes[expr.Ref]; typ != "" {
			return typ
		}
		if typ := l.signals[expr.Ref]; typ != "" {
			return typ
		}
	}
	if expr.Kind == "binop" && expr.BinOp != nil {
		left := l.inferExprType(&expr.BinOp.Left)
		right := l.inferExprType(&expr.BinOp.Right)
		switch expr.BinOp.Op {
		case "==", "!=", "<", ">", "<=", ">=", "&&", "||":
			return "Bool"
		case "+":
			if left == "String" || right == "String" {
				return "String"
			}
			return numericType(left, right)
		case "-", "*", "/", "%":
			return numericType(left, right)
		}
	}
	if expr.Kind == "cond" && expr.Cond != nil {
		thenType := l.inferExprType(&expr.Cond.Then)
		elseType := l.inferExprType(&expr.Cond.Else)
		if thenType == elseType {
			return thenType
		}
		return numericType(thenType, elseType)
	}
	if expr.Kind == "call" && expr.Call != nil {
		switch expr.Call.Callee {
		case "contains", "startsWith", "endsWith":
			return "Bool"
		case "len", "toInt":
			return "Int"
		case "toFloat":
			return "Double"
		case "split":
			return "[String]"
		case "upper", "lower", "trim", "replace", "substring", "join", "toString":
			return "String"
		}
	}
	return ""
}

func (l *lowerer) canonicalRef(ref string) string {
	ref = strings.TrimSpace(ref)
	if got := l.propRefs[ref]; got != "" {
		return got
	}
	prefix := l.propsName + "."
	if strings.HasPrefix(ref, prefix) {
		raw := strings.TrimPrefix(ref, prefix)
		if idx := strings.IndexByte(raw, '.'); idx >= 0 {
			head := raw[:idx]
			tail := raw[idx:]
			if got := l.propRefs[prefix+head]; got != "" {
				return got + tail
			}
		}
	}
	return ref
}

func (l *lowerer) exprScope(compScope *gosxir.ComponentScope, propsName string) *gosxir.ExprScope {
	scope := &gosxir.ExprScope{
		Signals:       make(map[string]bool),
		SignalAliases: make(map[string]string),
		Props:         make(map[string]bool),
		Handlers:      make(map[string]bool),
		EventFields: map[string]bool{
			"value":         true,
			"checked":       true,
			"key":           true,
			"selectedIndex": true,
		},
	}
	if propsName != "" {
		scope.Props[propsName] = true
	}
	if compScope == nil {
		return scope
	}
	for _, sig := range compScope.Signals {
		scope.Signals[sig.Name] = true
		if sig.Local != "" {
			scope.SignalAliases[sig.Local] = sig.Name
		}
	}
	for _, computed := range compScope.Computeds {
		scope.Signals[computed.Name] = true
	}
	for _, handler := range compScope.Handlers {
		scope.Handlers[handler.Name] = true
	}
	return scope
}

func collectStructs(root *gotreesitter.Node, src []byte, lang *gotreesitter.Language) map[string]structInfo {
	out := make(map[string]structInfo)
	var walk func(*gotreesitter.Node)
	walk = func(n *gotreesitter.Node) {
		if n == nil {
			return
		}
		if nodeType(n, lang) == "type_spec" {
			nameNode := fieldChild(n, lang, "name")
			typeNode := fieldChild(n, lang, "type")
			if nameNode != nil && typeNode != nil && nodeType(typeNode, lang) == "struct_type" {
				out[nodeText(src, nameNode)] = collectStructInfo(typeNode, src, lang)
			}
			return
		}
		for _, child := range namedChildren(n) {
			walk(child)
		}
	}
	walk(root)
	return out
}

func collectStructInfo(n *gotreesitter.Node, src []byte, lang *gotreesitter.Language) structInfo {
	info := structInfo{
		FieldNames: make(map[string]string),
		FieldTypes: make(map[string]string),
	}
	for _, field := range descendantsOfType(n, lang, "field_declaration") {
		typeNode := fieldChild(field, lang, "type")
		if typeNode == nil {
			continue
		}
		typ := nativeTypeFromGoType(nodeText(src, typeNode))
		for _, nameNode := range fieldChildren(field, lang, "name") {
			raw := nodeText(src, nameNode)
			if raw == "" {
				continue
			}
			canonical := canonicalFieldName(raw, fieldTag(src, field, lang))
			info.FieldNames[raw] = canonical
			info.FieldTypes[raw] = typ
			info.Fields = append(info.Fields, nir.PropField{Name: canonical, Type: typ})
		}
	}
	return info
}

func collectComponentSignatures(root *gotreesitter.Node, src []byte, lang *gotreesitter.Language) map[string]componentSignature {
	out := make(map[string]componentSignature)
	for _, fn := range descendantsOfType(root, lang, "function_declaration") {
		name := nodeText(src, fieldChild(fn, lang, "name"))
		if name == "" {
			continue
		}
		params := fieldChild(fn, lang, "parameters")
		if params == nil {
			continue
		}
		for _, param := range namedChildren(params) {
			if nodeType(param, lang) != "parameter_declaration" {
				continue
			}
			typeNode := fieldChild(param, lang, "type")
			nameNode := fieldChild(param, lang, "name")
			if typeNode == nil || nameNode == nil {
				continue
			}
			out[name] = componentSignature{
				PropsName: nodeText(src, nameNode),
				PropsType: nodeText(src, typeNode),
			}
			break
		}
	}
	return out
}

func handlersByName(scope *gosxir.ComponentScope) map[string]gosxir.HandlerInfo {
	out := make(map[string]gosxir.HandlerInfo)
	if scope == nil {
		return out
	}
	for _, handler := range scope.Handlers {
		out[handler.Name] = handler
	}
	return out
}

func nativeTag(tag string) string {
	switch tag {
	case "div", "section", "main", "article", "view":
		return "vstack"
	case "span", "p", "label", "text":
		return "text"
	case "textarea":
		return "textarea"
	case "select":
		return "select"
	case "option":
		return "option"
	case "input":
		return "textinput"
	default:
		return lowerFirst(tag)
	}
}

func nativeTagForElement(tag string, attrs []gosxir.Attr) string {
	if tag == "input" && staticAttrValue(attrs, "type") == "checkbox" {
		return "checkbox"
	}
	return nativeTag(tag)
}

func scene3DNativeTag(tag string) string {
	if tag == "Scene3D" {
		return "scene3d"
	}
	return lowerFirst(tag)
}

func scene3DHTMLTag(tag string) bool {
	return strings.EqualFold(tag, "html")
}

func isConditionalComponent(tag string) bool {
	switch tag {
	case "If", "Show", "When":
		return true
	default:
		return false
	}
}

func isLoopComponent(tag string) bool {
	switch tag {
	case "Each", "For":
		return true
	default:
		return false
	}
}

func nativeAttr(name string) bool {
	switch name {
	case "checked", "placeholder", "selectedIndex", "type", "value":
		return true
	default:
		return false
	}
}

func nativeEvent(name string) string {
	name = strings.TrimSpace(name)
	if strings.HasPrefix(name, "data-on-") {
		name = strings.TrimPrefix(name, "data-on-")
	}
	if strings.HasPrefix(name, "on") && len(name) > 2 {
		name = lowerFirst(name[2:])
	}
	switch strings.ToLower(name) {
	case "click", "tap":
		return "tap"
	default:
		return lowerFirst(name)
	}
}

func nativeTypeFromHint(hint string) string {
	switch hint {
	case "int":
		return "Int"
	case "float":
		return "Double"
	case "string":
		return "String"
	case "bool":
		return "Bool"
	default:
		return ""
	}
}

func nativeTypeFromGoType(typ string) string {
	typ = strings.TrimSpace(strings.TrimPrefix(typ, "*"))
	if strings.HasPrefix(typ, "[]") {
		elem := nativeTypeFromGoType(strings.TrimSpace(strings.TrimPrefix(typ, "[]")))
		if elem == "" {
			return typ
		}
		return "[" + elem + "]"
	}
	if strings.HasPrefix(typ, "map[") {
		key, value, ok := parseGoMapType(typ)
		if ok && key == "string" && (value == "any" || value == "interface{}") {
			return "Map<String, Any>"
		}
		return typ
	}
	switch typ {
	case "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64", "uintptr":
		return "Int"
	case "float32":
		return "Float"
	case "float64":
		return "Double"
	case "string":
		return "String"
	case "bool":
		return "Bool"
	default:
		return typ
	}
}

func parseGoMapType(typ string) (string, string, bool) {
	typ = strings.TrimSpace(typ)
	if !strings.HasPrefix(typ, "map[") {
		return "", "", false
	}
	end := strings.IndexByte(typ, ']')
	if end < 0 {
		return "", "", false
	}
	key := strings.TrimSpace(strings.TrimPrefix(typ[:end], "map["))
	value := strings.TrimSpace(typ[end+1:])
	if key == "" || value == "" {
		return "", "", false
	}
	return key, value, true
}

func numericType(left, right string) string {
	switch {
	case left == "Double" || right == "Double":
		return "Double"
	case left == "Float" || right == "Float":
		return "Float"
	case left == "Int" || right == "Int":
		return "Int"
	default:
		return ""
	}
}

func binOpText(op islandprogram.OpCode) string {
	switch op {
	case islandprogram.OpAdd:
		return "+"
	case islandprogram.OpSub:
		return "-"
	case islandprogram.OpMul:
		return "*"
	case islandprogram.OpDiv:
		return "/"
	case islandprogram.OpMod:
		return "%"
	case islandprogram.OpEq:
		return "=="
	case islandprogram.OpNeq:
		return "!="
	case islandprogram.OpLt:
		return "<"
	case islandprogram.OpGt:
		return ">"
	case islandprogram.OpLte:
		return "<="
	case islandprogram.OpGte:
		return ">="
	case islandprogram.OpAnd:
		return "&&"
	case islandprogram.OpOr:
		return "||"
	case islandprogram.OpConcat:
		return "+"
	default:
		return ""
	}
}

func portableCallName(op islandprogram.OpCode) string {
	switch op {
	case islandprogram.OpLen:
		return "len"
	case islandprogram.OpToUpper:
		return "upper"
	case islandprogram.OpToLower:
		return "lower"
	case islandprogram.OpTrim:
		return "trim"
	case islandprogram.OpSplit:
		return "split"
	case islandprogram.OpJoin:
		return "join"
	case islandprogram.OpReplace:
		return "replace"
	case islandprogram.OpSubstring:
		return "substring"
	case islandprogram.OpStartsWith:
		return "startsWith"
	case islandprogram.OpEndsWith:
		return "endsWith"
	case islandprogram.OpContains:
		return "contains"
	case islandprogram.OpToString:
		return "toString"
	case islandprogram.OpToInt:
		return "toInt"
	case islandprogram.OpToFloat:
		return "toFloat"
	default:
		return ""
	}
}

func canonicalFieldName(name, tag string) string {
	if jsonName := jsonTagName(tag); jsonName != "" && jsonName != "-" {
		return jsonName
	}
	return lowerFirst(name)
}

func jsonTagName(tag string) string {
	if tag == "" {
		return ""
	}
	raw, err := strconv.Unquote(tag)
	if err != nil {
		return ""
	}
	for _, part := range strings.Split(raw, " ") {
		if !strings.HasPrefix(part, "json:") {
			continue
		}
		value := strings.TrimPrefix(part, "json:")
		value, err := strconv.Unquote(value)
		if err != nil {
			return ""
		}
		name, _, _ := strings.Cut(value, ",")
		return name
	}
	return ""
}

func fieldTag(src []byte, n *gotreesitter.Node, lang *gotreesitter.Language) string {
	tag := fieldChild(n, lang, "tag")
	if tag == nil {
		return ""
	}
	return nodeText(src, tag)
}

func lowerFirst(s string) string {
	if s == "" {
		return ""
	}
	runes := []rune(s)
	runes[0] = unicode.ToLower(runes[0])
	return string(runes)
}

func upperFirst(s string) string {
	if s == "" {
		return ""
	}
	runes := []rune(s)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

func irSpan(s gosxir.Span) nir.Span {
	return nir.Span{
		StartLine: s.StartLine,
		StartCol:  s.StartCol,
		EndLine:   s.EndLine,
		EndCol:    s.EndCol,
	}
}

func nodeType(n *gotreesitter.Node, lang *gotreesitter.Language) string {
	if n == nil {
		return ""
	}
	return n.Type(lang)
}

func nodeText(src []byte, n *gotreesitter.Node) string {
	if n == nil {
		return ""
	}
	return n.Text(src)
}

func namedChildren(n *gotreesitter.Node) []*gotreesitter.Node {
	if n == nil {
		return nil
	}
	out := make([]*gotreesitter.Node, 0, n.NamedChildCount())
	for i := 0; i < n.ChildCount(); i++ {
		child := n.Child(i)
		if child != nil && child.IsNamed() {
			out = append(out, child)
		}
	}
	return out
}

func descendantsOfType(n *gotreesitter.Node, lang *gotreesitter.Language, typ string) []*gotreesitter.Node {
	var out []*gotreesitter.Node
	var walk func(*gotreesitter.Node)
	walk = func(cur *gotreesitter.Node) {
		if cur == nil {
			return
		}
		if nodeType(cur, lang) == typ {
			out = append(out, cur)
		}
		for _, child := range namedChildren(cur) {
			walk(child)
		}
	}
	walk(n)
	return out
}

func fieldChild(n *gotreesitter.Node, lang *gotreesitter.Language, field string) *gotreesitter.Node {
	if n == nil {
		return nil
	}
	return n.ChildByFieldName(field, lang)
}

func fieldChildren(n *gotreesitter.Node, lang *gotreesitter.Language, field string) []*gotreesitter.Node {
	if n == nil {
		return nil
	}
	var out []*gotreesitter.Node
	for i := 0; i < n.ChildCount(); i++ {
		if n.FieldNameForChild(i, lang) == field {
			out = append(out, n.Child(i))
		}
	}
	return out
}
