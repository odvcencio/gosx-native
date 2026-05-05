---
title: gosx-native — design
date: 2026-05-04
status: proposed
project: gosx-native
authors: odvcencio
---

# gosx-native — design

## Summary

gosx-native is the mobile counterpart to gosx. Where gosx is the React-like web framework, gosx-native is the React-Native-like mobile framework: same component model, same reactive primitives, same scene graph, different rendering targets.

Authors write in any of three sibling source languages — `go+gsx`, `swift+gsx`, `kotlin+gsx` — and gosx-native lowers all three to a shared target-agnostic IR (NIR), then emits idiomatic native source: SwiftUI for iOS, Jetpack Compose for Android. The existing gosx web emitter consumes the same NIR. A single `.gsx` file can produce a web bundle, an iOS archive, and an Android bundle from one build.

The framework owns at least 80% of any production app's surface (reactivity, navigation, data, auth, scene rendering, engine surfaces). The remaining 20% is structured escape hatches — source-level (`//gosx:native <target>` blocks) and project-level (hand-written native code via named extension points). The framework accretes coverage over time so the escape-hatch fraction shrinks per release.

## Goals

- **Cross-target parity by construction.** A capability that exists on web exists on iOS and Android, or it doesn't exist at all. Mismatches surface as compile-time diagnostics, never as silent no-ops.
- **Author once, ship anywhere.** A `.gsx` file written for web ships to mobile without modification when its primitives are portable; per-target overrides drop in cleanly when they aren't.
- **Idiomatic native output.** Emitted Swift looks like SwiftUI a Swift dev would write. Emitted Kotlin looks like Compose a Kotlin dev would write. No alien transpiler conventions.
- **RN-class deftness.** One-command init, sub-second source regen, working hot-reload from day one, clear diagnostics with source spans, easy native-module bridging.
- **Framework owns the surface.** Generated apps depend on a small, opinionated framework (`GSXNativeKit`, `gsxnative.aar`) that wraps platform reactivity and exposes a stable gosx-flavored API.

## Non-goals (v1)

- Hubs, CRDT, websocket presence, offline-first persistence. Deferred to v1.x.
- Server-side primitives on the device. Mobile is a pure client; data loaders and actions cross the wire as typed RPC, not as Go code.
- Custom scripting runtime on device. Production apps run native Swift/Kotlin only — no bytecode interpreter ships in release builds.
- Replacing Xcode or Android Studio. We integrate with them, not displace them.
- Desktop targets (Mac, Linux, Windows native). gosx-desktop owns the desktop story; gosx-native is mobile.

## Background

**gosx** parses `.gsx` files (Go with JSX-like markup) using a tree-sitter grammar built on **gotreesitter** (pure-Go tree-sitter runtime). Parsed CSTs lower into a flat-array IR (`gosx/ir/`), which two emitters consume: the HTML renderer for server output, and the island bytecode compiler for client interactivity. The web client runs a small WASM VM that interprets island bytecode against a signal-based reactive system.

**gotreesitter** ships pure-Go grammars composable via `grammargen.ExtendGrammar(base, customize)`. As of PR #58 (merged 2026-05-04), the registry includes programmatic Go DSLs for Go, Swift, and Kotlin grammars — plus an `external_scanner_binding` package for attaching external scanners to grammargen-built grammars. The mechanism gosx uses to make `go+gsx` now extends to Swift and Kotlin without prerequisites.

**Scope of this spec.** gosx-native v1: the framework, the compiler, the iOS and Android runtimes, the CLI, and the tooling. Coordinated changes in gosx (NIR extraction, `SceneRenderer` interface, `engine.Surface` abstraction) are in scope for design; their implementation lands in gosx itself.

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│ AUTHORING                                                                │
│   foo.gsx              Foo.swift+gsx          Foo.kt+gsx                 │
│   (Go + gsx markup)    (Swift + gsx markup)   (Kotlin + gsx markup)      │
└─────────┬────────────────────┬────────────────────────┬─────────────────┘
          ▼                    ▼                        ▼
┌─────────────────────────────────────────────────────────────────────────┐
│ PARSE  (gotreesitter + grammargen.ExtendGrammar — same mechanism x3)    │
│   ExtendGrammar(GoGrammar, gsx)   SwiftGrammar+gsx   KotlinGrammar+gsx  │
└─────────┬────────────────────┬────────────────────────┬─────────────────┘
          ▼                    ▼                        ▼
┌─────────────────────────────────────────────────────────────────────────┐
│ LOWER  (3 lowerers, all emit the same NIR)                              │
│   go-lower (gosx today)    swift-lower (new)         kotlin-lower (new) │
└──────────────────────────────────┬──────────────────────────────────────┘
                                   ▼
                  ┌────────────────────────────────────┐
                  │  NIR — target-agnostic IR          │
                  │  view tree, signals, computed,     │
                  │  handler bodies, slots, layout,    │
                  │  data-loader DECLS, action DECLS,  │
                  │  engine DECLS, scene.IR, route     │
                  │  table, capability requests        │
                  └────────────────┬───────────────────┘
                                   │
            ┌──────────────────────┼──────────────────────┐
            ▼                      ▼                      ▼
   ┌────────────────┐    ┌────────────────┐    ┌────────────────┐
   │ web emit       │    │ ios emit       │    │ android emit   │
   │ HTML + island  │    │ Swift+SwiftUI  │    │ Kotlin+Compose │
   │ bytecode       │    │ source         │    │ source         │
   │ (gosx today)   │    │ (gosx-native)  │    │ (gosx-native)  │
   └────────┬───────┘    └────────┬───────┘    └────────┬───────┘
            ▼                     ▼                     ▼
   ┌────────────────┐    ┌────────────────┐    ┌────────────────┐
   │ browser        │    │ GSXNativeKit   │    │ gsxnative.aar  │
   │ runtime        │    │ .framework     │    │ (Android lib)  │
   │ (gosx today)   │    │   • reactive   │    │   • reactive   │
   │                │    │   • nav        │    │   • nav        │
   │                │    │   • data clt   │    │   • data clt   │
   │                │    │   • bridge     │    │   • bridge     │
   │                │    │   • Metal      │    │   • Vulkan     │
   │                │    │     renderer   │    │     renderer   │
   └────────────────┘    └────────────────┘    └────────────────┘
```

### Repository ownership

- **`~/work/gosx`** — the React-like framework. Owns Go-side parsing, lowering to NIR, web emit, web runtime, server-side surfaces (data loaders, actions, auth, sessions).
- **`~/work/gotreesitter`** — pure-Go tree-sitter and grammargen. Owns the composable language grammars (Go, Swift, Kotlin).
- **`~/work/gosx-native`** (this project) — the React-Native-like framework. Owns the two new grammar extensions (`swift+gsx`, `kotlin+gsx`), the Swift and Kotlin lowerers, iOS/Android emitters, `GSXNativeKit.framework`, `gsxnative.aar`, the `gsxnative` CLI, project scaffolding, dev tooling.
- **`~/work/gosx-game`** (potential future extraction) — scene/game/sim/field. Both gosx and gosx-native consume it via the shared `SceneRenderer` interface and `scene.IR` contract. v1 plans for gosx-game in-place.

### Coordinated changes in gosx

These three changes land upstream in gosx because gosx itself benefits from them. gosx-native depends on them but does not own them.

1. **NIR extraction.** Rename `gosx/ir` to `gosx/nir`. Promote web-only details (HTML attribute names, browser event names) out of the IR and into the web emitter. The renamed package becomes the target-agnostic contract.
2. **`SceneRenderer` interface in `gosx/scene`.** The existing WebGL and WebGPU backends move behind a single Go interface so Metal and Vulkan backends plug in alongside.
3. **`engine.Surface` abstraction.** Replace `engine.Config.MountID` / `MountAttrs` (DOM-coupled) with a tagged `Surface` type that supports both DOM and native mounts.

Each change is small and additive on the gosx side. They unblock gosx-native without coupling it to gosx's release cycle.

### Long-term absorption

Per the original ask, gosx-native may fold into gosx as an optional module later. The repo split today buys independent iteration speed; folding becomes mechanical because the contract (NIR + `SceneRenderer` + `Surface`) holds either way.

## NIR — target-agnostic IR

NIR fully describes a renderable application without knowing which platform will render it. Every emitter consumes the same NIR. Every front-end produces it. NIR is the single thing that crosses from the authoring world to the platform world.

### Node categories

- **`Module`** — top-level: components, routes, data-loader declarations, action declarations, engine declarations, capability manifest, source-language tag.
- **`Component`** — name, typed props, slot signature, view-tree body.
- **View tree** — `Element`, `ComponentRef`, `Slot`, `Fragment`, `Conditional`, `Loop`, `ExprHole`. Tags are *symbolic* (`view`, `text`, `vstack`, `button`); emitters map symbolic tags to platform widgets. The Go lowerer normalizes HTML-named tags (`<div>`, `<span>`, `<a href>`) into the symbolic set, with a diagnostic when normalization is ambiguous.
- **Reactive primitives** — `SignalDecl{name, init}`, `ComputedDecl{name, body}`, `EffectDecl{deps, body}`. Each scoped to a component instance.
- **`Handler`** — named or anonymous body of statements, attached to symbolic events on view nodes (`onTap`, `onSubmit`, `onChange`).
- **`DataLoaderDecl` / `ActionDecl`** — name, typed params, typed return, transport hint (`http_json` in v1), URL/method (resolved by gosx routing). The implementation lives on the server; mobile sees the *declaration* and gets a typed client.
- **`EngineDecl`** — kind (`worker` | `surface` | `video`), capability requests, surface descriptor (`scene.IR` for 3D; `pixel_buffer` for raw 2D; `video` for video), bridge bindings.
- **`Route`** — path pattern, layout chain, component, params schema, data-loader bindings.

### The constrained expression language (`RxExpr`, `RxStmt`)

Reactive contexts (signal initializers, computed bodies, handler bodies, expression holes) accept only a portable subset:

- literals (number, string, bool, null, array, object)
- variable reference (signal, computed, prop, loop binding)
- signal mutation (`signal.set`, `signal.update`)
- arithmetic, comparison, boolean, string concat, ternary, `if`/`else`, `for`-of-array
- member access on typed structs/objects
- action and data-loader invocation (typed)
- platform-API calls — a curated namespace (`Date.now`, `Math.*`, `JSON.parse`) with equivalents on every target

This is gosx's existing island expression constraint, promoted to NIR's semantic boundary. Every target can produce native code for everything in the subset.

### The escape hatch — `//gosx:native <target>`

NIR's reactive expressions form a small sum type:

```
RxExpr =
  | Portable(...)                  // valid for ALL enabled targets
  | PerTarget(impls map[T]Opaque)  // explicit per-target native bodies
  | Single(target, Opaque)         // valid for exactly one target
```

The `//gosx:native <target>` directive triggers `PerTarget` or `Single`. Function-level form is the recommended style:

```go
//gosx:native swift
func formatDate(t Date) -> String { ... }

//gosx:native kotlin
fun formatDate(t: LocalDateTime): String { ... }

//gosx:native go
func formatDate(t time.Time) string { ... }
```

Inline-block form exists for one-liners but should be used sparingly:

```go
//gosx:native swift { Date.now.formatted(date: .abbreviated, time: .shortened) }
//gosx:native kotlin { LocalDateTime.now().format(...) }
formatted := derive(func() string { return formatDate(now) })
```

When a `PerTarget` declaration lacks an implementation for an enabled target, the compiler errors with a clear span and suggests either adding the missing target or replacing with portable code. Single-target apps (e.g., a `swift+gsx`-only project) bypass the constraint entirely — every reactive expression can be Swift because there's no other target to portability-check against.

### Per-language opaque payloads

For the small set of cases where source-language-specific code is legal in a single-target context — e.g., a Go-only library call in a server-component handler — NIR carries `OpaqueExpr{lang, source}`. The web emitter consumes Go opaque payloads; mobile emitters reject them with a span pointing at the source.

### Tagged source spans

Every NIR node carries `(file, byte_start, byte_end)`. Diagnostics, runtime errors (in dev builds), and tooling navigate back to the original `.gsx` line. This is non-negotiable for framework DX.

### Serialization

NIR is a Go struct in the `gosx/nir` package. JSON for tooling and inspection. A compact binary (extending the `GSX\x00` magic from gosx's island format with a new section type) for dev-mode hot-reload payloads. Production never reads serialized NIR — production reads compiled native source.

### Versioning

NIR carries a `version` field. Front-ends and emitters declare which versions they support. Minor versions add node kinds; major versions change semantics of existing kinds. Old emitters refuse new majors loudly; new emitters can opt to handle old majors.

## Front end — grammars and lowering

### Grammar extensions

All three follow the same recipe:

| Grammar | Base | Extension lives in | External scanner |
|---|---|---|---|
| `go+gsx` | `grammargen.GoGrammar()` | `gosx/grammar.go` (existing) | `gosx/gsx_attr_scanner.go` (existing) |
| `swift+gsx` | `grammargen.SwiftGrammar()` | `gosx-native/grammar/swift.go` (new) | `gosx-native/grammar/swift_gsx_scanner.go` (new) |
| `kotlin+gsx` | `grammargen.KotlinGrammar()` | `gosx-native/grammar/kotlin.go` (new) | `gosx-native/grammar/kotlin_gsx_scanner.go` (new) |

Each extension calls `grammargen.ExtendGrammar(<base>, customize)` with the same `jsx_*` rules: `jsx_element`, `jsx_self_closing_element`, `jsx_attribute`, `jsx_attribute_expression`, `jsx_text`, `jsx_fragment`, `jsx_spread_attribute`, plus a `jsx_*` alternative on the host's expression rule. Customization is structurally identical across grammars; only host-rule names differ.

External scanners port gosx's `gsx_attr_scanner.go` logic — handle the `jsx_attribute_expression` and `jsx_text` boundary tokens — coexisting with the host scanner's state machine. Swift's implicit semicolons and string interpolation, and Kotlin's same, require careful boundary handling but are bounded in scope.

### File extensions

- `.gsx` — Go (existing convention; no breaking change)
- `.swift.gsx` — Swift
- `.kt.gsx` — Kotlin

Distinct extensions matter for editor syntax highlighting, LSP language ID dispatch, build tooling discovery, and grep readability. The CLI infers source language from extension; no auto-detection.

### Lowerers

```
gosx/lower/go/             # existing gosx/ir/lower.go, refactored to emit nir.Module
gosx-native/lower/swift/   # new
gosx-native/lower/kotlin/  # new
```

Each lowerer takes a `*gotreesitter.Tree` plus source bytes and emits a `*nir.Module`. Shared JSX-handling lives in `nir/lower/jsx` since the JSX subtree shape is identical across all three grammars. Host-language statements (function declarations, props struct/data-class declarations, opaque blocks) are handled per-language.

### Component conventions per host language

| Language | Convention |
|---|---|
| Go | `func ComponentName(props PropsType) Node { return <...> }` |
| Swift | `func ComponentName(props: PropsType) -> Node { return <...> }` |
| Kotlin | `fun ComponentName(props: PropsType): Node { return <...> }` |

Capitalized name plus return-type-of-`Node` plus body returns markup → it's a component. No new annotation needed.

### Server components on mobile

By default, a server-rendered component (no `//gosx:island` marker; body returns a view tree) is treated as client-rendered on mobile. The view tree shape transfers; it just runs on the device instead of the server. The compiler emits a diagnostic only when the body contains opaque server-only code (e.g., `database/sql`, `net/http` server handlers, anything in a `gosx:server` namespace).

The `//gosx:island` marker is a no-op on mobile. Every component is client-runtime on mobile by definition. The marker remains legal so a single `.gsx` file works unchanged across web and mobile.

### Repo layout (gosx-native top level)

```
gosx-native/
  grammar/        # grammar extensions for Swift, Kotlin
  lower/          # lowerers (swift, kotlin)
  emit/           # emitters (ios, android)
  runtime/        # framework sources (NOT Go code)
    ios/          # GSXNativeKit.framework sources (Swift)
    android/      # gsxnative.aar sources (Kotlin)
  scaffold/       # project templates
  cmd/gsxnative/  # CLI entry point
  internal/       # diagnostics, config, shared infra
  examples/       # demo apps
  docs/           # this spec lives under docs/superpowers/specs
```

`runtime/ios` and `runtime/android` hold actual Swift and Kotlin source — versioned with the Go transpiler so they evolve in lockstep. The CLI build embeds them as resources for `init` to copy out.

## Back end — emit and framework runtimes

### Generated code shape

A trivial counter authored once:

```go
// counter.gsx (Go)
//gosx:island
func Counter(props struct{ Start int }) Node {
    count := signal.New(props.Start)
    return <vstack>
        <button onTap={func(){ count.Set(count.Get() - 1) }}>-</button>
        <text>{count.Get()}</text>
        <button onTap={func(){ count.Set(count.Get() + 1) }}>+</button>
    </vstack>
}
```

Emits to Swift+SwiftUI:

```swift
// Counter.swift (generated)
import GSXNativeKit
import SwiftUI

public struct Counter: GSXComponent {
    public struct Props { public var start: Int }
    public let props: Props
    @GSXSignal private var count: Int

    public init(props: Props) {
        self.props = props
        self._count = GSXSignal(wrappedValue: props.start)
    }

    public var body: some View {
        VStack {
            Button("-") { count = count - 1 }
            Text("\(count)")
            Button("+") { count = count + 1 }
        }
    }
}
```

And to Kotlin+Compose:

```kotlin
// Counter.kt (generated)
package app

import com.gosx.native.*
import androidx.compose.foundation.layout.*
import androidx.compose.material3.*
import androidx.compose.runtime.*

data class CounterProps(val start: Int)

@GSXComponent
@Composable
fun Counter(props: CounterProps) {
    var count by gsxSignal(props.start)
    Column {
        Button(onClick = { count -= 1 }) { Text("-") }
        Text("$count")
        Button(onClick = { count += 1 }) { Text("+") }
    }
}
```

A SwiftUI/Compose dev opening the output sees normal native code that uses a small framework. No NIR. No bytecode loader. No interpreter shim.

### Symbolic tag mapping (v1)

| NIR tag | SwiftUI | Compose |
|---|---|---|
| `view` | `Group` / `ZStack` | `Box` |
| `vstack` | `VStack` | `Column` |
| `hstack` | `HStack` | `Row` |
| `text` | `Text` | `Text` |
| `image` | `Image` | `Image` |
| `button` | `Button` | `Button` |
| `textfield` | `TextField` | `OutlinedTextField` |
| `scrollview` | `ScrollView` | `Modifier.verticalScroll` wrapper |
| `list` | `List` (or `LazyVStack` for perf) | `LazyColumn` |
| `lazygrid` | `LazyVGrid` | `LazyVerticalGrid` |
| `divider` | `Divider` | `HorizontalDivider` |
| `spacer` | `Spacer` | `Spacer(Modifier.weight(1f))` |
| `link` | `Link` (URL) / `NavigationLink` (route) | `ClickableText` / `NavController.navigate` |
| `surface` | engine-managed view (Metal) | engine-managed view (Vulkan) |

HTML-leaning tags from Go `.gsx` (`<div>`, `<span>`, `<p>`, `<a>`) normalize at lower-time. Anything that doesn't normalize cleanly produces a diagnostic.

### Framework runtime libraries

`GSXNativeKit.framework` (Swift package) and `gsxnative.aar` (Android library) both provide:

1. **Reactive primitives** — `GSXSignal<T>` / `gsxSignal(initial)` / `gsxComputed { ... }` / `gsxEffect(deps) { ... }`. Each wraps the platform's native reactivity (`@State`/`@Observable` on iOS; `mutableStateOf`/`derivedStateOf`/`LaunchedEffect` on Android). The wrappers exist so generated code calls a stable gosx-shaped API and the framework can layer behavior (hot-reload tracing, devtools, time-travel debug) without touching emit.
2. **Component protocol/annotation** — `GSXComponent` marker (Swift protocol with associated `Props`; Kotlin annotation). Used by tooling (inspector, debugger, hot-reload).
3. **Slot machinery** — typed `@ViewBuilder` lambdas (Swift) / `@Composable () -> Unit` lambdas (Kotlin).
4. **Navigation router** — wraps `NavigationStack` / `NavHost`. Consumes the NIR route table at boot; exposes a `GSXRouter` singleton.
5. **Data client + action client** — generated typed clients for every `DataLoaderDecl` / `ActionDecl` in NIR. Handles auth header attach, JSON codecs, retries, in-memory caching with explicit invalidation.
6. **Native bridge** — typed RPC envelope adapted from `gosx/desktop/bridge/router.go`. Same Go-side service registration; native HTTP/WS transport.
7. **Engine surface API** — `GSXSurface` (UIView-backed `CAMetalLayer` on iOS; `SurfaceView`-backed Vulkan on Android). The runtime owns GPU lifecycle.
8. **Scene renderer** — Metal backend (iOS) and Vulkan backend (Android) implementing the `SceneRenderer` interface. Consumes `scene.IR` directly. Same data, different GPU.

### Generated project layout

`gsxnative init ios myapp/` produces:

```
myapp/ios/
  MyApp.xcodeproj/
  Package.swift                     # SwiftPM manifest, depends on GSXNativeKit
  Sources/MyApp/
    GSXAppMain.swift                # @main App entry, mounts router
    Generated/                      # NEVER edit by hand; gsxnative regenerates wholesale
      Counter.swift
      Profile.swift
      Routes.swift
      DataClient.swift
      ActionClient.swift
      ...
    Native/                         # hand-written escape hatches live here
    Resources/
      Assets.xcassets
      gsx-bundle.json               # capability manifest, server URLs per env
  Tests/
  .gsxnative/
    config.json                     # build flags, signing pointer
```

Symmetric for Android. The `Generated/` directory is regenerated wholesale on every build — strict, no exceptions, no drift. Hand-written code lives in `Native/` (or any sibling directory). Generated code freely references hand-written code; the boundary is unambiguous.

### Where the emit code lives

```
gosx-native/emit/
  ios/         # Swift source emitter
  android/     # Kotlin source emitter
  shared/      # tag-mapping tables, identifier mangling, common printer infra
```

Both emitters consume `*nir.Module` and write a tree of source files to a target directory. Pure functions modulo I/O. Trivially testable via golden snapshots.

## Binding principles

### Cross-target portability

Anything gosx exposes as a *capability* (component, signal, route, action, engine, scene3d primitive, physics body, animation track, postFX pass, material) is described by NIR plus a small set of side IRs (`scene.IR`, `engine.Config`, the route table). A new render target is **conformant** if it implements the full surface of those IRs. Targets cannot silently no-op a primitive — unsupported features produce compile-time diagnostics. Parity is a property of the architecture, not a goal we keep restating.

### Scene3D — full surface, conformance-tested

`scene.IR` carries camera, lights, nodes, meshes, materials (PBR), animations, physics declarations, postFX passes. Four backends consume it through the new `SceneRenderer` interface:

| Target | Renderer | Lives in |
|---|---|---|
| Web (default) | WebGL backend | gosx |
| Web (modern) | WebGPU backend (Go→WASM) | gosx |
| Desktop (Windows) | WebView2 reuses web stack | gosx-desktop |
| iOS | Metal backend | gosx-native/runtime/ios |
| Android | Vulkan backend | gosx-native/runtime/android |

Mobile renderer scope for v1 is **full `scene.IR` surface, not a subset**. When gosx ships a new scene.IR feature (recent webgpu PBR + sequenced animation work, for instance), the mobile backends gain a corresponding implementation in the same release window. Failure mode if we slip: a clear capability-mismatch diagnostic on the mobile target, never a degraded render.

Shader portability: WGSL written for the web backends compiles to MSL (iOS) and SPIR-V (Vulkan/Android) via existing toolchain (`naga` or `SPIRV-Cross` — choice goes in the implementation plan). Per-target shader overrides ride the same `//gosx:native <target>` directive.

### The 80/20 framework principle

The framework owns at least 80% of any production app's surface. Anything ≥20% native signals a missing primitive — file an issue, prioritize for the next minor. The framework accretes per release; users don't ship around it forever.

The 20% native escape hatch is at two levels, both first-class:

1. **Source-level** (`//gosx:native <target>`) — covered above. A single component or expression drops into platform code without leaving `.gsx`.
2. **Project-level** (hand-written Swift/Kotlin alongside `Generated/`) — for whole capabilities the framework doesn't yet wrap. The framework exposes named extension points so this code integrates cleanly.

### Extension points (the structured 20%)

| Extension point | What it lets you bring in |
|---|---|
| `GSXNativeView` | Wrap any UIView / Android View as a gsx component, mountable from JSX |
| `GSXEngine` protocol | Custom Metal/Vulkan engine; framework owns the surface lifecycle, you own the render |
| `GSXTransport` | Non-HTTP data transports (gRPC, websocket-only, mock for tests) |
| `GSXCapabilityProvider` | Vend a device capability the framework's standard providers don't cover |
| `GSXNavigationTransition` | Custom navigation transition not in the built-in set |
| `GSXBridge` | Register a native service callable from gosx server-side via the bridge envelope |

Even when you escape-hatch, you do it through a contract the framework recognizes. Generated code knows how to find your code; tooling (LSP, inspector, hot-reload) keeps working.

### v1 framework surface (the initial 80%)

- Full NIR view tree (all symbolic tags from the table above)
- Reactive primitives (`Signal`, `Computed`, `Effect` with full gosx semantics)
- Navigation router (push, pop, replace, deeplink, back-handling)
- Data client + action client (HTTP/JSON v1, auth header attach, retry policy, in-memory cache with explicit invalidation, optimistic updates)
- Auth (token storage in Keychain / EncryptedSharedPreferences, refresh flow)
- Forms (validation, submission, error display, field bindings to signals)
- Lists with infinite scroll and pull-to-refresh
- Images with caching and placeholders
- Sheets, alerts, dialogs
- Animation primitives (transition + gesture)
- Theming (colors, typography, dark mode)
- Accessibility (screen reader, dynamic type)
- i18n hooks
- Engine surfaces (Metal + Vulkan, native lifecycle)
- Scene3D rendering (Metal + Vulkan, full `scene.IR` parity)

### v1.x accretion roadmap

These are extension-point candidates in v1, framework-owned in later minors:

- Camera, microphone, location, contacts, notifications, biometrics, share sheet, document picker
- Payments (StoreKit, Google Play Billing)
- Background tasks
- Push notification channels
- Hubs (WS, CRDT, presence) — the deferred multiplayer surface
- Wear, CarPlay, widgets
- Persistent disk cache for data client
- Offline-first persistence (CRDT-backed, paired with Hubs)

Each ships independently in a minor without breaking earlier apps.

## Server interop

The principle from above: shape crosses, server implementation stays where it is, mobile gets a typed client.

### Routing

gosx's file-based routing lowers into NIR's `Route` table at compile time. Per route: path pattern, layout chain, component, typed params schema, list of bound data-loader declarations, navigation flags (push, modal, tab, replace).

| NIR concept | iOS | Android |
|---|---|---|
| Route table | `GSXRouter` (singleton, owns `NavigationPath`) | `GSXRouter` (singleton, owns `NavController`) |
| Push route | `router.push(.profile(id: 42))` | `router.push(Profile(id = 42))` |
| Modal route | `.sheet(...)` presentation | `ModalBottomSheet` / `Dialog` |
| Tab layout | `TabView` with tag binding | `Scaffold` + `BottomNavigation` |
| Layout chain | nested SwiftUI views | nested composables |
| Deeplinks | URL scheme + Universal Links | App Links + intent filter |
| Back handling | `NavigationStack` native swipe-back | `BackHandler` hook |

The router is a runtime singleton, not generated code. Generated `Routes.swift` / `Routes.kt` declares the route enum, parameter types, and registers them at app boot.

### Data loaders

Every `DataLoaderDecl` in NIR generates one method on `dataClient`:

```swift
public extension GSXDataClient {
    func getUser(id: String) async throws -> User { /* HTTP GET /api/loaders/getUser?id=... */ }
    func listPosts(after: String?) async throws -> PostPage { ... }
}
```

```kotlin
suspend fun GSXDataClient.getUser(id: String): User = ...
suspend fun GSXDataClient.listPosts(after: String?): PostPage = ...
```

Routes auto-bind data loaders via NIR `bind:` declarations. When the route mounts, the framework dispatches loader calls (parallel by default) and exposes results as signals on the component. Loading and error states are framework-owned via `@GSXResource` (Swift propertyWrapper) / `gsxResource { ... }` (Compose helper) that exposes `.loading | .success(value) | .failure(err)`.

### Actions

Same shape as loaders, plus mutation semantics:

```swift
public extension GSXActionClient {
    @discardableResult
    func createPost(input: PostInput) async throws -> Post { /* POST /api/actions/createPost */ }
}
```

NIR `ActionDecl` carries two metadata pieces that affect mobile behavior:

1. **`invalidates`** — list of data loaders whose caches bust on success. Framework auto-refetches subscribed views.
2. **`optimistic`** — optional optimistic-update closure expressed in the portable subset. Framework applies the predicted state locally pending server response, rolls back on failure with a flash error.

This makes "feels native and snappy" the framework default rather than something every app rebuilds.

### Auth

Token-based. Framework owns:

- Token storage (Keychain / EncryptedSharedPreferences)
- Auto-attach as `Authorization: Bearer ...` on every request
- 401 interceptor → refresh flow → retry original request once
- Logout = token wipe + navigate-to-login (configurable)

The server already has sessions, OAuth, WebAuthn. v1 mobile assumes a `/api/auth/exchange` endpoint that takes whatever credential and returns a mobile token. Framework provides `GSXAuth.signIn(strategy: .password(email, pw))` / `.signIn(strategy: .oauth(provider))`.

### Bridge envelope

Adapted from `gosx/desktop/bridge/router.go`. Typed RPC: Go-side `App.Bind("Vault", handler)` exposes a service; mobile calls `bridge.call("Vault.encrypt", args)` returning `async throws ResultType`. HTTP transport in v1; WebSocket lane lands when hubs do.

Bridge calls are RPC to a server method, not declarative data ops. The right vehicle for "encrypt this payload server-side," "kick off a long-running job," "subscribe to a stream." Actions and loaders remain the right vehicle for typical CRUD and queries. Both share the auth/transport plumbing.

### Capability negotiation

The app declares required capabilities in its NIR capability manifest at compile time. At boot, the framework calls `/api/capabilities` and validates the server still has everything declared. Missing capability → app refuses to start with a clear "server is missing X" message. Prevents the "deployed app, rolled back server, runtime mystery" failure.

### Offline and network

v1 keeps this honest:

- Framework exposes `GSXNetwork.status` as a signal (`.online(.wifi)` / `.online(.cellular)` / `.offline`)
- Default: assume network. Loaders and actions surface failures cleanly.
- Apps can use the signal to show offline banners or queue user intent.
- No built-in offline-first persistence in v1. That's the deferred hubs/CRDT surface.

### Caching

- In-memory only (per-session) in v1
- Per-loader TTL declared in NIR (lowered from gosx's existing cache-header conventions)
- Action `invalidates` busts caches; framework auto-refetches subscribed views
- Manual: `dataClient.invalidate(\.getUser(id: 42))`
- Persistent disk cache lands in v1.x

### Generated client code layout

```
Generated/
  Routes.{swift,kt}        # route enum, parameter types, registration
  DataClient.{swift,kt}    # one method per DataLoaderDecl
  ActionClient.{swift,kt}  # one method per ActionDecl
  Bridge.{swift,kt}        # bridge service stubs
  Capabilities.{swift,kt}  # capability manifest
  Models.{swift,kt}        # shared types referenced by clients
```

Everything else (router state, transport, auth, cache) lives in `GSXNativeKit` / `gsxnative.aar` — framework code, not generated.

## Tooling

### CLI

Two entry points, same code path:

```bash
# Standalone — for mobile-only projects or CI/scripts
gsxnative init myapp --targets ios,android
gsxnative dev ios
gsxnative build android --release

# Subcommand — for gosx apps that pulled in gosx-native as a dep
gosx mobile init --targets ios,android
gosx mobile dev ios
gosx mobile build android --release
```

`gosx mobile` is a thin shim. When `github.com/odvcencio/gosx-native` is in `go.mod`, gosx auto-discovers and registers the subcommands. When it isn't, `gosx mobile` prints a "run `go get github.com/odvcencio/gosx-native` to add mobile support" message.

### Command set

| Command | Does |
|---|---|
| `init <name> [--targets ios,android,web] [--template basic\|tabs\|auth\|scene3d]` | Scaffold a new project (or add mobile to an existing gosx app) |
| `dev [--target ios\|android\|web\|all] [path]` | File-watch, regenerate `Generated/`, hot-reload |
| `build <target> [--release\|--debug] [path]` | Produce signed app artifact |
| `compile <file.gsx>` | Lower to NIR, dump JSON for inspection |
| `emit <target> <file.gsx>` | Emit native source to stdout |
| `check <file.gsx> [--target ios\|android\|web]` | Parse + lower + per-target diagnostics |
| `fmt <file.gsx>` | Format `.gsx` / `.swift.gsx` / `.kt.gsx` |
| `lsp [--language go+gsx\|swift+gsx\|kotlin+gsx]` | Language server |
| `scene-conform [--target ios\|android]` | Scene3D conformance suite against a backend |

### Project scaffold

**Adding mobile to an existing gosx app** (the common case): `gsxnative init` auto-edits `go.mod` to require `github.com/odvcencio/gosx-native` and creates `ios/` and `android/` directories with the structure shown in the back-end section.

**Mobile-only project** (someone authoring in `swift+gsx` or `kt+gsx` from scratch): same shape minus the existing `app/` sources. `gsxnative init` creates an `app/` with starter `App.swift.gsx` (or `App.kt.gsx`) and a route file.

**v1 templates:**

- `basic` — single screen, signal counter, one route
- `tabs` — three tabs, routing, sample data loader
- `auth` — login + protected home, full auth flow
- `scene3d` — embedded Scene3D engine with UI overlay; exercises the engine surface contract

### Dev workflow

`gsxnative dev ios` runs:

1. **File watcher** on `.gsx` / `.swift.gsx` / `.kt.gsx` sources, the route directory, and the capability manifest.
2. **On change:** parse → lower → emit → write to `Generated/`. Diagnostics print with source spans; emit aborts on error severity.
3. **Reload trigger** for the running app (see hot-reload below).
4. **Server proxy:** if a gosx server is running locally, the dev iOS/Android app's data client points at `http://localhost:<port>` (configurable). For physical-device testing, the watcher prints the LAN address.

`gsxnative dev all` runs the gosx web dev server, iOS Simulator, and Android Emulator side-by-side. One source change updates all three. This is the headline DX moment — *one edit, three live previews*.

### Hot reload — v1 honest scope

Two layers:

1. **Source regeneration** — fast and reliable. Edit `.gsx`, watcher regenerates `Generated/` in milliseconds.
2. **Runtime swap** — the slower "do I need to rebuild?" question. v1 lean:
   - **Platform tools by default.** Xcode 26+ has improved code injection; the *Inject* library (Krzysztof Zabłocki) is the established path for SwiftUI hot-reload during dev. Android Studio's Apply Code Changes works for most edits. Both are mature, free, and don't require us to ship runtime infrastructure.
   - **Dev-mode NIR-binary patch protocol** as a v1.x add. We bundle a small dev-only runtime that loads compiled NIR patches over a local socket and hot-swaps affected components without recompile. Reuses the GSX binary format from above. Drops out of release builds entirely. Not v1, but the architecture supports it cleanly.

Honest tradeoff for v1: full source regen on save (fast) plus platform-tool hot-reload for the actual rebuild (works but has rough edges). Working DX, not magical. Magical is v1.x.

### Build artifacts

```bash
gsxnative build ios --release
# → ios/build/MyApp.xcarchive (signed if signing config present)
# → ios/build/MyApp.ipa for ad-hoc

gsxnative build android --release
# → android/app/build/outputs/bundle/release/MyApp.aab (signed)
# → android/app/build/outputs/apk/release/MyApp.apk for ad-hoc
```

Wraps `xcodebuild` and `gradle assembleRelease` / `bundleRelease`. Signing config lives in `.gsxnative/signing.json` (gitignored) or env vars.

`gsxnative build all --release` parallelizes web + iOS + Android. CI calls this; produces all three artifacts in one pass.

### LSP

Single binary, all three source languages, dispatched by file extension. Provides:

- Syntax highlighting (via tree-sitter grammars; LSP forwards highlight queries)
- Diagnostics (parse errors, lower errors, NIR-level type errors, unsupported-on-target warnings)
- Go-to-definition across `.gsx` files, into framework runtime headers, into generated client methods
- Hover (types, capability requirements, "ships on web/iOS/android: ✓✓✓")
- Completion (component props, signal/computed/effect, framework primitives)
- Format-on-save

v1 ships a **VS Code extension** wrapping the LSP. Xcode and Android Studio integration deferred to v1.x.

### Cross-target build orchestration

`gsxnative build all` is the killer demo: web bundle + iOS archive + Android bundle from one source tree, one command, one parity guarantee. CI just runs that.

## Diagnostics and testing

### Diagnostic format

Every NIR node carries `(file, byte_start, byte_end)`. Every diagnostic uses it. Format mirrors Rust and Swift — point at the offending span with carets, give the cause, suggest the fix:

```
error[E2103]: missing per-target implementation
  --> app/DateBadge.swift+gsx:42:5
   |
42 |     formatted = formatDate(now)
   |                 ^^^^^^^^^^ no `//gosx:native android` body for this function
   |
help: provide an Android implementation, or replace with a portable subset call
   |
40 + //gosx:native android
41 + fun formatDate(t: LocalDateTime): String = ...
   |
```

### Diagnostic categories

Stable `E####` codes so docs, IDE quick-fixes, and tooling can hang off them.

| Category | Class | Examples |
|---|---|---|
| Parse | E1xxx | Unterminated JSX, mismatched tags, scanner failure |
| Lower (CST→NIR) | E2xxx | Unknown HTML tag with no symbolic mapping; component lacks return type; slot used outside a component body |
| NIR validation | E3xxx | Signal referenced but not declared; component invoked with wrong props shape; route missing a layout |
| Cross-target | E4xxx | `PerTarget` missing implementation for an enabled target; server-only API used in a reactive context; capability declared but target lacks it |
| Emit | E5xxx | Symbolic tag has no mapping for this target's runtime version; per-target opaque body fails platform parser |
| Runtime (dev only) | E6xxx | Bridge call to unregistered service; data loader returned wrong shape; capability mismatch at boot |

Severity: **error** (blocks emit), **warning** (allows emit, flagged), **info** (LSP hint, never blocks). JSON output mode (`--diagnostics=json`) for editor integration.

### Common-mistake catches (hard-coded from day one)

- HTML tags in Go `.gsx` targeting mobile that have a clean symbolic equivalent → silent normalization with an info note
- Server-only imports in a reactive context → error pointing at both the import and the usage site
- Engine capabilities the target doesn't support → error at the engine declaration with concrete options
- Forgotten signal declaration → diagnostic with scope-of-declaration and a one-keystroke fix
- Cross-component signal access → error suggesting either lifting state or using a shared signal key

### Runtime errors

Framework provides a unified `GSXError` shape:

```swift
public enum GSXError: Error {
    case bridgeFailed(service: String, method: String, underlying: Error)
    case dataLoader(name: String, kind: DataLoaderErrorKind)
    case actionFailed(name: String, kind: ActionErrorKind)
    case capabilityMissing(name: String)
    case sourceTrace(span: GSXSourceSpan, underlying: Error)  // dev builds wrap with original .gsx span
}
```

Dev builds carry the source span back to the original `.gsx` line — clicking the error in Xcode/AS opens the source. Release builds strip the span (privacy + binary size). Both platforms route `GSXError` to the framework's `ErrorBoundary` view (configurable per route), so apps default to handling rather than crashing.

### Testing strategy

Five layers:

1. **Grammar conformance.** Per source language: a corpus of valid + invalid `.gsx` files with expected parse trees / expected diagnostics. Lives in `testdata/corpus/{go,swift,kotlin}/`. Reuses gotreesitter's existing test plumbing.
2. **Lowering snapshots.** For each corpus file, snapshot the resulting `nir.Module` as JSON. `testdata/expected/nir/`.
3. **Emit golden tests.** For each NIR snapshot, snapshot the per-target emitted source. Reviewable diffs in PR. `testdata/expected/emit/{ios,android}/`.
4. **Build smoke tests.** CI matrix runs `xcodebuild` (Mac runner) and `gradle assembleDebug` (Linux runner) against the emitted snapshots. Catches "code looks right but doesn't compile."
5. **Runtime integration tests.** XCTest (iOS Simulator) and Espresso (Android Emulator) launch demo apps from `examples/` and assert behavior. Runs nightly + on release branches (cost too high for every PR).

**Plus the Scene3D conformance harness:** corpus of `scene.IR` documents rendered through every backend (WebGL, WebGPU, Metal, Vulkan), pixel-diffed against tolerance. Lives in `gosx/scene/conform/` so all backends are tested together. Runs in gosx-native CI as well as gosx CI.

**Cross-target parity tests** (the framework's headline guarantee): demo apps from `examples/` are emitted to all three targets, the rendered view tree is captured (snapshot of widget tree, not pixels), and structural equivalence is asserted modulo platform-specific styling. Catches drift in symbolic-tag mappings.

**Performance suite** (tracked over time, gates merges):

- NIR lower throughput (modules/sec)
- Emit throughput per target
- Generated app cold-boot time (release build, on-device)
- Signal update propagation cost (10k signal microbenchmark)
- Memory at idle / after first interaction

### Test code layout

```
gosx-native/
  testdata/
    corpus/{go,swift,kotlin}/        # .gsx source files
    expected/
      nir/                           # JSON snapshots
      emit/{ios,android}/            # Swift/Kotlin source snapshots
  test/
    grammar/      # parse conformance per language
    lower/        # CST → NIR
    emit/         # NIR → source golden
    build/        # xcodebuild + gradle smoke
    runtime/      # XCTest + Espresso integration
    parity/       # cross-target structural equivalence
    perf/         # benchmark suite
```

### CI matrix

- **Linux runner:** grammar + lower + emit + Android build smoke + Android runtime (Emulator)
- **Mac runner:** iOS build smoke + iOS runtime (Simulator) + Scene3D Metal conformance
- **Linux + Mac shared:** Scene3D Vulkan conformance (Mac via MoltenVK), parity tests

### Deferred to v1.x

- Fuzz harness (random NIR mutations through emit)
- Visual diff with strict tolerance (vs. structural equivalence in v1)
- On-device performance profiling integrated with the perf suite
- Network-fault injection (offline transitions, slow loaders)

## DX commitments — RN-class deftness

The framework should feel deft. These are operational commitments, not aspirations. We measure them and they gate releases.

- **Time from `gsxnative init myapp` to running app on simulator:** under 60 seconds on a clean machine.
- **File-save to `Generated/` regen:** under 500 ms for a 1000-component project on a developer laptop.
- **File-save to visible UI change:** under 3 seconds with platform hot-reload (Inject / Apply Code Changes); under 1 second with v1.x NIR-binary patch protocol.
- **Cold-boot to first interaction (release build, mid-tier device):** under 1 second for a basic-template app.
- **Diagnostic clarity:** every error code has a one-paragraph documentation entry with the cause, the fix, and a worked example.
- **Native-module bridging:** wrap a UIView / Android View as a gsx component in under 50 lines of native code via `GSXNativeView`.
- **First-class starter templates:** every template `init`s clean, builds clean, and ships at least one screen that exercises navigation, signals, and a data loader.

These targets live in the perf suite and CI gates them.

## Coordinated changes in gosx (recap)

Three small, additive changes land upstream in gosx itself before gosx-native v1:

1. **NIR extraction.** Rename `gosx/ir` to `gosx/nir`. Promote web-only details out of the IR and into the web emitter.
2. **`SceneRenderer` interface in `gosx/scene`.** Existing WebGL and WebGPU backends move behind a Go interface so Metal and Vulkan plug in alongside.
3. **`engine.Surface` abstraction.** Replace DOM-coupled mount fields with a tagged `Surface` type supporting both DOM and native mounts.

gosx and gosx-native release in coordination for v1; subsequent releases can decouple as the contracts stabilize.

## Roadmap

### v1 — the framework, two targets, full base surface

- Three input grammars (`go+gsx`, `swift+gsx`, `kotlin+gsx`)
- NIR + JSON serialization
- Three lowerers
- Two emitters (iOS, Android)
- Two runtime libraries (`GSXNativeKit`, `gsxnative.aar`) with full v1 framework surface
- CLI (`gsxnative` + `gosx mobile` shim)
- Project scaffolding for four templates
- LSP + VS Code extension
- Build smoke + runtime integration + parity + Scene3D conformance harnesses
- Documentation site

### v1.x — accretion

- Native-module wrappers for camera, location, biometrics, payments, notifications, etc. (one minor each as appropriate)
- Persistent disk cache for data client
- Dev-mode NIR-binary patch protocol for hot-reload
- Xcode and Android Studio integrations of the LSP
- Visual-diff Scene3D conformance (vs. structural equivalence in v1)

### v2 — multiplayer

- Hubs (WS, CRDT, presence)
- Offline-first persistence with CRDT sync
- Shared signals across users (the `signal.NewShared` web concept extends to mobile)

### Future targets

The architecture supports adding render targets without touching gosx-native v1 internals. Candidate future targets:

- Compose-Desktop (covers Mac, Linux, Windows native)
- Embedded (Linux frame-buffer, automotive infotainment)
- TV (tvOS, Android TV)

Each requires implementing `SceneRenderer`, the `SurfaceHost`, the runtime primitives, and a per-target emitter. No core change.

## Open questions

The brainstorm answered the consequential design questions. These remain for the implementation plan:

- **Verification of `grammargen.SwiftGrammar()` / `KotlinGrammar()` round-trip — first plan milestone.** PR #58 landed the DSLs and regenerated blobs; the entire front end depends on `ExtendGrammar()` over those bases producing a working `swift+gsx` / `kotlin+gsx` parser. The implementation plan starts with this smoke test.
- **Type-system mapping into the constrained expression language.** The portable subset enumerates allowed forms but doesn't pin how host-language type systems map into NIR's type model (e.g., does Swift `Optional<T>` map to a NIR nullable, or stay opaque? How does Go's interface type land in Swift/Kotlin?). The lowerers will need this nailed down — implementation-plan input.
- **Shader cross-compilation choice.** `naga` (Rust, cross-platform, no CGo) vs. `SPIRV-Cross` (C++). Pick during implementation based on the CGo posture we want.
- **Bridge transport for streaming.** WebSocket lane for streaming bridge calls — bring forward into v1 or wait for hubs?
- **Concrete struct shapes for NIR.** This spec describes node categories; the implementation plan pins the Go types.
- **Symbolic tag mapping coverage.** The v1 table covers the basics. The implementation plan fills in the long tail (specific list-row semantics, TextField input modes, etc.) by audit.

## Decisions log

The choices we made and why, for future readers and our future selves:

- **Path C (three sibling grammars → shared NIR → per-target emitters), not Path A (Go-only transpiler).** Bigger upfront commitment, but it's the architecturally honest answer and the prerequisite (grammargen Swift/Kotlin DSLs) landed in gotreesitter PR #58 the same day. Path A would have shipped sooner but boxed us in.
- **C-prime (native source for everything, no production VM), not C-with-VM.** The web VM exists for payload reasons that don't transfer to mobile. Native code is faster, more debuggable, more App-Store-friendly. Framework is still substantial — it just doesn't include an interpreter.
- **Constrained expression language as NIR's semantic boundary, with `//gosx:native` escape hatches.** Default is portable; escape hatches are first-class and per-target. Single-target apps pay no portability tax.
- **Strict `Generated/`, soft everywhere else.** Unambiguous boundary between throwaway and durable code.
- **Framework owns reactive primitives** (`@GSXSignal`, `gsxSignal`), not raw `@State` / `mutableStateOf` in generated code. Stable API surface; room for tooling.
- **HTTP/JSON for v1 transport, optimistic updates as a v1 commitment.** JSON is universal and debuggable; binary lane is mechanical to add later. Optimistic updates are what separates "feels native" from "feels transpiled."
- **Both `gsxnative` standalone and `gosx mobile` subcommands.** Same code path; ergonomics matter for two distinct user journeys.
- **Runtime integration tests in CI nightly + on release branches**, not every PR. Keeps PR latency reasonable; catches regressions before they ship.
- **Scene3D mobile renderer scope is full `scene.IR` parity in v1**, not a subset. Anything less makes the cross-target parity promise dishonest.

## Appendix A — file extension cheat sheet

| Extension | Source language | Parsed by |
|---|---|---|
| `.gsx` | Go + gsx | `grammargen.ExtendGrammar(GoGrammar(), customize)` |
| `.swift.gsx` | Swift + gsx | `grammargen.ExtendGrammar(SwiftGrammar(), customize)` |
| `.kt.gsx` | Kotlin + gsx | `grammargen.ExtendGrammar(KotlinGrammar(), customize)` |
| `.gsxbin` | NIR binary (dev/tooling) | gosx-native binary loader |

## Appendix B — CLI command cheat sheet

```bash
gsxnative init myapp --targets ios,android,web --template tabs
gsxnative dev all
gsxnative build all --release
gsxnative compile app/Counter.gsx     # dump NIR JSON
gsxnative emit ios app/Counter.gsx    # emit Swift to stdout
gsxnative check app/Counter.gsx --target android
gsxnative fmt app/
gsxnative lsp                         # for editor integration
gsxnative scene-conform --target ios  # Scene3D backend conformance
```

## Appendix C — directive cheat sheet

```go
//gosx:island                 // marks a component as client-runtime on web (no-op on mobile)
//gosx:native swift           // function or block: provides Swift implementation
//gosx:native kotlin          // function or block: provides Kotlin implementation
//gosx:native go              // function or block: provides Go implementation
//gosx:targets ios,android    // restrict a function/component to listed targets
//gosx:web-only               // explicitly opt out of mobile emit
//gosx:server                 // namespace marker for server-only operations
```
