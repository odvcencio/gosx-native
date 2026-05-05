package gosx

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"

	"github.com/odvcencio/gosx/nir"
)

func TestLowerCounterMatchesSwiftCounterSemantics(t *testing.T) {
	src, err := os.ReadFile("../../testdata/corpus/go/counter.gsx")
	if err != nil {
		t.Fatalf("read GoSX counter: %v", err)
	}
	got, err := LowerSource(src)
	if err != nil {
		t.Fatalf("lower GoSX counter: %v", err)
	}

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

func normalizeModule(mod *nir.Module) {
	mod.SourceLanguage = ""
	for _, component := range mod.Components {
		component.Span = nir.Span{}
		for _, signal := range component.Signals {
			signal.Span = nir.Span{}
			normalizeRxExpr(signal.Init)
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
