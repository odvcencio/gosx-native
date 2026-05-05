package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/odvcencio/gosx-native/grammar"
	swiftlower "github.com/odvcencio/gosx-native/lower/swift"
	"github.com/odvcencio/gosx/nir"
	gotreesitter "github.com/odvcencio/gotreesitter"
)

func runCompile(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: gsxnative compile <file.swift.gsx>")
	}
	mod, err := compileSwift(args[0])
	if err != nil {
		return err
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(mod)
}

func compileSwift(path string) (*nir.Module, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	lang, err := grammar.SwiftGSXLanguage()
	if err != nil {
		return nil, err
	}
	parser := gotreesitter.NewParser(lang)
	tree, err := parser.Parse(src)
	if err != nil {
		return nil, err
	}
	defer tree.Release()
	if tree.RootNode().HasError() {
		return nil, fmt.Errorf("parse error in %s:\n%s", path, tree.RootNode().SExpr(lang))
	}
	return swiftlower.Lower(tree.RootNode(), src, lang)
}
