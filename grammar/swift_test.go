package grammar

import (
	"os"
	"testing"

	gotreesitter "github.com/odvcencio/gotreesitter"
)

func TestParsesTrivialJSXInSwift(t *testing.T) {
	lang, err := SwiftGSXLanguage()
	if err != nil {
		t.Fatalf("compile language: %v", err)
	}
	parser := gotreesitter.NewParser(lang)

	src := []byte(`func body() -> Node { return <text>hi</text> }`)
	tree, err := parser.Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	defer tree.Release()
	if tree.RootNode().HasError() {
		t.Fatalf("parse errors:\n%s", tree.RootNode().SExpr(lang))
	}
	if !containsType(tree.RootNode(), lang, "jsx_element") {
		t.Fatalf("no jsx_element found in tree:\n%s", tree.RootNode().SExpr(lang))
	}
}

func TestParsesCounter(t *testing.T) {
	src, err := os.ReadFile("../testdata/corpus/swift/counter.swift.gsx")
	if err != nil {
		t.Fatalf("read corpus: %v", err)
	}
	lang, err := SwiftGSXLanguage()
	if err != nil {
		t.Fatalf("compile language: %v", err)
	}
	parser := gotreesitter.NewParser(lang)
	tree, err := parser.Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	defer tree.Release()
	if tree.RootNode().HasError() {
		t.Fatalf("parse errors:\n%s", tree.RootNode().SExpr(lang))
	}
}

func containsType(n *gotreesitter.Node, lang *gotreesitter.Language, want string) bool {
	if n == nil {
		return false
	}
	if n.Type(lang) == want {
		return true
	}
	for i := 0; i < n.ChildCount(); i++ {
		if containsType(n.Child(i), lang, want) {
			return true
		}
	}
	return false
}
