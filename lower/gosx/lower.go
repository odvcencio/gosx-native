package gosx

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"

	gosxlang "github.com/odvcencio/gosx"
	gosxir "github.com/odvcencio/gosx/ir"
	islandprogram "github.com/odvcencio/gosx/island/program"
	"github.com/odvcencio/gosx/nir"
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
		if isConditionalComponent(node.Tag) {
			return l.lowerConditionalView(node)
		}
		return l.lowerElementView(node)
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

func (l *lowerer) lowerElementView(node *gosxir.Node) (nir.View, error) {
	element := &nir.Element{
		Tag:  nativeTag(node.Tag),
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
		islandprogram.OpAnd, islandprogram.OpOr:
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
	default:
		return nil, fmt.Errorf("unsupported GoSX expression opcode %d", expr.Op)
	}
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
	case "input":
		return "textinput"
	default:
		return lowerFirst(tag)
	}
}

func isConditionalComponent(tag string) bool {
	switch tag {
	case "If", "Show", "When":
		return true
	default:
		return false
	}
}

func nativeAttr(name string) bool {
	switch name {
	case "placeholder", "type", "value":
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
