package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	nativescene3d "github.com/odvcencio/gosx-native/scene3d"
	"github.com/odvcencio/gosx-native/target"
	"github.com/odvcencio/gosx/nir"
)

type sceneConformOptions struct {
	target    string
	update    bool
	goldenDir string
}

func runSceneConform(args []string) error {
	root, err := repoRoot()
	if err != nil {
		return err
	}
	opts, sources, err := parseSceneConformOptions(root, args)
	if err != nil {
		return err
	}
	if len(sources) == 0 {
		sources = defaultSceneConformSources(root)
	}
	for _, source := range sources {
		if err := sceneConformSource(source, opts); err != nil {
			return err
		}
	}
	return nil
}

func defaultSceneConformSources(root string) []string {
	return []string{
		filepath.Join(root, "testdata/corpus/go/scene3d.gsx"),
		filepath.Join(root, "testdata/corpus/go/scene3d_instancing.gsx"),
		filepath.Join(root, "testdata/corpus/go/scene3d_postfx.gsx"),
		filepath.Join(root, "testdata/corpus/go/scene3d_compute.gsx"),
		filepath.Join(root, "testdata/corpus/go/scene3d_html.gsx"),
	}
}

func parseSceneConformOptions(root string, args []string) (sceneConformOptions, []string, error) {
	opts := sceneConformOptions{
		target:    "all",
		goldenDir: filepath.Join(root, "testdata/expected/scene3d"),
	}
	fs := flag.NewFlagSet("gsxnative scene-conform", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&opts.target, "target", opts.target, "target to check: ios, android, or all")
	fs.BoolVar(&opts.update, "update", false, "rewrite conformance goldens")
	fs.StringVar(&opts.goldenDir, "golden-dir", opts.goldenDir, "directory containing Scene3D conformance goldens")
	if err := fs.Parse(args); err != nil {
		return sceneConformOptions{}, nil, err
	}
	opts.target = strings.ToLower(strings.TrimSpace(opts.target))
	switch opts.target {
	case "all", string(target.IOS), string(target.Android):
	default:
		return sceneConformOptions{}, nil, fmt.Errorf("unknown scene-conform target: %s (supported: ios, android, all)", opts.target)
	}
	return opts, fs.Args(), nil
}

func sceneConformSource(source string, opts sceneConformOptions) error {
	mod, err := compileFile(source)
	if err != nil {
		return err
	}
	sceneElement, err := findScene3DElement(mod)
	if err != nil {
		return err
	}
	ir, err := nativescene3d.CanonicalIR(sceneElement)
	if err != nil {
		return err
	}
	irJSON, err := json.MarshalIndent(ir, "", "  ")
	if err != nil {
		return err
	}
	irJSON = append(irJSON, '\n')
	base := sceneConformBase(source)
	if err := compareOrUpdate(filepath.Join(opts.goldenDir, base+".ir.json"), irJSON, opts.update); err != nil {
		return err
	}
	for _, tgt := range sceneConformTargets(opts.target) {
		if err := target.Validate(mod, tgt); err != nil {
			return err
		}
		sourceBytes, err := emitNativeSource(tgt, mod)
		if err != nil {
			return err
		}
		goldenPath := filepath.Join(opts.goldenDir, base+"."+string(tgt)+sceneConformExt(tgt))
		if err := compareOrUpdate(goldenPath, sourceBytes, opts.update); err != nil {
			return err
		}
	}
	fmt.Fprintf(os.Stdout, "scene-conform: %s OK\n", source)
	return nil
}

func findScene3DElement(mod *nir.Module) (*nir.Element, error) {
	if mod == nil {
		return nil, fmt.Errorf("nil NIR module")
	}
	for _, component := range mod.Components {
		if component == nil {
			continue
		}
		if element := findScene3DView(component.Body); element != nil {
			return element, nil
		}
	}
	return nil, fmt.Errorf("no <Scene3D> element found")
}

func findScene3DView(view nir.View) *nir.Element {
	switch n := view.(type) {
	case *nir.Element:
		if n.Tag == "scene3d" {
			return n
		}
		for _, child := range n.Children {
			if element := findScene3DView(child); element != nil {
				return element
			}
		}
	case *nir.Conditional:
		for _, child := range n.Then {
			if element := findScene3DView(child); element != nil {
				return element
			}
		}
		for _, child := range n.Else {
			if element := findScene3DView(child); element != nil {
				return element
			}
		}
	case *nir.ComponentRef:
		for _, child := range n.Children {
			if element := findScene3DView(child); element != nil {
				return element
			}
		}
	case *nir.Loop:
		for _, child := range n.Body {
			if element := findScene3DView(child); element != nil {
				return element
			}
		}
		for _, child := range n.Empty {
			if element := findScene3DView(child); element != nil {
				return element
			}
		}
	}
	return nil
}

func compareOrUpdate(path string, got []byte, update bool) error {
	if update {
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}
		return os.WriteFile(path, got, 0644)
	}
	want, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if !bytes.Equal(bytes.TrimSpace(got), bytes.TrimSpace(want)) {
		return fmt.Errorf("scene conformance mismatch for %s\nGot:\n%s\nExpected:\n%s", path, got, want)
	}
	return nil
}

func sceneConformTargets(name string) []target.Target {
	switch name {
	case string(target.IOS):
		return []target.Target{target.IOS}
	case string(target.Android):
		return []target.Target{target.Android}
	default:
		return []target.Target{target.IOS, target.Android}
	}
}

func sceneConformExt(tgt target.Target) string {
	switch tgt {
	case target.IOS:
		return ".swift"
	case target.Android:
		return ".kt"
	default:
		return ".txt"
	}
}

func sceneConformBase(source string) string {
	base := filepath.Base(source)
	base = strings.TrimSuffix(base, ".swift.gsx")
	base = strings.TrimSuffix(base, ".gsx")
	return base
}
