# gosx-native — Milestone 1: vertical slice

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Prove the gosx-native pipeline end-to-end with the smallest real demo: a `Counter.swift.gsx` file parses through a Swift+gsx tree-sitter grammar, lowers to NIR, emits idiomatic SwiftUI source, builds with Xcode, and runs reactively in iOS Simulator.

**Architecture:** Three pipeline stages mirror the spec. (1) `grammargen.ExtendGrammar(SwiftGrammar(), customize)` produces a `swift+gsx` grammar that the existing gotreesitter runtime parses. (2) A new `lower/swift` package walks the CST and emits a minimal `nir.Module`. (3) A new `emit/ios` package walks the NIR and prints SwiftUI source files. The runtime library `GSXNativeKit` provides `@GSXSignal` over SwiftUI's `@State` so generated code calls a stable framework API. A demo Xcode project exercises the whole chain.

**Tech Stack:** Go 1.24+ (lowerer, emitter, CLI), gotreesitter (parser runtime + grammars), Swift 5.10+ / SwiftUI (runtime + generated code), Xcode 26+ (build), GitHub Actions (CI).

**Reference skills (invoke as needed):**
- @superpowers:test-driven-development for the TDD discipline this plan depends on
- @superpowers:systematic-debugging when a step doesn't behave as written
- @superpowers:verification-before-completion before claiming any task done

---

## Scope

**In M1:**
- Single source language: `swift+gsx` (Swift host + JSX-style markup)
- Single target: iOS / SwiftUI
- Single demo component: a Counter (initial value, increment/decrement buttons, live reactive count display)
- Minimum viable NIR (just enough for Counter)
- Minimum viable framework (just `@GSXSignal` and `GSXComponent`)
- Working CLI commands: `gsxnative compile` and `gsxnative emit`
- End-to-end test that builds and launches the demo on iOS Simulator

**Explicitly out of M1** (each lands in its own subsequent milestone plan):
- `go+gsx` and `kotlin+gsx` front ends
- Android / Compose target
- Routing, data loaders, actions, auth, bridge
- Engines, scene3d, Metal/Vulkan renderers
- LSP, formatter, dev hot-reload
- Project scaffolding (`gsxnative init`)
- Optimistic updates, capability negotiation
- The `gosx mobile` subcommand shim
- The full NIR rename in gosx (this milestone uses a minimal additive `gosx/nir/`)

**Subsequent milestones** (sketched at end of this doc):
- M2 — Android target (Kotlin emit + Compose runtime, parallel of M1)
- M3 — `go+gsx` front end (lower existing gosx IR into NIR)
- M4 — Routing + navigation
- M5 — Data + actions + bridge
- M6 — Engine surface + scene3d (Metal)
- M7 — `gsxnative init`, project scaffolding, dev workflow
- M8 — Full NIR rename + capability negotiation + diagnostics polish
- M9+ — v1.x accretion roadmap from spec

---

## File structure

### Created in `~/work/gosx` (upstream additions; no breaking changes)

| File | Responsibility |
|---|---|
| `gosx/nir/nir.go` | Minimal NIR types for M1: `Module`, `Component`, `View`, `Element`, `Text`, `Slot`, `SignalDecl`, `Handler`, `Span`. Additive — does NOT touch `gosx/ir/`. |
| `gosx/nir/nir_test.go` | Round-trip tests for NIR JSON serialization. |

### Created in `~/work/gosx-native`

| File | Responsibility |
|---|---|
| `.gitignore` | Standard Go + Xcode + Android Studio ignores |
| `README.md` | One-paragraph project description, license, "M1 vertical slice" status |
| `LICENSE` | MIT |
| `go.mod` | Module path `github.com/odvcencio/gosx-native`, Go 1.24, requires gotreesitter + gosx |
| `go.sum` | Generated |
| `grammar/swift.go` | `SwiftGSXGrammar()` — `ExtendGrammar(SwiftGrammar(), customize)` adding `jsx_*` rules to Swift's expression rule |
| `grammar/swift_scanner.go` | External scanner for `jsx_attribute_expression` and `jsx_text` in Swift context |
| `grammar/swift_test.go` | Parses sample `swift+gsx` snippets, asserts CST shape |
| `lower/swift/lower.go` | `Lower(*Tree, []byte, *Language) (*nir.Module, error)` |
| `lower/swift/lower_test.go` | CST → NIR snapshot tests against testdata corpus |
| `emit/shared/tags.go` | Symbolic tag → SwiftUI/Compose mapping table; M1 fills in only the tags Counter uses |
| `emit/ios/emit.go` | `Emit(*nir.Module, io.Writer) error` — NIR → SwiftUI source |
| `emit/ios/emit_test.go` | NIR → Swift source golden tests |
| `internal/diagnostics/diagnostics.go` | `Diagnostic{Code, Severity, Span, Message, Help}` and `String()` formatter |
| `cmd/gsxnative/main.go` | CLI entry, subcommand dispatch |
| `cmd/gsxnative/compile.go` | `compile <file>` — parse + lower + dump NIR JSON |
| `cmd/gsxnative/emit.go` | `emit ios <file>` — parse + lower + emit SwiftUI to stdout |
| `runtime/ios/Package.swift` | SwiftPM manifest for GSXNativeKit |
| `runtime/ios/Sources/GSXNativeKit/GSXSignal.swift` | `@GSXSignal` propertyWrapper over `@State` |
| `runtime/ios/Sources/GSXNativeKit/GSXComponent.swift` | `GSXComponent` marker protocol |
| `runtime/ios/Tests/GSXNativeKitTests/GSXSignalTests.swift` | XCTest: signal updates trigger view recomposition |
| `testdata/corpus/swift/counter.swift.gsx` | The Counter source under test |
| `testdata/expected/nir/counter.json` | Snapshot of lowered Counter NIR |
| `testdata/expected/emit/ios/Counter.swift` | Snapshot of emitted Counter Swift source |
| `examples/counter-ios/CounterDemo.xcodeproj/` | Minimal Xcode project that uses GSXNativeKit + the emitted Counter |
| `examples/counter-ios/CounterDemo/CounterDemoApp.swift` | `@main App` entry, hosts Counter |
| `examples/counter-ios/CounterDemo/Generated/Counter.swift` | Wholly generated by the pipeline; checked in for reproducibility |
| `test/e2e/counter_test.go` | End-to-end: parse → lower → emit → xcodebuild → assert success |
| `.github/workflows/ci.yml` | Lint + unit tests on Linux; Xcode build smoke on macOS |
| `Makefile` | `make test`, `make smoke`, `make demo` convenience targets |

### Replace directives

During M1 we develop against local working copies of gosx and gotreesitter. `go.mod` includes:

```
replace github.com/odvcencio/gosx => ../gosx
replace github.com/odvcencio/gotreesitter => ../gotreesitter
```

These get stripped before tagging the first public release.

---

## Tasks

### Task 1: Verify grammargen Swift round-trip

Confirms PR #58's `SwiftGrammar()` actually composes through `ExtendGrammar()` with a trivial customize function, before we sink time into the real grammar work. This is the open question the spec elevated to first milestone.

**Files:**
- Create: `~/work/gotreesitter/grammargen/swift_extend_smoke_test.go`

- [ ] **Step 1: Write the smoke test**

```go
package grammargen

import (
	"testing"
	gotreesitter "github.com/odvcencio/gotreesitter"
)

// Verifies that grammargen.SwiftGrammar() composes through ExtendGrammar()
// and produces a runnable Language that parses trivial Swift.
func TestSwiftGrammarExtendSmoke(t *testing.T) {
	g := ExtendGrammar("swift_smoke", SwiftGrammar(), func(g *Grammar) {
		// no-op customize: just verify the round-trip works
	})
	lang, err := g.Compile()
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	parser := gotreesitter.NewParser(lang)
	defer parser.Release()
	tree, err := parser.Parse([]byte(`let x = 1`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	defer tree.Release()
	if tree.Root().HasError() {
		t.Fatalf("parse error in trivial Swift")
	}
}
```

- [ ] **Step 2: Run it to verify it passes**

```bash
cd ~/work/gotreesitter && go test ./grammargen -run TestSwiftGrammarExtendSmoke -v
```

Expected: PASS. If FAIL: stop and surface to user — the rest of M1 depends on this working. Do NOT paper over with workarounds.

- [ ] **Step 3: Repeat for Kotlin**

Add a `TestKotlinGrammarExtendSmoke` mirror, run it, confirm PASS. We don't use Kotlin in M1 but the spec depends on it for M2; cheap to verify now.

- [ ] **Step 4: Commit**

```bash
cd ~/work/gotreesitter
git add grammargen/swift_extend_smoke_test.go
buckley commit --yes -min  # if buckley installed; otherwise: git commit -m "test(grammargen): smoke-test Swift+Kotlin ExtendGrammar round-trip"
```

---

### Task 2: Add minimal NIR package to gosx

Adds an additive `gosx/nir/` package with just the node types Counter needs. Does not touch `gosx/ir/`.

**Files:**
- Create: `~/work/gosx/nir/nir.go`
- Create: `~/work/gosx/nir/nir_test.go`

- [ ] **Step 1: Write the JSON round-trip test**

```go
package nir

import (
	"encoding/json"
	"testing"
)

func TestModuleRoundTrip(t *testing.T) {
	in := &Module{
		SourceLanguage: "swift",
		Components: []*Component{{
			Name: "Counter",
			Body: &Element{
				Tag: "vstack",
				Children: []View{
					&Element{Tag: "text", Children: []View{&Text{Value: "hi"}}},
				},
			},
		}},
	}
	data, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out Module
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(out.Components) != 1 || out.Components[0].Name != "Counter" {
		t.Fatalf("round-trip mismatch: %+v", out)
	}
}
```

- [ ] **Step 2: Run to verify it fails (package doesn't exist)**

```bash
cd ~/work/gosx && go test ./nir -v
```

Expected: FAIL with `no Go files`.

- [ ] **Step 3: Implement minimal NIR types**

```go
// Package nir is the target-agnostic intermediate representation for gosx-native.
// M1 ships only the node types Counter needs; types accrete per subsequent milestone.
package nir

type Module struct {
	Version        int          `json:"version"`
	SourceLanguage string       `json:"source_language"`
	Components     []*Component `json:"components"`
}

type Component struct {
	Name  string  `json:"name"`
	Props *Props  `json:"props,omitempty"`
	Body  View    `json:"body"`
	Span  Span    `json:"span"`
}

type Props struct {
	Fields []PropField `json:"fields"`
}

type PropField struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// View is the sum type for view-tree nodes. M1 ships only Element, Text,
// and ExprHole — Slot/Fragment/Conditional/Loop/ComponentRef land in later
// milestones as the corpus grows.
type View interface {
	isView()
}

type Element struct {
	Tag      string     `json:"tag"`
	Attrs    []Attr     `json:"attrs,omitempty"`
	Handlers []Handler  `json:"handlers,omitempty"`
	Children []View     `json:"children,omitempty"`
	Span     Span       `json:"span"`
}

func (*Element) isView() {}

type Text struct {
	Value string `json:"value"`
	Span  Span   `json:"span"`
}

func (*Text) isView() {}

type ExprHole struct {
	Expr RxExpr `json:"expr"`
	Span Span   `json:"span"`
}

func (*ExprHole) isView() {}

type Attr struct {
	Name  string `json:"name"`
	Value RxExpr `json:"value"`
	Span  Span   `json:"span"`
}

type Handler struct {
	Event string  `json:"event"`         // e.g. "tap"
	Body  RxBlock `json:"body"`
	Span  Span    `json:"span"`
}

// RxExpr is the constrained reactive-expression sum type from the spec §2.
// M1 covers only Portable variants; PerTarget and Single land in later milestones.
type RxExpr struct {
	Kind    string   `json:"kind"`              // "literal" | "ref" | "binop" | "call"
	Literal *Literal `json:"literal,omitempty"`
	Ref     string   `json:"ref,omitempty"`
	BinOp   *BinOp   `json:"binop,omitempty"`
	Call    *Call    `json:"call,omitempty"`
	Span    Span     `json:"span"`
}

type Literal struct {
	Type  string `json:"type"`  // "int" | "string" | "bool"
	Value string `json:"value"` // string-encoded; type tells you how to parse
}

type BinOp struct {
	Op    string `json:"op"` // "+" "-" "*" "/" "==" etc.
	Left  RxExpr `json:"left"`
	Right RxExpr `json:"right"`
}

type Call struct {
	Callee string   `json:"callee"`
	Args   []RxExpr `json:"args"`
}

type RxBlock struct {
	Stmts []RxStmt `json:"stmts"`
}

type RxStmt struct {
	Kind   string  `json:"kind"`           // "expr" | "signal_set"
	Expr   *RxExpr `json:"expr,omitempty"`
	Target string  `json:"target,omitempty"` // signal name for signal_set
	Value  *RxExpr `json:"value,omitempty"`  // new value for signal_set
}

type SignalDecl struct {
	Name string  `json:"name"`
	Type string  `json:"type"`
	Init *RxExpr `json:"init"`
	Span Span    `json:"span"`
}

type Span struct {
	File      string `json:"file"`
	StartByte int    `json:"start_byte"`
	EndByte   int    `json:"end_byte"`
	StartLine int    `json:"start_line"`
	StartCol  int    `json:"start_col"`
	EndLine   int    `json:"end_line"`
	EndCol    int    `json:"end_col"`
}
```

Note: For M1 simplicity, `View` is a struct-tagged interface. The View slice in JSON uses each concrete type's natural shape; we'll add a discriminator in M3 when more node types land.

- [ ] **Step 4: Run test to verify it passes**

```bash
cd ~/work/gosx && go test ./nir -v
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
cd ~/work/gosx
git add nir/
buckley commit --yes -min  # or git commit -m "feat(nir): add minimal NIR package for gosx-native M1"
```

---

### Task 3: Add Component scope (signals + handlers) to NIR

Counter needs signals; signals are component-scoped. Add the minimum for Counter.

**Files:**
- Modify: `~/work/gosx/nir/nir.go`
- Modify: `~/work/gosx/nir/nir_test.go`

- [ ] **Step 1: Write the scope test**

```go
func TestComponentSignals(t *testing.T) {
	c := &Component{
		Name: "Counter",
		Signals: []*SignalDecl{{
			Name: "count",
			Type: "Int",
			Init: &RxExpr{Kind: "ref", Ref: "props.start"},
		}},
	}
	if c.Signals[0].Name != "count" {
		t.Fatalf("signal field broken: %+v", c.Signals[0])
	}
}
```

- [ ] **Step 2: Run, expect FAIL** (`Component` has no `Signals` field).

- [ ] **Step 3: Add `Signals []*SignalDecl` field to `Component`**

- [ ] **Step 4: Run, expect PASS**

- [ ] **Step 5: Commit**

```bash
cd ~/work/gosx && git add nir/ && buckley commit --yes -min
```

---

### Task 4: Bootstrap gosx-native repo

Repo init, go.mod, .gitignore, README, LICENSE, basic Makefile.

**Files:**
- Create: `~/work/gosx-native/.gitignore`
- Create: `~/work/gosx-native/README.md`
- Create: `~/work/gosx-native/LICENSE`
- Create: `~/work/gosx-native/go.mod`
- Create: `~/work/gosx-native/Makefile`

- [ ] **Step 1: `git init`**

```bash
cd ~/work/gosx-native && git init -b main
```

- [ ] **Step 2: Create `.gitignore`**

```
# Go
*.test
*.out
/bin/
/dist/

# Xcode
**/build/
*.xcuserstate
**/xcuserdata/
**/DerivedData/

# Android Studio / Gradle
**/.gradle/
**/local.properties
**/*.iml
**/captures/

# Editors
.vscode/
.idea/
*.swp

# OS
.DS_Store
```

- [ ] **Step 3: Create `LICENSE` (MIT)**

Standard MIT text with `Copyright (c) 2026 Oscar Villavicencio`.

- [ ] **Step 4: Create `README.md`**

```markdown
# gosx-native

The mobile counterpart to [gosx](https://github.com/odvcencio/gosx). React Native is to React what gosx-native is to gosx: same component model, same reactive primitives, same scene graph, different rendering targets.

**Status: M1 — vertical slice in progress.** Not ready for use.

See [`docs/superpowers/specs/2026-05-04-gosx-native-design.md`](docs/superpowers/specs/2026-05-04-gosx-native-design.md) for the design.

## License

MIT
```

- [ ] **Step 5: Create `go.mod`**

```bash
cd ~/work/gosx-native && go mod init github.com/odvcencio/gosx-native
```

The complete `go.mod` after edits should look like this — append the require + replace blocks to the auto-generated file:

```
module github.com/odvcencio/gosx-native

go 1.24

require (
	github.com/odvcencio/gosx v0.0.0
	github.com/odvcencio/gotreesitter v0.0.0
)

replace (
	github.com/odvcencio/gosx => ../gosx
	github.com/odvcencio/gotreesitter => ../gotreesitter
)
```

Then run `go mod tidy` to populate `go.sum`. If tidy fails because we haven't yet imported anything from gosx/gotreesitter, that's fine — `go.sum` will populate after Task 5 imports them.

- [ ] **Step 6: Create `Makefile`**

```makefile
.PHONY: test smoke demo lint

test:
	go test ./...

smoke:
	cd test/e2e && go test -tags smoke -v ./...

demo:
	@echo "Running M1 vertical slice demo..."
	go run ./cmd/gsxnative emit ios testdata/corpus/swift/counter.swift.gsx > /tmp/Counter.swift
	@echo "Generated /tmp/Counter.swift"

lint:
	go vet ./...
	@unformatted=$$(gofmt -l .); if [ -n "$$unformatted" ]; then echo "Unformatted files:"; echo "$$unformatted"; exit 1; fi
```

- [ ] **Step 7: Initial commit**

```bash
cd ~/work/gosx-native
git add .
buckley commit --yes -min  # or git commit -m "init: gosx-native repo bootstrap"
```

---

### Task 5: Swift+gsx grammar — minimum to parse a Counter

The grammar customize function adds `jsx_*` rules and appends them to Swift's expression rule. M1 covers: `<vstack>`, `<button onTap={...}>`, `<text>`, expression holes `{...}`, attributes (string and expression).

**Files:**
- Create: `~/work/gosx-native/grammar/swift.go`

- [ ] **Step 1: Write the grammar extension**

Use `~/work/gosx/grammar.go` as the template. Replace `GoGrammar()` with `SwiftGrammar()`. The extension point is Swift's `_primary_expression` rule (verified at `~/work/gotreesitter/grammargen/swift_grammar.go:1466`); JSX becomes a peer of `tuple_expression`, `_basic_literal`, etc. Add a JSX choice via `AppendChoice` on that rule.

```go
package grammar

import (
	gotreesitter "github.com/odvcencio/gotreesitter"
	"github.com/odvcencio/gotreesitter/grammargen"
)

// SwiftGSXGrammar extends Swift with JSX-style markup as expressions.
// File extension: .swift.gsx
func SwiftGSXGrammar() *grammargen.Grammar {
	return grammargen.ExtendGrammar("swift_gsx", grammargen.SwiftGrammar(), func(g *grammargen.Grammar) {
		// Externals — ordered must match swiftScanner constants
		g.Externals = append(g.Externals,
			grammargen.Sym("jsx_attribute_expression"),
			grammargen.Sym("jsx_text"),
		)

		// jsx_identifier, jsx_html_tag_name, jsx_dotted_name, jsx_tag_name
		// (copy from gosx/grammar.go — these are language-agnostic)
		// ...

		// jsx_string_literal, jsx_expression_container
		// jsx_attribute, jsx_spread_attribute, jsx_attributes
		// jsx_element, jsx_self_closing_element, jsx_fragment
		// (copy from gosx/grammar.go)

		// Extend `_primary_expression` (Swift's leaf-expression alternatives:
		// tuple_expression, _basic_literal, lambda_literal, etc.).
		// Verified location: ~/work/gotreesitter/grammargen/swift_grammar.go:1466.
		// JSX as a primary expression matches the gosx model (gosx adds JSX as
		// an alternative to Go's _expression rule — same shape, different host).
		grammargen.AppendChoice(g, "_primary_expression",
			grammargen.Choice(
				grammargen.Sym("jsx_element"),
				grammargen.Sym("jsx_self_closing_element"),
				grammargen.Sym("jsx_fragment"),
			),
		)
	})
}

func SwiftGSXLanguage() (*gotreesitter.Language, error) {
	g := SwiftGSXGrammar()
	g.SetExternalScanner(&swiftScanner{})
	return g.Compile()
}
```

Note: `AppendChoice` is a package-level function in grammargen (`func AppendChoice(g *Grammar, name string, rule *Rule)`) — confirmed at `~/work/gotreesitter/grammargen/grammar.go:100`. Pass a `Choice(...)` containing the new alternatives, not bare `Sym` calls.

- [ ] **Step 2: Compile-check the grammar package** (no test yet — scanner + tests come next)

```bash
cd ~/work/gosx-native && go build ./grammar/
```

Expected: builds. If it doesn't, the wrong rule was extended or external scanner reference is misordered.

- [ ] **Step 3: Commit**

```bash
git add grammar/swift.go && buckley commit --yes -min
```

---

### Task 6: Swift+gsx external scanner

Port `~/work/gosx/gsx_attr_scanner.go` to Swift context. Same two boundary tokens; the scanning logic is mostly language-agnostic (it's about JSX text and `{` boundaries).

**Files:**
- Create: `~/work/gosx-native/grammar/swift_scanner.go`

- [ ] **Step 1: Copy and adapt `gsx_attr_scanner.go`**

Read `~/work/gosx/gsx_attr_scanner.go` in full. The logic is:
- `scanAttributeExpression`: when we see `{` in attribute position, scan forward respecting nested braces, emit `jsx_attribute_expression` covering the matched range.
- `scanGSXText`: scan text until we hit `<` or `{`, emit `jsx_text`.

Adapt by:
- Changing the package name to `grammar`.
- Renaming the type `gsxScanner` → `swiftScanner`.
- Renaming the constants `gsxExternalAttributeExpression` → `swiftExternalAttributeExpression` and `gsxExternalText` → `swiftExternalText` (kept as a `const ( ... = iota ... )` block so the implicit indices align with Task 5's `g.Externals` order: `jsx_attribute_expression` is index 0, `jsx_text` is index 1 — verified against `~/work/gosx/gsx_attr_scanner.go:14-15`).
- Updating any internal references from `gsx*` to `swift*`.

The scanning bytes-and-state logic itself does NOT change.

- [ ] **Step 2: Write a parse test for a trivial JSX inside Swift**

```go
// grammar/swift_test.go
package grammar

import (
	"testing"
	gotreesitter "github.com/odvcencio/gotreesitter"
)

func TestParsesTrivialJSXInSwift(t *testing.T) {
	lang, err := SwiftGSXLanguage()
	if err != nil {
		t.Fatalf("compile language: %v", err)
	}
	parser := gotreesitter.NewParser(lang)
	defer parser.Release()

	src := []byte(`func body() -> Node { return <text>hi</text> }`)
	tree, err := parser.Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	defer tree.Release()
	if tree.Root().HasError() {
		t.Fatalf("parse errors:\n%s", tree.Root().String(lang))
	}
	// Spot-check: find a jsx_element in the tree
	if !containsType(tree.Root(), lang, "jsx_element") {
		t.Fatalf("no jsx_element found in tree:\n%s", tree.Root().String(lang))
	}
}

func containsType(n *gotreesitter.Node, lang *gotreesitter.Language, want string) bool {
	if n.Type(lang) == want {
		return true
	}
	for i := uint32(0); i < n.ChildCount(); i++ {
		if containsType(n.Child(i), lang, want) {
			return true
		}
	}
	return false
}
```

- [ ] **Step 3: Run, expect FAIL or PASS depending on grammar correctness**

```bash
cd ~/work/gosx-native && go test ./grammar -run TestParsesTrivialJSXInSwift -v
```

If FAIL with parse errors: examine the printed CST. Most likely the `_primary_expression` choice didn't compose correctly or the externals are in the wrong order. Use @superpowers:systematic-debugging.

- [ ] **Step 4: Iterate until PASS**

- [ ] **Step 5: Add a test for the full Counter shape**

```go
func TestParsesCounter(t *testing.T) {
	src, err := os.ReadFile("../testdata/corpus/swift/counter.swift.gsx")
	if err != nil {
		t.Fatalf("read corpus: %v", err)
	}
	lang, _ := SwiftGSXLanguage()
	parser := gotreesitter.NewParser(lang)
	defer parser.Release()
	tree, err := parser.Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	defer tree.Release()
	if tree.Root().HasError() {
		t.Fatalf("parse errors:\n%s", tree.Root().String(lang))
	}
}
```

- [ ] **Step 6: Create the corpus file** at `testdata/corpus/swift/counter.swift.gsx`:

```swift
struct CounterProps { var start: Int }

func Counter(props: CounterProps) -> Node {
    let count = signal(props.start)
    return <vstack>
        <button onTap={ count.set(count.get() - 1) }>-</button>
        <text>{count.get()}</text>
        <button onTap={ count.set(count.get() + 1) }>+</button>
    </vstack>
}
```

- [ ] **Step 7: Run, fix grammar gaps until PASS**

- [ ] **Step 8: Commit**

```bash
git add grammar/ testdata/corpus/swift/ && buckley commit --yes -min
```

---

### Task 7: Swift → NIR lowerer skeleton

Walks the parsed CST and emits a `*nir.Module`. M1 lowerer handles only the Counter shape; subsequent milestones extend it.

**Files:**
- Create: `~/work/gosx-native/lower/swift/lower.go`
- Create: `~/work/gosx-native/lower/swift/lower_test.go`

- [ ] **Step 1: Write the lowerer test (snapshot)**

```go
package swift

import (
	"encoding/json"
	"os"
	"testing"

	gotreesitter "github.com/odvcencio/gotreesitter"
	"github.com/odvcencio/gosx-native/grammar"
)

func TestLowerCounter(t *testing.T) {
	src, err := os.ReadFile("../../testdata/corpus/swift/counter.swift.gsx")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	lang, _ := grammar.SwiftGSXLanguage()
	parser := gotreesitter.NewParser(lang)
	defer parser.Release()
	tree, _ := parser.Parse(src)
	defer tree.Release()

	mod, err := Lower(tree.Root(), src, lang)
	if err != nil {
		t.Fatalf("lower: %v", err)
	}

	got, err := json.MarshalIndent(mod, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	expected, err := os.ReadFile("../../testdata/expected/nir/counter.json")
	if os.IsNotExist(err) {
		// First run: write the golden file. Inspect it for correctness, then re-run.
		_ = os.WriteFile("../../testdata/expected/nir/counter.json", got, 0644)
		t.Fatalf("wrote initial golden — inspect testdata/expected/nir/counter.json and re-run")
	}
	if err != nil {
		t.Fatalf("read expected: %v", err)
	}
	if string(got) != string(expected) {
		t.Fatalf("NIR mismatch.\nGot:\n%s\n\nExpected:\n%s", got, expected)
	}
}
```

- [ ] **Step 2: Run, expect FAIL** (`Lower` doesn't exist)

- [ ] **Step 3: Implement minimal `Lower`**

Use `~/work/gosx/ir/lower.go` as the template. The walking pattern is the same; only the host-language constructs differ. Concretely:

1. Find function declarations (Swift's `function_declaration` — confirm via `tree.Root().String(lang)`).
2. For each function whose body returns `<...>`, treat as a Component:
   - Component name = function name
   - Component props = parsed from the parameter list; M1 supports a single struct parameter
   - Component body = lower the JSX expression to a `View`
   - Component signals = collect `let X = signal(...)` declarations from the function body
3. JSX element → `nir.Element`; JSX text content → `nir.Text`; expression holes → `nir.ExprHole`; attributes → `nir.Attr`; event attributes → `nir.Handler` (see canonical event names below).
4. Reactive expressions: parse the host-language expression body into the constrained `RxExpr` form. For M1, support: integer literal, signal read, signal write, binary `+`/`-`, simple references.

#### M1 lowerer conventions (binding contract with the emitter)

These conventions pin how host-language constructs map into NIR. The lowerer MUST follow them; the emitter (Task 9) MUST assume them. Drift between the two breaks the pipeline silently.

**A. Signal factory.** A `let NAME = signal(INIT_EXPR)` statement in the component body lowers to:

```go
&nir.SignalDecl{
    Name: "NAME",
    Type: <inferred from INIT_EXPR — see type-inference rules below>,
    Init: <lowered INIT_EXPR as RxExpr>,
}
```

Type-inference rules for M1 (in priority order):
1. INIT_EXPR is an integer literal → `"Int"`
2. INIT_EXPR is a string literal → `"String"`
3. INIT_EXPR is a bool literal → `"Bool"`
4. INIT_EXPR is a `ref` to `props.FIELD` where FIELD is a known prop → use the prop field's declared type
5. Otherwise → emit a diagnostic ("M1 cannot infer type for signal NAME; explicit annotation required") and bail. Future milestones add full type inference.

NOT to a Call expression. The `signal(...)` invocation is recognized as a factory by the lowerer and consumed.

**B. Signal read.** A `NAME.get()` call where `NAME` is a known signal lowers to:

```go
&nir.RxExpr{Kind: "ref", Ref: "NAME"}
```

NOT to `Call{callee: "NAME.get"}`. The emitter relies on this so it can produce `count` (the propertyWrapper's wrappedValue) instead of `count.get()` (which doesn't exist on `@GSXSignal`).

**C. Signal write.** A `NAME.set(VALUE_EXPR)` call where `NAME` is a known signal lowers to a `signal_set` statement:

```go
nir.RxStmt{
    Kind:   "signal_set",
    Target: "NAME",
    Value:  <lowered VALUE_EXPR>,
}
```

NOT to a `Call` expression. The emitter produces `count = <expr>` for this (the propertyWrapper's setter is `nonmutating`).

**D. Event name canonicalization.** JSX attributes that begin with `on` followed by a capital letter (`onTap`, `onSubmit`, `onChange`) lower to handlers with the lowercased event name *without* the `on` prefix:

```
onTap   → Handler{Event: "tap"}
onSubmit → Handler{Event: "submit"}
onChange → Handler{Event: "change"}
```

The handler value (the `{...}` body) lowers to `RxBlock`. The emitter uses `Event: "tap"` to recognize tap-handler elements as `Button`.

**E. Component classification.** A function `Foo` with return type `Node` and a body that returns a JSX expression is a component. Its parameters become props. M1 supports a single struct parameter named `props` — multi-arg components and prop-spread are out of M1.

#### Implementation order

Build the lowerer function-by-function, running the test after each addition:

1. Function discovery + Component shell
2. Props extraction
3. JSX → Element/Text walking
4. Signal factory detection (convention A)
5. Handler extraction + event name canonicalization (convention D)
6. RxExpr lowering: literals, signal reads (B), binops, signal writes (C)

After each, run `go test ./lower/swift -run TestLowerCounter -v` and inspect the produced NIR before moving on.

- [ ] **Step 4: Iterate to PASS**

This is the longest single task in M1. Use @superpowers:test-driven-development discipline: don't lower anything you don't have a test for. Don't try to handle every Swift expression — only what Counter contains.

- [ ] **Step 5: Inspect the golden file**

Before committing, manually open `testdata/expected/nir/counter.json` and verify:
- `Components[0].Name == "Counter"`
- `Signals[0].Name == "count"`
- View tree has a vstack containing two buttons + a text expr-hole
- Handlers contain `signal_set` statements

If anything looks wrong, fix the lowerer (don't just adjust the golden).

- [ ] **Step 6: Commit**

```bash
git add lower/ testdata/expected/nir/ && buckley commit --yes -min
```

---

### Task 8: Symbolic tag mapping

Tiny package that maps NIR symbolic tags to platform widgets. M1 fills only what Counter needs.

**Files:**
- Create: `~/work/gosx-native/emit/shared/tags.go`
- Create: `~/work/gosx-native/emit/shared/tags_test.go`

- [ ] **Step 1: Write the test**

```go
package shared

import "testing"

func TestSwiftUITagMapping(t *testing.T) {
	cases := map[string]string{
		"vstack": "VStack",
		"hstack": "HStack",
		"text":   "Text",
		"button": "Button",
		"view":   "Group",
	}
	for nir, want := range cases {
		got := SwiftUITag(nir)
		if got != want {
			t.Errorf("SwiftUITag(%q) = %q, want %q", nir, got, want)
		}
	}
}

func TestUnknownTagReportsError(t *testing.T) {
	got := SwiftUITag("nonexistent_tag")
	if got != "" {
		t.Errorf("expected empty for unknown, got %q", got)
	}
}
```

- [ ] **Step 2: Run, FAIL**

- [ ] **Step 3: Implement**

```go
package shared

var swiftuiTags = map[string]string{
	"view":   "Group",
	"vstack": "VStack",
	"hstack": "HStack",
	"text":   "Text",
	"button": "Button",
}

// SwiftUITag returns the SwiftUI widget name for a symbolic NIR tag,
// or "" if the tag has no mapping (caller should emit a diagnostic).
func SwiftUITag(nirTag string) string {
	return swiftuiTags[nirTag]
}
```

- [ ] **Step 4: Run, PASS**

- [ ] **Step 5: Commit**

```bash
git add emit/shared/ && buckley commit --yes -min
```

---

### Task 9: iOS emitter

Walks NIR and prints SwiftUI source.

**Files:**
- Create: `~/work/gosx-native/emit/ios/emit.go`
- Create: `~/work/gosx-native/emit/ios/emit_test.go`

- [ ] **Step 1: Write the snapshot test**

```go
package ios

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/odvcencio/gosx/nir"
)

func TestEmitCounter(t *testing.T) {
	data, _ := os.ReadFile("../../testdata/expected/nir/counter.json")
	var mod nir.Module
	if err := json.Unmarshal(data, &mod); err != nil {
		t.Fatalf("unmarshal nir: %v", err)
	}
	var buf bytes.Buffer
	if err := Emit(&mod, &buf); err != nil {
		t.Fatalf("emit: %v", err)
	}

	expected, err := os.ReadFile("../../testdata/expected/emit/ios/Counter.swift")
	if os.IsNotExist(err) {
		_ = os.WriteFile("../../testdata/expected/emit/ios/Counter.swift", buf.Bytes(), 0644)
		t.Fatalf("wrote initial golden — inspect and re-run")
	}
	if err != nil {
		t.Fatalf("read expected: %v", err)
	}
	if !bytes.Equal(buf.Bytes(), expected) {
		t.Fatalf("emit mismatch.\nGot:\n%s\n\nExpected:\n%s", buf.String(), expected)
	}
}
```

- [ ] **Step 2: Run, FAIL**

- [ ] **Step 3: Implement `Emit`**

```go
package ios

import (
	"fmt"
	"io"
	"strings"

	"github.com/odvcencio/gosx/nir"
	"github.com/odvcencio/gosx-native/emit/shared"
)

// Emit writes SwiftUI source for every component in mod.
func Emit(mod *nir.Module, w io.Writer) error {
	fmt.Fprintln(w, "// Code generated by gsxnative. DO NOT EDIT.")
	fmt.Fprintln(w, "import GSXNativeKit")
	fmt.Fprintln(w, "import SwiftUI")
	fmt.Fprintln(w)
	for _, c := range mod.Components {
		if err := emitComponent(w, c); err != nil {
			return err
		}
	}
	return nil
}

func emitComponent(w io.Writer, c *nir.Component) error {
	fmt.Fprintf(w, "public struct %s: GSXComponent {\n", c.Name)
	if c.Props != nil {
		fmt.Fprintf(w, "    public struct Props {\n")
		for _, f := range c.Props.Fields {
			fmt.Fprintf(w, "        public var %s: %s\n", f.Name, f.Type)
		}
		fmt.Fprintf(w, "    }\n")
		fmt.Fprintf(w, "    public let props: Props\n")
	}
	for _, sig := range c.Signals {
		fmt.Fprintf(w, "    @GSXSignal private var %s: %s\n", sig.Name, sig.Type)
	}
	fmt.Fprintf(w, "\n    public init(props: Props) {\n")
	fmt.Fprintf(w, "        self.props = props\n")
	for _, sig := range c.Signals {
		fmt.Fprintf(w, "        self._%s = GSXSignal(wrappedValue: %s)\n", sig.Name, emitRxExpr(sig.Init))
	}
	fmt.Fprintf(w, "    }\n\n")
	fmt.Fprintf(w, "    public var body: some View {\n")
	fmt.Fprintf(w, "        %s\n", emitView(c.Body, 2))
	fmt.Fprintf(w, "    }\n}\n")
	return nil
}

func emitView(v nir.View, indent int) string {
	pad := strings.Repeat("    ", indent)
	switch n := v.(type) {
	case *nir.Element:
		tag := shared.SwiftUITag(n.Tag)
		// Elements with onTap handler → Button (M1 special case)
		for _, h := range n.Handlers {
			if h.Event == "tap" {
				label := emitChildrenInline(n.Children)
				return fmt.Sprintf(`Button("%s") { %s }`, label, emitRxBlock(h.Body))
			}
		}
		// vstack / hstack → wraps children
		var sb strings.Builder
		sb.WriteString(tag)
		sb.WriteString(" {\n")
		for _, c := range n.Children {
			sb.WriteString(pad)
			sb.WriteString("    ")
			sb.WriteString(emitView(c, indent+1))
			sb.WriteString("\n")
		}
		sb.WriteString(pad)
		sb.WriteString("}")
		return sb.String()
	case *nir.Text:
		return fmt.Sprintf(`Text("%s")`, n.Value)
	case *nir.ExprHole:
		return fmt.Sprintf(`Text("\(%s)")`, emitRxExpr(&n.Expr))
	}
	return "/* unsupported */"
}

func emitChildrenInline(children []nir.View) string {
	for _, c := range children {
		if t, ok := c.(*nir.Text); ok {
			return t.Value
		}
	}
	return ""
}

func emitRxExpr(e *nir.RxExpr) string {
	if e == nil {
		return ""
	}
	switch e.Kind {
	case "literal":
		return e.Literal.Value
	case "ref":
		return e.Ref
	case "binop":
		return fmt.Sprintf("%s %s %s", emitRxExpr(&e.BinOp.Left), e.BinOp.Op, emitRxExpr(&e.BinOp.Right))
	case "call":
		args := make([]string, len(e.Call.Args))
		for i := range e.Call.Args {
			args[i] = emitRxExpr(&e.Call.Args[i])
		}
		return fmt.Sprintf("%s(%s)", e.Call.Callee, strings.Join(args, ", "))
	}
	return ""
}

func emitRxBlock(b nir.RxBlock) string {
	parts := make([]string, len(b.Stmts))
	for i, s := range b.Stmts {
		switch s.Kind {
		case "expr":
			parts[i] = emitRxExpr(s.Expr)
		case "signal_set":
			parts[i] = fmt.Sprintf("%s = %s", s.Target, emitRxExpr(s.Value))
		}
	}
	return strings.Join(parts, "; ")
}
```

- [ ] **Step 4: Run, inspect golden**

Open `testdata/expected/emit/ios/Counter.swift` after the first run. It should look very close to the spec's example emit (modulo formatting). If wrong, fix `Emit` — don't adjust the golden.

- [ ] **Step 5: Iterate to PASS**

- [ ] **Step 6: Commit**

```bash
git add emit/ testdata/expected/emit/ && buckley commit --yes -min
```

---

### Task 10: Diagnostics infrastructure

Minimal diagnostic type used by lower and emit. Full E#### system lands in M8.

**Files:**
- Create: `~/work/gosx-native/internal/diagnostics/diagnostics.go`
- Create: `~/work/gosx-native/internal/diagnostics/diagnostics_test.go`

- [ ] **Step 1: Test**

```go
package diagnostics

import "testing"

func TestFormatHumanReadable(t *testing.T) {
	d := Diagnostic{
		Code:     "E2103",
		Severity: SeverityError,
		Span:     Span{File: "foo.gsx", StartLine: 42, StartCol: 5},
		Message:  "missing per-target implementation",
		Help:     "provide an Android implementation",
	}
	got := d.String()
	want := "error[E2103]: missing per-target implementation\n  --> foo.gsx:42:5\nhelp: provide an Android implementation\n"
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}
```

- [ ] **Step 2-4: Implement to pass, commit**

```go
package diagnostics

import (
	"fmt"
	"strings"
)

type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
	SeverityInfo    Severity = "info"
)

type Span struct {
	File      string
	StartLine int
	StartCol  int
}

type Diagnostic struct {
	Code     string
	Severity Severity
	Span     Span
	Message  string
	Help     string
}

func (d Diagnostic) String() string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "%s[%s]: %s\n", d.Severity, d.Code, d.Message)
	fmt.Fprintf(&sb, "  --> %s:%d:%d\n", d.Span.File, d.Span.StartLine, d.Span.StartCol)
	if d.Help != "" {
		fmt.Fprintf(&sb, "help: %s\n", d.Help)
	}
	return sb.String()
}
```

```bash
git add internal/diagnostics/ && buckley commit --yes -min
```

---

### Task 11: CLI — `compile` and `emit` subcommands

**Files:**
- Create: `~/work/gosx-native/cmd/gsxnative/main.go`
- Create: `~/work/gosx-native/cmd/gsxnative/compile.go`
- Create: `~/work/gosx-native/cmd/gsxnative/emit.go`

- [ ] **Step 1: Test the CLI end-to-end (table-driven)**

```go
// cmd/gsxnative/main_test.go
package main

import (
	"bytes"
	"os/exec"
	"strings"
	"testing"
)

func TestCompileCounterPrintsNIR(t *testing.T) {
	cmd := exec.Command("go", "run", ".", "compile", "../../testdata/corpus/swift/counter.swift.gsx")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		t.Fatalf("run: %v", err)
	}
	if !strings.Contains(out.String(), `"name": "Counter"`) {
		t.Fatalf("expected Counter in NIR JSON, got:\n%s", out.String())
	}
}

func TestEmitIOSCounterPrintsSwift(t *testing.T) {
	cmd := exec.Command("go", "run", ".", "emit", "ios", "../../testdata/corpus/swift/counter.swift.gsx")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		t.Fatalf("run: %v", err)
	}
	if !strings.Contains(out.String(), "struct Counter: GSXComponent") {
		t.Fatalf("expected Counter struct in emitted Swift, got:\n%s", out.String())
	}
}
```

- [ ] **Step 2: Run, FAIL** (no `main.go`)

- [ ] **Step 3: Implement `main.go` dispatch**

```go
package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: gsxnative <compile|emit> ...")
		os.Exit(2)
	}
	switch os.Args[1] {
	case "compile":
		if err := runCompile(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "emit":
		if err := runEmit(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand: %s\n", os.Args[1])
		os.Exit(2)
	}
}
```

- [ ] **Step 4: Implement `compile.go`**

```go
package main

import (
	"encoding/json"
	"fmt"
	"os"

	gotreesitter "github.com/odvcencio/gotreesitter"
	"github.com/odvcencio/gosx-native/grammar"
	"github.com/odvcencio/gosx-native/lower/swift"
)

func runCompile(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: gsxnative compile <file.swift.gsx>")
	}
	src, err := os.ReadFile(args[0])
	if err != nil {
		return err
	}
	lang, err := grammar.SwiftGSXLanguage()
	if err != nil {
		return err
	}
	parser := gotreesitter.NewParser(lang)
	defer parser.Release()
	tree, err := parser.Parse(src)
	if err != nil {
		return err
	}
	defer tree.Release()
	mod, err := swift.Lower(tree.Root(), src, lang)
	if err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(mod)
}
```

- [ ] **Step 5: Implement `emit.go`**

```go
package main

import (
	"fmt"
	"os"

	gotreesitter "github.com/odvcencio/gotreesitter"
	"github.com/odvcencio/gosx-native/emit/ios"
	"github.com/odvcencio/gosx-native/grammar"
	"github.com/odvcencio/gosx-native/lower/swift"
)

func runEmit(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: gsxnative emit <ios|android> <file.swift.gsx>")
	}
	target, file := args[0], args[1]
	src, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	lang, err := grammar.SwiftGSXLanguage()
	if err != nil {
		return err
	}
	parser := gotreesitter.NewParser(lang)
	defer parser.Release()
	tree, _ := parser.Parse(src)
	defer tree.Release()
	mod, err := swift.Lower(tree.Root(), src, lang)
	if err != nil {
		return err
	}
	switch target {
	case "ios":
		return ios.Emit(mod, os.Stdout)
	default:
		return fmt.Errorf("unknown target: %s (M1 supports: ios)", target)
	}
}
```

- [ ] **Step 6: Run tests, PASS**

- [ ] **Step 7: Commit**

```bash
git add cmd/ && buckley commit --yes -min
```

---

### Task 12: GSXNativeKit — `@GSXSignal` propertyWrapper

**Files:**
- Create: `~/work/gosx-native/runtime/ios/Package.swift`
- Create: `~/work/gosx-native/runtime/ios/Sources/GSXNativeKit/GSXSignal.swift`
- Create: `~/work/gosx-native/runtime/ios/Sources/GSXNativeKit/GSXComponent.swift`
- Create: `~/work/gosx-native/runtime/ios/Tests/GSXNativeKitTests/GSXSignalTests.swift`

- [ ] **Step 1: Create `Package.swift`**

```swift
// swift-tools-version:5.10
import PackageDescription

let package = Package(
    name: "GSXNativeKit",
    platforms: [.iOS(.v17), .macOS(.v14)],
    products: [
        .library(name: "GSXNativeKit", targets: ["GSXNativeKit"]),
    ],
    targets: [
        .target(name: "GSXNativeKit"),
        .testTarget(name: "GSXNativeKitTests", dependencies: ["GSXNativeKit"]),
    ]
)
```

- [ ] **Step 2: Write the failing test**

```swift
// Tests/GSXNativeKitTests/GSXSignalTests.swift
import XCTest
import SwiftUI
@testable import GSXNativeKit

final class GSXSignalTests: XCTestCase {
    func testSignalReadWrite() {
        // GSXSignal wraps @State; we can exercise the wrappedValue path directly.
        var s = GSXSignal(wrappedValue: 5)
        XCTAssertEqual(s.wrappedValue, 5)
        s.wrappedValue = 7
        XCTAssertEqual(s.wrappedValue, 7)
    }
}
```

- [ ] **Step 3: Run, FAIL** (`GSXSignal` doesn't exist)

```bash
cd ~/work/gosx-native/runtime/ios && swift test
```

- [ ] **Step 4: Implement `GSXSignal`**

```swift
// Sources/GSXNativeKit/GSXSignal.swift
import SwiftUI

/// GSXSignal exposes gosx-flavored reactive state on top of SwiftUI's @State.
/// Generated code uses @GSXSignal so the framework owns the API surface.
@propertyWrapper
public struct GSXSignal<T>: DynamicProperty {
    @State private var value: T

    public init(wrappedValue: T) {
        self._value = State(initialValue: wrappedValue)
    }

    public var wrappedValue: T {
        get { value }
        nonmutating set { value = newValue }
    }

    public var projectedValue: Binding<T> {
        $value
    }
}
```

- [ ] **Step 5: Implement `GSXComponent`**

```swift
// Sources/GSXNativeKit/GSXComponent.swift
import SwiftUI

/// Marker protocol for gosx-native generated components. Used by tooling
/// (inspector, debugger, hot-reload). Not load-bearing for execution.
public protocol GSXComponent: View {
    associatedtype Props
    var props: Props { get }
}
```

- [ ] **Step 6: Run, PASS**

- [ ] **Step 7: Commit**

```bash
cd ~/work/gosx-native && git add runtime/ios/ && buckley commit --yes -min
```

---

### Task 13: Demo Xcode project that hosts the generated Counter

This task uses **xcodegen** to produce the Xcode project from a YAML spec — interactive Xcode steps don't fit a scripted plan.

**Files:**
- Create: `~/work/gosx-native/examples/counter-ios/project.yml` (xcodegen spec)
- Create: `~/work/gosx-native/examples/counter-ios/CounterDemo/CounterDemoApp.swift`
- Create: `~/work/gosx-native/examples/counter-ios/CounterDemo/Generated/Counter.swift` (regenerated by pipeline; checked in)
- Generated by xcodegen: `~/work/gosx-native/examples/counter-ios/CounterDemo.xcodeproj/`

- [ ] **Step 1: Install xcodegen if missing**

```bash
which xcodegen || brew install xcodegen
```

- [ ] **Step 2: Create `project.yml`**

```yaml
# examples/counter-ios/project.yml
name: CounterDemo
options:
  bundleIdPrefix: com.gosx.native.demo
  deploymentTarget:
    iOS: "17.0"
  createIntermediateGroups: true

packages:
  GSXNativeKit:
    path: ../../runtime/ios

targets:
  CounterDemo:
    type: application
    platform: iOS
    sources:
      - CounterDemo
    dependencies:
      - package: GSXNativeKit
        product: GSXNativeKit
    settings:
      base:
        INFOPLIST_KEY_UIApplicationSceneManifest_Generation: YES
        INFOPLIST_KEY_UILaunchScreen_Generation: YES
        INFOPLIST_KEY_UISupportedInterfaceOrientations_iPhone: "UIInterfaceOrientationPortrait"
        TARGETED_DEVICE_FAMILY: "1,2"
```

- [ ] **Step 3: Create `CounterDemoApp.swift`**

```swift
import SwiftUI
import GSXNativeKit

@main
struct CounterDemoApp: App {
    var body: some Scene {
        WindowGroup {
            Counter(props: .init(start: 0))
        }
    }
}
```

- [ ] **Step 4: Generate Counter.swift via the pipeline**

```bash
cd ~/work/gosx-native
mkdir -p examples/counter-ios/CounterDemo/Generated
go run ./cmd/gsxnative emit ios testdata/corpus/swift/counter.swift.gsx \
  > examples/counter-ios/CounterDemo/Generated/Counter.swift
```

Verify the file is valid Swift by inspecting it. If it looks wrong, fix the emitter — don't hand-edit Generated/.

- [ ] **Step 5: Generate the Xcode project**

```bash
cd ~/work/gosx-native/examples/counter-ios && xcodegen generate
```

Expected output: `CounterDemo.xcodeproj/` directory created.

- [ ] **Step 6: Build the project**

```bash
cd ~/work/gosx-native/examples/counter-ios
xcodebuild -project CounterDemo.xcodeproj -scheme CounterDemo \
  -destination "platform=iOS Simulator,name=$${IOS_SIMULATOR_NAME:-iPhone 15}" \
  build
```

Expected: BUILD SUCCEEDED. If FAIL, the most likely causes:
- `GSXNativeKit` path not resolved → check the `packages:` section in `project.yml`, ensure `../../runtime/ios` exists relative to `project.yml`
- Generated/Counter.swift has a typo from emit → fix the emitter, regenerate, do not hand-edit
- Deployment target mismatch → confirm `deploymentTarget.iOS` in `project.yml`

- [ ] **Step 7: Commit (project.yml + Generated/, NOT the .xcodeproj)**

The `.xcodeproj` is regenerated by xcodegen on every clean checkout, so don't commit it. Add it to `.gitignore`:

```bash
cd ~/work/gosx-native
echo "examples/counter-ios/CounterDemo.xcodeproj/" >> .gitignore
git add examples/counter-ios/project.yml \
        examples/counter-ios/CounterDemo/CounterDemoApp.swift \
        examples/counter-ios/CounterDemo/Generated/Counter.swift \
        .gitignore
buckley commit --yes -min
```

---

### Task 14: End-to-end test

A Go test that runs the full pipeline and confirms the demo builds. Skipped on Linux runners; runs on macOS.

**Files:**
- Create: `~/work/gosx-native/test/e2e/counter_test.go`

- [ ] **Step 1: Write the e2e test**

The test compares freshly-emitted source against the checked-in `Generated/Counter.swift` to verify they match (catches emit drift), then builds the demo. It does NOT mutate the tracked file.

```go
//go:build smoke && darwin

package e2e

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestCounterEndToEnd(t *testing.T) {
	repoRoot, _ := filepath.Abs("../..")

	// 1. Regenerate Counter.swift in-memory and confirm it matches the
	//    checked-in Generated/Counter.swift (catches emit drift).
	out, err := exec.Command("go", "run",
		filepath.Join(repoRoot, "cmd/gsxnative"),
		"emit", "ios",
		filepath.Join(repoRoot, "testdata/corpus/swift/counter.swift.gsx"),
	).Output()
	if err != nil {
		t.Fatalf("emit: %v", err)
	}
	tracked := filepath.Join(repoRoot, "examples/counter-ios/CounterDemo/Generated/Counter.swift")
	expected, err := os.ReadFile(tracked)
	if err != nil {
		t.Fatalf("read tracked Generated/Counter.swift: %v", err)
	}
	if !bytes.Equal(out, expected) {
		t.Fatalf("emit drift detected: regenerated Counter.swift differs from %s.\n"+
			"Run `make demo` and commit the regenerated file if the change is intentional.", tracked)
	}

	// 2. Generate the Xcode project (xcodegen reads project.yml).
	gen := exec.Command("xcodegen", "generate")
	gen.Dir = filepath.Join(repoRoot, "examples/counter-ios")
	gen.Stdout = os.Stdout
	gen.Stderr = os.Stderr
	if err := gen.Run(); err != nil {
		t.Fatalf("xcodegen: %v", err)
	}

	// 3. Build the iOS demo.
	build := exec.Command("xcodebuild",
		"-project", filepath.Join(repoRoot, "examples/counter-ios/CounterDemo.xcodeproj"),
		"-scheme", "CounterDemo",
		"-destination", "platform=iOS Simulator,name=" + simName(),
		"build",
	)
	build.Stdout = os.Stdout
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		t.Fatalf("xcodebuild: %v", err)
	}
}

// simName returns the iOS Simulator name to build against.
// Override via the IOS_SIMULATOR_NAME environment variable so CI can
// follow whatever destination the runner provides without a code change.
func simName() string {
	if n := os.Getenv("IOS_SIMULATOR_NAME"); n != "" {
		return n
	}
	return "iPhone 15"
}
```

- [ ] **Step 2: Run on macOS**

```bash
cd ~/work/gosx-native && go test -tags smoke -v ./test/e2e/...
```

Expected: PASS. If FAIL, this is real-world validation that the chain holds together — fix whatever broke (most likely emit drift) and iterate.

- [ ] **Step 3: Commit**

```bash
git add test/e2e/ && buckley commit --yes -min
```

---

### Task 15: CI workflow + GitHub remote

GitHub Actions: Linux job for unit tests + lint, macOS job for build smoke. Also creates the GitHub remote and pushes.

**Files:**
- Create: `~/work/gosx-native/.github/workflows/ci.yml`

- [ ] **Step 1: Verify `go test ./...` passes on Linux locally before adding CI**

```bash
cd ~/work/gosx-native && go test ./... -v
```

Expected: all tests PASS. gotreesitter is pure-Go (no CGo), so this should work cleanly on any Go-supporting OS. If something fails because of platform-specific code, fix locally before adding the CI workflow that depends on this passing.

- [ ] **Step 2: Write the workflow**

```yaml
name: CI

on:
  push:
    branches: [main]
  pull_request:

jobs:
  unit-linux:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.24'
      - name: Test
        run: go test -race ./...
      - name: Vet
        run: go vet ./...
      - name: Format check
        run: |
          unformatted=$(gofmt -l .)
          if [ -n "$unformatted" ]; then
            echo "Unformatted files:"
            echo "$unformatted"
            exit 1
          fi

  build-smoke-macos:
    runs-on: macos-14
    needs: unit-linux
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.24'
      - name: Select Xcode
        run: sudo xcode-select -s /Applications/Xcode_15.4.app
      - name: E2E
        run: go test -tags smoke -v ./test/e2e/...
```

- [ ] **Step 3: Create the GitHub remote**

```bash
cd ~/work/gosx-native
gh repo create odvcencio/gosx-native --public --source=. --description "The mobile counterpart to gosx — same component model, native iOS/Android targets."
```

If `gh` is not available or authenticated, prompt the user to create the remote manually and run `git remote add origin git@github.com:odvcencio/gosx-native.git` instead.

- [ ] **Step 4: Push and verify CI green**

```bash
git add .github/ && buckley commit --yes -min
git push -u origin main
```

Expected: both jobs go green on first push. If they don't, fix locally first — don't iterate by pushing.

---

### Task 16: M1 milestone summary commit

Final pass: refresh the README to note M1 done.

**Files:**
- Modify: `~/work/gosx-native/README.md`

- [ ] **Step 1: Update README status line**

Change "M1 — vertical slice in progress" to "M1 — vertical slice complete. Counter demo runs on iOS Simulator. M2 (Android) next."

- [ ] **Step 2: Verify nothing is broken**

```bash
cd ~/work/gosx-native && make test && make smoke  # smoke runs only on macOS
```

Both pass. Use @superpowers:verification-before-completion before marking M1 done.

- [ ] **Step 3: Commit**

```bash
git add README.md && buckley commit --yes -min
```

---

## Subsequent milestones (high-level)

Each gets its own plan document when its turn comes.

### M2 — Android target (Kotlin emit + Compose runtime)
Mirror of M1 on Android. Same Counter, same NIR, new emit (`emit/android`), new runtime (`runtime/android`), Gradle build smoke. The Counter Compose source emitter and runtime stubs are in place; `docs/superpowers/plans/2026-05-05-android-counter-build-smoke.md` tracks the Android debug APK compile proof and the remaining emulator/AAR work.

### M3 — `go+gsx` front end
Lower the existing gosx Go-side IR (or refactor `gosx/ir/lower.go` to emit NIR directly) so a `.gsx` file authored in Go can target iOS and Android. Counter authored in Go produces equivalent iOS and Android apps.

### M4 — Routing + navigation
File-based routes lower into NIR's `Route` table. Runtime `GSXRouter` wraps `NavigationStack` / `NavHost`. Demo: a two-screen app with push navigation.

### M5 — Data + actions + bridge
Generated typed data/action client (HTTP/JSON). Bridge envelope. Auth (token storage, refresh). Demo: app authenticates, fetches a list, mutates with an optimistic update.

### M6 — Engine surface + Scene3D Metal backend
`GSXSurface` (Metal-backed CAMetalLayer view), `SceneRenderer` interface in gosx, Metal implementation, conformance harness against existing WebGL/WebGPU output. Demo: rotating cube.

### M7 — Project scaffolding (`gsxnative init`)
Templates (`basic`, `tabs`, `auth`, `scene3d`), `init` command, dev workflow with file-watcher + platform hot-reload integration.

### M8 — Full NIR rename in gosx + diagnostic polish
Migrate all of `gosx/ir/` into `gosx/nir/` (or evolve in place). Stable E#### diagnostic codes. JSON diagnostic mode. Per-language opaque payload support (`//gosx:native <target>` directive).

### M9 — Vulkan backend (Android Scene3D)
Mirror of M6 on Android. Closes the Scene3D parity claim.

### M10+ — v1.x accretion
Native module wrappers (camera, location, biometrics, etc.), persistent disk cache, dev-mode NIR-binary patch hot-reload, LSP IDE integrations. Each ships independently per the spec roadmap.

---

## Notes for the implementer

- **Start with Task 1.** If grammargen Swift round-trip doesn't work, stop and surface it. Everything downstream depends on it.
- **The lowerer (Task 7) is the longest single task.** Pace yourself, write small tests, don't try to lower constructs that aren't in the Counter corpus.
- **Resist scope creep into M2/M3.** This plan deliberately ships only iOS / only `swift+gsx` / only Counter. Adding Android or Go-source authoring "while you're at it" will slip M1.
- **Commits are frequent and small.** Every task ends in a commit. Use `buckley commit --yes -min` (no `-graft` — gosx-native is a standard public GitHub repo, not graft).
- **Verification before completion is non-negotiable.** Don't mark a task done without running the test/build it specifies. Use @superpowers:verification-before-completion as the discipline check.
