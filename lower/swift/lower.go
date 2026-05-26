package swift

import (
	"fmt"
	"strings"
	"unicode"

	"m31labs.dev/gosx/nir"
	gotreesitter "github.com/odvcencio/gotreesitter"
)

type lowerer struct {
	src       []byte
	lang      *gotreesitter.Language
	structs   map[string][]nir.PropField
	propTypes map[string]string
	signals   map[string]string
	component *nir.Component
}

func Lower(root *gotreesitter.Node, src []byte, lang *gotreesitter.Language) (*nir.Module, error) {
	l := &lowerer{
		src:     src,
		lang:    lang,
		structs: make(map[string][]nir.PropField),
		signals: make(map[string]string),
	}
	for _, child := range namedChildren(root) {
		if child.Type(lang) == "struct_declaration" {
			if err := l.collectStruct(child); err != nil {
				return nil, err
			}
		}
	}

	mod := &nir.Module{Version: 1, SourceLanguage: "swift"}
	for _, child := range namedChildren(root) {
		if child.Type(lang) != "function_declaration" {
			continue
		}
		component, err := l.lowerComponent(child)
		if err != nil {
			return nil, err
		}
		if component != nil {
			mod.Components = append(mod.Components, component)
		}
	}
	return mod, nil
}

func (l *lowerer) collectStruct(n *gotreesitter.Node) error {
	name := text(l.src, fieldChild(n, l.lang, "name"))
	if name == "" {
		return fmt.Errorf("struct declaration missing name")
	}
	var fields []nir.PropField
	for _, member := range fieldChildren(n, l.lang, "members") {
		if member.Type(l.lang) != "property_declaration" {
			continue
		}
		fieldName := text(l.src, fieldChild(member, l.lang, "name"))
		fieldType := text(l.src, fieldChild(member, l.lang, "type"))
		if fieldName != "" && fieldType != "" {
			fields = append(fields, nir.PropField{Name: fieldName, Type: fieldType})
		}
	}
	l.structs[name] = fields
	return nil
}

func (l *lowerer) lowerComponent(n *gotreesitter.Node) (*nir.Component, error) {
	name := text(l.src, fieldChild(n, l.lang, "name"))
	if name == "" {
		return nil, fmt.Errorf("function declaration missing name")
	}
	params := firstNamedChild(n, l.lang, "function_value_parameters")
	param := firstNamedChild(params, l.lang, "function_parameter")
	paramName := text(l.src, fieldChild(param, l.lang, "name"))
	paramType := text(l.src, fieldChild(param, l.lang, "type"))

	props := &nir.Props{Fields: l.structs[paramType]}
	l.propTypes = make(map[string]string)
	for _, field := range props.Fields {
		l.propTypes[paramName+"."+field.Name] = field.Type
	}

	c := &nir.Component{
		Name:  name,
		Props: props,
		Span:  span(n),
	}
	l.component = c
	l.signals = make(map[string]string)

	body := fieldChild(n, l.lang, "body")
	for _, stmt := range namedChildren(body) {
		if stmt.Type(l.lang) != "property_declaration" || !strings.HasPrefix(strings.TrimSpace(text(l.src, stmt)), "let ") {
			continue
		}
		signal, err := l.lowerSignal(stmt)
		if err != nil {
			return nil, err
		}
		if signal != nil {
			c.Signals = append(c.Signals, signal)
			l.signals[signal.Name] = signal.Type
		}
	}

	for _, stmt := range namedChildren(body) {
		if stmt.Type(l.lang) != "control_transfer_statement" {
			continue
		}
		result := fieldChild(stmt, l.lang, "result")
		if result == nil || !isJSX(result, l.lang) {
			continue
		}
		view, err := l.lowerView(result)
		if err != nil {
			return nil, err
		}
		c.Body = view
		return c, nil
	}
	return nil, nil
}

func (l *lowerer) lowerSignal(n *gotreesitter.Node) (*nir.SignalDecl, error) {
	name := text(l.src, fieldChild(n, l.lang, "name"))
	value := fieldChild(n, l.lang, "value")
	if name == "" || value == nil || value.Type(l.lang) != "call_expression" {
		return nil, nil
	}
	if text(l.src, fieldChild(value, l.lang, "function")) != "signal" {
		return nil, nil
	}
	args := fieldChildren(value, l.lang, "argument")
	if len(args) != 1 {
		return nil, fmt.Errorf("signal %s expects one initializer", name)
	}
	init, err := l.lowerRxExpr(args[0])
	if err != nil {
		return nil, err
	}
	typ, err := l.inferSignalType(init)
	if err != nil {
		return nil, fmt.Errorf("signal %s: %w", name, err)
	}
	return &nir.SignalDecl{Name: name, Type: typ, Init: init, Span: span(n)}, nil
}

func (l *lowerer) inferSignalType(expr *nir.RxExpr) (string, error) {
	if expr == nil {
		return "", fmt.Errorf("missing initializer")
	}
	if expr.Kind == "literal" && expr.Literal != nil {
		switch expr.Literal.Type {
		case "int":
			return "Int", nil
		case "string":
			return "String", nil
		case "bool":
			return "Bool", nil
		}
	}
	if expr.Kind == "ref" {
		if typ := l.propTypes[expr.Ref]; typ != "" {
			return typ, nil
		}
	}
	return "", fmt.Errorf("M1 cannot infer type")
}

func (l *lowerer) lowerView(n *gotreesitter.Node) (nir.View, error) {
	switch n.Type(l.lang) {
	case "jsx_element":
		open := fieldChild(n, l.lang, "open")
		element := &nir.Element{
			Tag:  tagName(l.src, fieldChild(open, l.lang, "name")),
			Span: span(n),
		}
		for _, attr := range fieldChildren(open, l.lang, "attributes") {
			name := text(l.src, fieldChild(attr, l.lang, "name"))
			value := fieldChild(attr, l.lang, "value")
			if isEventName(name) {
				block, err := l.lowerHandler(value)
				if err != nil {
					return nil, err
				}
				element.Handlers = append(element.Handlers, nir.Handler{
					Event: canonicalEventName(name),
					Body:  block,
					Span:  span(attr),
				})
				continue
			}
			expr, err := l.lowerExpressionContainer(value)
			if err != nil {
				return nil, err
			}
			element.Attrs = append(element.Attrs, nir.Attr{Name: name, Value: valueOrZero(expr), Span: span(attr)})
		}
		for _, child := range fieldChildren(n, l.lang, "children") {
			view, err := l.lowerJSXChild(child)
			if err != nil {
				return nil, err
			}
			if view != nil {
				element.Children = append(element.Children, view)
			}
		}
		return element, nil
	case "jsx_self_closing_element":
		return &nir.Element{
			Tag:  tagName(l.src, fieldChild(n, l.lang, "name")),
			Span: span(n),
		}, nil
	default:
		return nil, fmt.Errorf("unsupported view node %s", n.Type(l.lang))
	}
}

func (l *lowerer) lowerJSXChild(n *gotreesitter.Node) (nir.View, error) {
	switch n.Type(l.lang) {
	case "jsx_text":
		value := strings.TrimSpace(text(l.src, n))
		if value == "" {
			return nil, nil
		}
		return &nir.Text{Value: value, Span: span(n)}, nil
	case "jsx_expression_container":
		expr, err := l.lowerExpressionContainer(n)
		if err != nil {
			return nil, err
		}
		if expr == nil {
			return nil, nil
		}
		return &nir.ExprHole{Expr: *expr, Span: span(n)}, nil
	case "jsx_element", "jsx_self_closing_element":
		return l.lowerView(n)
	default:
		return nil, nil
	}
}

func (l *lowerer) lowerHandler(n *gotreesitter.Node) (nir.RxBlock, error) {
	expr := fieldChild(n, l.lang, "expression")
	stmt, err := l.lowerRxStmt(expr)
	if err != nil {
		return nir.RxBlock{}, err
	}
	return nir.RxBlock{Stmts: []nir.RxStmt{stmt}}, nil
}

func (l *lowerer) lowerExpressionContainer(n *gotreesitter.Node) (*nir.RxExpr, error) {
	if n == nil {
		return nil, nil
	}
	expr := fieldChild(n, l.lang, "expression")
	if expr == nil {
		return nil, nil
	}
	return l.lowerRxExpr(expr)
}

func (l *lowerer) lowerRxStmt(n *gotreesitter.Node) (nir.RxStmt, error) {
	if n != nil && n.Type(l.lang) == "call_expression" {
		if target, method, ok := l.navigationParts(fieldChild(n, l.lang, "function")); ok && method == "set" && l.signals[target] != "" {
			args := fieldChildren(n, l.lang, "argument")
			if len(args) != 1 {
				return nir.RxStmt{}, fmt.Errorf("signal set expects one value")
			}
			value, err := l.lowerRxExpr(args[0])
			if err != nil {
				return nir.RxStmt{}, err
			}
			return nir.RxStmt{Kind: "signal_set", Target: target, Value: value}, nil
		}
	}
	expr, err := l.lowerRxExpr(n)
	if err != nil {
		return nir.RxStmt{}, err
	}
	return nir.RxStmt{Kind: "expr", Expr: expr}, nil
}

func (l *lowerer) lowerRxExpr(n *gotreesitter.Node) (*nir.RxExpr, error) {
	if n == nil {
		return nil, fmt.Errorf("missing expression")
	}
	switch n.Type(l.lang) {
	case "integer_literal":
		return &nir.RxExpr{
			Kind:    "literal",
			Literal: &nir.Literal{Type: "int", Value: text(l.src, n)},
			Span:    span(n),
		}, nil
	case "string_literal":
		return &nir.RxExpr{
			Kind:    "literal",
			Literal: &nir.Literal{Type: "string", Value: strings.Trim(text(l.src, n), `"`)},
			Span:    span(n),
		}, nil
	case "simple_identifier":
		return &nir.RxExpr{Kind: "ref", Ref: text(l.src, n), Span: span(n)}, nil
	case "navigation_expression":
		return &nir.RxExpr{Kind: "ref", Ref: navigationText(l.src, n, l.lang), Span: span(n)}, nil
	case "call_expression":
		if target, method, ok := l.navigationParts(fieldChild(n, l.lang, "function")); ok && method == "get" && l.signals[target] != "" {
			return &nir.RxExpr{Kind: "ref", Ref: target, Span: span(n)}, nil
		}
		args := fieldChildren(n, l.lang, "argument")
		call := &nir.Call{Callee: exprText(l.src, fieldChild(n, l.lang, "function"), l.lang)}
		for _, arg := range args {
			expr, err := l.lowerRxExpr(arg)
			if err != nil {
				return nil, err
			}
			call.Args = append(call.Args, *expr)
		}
		return &nir.RxExpr{Kind: "call", Call: call, Span: span(n)}, nil
	case "binary_expression":
		left, err := l.lowerRxExpr(fieldChild(n, l.lang, "left"))
		if err != nil {
			return nil, err
		}
		right, err := l.lowerRxExpr(fieldChild(n, l.lang, "right"))
		if err != nil {
			return nil, err
		}
		return &nir.RxExpr{
			Kind: "binop",
			BinOp: &nir.BinOp{
				Op:    text(l.src, fieldChild(n, l.lang, "operator")),
				Left:  *left,
				Right: *right,
			},
			Span: span(n),
		}, nil
	default:
		return nil, fmt.Errorf("unsupported expression %s (%q)", n.Type(l.lang), text(l.src, n))
	}
}

func (l *lowerer) navigationParts(n *gotreesitter.Node) (string, string, bool) {
	if n == nil || n.Type(l.lang) != "navigation_expression" {
		return "", "", false
	}
	target := exprText(l.src, fieldChild(n, l.lang, "target"), l.lang)
	suffix := text(l.src, fieldChild(n, l.lang, "suffix"))
	return target, suffix, target != "" && suffix != ""
}

func isJSX(n *gotreesitter.Node, lang *gotreesitter.Language) bool {
	if n == nil {
		return false
	}
	switch n.Type(lang) {
	case "jsx_element", "jsx_self_closing_element", "jsx_fragment":
		return true
	default:
		return false
	}
}

func isEventName(name string) bool {
	if len(name) <= 2 || !strings.HasPrefix(name, "on") {
		return false
	}
	return unicode.IsUpper([]rune(name[2:])[0])
}

func canonicalEventName(name string) string {
	runes := []rune(name[2:])
	if len(runes) == 0 {
		return ""
	}
	runes[0] = unicode.ToLower(runes[0])
	return string(runes)
}

func valueOrZero(expr *nir.RxExpr) nir.RxExpr {
	if expr == nil {
		return nir.RxExpr{}
	}
	return *expr
}

func tagName(src []byte, n *gotreesitter.Node) string {
	if n == nil {
		return ""
	}
	for _, child := range namedChildren(n) {
		if got := tagName(src, child); got != "" {
			return got
		}
	}
	return text(src, n)
}

func navigationText(src []byte, n *gotreesitter.Node, lang *gotreesitter.Language) string {
	target := exprText(src, fieldChild(n, lang, "target"), lang)
	suffix := text(src, fieldChild(n, lang, "suffix"))
	if target == "" {
		return suffix
	}
	if suffix == "" {
		return target
	}
	return target + "." + suffix
}

func exprText(src []byte, n *gotreesitter.Node, lang *gotreesitter.Language) string {
	if n == nil {
		return ""
	}
	if n.Type(lang) == "navigation_expression" {
		return navigationText(src, n, lang)
	}
	return text(src, n)
}

func text(src []byte, n *gotreesitter.Node) string {
	if n == nil {
		return ""
	}
	return n.Text(src)
}

func span(n *gotreesitter.Node) nir.Span {
	if n == nil {
		return nir.Span{}
	}
	start := n.StartPoint()
	end := n.EndPoint()
	return nir.Span{
		StartByte: int(n.StartByte()),
		EndByte:   int(n.EndByte()),
		StartLine: int(start.Row) + 1,
		StartCol:  int(start.Column) + 1,
		EndLine:   int(end.Row) + 1,
		EndCol:    int(end.Column) + 1,
	}
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

func firstNamedChild(n *gotreesitter.Node, lang *gotreesitter.Language, typ string) *gotreesitter.Node {
	for _, child := range namedChildren(n) {
		if child.Type(lang) == typ {
			return child
		}
	}
	return nil
}

func fieldChild(n *gotreesitter.Node, lang *gotreesitter.Language, field string) *gotreesitter.Node {
	if n == nil {
		return nil
	}
	for i := 0; i < n.ChildCount(); i++ {
		if n.FieldNameForChild(i, lang) == field {
			return n.Child(i)
		}
	}
	return nil
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
