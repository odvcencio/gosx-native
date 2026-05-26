package swift

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"m31labs.dev/gosx-native/grammar"
	gotreesitter "github.com/odvcencio/gotreesitter"
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
