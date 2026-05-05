package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/odvcencio/gosx-native/grammar"
	gosxlower "github.com/odvcencio/gosx-native/lower/gosx"
	swiftlower "github.com/odvcencio/gosx-native/lower/swift"
	"github.com/odvcencio/gosx/nir"
	gotreesitter "github.com/odvcencio/gotreesitter"
)

func runCompile(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: gsxnative compile <file.gsx|file.swift.gsx>")
	}
	mod, err := compileFile(args[0])
	if err != nil {
		return err
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(mod)
}

func compileFile(path string) (*nir.Module, error) {
	switch {
	case strings.HasSuffix(path, ".swift.gsx"):
		return compileSwift(path)
	case strings.HasSuffix(path, ".gsx"):
		return compileGoSX(path)
	default:
		return nil, fmt.Errorf("unsupported source file %q (supported: .gsx, .swift.gsx)", path)
	}
}

func compileGoSX(path string) (*nir.Module, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return gosxlower.LowerSource(src)
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
