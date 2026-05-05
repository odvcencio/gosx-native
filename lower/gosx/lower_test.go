package gosx

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/odvcencio/gosx/nir"
)

func TestLowerCounterMatchesSwiftCounterSemantics(t *testing.T) {
	got := lowerFixture(t, "counter.gsx")

	expectedData, err := os.ReadFile("../../testdata/expected/nir/counter.json")
	if err != nil {
		t.Fatalf("read expected NIR: %v", err)
	}
	var expected nir.Module
	if err := json.Unmarshal(expectedData, &expected); err != nil {
		t.Fatalf("unmarshal expected NIR: %v", err)
	}

	normalizeModule(got)
	normalizeModule(&expected)
	if !reflect.DeepEqual(got, &expected) {
		gotJSON, _ := json.MarshalIndent(got, "", "  ")
		expectedJSON, _ := json.MarshalIndent(expected, "", "  ")
		t.Fatalf("GoSX NIR semantics mismatch.\nGot:\n%s\n\nExpected:\n%s", gotJSON, expectedJSON)
	}
}

func TestLowerPanelCoversInlineAndMultiSignalHandlers(t *testing.T) {
	mod := lowerFixture(t, "panel.gsx")
	if got := len(mod.Components); got != 1 {
		t.Fatalf("component count = %d, want 1", got)
	}
	component := mod.Components[0]
	if component.Name != "Panel" {
		t.Fatalf("component name = %q, want Panel", component.Name)
	}
	if got, want := component.Props.Fields, []nir.PropField{
		{Name: "start", Type: "Int"},
		{Name: "label", Type: "String"},
	}; !reflect.DeepEqual(got, want) {
		t.Fatalf("props = %+v, want %+v", got, want)
	}
	if got := len(component.Signals); got != 2 {
		t.Fatalf("signals = %d, want 2", got)
	}
	assertSignal(t, component.Signals[0], "count", "Int", "props.start")
	assertSignal(t, component.Signals[1], "label", "String", "props.label")

	root := requireElement(t, component.Body, "vstack")
	if got := len(root.Children); got != 2 {
		t.Fatalf("root children = %d, want 2", got)
	}
	hstack := requireElement(t, root.Children[1], "hstack")
	if got := len(hstack.Children); got != 4 {
		t.Fatalf("hstack children = %d, want 4", got)
	}

	reset := requireElement(t, hstack.Children[0], "button")
	requireHandlerTargets(t, reset, []string{"count", "label"})
	requireRefValue(t, reset.Handlers[0].Body.Stmts[0].Value, "props.start")
	requireRefValue(t, reset.Handlers[0].Body.Stmts[1].Value, "props.label")

	inlineIncrement := requireElement(t, hstack.Children[2], "button")
	requireHandlerTargets(t, inlineIncrement, []string{"count"})
	requireBinOpValue(t, inlineIncrement.Handlers[0].Body.Stmts[0].Value, "+")

	advance := requireElement(t, hstack.Children[3], "button")
	requireHandlerTargets(t, advance, []string{"count", "label"})
	requireBinOpValue(t, advance.Handlers[0].Body.Stmts[0].Value, "+")
	requireLiteralValue(t, advance.Handlers[0].Body.Stmts[1].Value, "string", "advanced")
}

func TestLowerGreeterCoversEventValueTextInput(t *testing.T) {
	mod := lowerFixture(t, "greeter.gsx")
	if got := len(mod.Components); got != 1 {
		t.Fatalf("component count = %d, want 1", got)
	}
	component := mod.Components[0]
	if component.Name != "Greeter" {
		t.Fatalf("component name = %q, want Greeter", component.Name)
	}
	if got, want := component.Props.Fields, []nir.PropField{
		{Name: "initialName", Type: "String"},
	}; !reflect.DeepEqual(got, want) {
		t.Fatalf("props = %+v, want %+v", got, want)
	}
	if got := len(component.Signals); got != 1 {
		t.Fatalf("signals = %d, want 1", got)
	}
	assertSignal(t, component.Signals[0], "name", "String", "props.initialName")

	root := requireElement(t, component.Body, "vstack")
	if got := len(root.Children); got != 2 {
		t.Fatalf("root children = %d, want 2", got)
	}
	input := requireElement(t, root.Children[0], "textinput")
	requireAttrRef(t, input, "value", "name")
	requireAttrLiteral(t, input, "placeholder", "string", "Name")
	requireHandlerEventTargets(t, input, "input", []string{"name"})
	requireRefValue(t, input.Handlers[0].Body.Stmts[0].Value, "event.value")
}

func TestLowerFormControlsCoversNativeEventFields(t *testing.T) {
	mod := lowerFixture(t, "form_controls.gsx")
	if got := len(mod.Components); got != 1 {
		t.Fatalf("component count = %d, want 1", got)
	}
	component := mod.Components[0]
	if component.Name != "FormControls" {
		t.Fatalf("component name = %q, want FormControls", component.Name)
	}
	if got, want := component.Props.Fields, []nir.PropField{
		{Name: "bio", Type: "String"},
		{Name: "enabled", Type: "Bool"},
		{Name: "selectedIndex", Type: "Int"},
	}; !reflect.DeepEqual(got, want) {
		t.Fatalf("props = %+v, want %+v", got, want)
	}
	if got := len(component.Signals); got != 4 {
		t.Fatalf("signals = %d, want 4", got)
	}
	assertSignal(t, component.Signals[0], "bio", "String", "props.bio")
	assertSignal(t, component.Signals[1], "enabled", "Bool", "props.enabled")
	if component.Signals[2].Name != "lastKey" || component.Signals[2].Type != "String" {
		t.Fatalf("signal = %+v, want name=lastKey type=String", component.Signals[2])
	}
	requireLiteralValue(t, component.Signals[2].Init, "string", "")
	assertSignal(t, component.Signals[3], "choice", "Int", "props.selectedIndex")

	root := requireElement(t, component.Body, "vstack")
	if got := len(root.Children); got != 8 {
		t.Fatalf("root children = %d, want 8", got)
	}

	textarea := requireElement(t, root.Children[0], "textarea")
	requireAttrRef(t, textarea, "value", "bio")
	requireAttrLiteral(t, textarea, "placeholder", "string", "Bio")
	requireHandlerValueRef(t, textarea, "input", "bio", "event.value")

	checkbox := requireElement(t, root.Children[1], "checkbox")
	requireAttrLiteral(t, checkbox, "type", "string", "checkbox")
	requireAttrRef(t, checkbox, "checked", "enabled")
	requireHandlerValueRef(t, checkbox, "change", "enabled", "event.checked")

	keyInput := requireElement(t, root.Children[2], "textinput")
	requireAttrRef(t, keyInput, "value", "lastKey")
	requireAttrLiteral(t, keyInput, "placeholder", "string", "Press key")
	requireHandlerValueRef(t, keyInput, "key", "lastKey", "event.key")

	selectInput := requireElement(t, root.Children[3], "select")
	requireAttrRef(t, selectInput, "selectedIndex", "choice")
	requireHandlerValueRef(t, selectInput, "change", "choice", "event.selectedIndex")
	if got := len(selectInput.Children); got != 2 {
		t.Fatalf("select options = %d, want 2", got)
	}
	requireElement(t, selectInput.Children[0], "option")
	requireElement(t, selectInput.Children[1], "option")
}

func TestLowerDerivedCoversComputedDecl(t *testing.T) {
	mod := lowerFixture(t, "derived.gsx")
	if got := len(mod.Components); got != 1 {
		t.Fatalf("component count = %d, want 1", got)
	}
	component := mod.Components[0]
	if component.Name != "Derived" {
		t.Fatalf("component name = %q, want Derived", component.Name)
	}
	if got, want := component.Props.Fields, []nir.PropField{
		{Name: "start", Type: "Int"},
	}; !reflect.DeepEqual(got, want) {
		t.Fatalf("props = %+v, want %+v", got, want)
	}
	if got := len(component.Signals); got != 1 {
		t.Fatalf("signals = %d, want 1", got)
	}
	assertSignal(t, component.Signals[0], "count", "Int", "props.start")
	if got := len(component.Computeds); got != 1 {
		t.Fatalf("computeds = %d, want 1", got)
	}
	computed := component.Computeds[0]
	if computed.Name != "doubled" || computed.Type != "Int" {
		t.Fatalf("computed = %+v, want doubled Int", computed)
	}
	requireBinOpValue(t, computed.Body, "*")

	root := requireElement(t, component.Body, "vstack")
	if got := len(root.Children); got != 2 {
		t.Fatalf("root children = %d, want 2", got)
	}
	text := requireElement(t, root.Children[0], "text")
	requireExprHoleRef(t, text.Children[0], "doubled")
	button := requireElement(t, root.Children[1], "button")
	requireHandlerTargets(t, button, []string{"count"})
	requireBinOpValue(t, button.Handlers[0].Body.Stmts[0].Value, "+")
}

func TestLowerConditionalCoversIfViewNode(t *testing.T) {
	mod := lowerFixture(t, "conditional.gsx")
	if got := len(mod.Components); got != 1 {
		t.Fatalf("component count = %d, want 1", got)
	}
	component := mod.Components[0]
	if component.Name != "Toggle" {
		t.Fatalf("component name = %q, want Toggle", component.Name)
	}
	if got, want := component.Props.Fields, []nir.PropField{
		{Name: "initiallyVisible", Type: "Bool"},
	}; !reflect.DeepEqual(got, want) {
		t.Fatalf("props = %+v, want %+v", got, want)
	}
	if got := len(component.Signals); got != 1 {
		t.Fatalf("signals = %d, want 1", got)
	}
	assertSignal(t, component.Signals[0], "visible", "Bool", "props.initiallyVisible")

	root := requireElement(t, component.Body, "vstack")
	if got := len(root.Children); got != 2 {
		t.Fatalf("root children = %d, want 2", got)
	}
	conditional := requireConditional(t, root.Children[0], "visible")
	if got := len(conditional.Then); got != 1 {
		t.Fatalf("then children = %d, want 1", got)
	}
	visibleText := requireElement(t, conditional.Then[0], "text")
	if got := len(visibleText.Children); got != 1 {
		t.Fatalf("text children = %d, want 1", got)
	}
	visible, ok := visibleText.Children[0].(*nir.Text)
	if !ok || visible.Value != "Visible" {
		t.Fatalf("conditional text = %+v, want Visible", visibleText.Children[0])
	}

	button := requireElement(t, root.Children[1], "button")
	requireHandlerTargets(t, button, []string{"visible"})
	requireLiteralValue(t, button.Handlers[0].Body.Stmts[0].Value, "bool", "false")
}

func TestLowerComponentRefCoversNestedComponentCall(t *testing.T) {
	mod := lowerFixture(t, "component_ref.gsx")
	if got := len(mod.Components); got != 2 {
		t.Fatalf("component count = %d, want 2", got)
	}
	badge := findComponent(t, mod, "Badge")
	if got, want := badge.Props.Fields, []nir.PropField{
		{Name: "label", Type: "String"},
	}; !reflect.DeepEqual(got, want) {
		t.Fatalf("badge props = %+v, want %+v", got, want)
	}
	badgeBody := requireElement(t, badge.Body, "text")
	requireExprHoleRef(t, badgeBody.Children[0], "props.label")

	profile := findComponent(t, mod, "Profile")
	if got, want := profile.Props.Fields, []nir.PropField{
		{Name: "name", Type: "String"},
	}; !reflect.DeepEqual(got, want) {
		t.Fatalf("profile props = %+v, want %+v", got, want)
	}
	root := requireElement(t, profile.Body, "vstack")
	if got := len(root.Children); got != 1 {
		t.Fatalf("root children = %d, want 1", got)
	}
	ref := requireComponentRef(t, root.Children[0], "Badge")
	if got := len(ref.Props); got != 1 {
		t.Fatalf("component props = %d, want 1", got)
	}
	if ref.Props[0].Name != "label" {
		t.Fatalf("prop name = %q, want label", ref.Props[0].Name)
	}
	requireRefValue(t, &ref.Props[0].Value, "props.name")
}

func TestLowerLoopCoversEachViewNode(t *testing.T) {
	mod := lowerFixture(t, "loop.gsx")
	if got := len(mod.Components); got != 1 {
		t.Fatalf("component count = %d, want 1", got)
	}
	component := mod.Components[0]
	if component.Name != "Roster" {
		t.Fatalf("component name = %q, want Roster", component.Name)
	}
	if got, want := component.Props.Fields, []nir.PropField{
		{Name: "items", Type: "[String]"},
	}; !reflect.DeepEqual(got, want) {
		t.Fatalf("props = %+v, want %+v", got, want)
	}
	root := requireElement(t, component.Body, "vstack")
	if got := len(root.Children); got != 1 {
		t.Fatalf("root children = %d, want 1", got)
	}
	loop := requireLoop(t, root.Children[0], "props.items", "item")
	if got := len(loop.Body); got != 1 {
		t.Fatalf("loop body children = %d, want 1", got)
	}
	text := requireElement(t, loop.Body[0], "text")
	requireExprHoleRef(t, text.Children[0], "item")
}

func lowerFixture(t *testing.T, name string) *nir.Module {
	t.Helper()
	src, err := os.ReadFile(filepath.Join("../../testdata/corpus/go", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	mod, err := LowerSource(src)
	if err != nil {
		t.Fatalf("lower fixture %s: %v", name, err)
	}
	return mod
}

func findComponent(t *testing.T, mod *nir.Module, name string) *nir.Component {
	t.Helper()
	for _, component := range mod.Components {
		if component.Name == name {
			return component
		}
	}
	t.Fatalf("missing component %q", name)
	return nil
}

func assertSignal(t *testing.T, sig *nir.SignalDecl, name, typ, initRef string) {
	t.Helper()
	if sig.Name != name || sig.Type != typ {
		t.Fatalf("signal = %+v, want name=%s type=%s", sig, name, typ)
	}
	requireRefValue(t, sig.Init, initRef)
}

func requireElement(t *testing.T, view nir.View, tag string) *nir.Element {
	t.Helper()
	element, ok := view.(*nir.Element)
	if !ok {
		t.Fatalf("view = %T, want *nir.Element", view)
	}
	if element.Tag != tag {
		t.Fatalf("element tag = %q, want %q", element.Tag, tag)
	}
	return element
}

func requireHandlerTargets(t *testing.T, element *nir.Element, targets []string) {
	t.Helper()
	requireHandlerEventTargets(t, element, "tap", targets)
}

func requireHandlerEventTargets(t *testing.T, element *nir.Element, event string, targets []string) {
	t.Helper()
	if len(element.Handlers) != 1 {
		t.Fatalf("handler count = %d, want 1", len(element.Handlers))
	}
	handler := element.Handlers[0]
	if handler.Event != event {
		t.Fatalf("event = %q, want %q", handler.Event, event)
	}
	if len(handler.Body.Stmts) != len(targets) {
		t.Fatalf("stmt count = %d, want %d", len(handler.Body.Stmts), len(targets))
	}
	for i, target := range targets {
		stmt := handler.Body.Stmts[i]
		if stmt.Kind != "signal_set" || stmt.Target != target {
			t.Fatalf("stmt[%d] = %+v, want signal_set target=%s", i, stmt, target)
		}
	}
}

func requireHandlerValueRef(t *testing.T, element *nir.Element, event, target, ref string) {
	t.Helper()
	requireHandlerEventTargets(t, element, event, []string{target})
	requireRefValue(t, element.Handlers[0].Body.Stmts[0].Value, ref)
}

func requireAttrRef(t *testing.T, element *nir.Element, name, ref string) {
	t.Helper()
	requireRefValue(t, requireAttr(t, element, name), ref)
}

func requireAttrLiteral(t *testing.T, element *nir.Element, name, typ, value string) {
	t.Helper()
	requireLiteralValue(t, requireAttr(t, element, name), typ, value)
}

func requireAttr(t *testing.T, element *nir.Element, name string) *nir.RxExpr {
	t.Helper()
	for i := range element.Attrs {
		if element.Attrs[i].Name == name {
			return &element.Attrs[i].Value
		}
	}
	t.Fatalf("missing attr %q on %+v", name, element.Attrs)
	return nil
}

func requireExprHoleRef(t *testing.T, view nir.View, ref string) {
	t.Helper()
	hole, ok := view.(*nir.ExprHole)
	if !ok {
		t.Fatalf("view = %T, want *nir.ExprHole", view)
	}
	requireRefValue(t, &hole.Expr, ref)
}

func requireConditional(t *testing.T, view nir.View, conditionRef string) *nir.Conditional {
	t.Helper()
	conditional, ok := view.(*nir.Conditional)
	if !ok {
		t.Fatalf("view = %T, want *nir.Conditional", view)
	}
	requireRefValue(t, &conditional.Condition, conditionRef)
	return conditional
}

func requireComponentRef(t *testing.T, view nir.View, name string) *nir.ComponentRef {
	t.Helper()
	ref, ok := view.(*nir.ComponentRef)
	if !ok {
		t.Fatalf("view = %T, want *nir.ComponentRef", view)
	}
	if ref.Name != name {
		t.Fatalf("component ref = %q, want %q", ref.Name, name)
	}
	return ref
}

func requireLoop(t *testing.T, view nir.View, itemsRef, itemName string) *nir.Loop {
	t.Helper()
	loop, ok := view.(*nir.Loop)
	if !ok {
		t.Fatalf("view = %T, want *nir.Loop", view)
	}
	requireRefValue(t, &loop.Items, itemsRef)
	if loop.ItemName != itemName {
		t.Fatalf("loop item = %q, want %q", loop.ItemName, itemName)
	}
	return loop
}

func requireRefValue(t *testing.T, expr *nir.RxExpr, ref string) {
	t.Helper()
	if expr == nil || expr.Kind != "ref" || expr.Ref != ref {
		t.Fatalf("expr = %+v, want ref %q", expr, ref)
	}
}

func requireBinOpValue(t *testing.T, expr *nir.RxExpr, op string) {
	t.Helper()
	if expr == nil || expr.Kind != "binop" || expr.BinOp == nil || expr.BinOp.Op != op {
		t.Fatalf("expr = %+v, want binop %q", expr, op)
	}
}

func requireLiteralValue(t *testing.T, expr *nir.RxExpr, typ, value string) {
	t.Helper()
	if expr == nil || expr.Kind != "literal" || expr.Literal == nil || expr.Literal.Type != typ || expr.Literal.Value != value {
		t.Fatalf("expr = %+v, want literal %s %q", expr, typ, value)
	}
}

func normalizeModule(mod *nir.Module) {
	mod.SourceLanguage = ""
	for _, component := range mod.Components {
		component.Span = nir.Span{}
		for _, signal := range component.Signals {
			signal.Span = nir.Span{}
			normalizeRxExpr(signal.Init)
		}
		for _, computed := range component.Computeds {
			computed.Span = nir.Span{}
			normalizeRxExpr(computed.Body)
		}
		normalizeView(component.Body)
	}
}

func normalizeView(view nir.View) {
	switch node := view.(type) {
	case *nir.Element:
		node.Span = nir.Span{}
		for i := range node.Attrs {
			node.Attrs[i].Span = nir.Span{}
			normalizeRxExpr(&node.Attrs[i].Value)
		}
		for i := range node.Handlers {
			node.Handlers[i].Span = nir.Span{}
			for j := range node.Handlers[i].Body.Stmts {
				normalizeRxStmt(&node.Handlers[i].Body.Stmts[j])
			}
		}
		for _, child := range node.Children {
			normalizeView(child)
		}
	case *nir.Text:
		node.Span = nir.Span{}
	case *nir.ExprHole:
		node.Span = nir.Span{}
		normalizeRxExpr(&node.Expr)
	case *nir.Conditional:
		node.Span = nir.Span{}
		normalizeRxExpr(&node.Condition)
		for _, child := range node.Then {
			normalizeView(child)
		}
		for _, child := range node.Else {
			normalizeView(child)
		}
	case *nir.ComponentRef:
		node.Span = nir.Span{}
		for i := range node.Props {
			node.Props[i].Span = nir.Span{}
			normalizeRxExpr(&node.Props[i].Value)
		}
		for _, child := range node.Children {
			normalizeView(child)
		}
	case *nir.Loop:
		node.Span = nir.Span{}
		normalizeRxExpr(&node.Items)
		for _, child := range node.Body {
			normalizeView(child)
		}
		for _, child := range node.Empty {
			normalizeView(child)
		}
	}
}

func normalizeRxStmt(stmt *nir.RxStmt) {
	normalizeRxExpr(stmt.Expr)
	normalizeRxExpr(stmt.Value)
}

func normalizeRxExpr(expr *nir.RxExpr) {
	if expr == nil {
		return
	}
	expr.Span = nir.Span{}
	if expr.BinOp != nil {
		normalizeRxExpr(&expr.BinOp.Left)
		normalizeRxExpr(&expr.BinOp.Right)
	}
	if expr.Call != nil {
		for i := range expr.Call.Args {
			normalizeRxExpr(&expr.Call.Args[i])
		}
	}
}
