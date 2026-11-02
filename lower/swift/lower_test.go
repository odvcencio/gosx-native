package swift

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	gotreesitter "github.com/odvcencio/gotreesitter"
	"m31labs.dev/gosx-native/grammar"
	"m31labs.dev/gosx-native/nir"
)

func TestLowerCounter(t *testing.T) {
	src, err := os.ReadFile("../../testdata/corpus/swift/counter.swift.gsx")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	lang, err := grammar.SwiftGSXLanguage()
	if err != nil {
		t.Fatalf("language: %v", err)
	}
	parser := gotreesitter.NewParser(lang)
	tree, err := parser.Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	defer tree.Release()

	mod, err := Lower(tree.RootNode(), src, lang)
	if err != nil {
		t.Fatalf("lower: %v", err)
	}

	got, err := json.MarshalIndent(mod, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	expected, err := os.ReadFile("../../testdata/expected/nir/counter.json")
	if err != nil {
		t.Fatalf("read expected: %v", err)
	}
	if strings.TrimSpace(string(got)) != strings.TrimSpace(string(expected)) {
		t.Fatalf("NIR mismatch.\nGot:\n%s\n\nExpected:\n%s", got, expected)
	}
}

func TestLowerEscapedString(t *testing.T) {
	src := []byte(`func Greeting() -> Node { return <text>{"hello \"world\""}</text> }`)
	lang, err := grammar.SwiftGSXLanguage()
	if err != nil {
		t.Fatal(err)
	}
	parser := gotreesitter.NewParser(lang)
	tree, err := parser.Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	defer tree.Release()
	if tree.RootNode().HasError() {
		t.Fatal(tree.RootNode().SExpr(lang))
	}
	mod, err := Lower(tree.RootNode(), src, lang)
	if err != nil {
		t.Fatal(err)
	}
	if len(mod.Components) != 1 {
		t.Fatalf("components = %d", len(mod.Components))
	}
	element, ok := mod.Components[0].Body.(*nir.Element)
	if !ok || len(element.Children) != 1 {
		t.Fatalf("body = %#v", mod.Components[0].Body)
	}
	hole, ok := element.Children[0].(*nir.ExprHole)
	if !ok || hole.Expr.Literal == nil || hole.Expr.Literal.Value != `hello "world"` {
		t.Fatalf("child = %#v", element.Children[0])
	}
}
