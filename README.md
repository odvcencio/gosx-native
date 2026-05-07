# gosx-native

The mobile counterpart to [gosx](https://github.com/odvcencio/gosx). React Native is to React what gosx-native is to gosx: same component model, same reactive primitives, same scene graph, different rendering targets.

## Agent Skill

Agents helping someone use gosx-native should read the GoSX ecosystem skill: [using-gosx-ecosystem](https://github.com/odvcencio/m31labs-skills/blob/main/skills/using-gosx-ecosystem/SKILL.md).

**Status: Android and iOS counter vertical slices compile. The iOS demo builds and passes a Simulator UI smoke test; the Android demo regenerates Compose source and assembles a debug APK in CI. GoSX `.gsx` Counter, Panel, Greeter, Derived, Toggle, Profile, Roster, FormControls, Expressions, and Scene3D fixtures now lower through the shared NIR and emit deterministic SwiftUI/Compose source. `gsxnative init` now scaffolds a real dual-platform app shell with `gosxnative.json`, generated SwiftUI/Compose sources, generated route/data/action/bridge declaration clients, capability manifests, project-level native bridge/capability skeletons, XcodeGen config, Gradle config, release signing placeholders, dev server URL metadata hooks, an executable reload adapter script, and checked default GoSX source. `gsxnative build` discovers that config from the app directory or descendants, derives route/data/action/capability/model/bridge declarations from source directives with config fallback, validates declared routes against compiled components, validates action invalidation targets, validates custom model references and recursive model cycles, validates capability targets and bridge schemas, validates `//gosx:native` escape-hatch target coverage, forwards release environment metadata to Xcode/Gradle, forwards optional `.gsxnative/signing.json` signing settings to Xcode/Gradle with Android secrets sourced from env vars and redacted from manifests, forwards validated server URLs to Xcode/Gradle, derives Android flavor build tasks/artifacts, can publish expected APK/AAB/archive/export artifacts into deterministic target subdirectories, can keep stable Xcode derived data for simulator app artifacts, and regenerates both component and support declaration sources. `gsxnative store-plan` turns release artifact manifests into deterministic App Store Connect and Google Play submission handoff plans with selected artifacts, tracks, required secret names, and readiness warnings. `gsxnative check` and `gsxnative emit` also fail before emission when a native escape hatch is missing the requested Swift/Kotlin target body. `gsxnative dev` now provides a source/config polling codegen loop, with `--once` for CI and scripts, `--build` when a dev loop should invoke Xcode/Gradle after regeneration, `--install`/`--launch` orchestration for Android `adb` and iOS Simulator `simctl` with selected device, package/activity, app path, and bundle-id overrides, and reload trigger hooks through JSON stamp files plus generic or target-specific shell commands. `gsxnative devices` lists Android and iOS targets from `adb devices -l` and `xcrun simctl list devices --json`, including JSON output for scripts. Generated support clients now carry endpoint and bridge specs, typed primitive, custom-model, and one-dimensional collection param/input/output schemas, generated Swift `Codable` models, generated Kotlin JSON model adapters with nested custom-model and array encode/decode support, per-endpoint auth/cache/retry/optimistic metadata, route auth metadata, path/query param resolution, action invalidation policies, generated capability specs, generated bridge callers, and generated capability-negotiation helpers. Runtime kit coverage now includes component/signal primitives, router stack primitives, route guards, data transport/client primitives, mobile auth token exchange clients, bearer auth with refresh, signed request transports, structured diagnostics, bridge client primitives, bridge dispatch envelopes, capability checking, `/api/capabilities` negotiation, project-level capability providers, bridge service registries, named in-memory loader cache, retry handling, validation-failure surfacing, action closures, and native Scene3D surfaces. Scene3D lowers into a typed NIR payload, maps into the canonical `gosx/scene.IR` conformance contract, renders static meshes/models/points, instanced mesh batches, compute-particle placeholders, WKWebView/WebView-backed HTML overlays, map-backed Scene3D spread props, and Canvas-level post-fx visualization through runtime views. Runtime Scene3D defaults to native GPU-backed surfaces: SceneKit on iOS and an OpenGL ES `GLSurfaceView` bridge on Android, with the old Canvas renderer available via `backend="canvas"`. `scene-conform` gates canonical IR, deterministic renderer-agnostic render signatures, and iOS/Android source goldens for the Scene3D static, instancing, compute, HTML, post-fx, Canvas fallback, and spread-props fixtures. CI regenerates, diffs, compiles, hosts, UI-asserts, and screenshot pixel-smokes checked-in Scene3D demo sources for both native app shells; it also emits every valid Scene3D fixture into temporary native app sources and compiler-checks them on iOS and Android. No currently enumerated Scene3D native target tags fail validation, but full semantic route/data/model/bridge lowering into NIR, renderer-grade post-fx passes, real GPU compute, golden native render-output conformance, and full Metal/Vulkan-grade `scene.IR` parity remain open.**

See [`docs/superpowers/specs/2026-05-04-gosx-native-design.md`](docs/superpowers/specs/2026-05-04-gosx-native-design.md) for the design.

## Production Readiness Gaps

- **Source-owned declarations:** `//gosx:route`, `//gosx:data`, `//gosx:action`, `//gosx:capability`, and `//gosx:bridge` directives now generate native support declarations from source; the remaining work is full semantic lowering into NIR instead of directive scanning.
- **Typed data/action codecs:** generated clients now support primitive, named custom-model, and one-dimensional collection schemas, JSON input/output models, validation failures, named cache TTLs, action invalidation metadata, retry counts, and optimistic metadata. Remaining work: portable optimistic-update closures, route-bound resource state, and semantic lowering into NIR instead of directive scanning.
- **Auth and session plumbing:** native runtimes now include in-memory token stores, secure platform token stores, refreshable token-store contracts, bearer-auth transport wrappers with 401 refresh/retry, mobile token exchange clients for `/api/auth/exchange`, HMAC request-signing transports, generated client constructors that accept token stores, per-endpoint auth policy metadata, generated route auth metadata, and route guards; higher-level OAuth/WebAuthn templates are still open.
- **Native bridge and escape hatches:** generated bridge specs, typed Swift/Kotlin bridge callers, capability manifests, `/api/capabilities` negotiation, runtime capability providers, bridge dispatch envelopes, bridge service registries, scaffolded native-module skeletons, and target coverage checks for `//gosx:native <target>` implementations are present. Remaining work: secure local capability implementations and full opaque native payload lowering.
- **Dev workflow:** `gsxnative dev` now watches GoSX source/config files and regenerates native sources through the build pipeline; `--build` lets the loop invoke Xcode/Gradle and preserves forwarded simulator/Gradle build flags, including validated `--server-url` metadata for local server-backed clients. `--install` and `--launch` now orchestrate Android `adb install`/`am start` and iOS Simulator `simctl install`/`launch`, with shared and target-specific device selectors plus package/activity, APK, app, and bundle-id overrides. `--reload-stamp`, `--reload-command`, `--android-reload-command`, and `--ios-reload-command` provide deterministic post-cycle hooks for platform reload adapters. `gsxnative devices` discovers Android and iOS device targets for humans and scripts. Runtime-native incremental patching is still open.
- **Release packaging:** Android release tasks, Android flavor-aware default tasks/artifact paths, iOS release archive/export defaults, Xcode/Gradle environment forwarding, env-backed release signing config, generated Android Gradle signing hooks, deterministic local artifact publishing, redacted JSON artifact manifests, and store submission handoff plans are available through `gsxnative build --release` and `gsxnative store-plan`; direct store upload execution is still external.
- **Scene3D renderer fidelity:** renderer-grade post-fx, real GPU compute, golden visual conformance, and full Metal/Vulkan-grade `scene.IR` parity remain open.
- **Operational hardening:** named in-memory caching, validation failures, retry counts, exponential retry backoff controls, telemetry-safe structured diagnostics, offline/network policy runtime APIs with platform status providers, and provider-neutral crash reporting hooks are present; provider-specific crash SDK templates still need production integration.

## Counter Demos

- New app scaffold: `go run ./cmd/gsxnative init /tmp/MyApp --name MyApp --module com.example.myapp`
- Build a scaffolded app: `cd /tmp/MyApp && gsxnative build all`
- Build a staging Android release flavor: `gsxnative build android --release --env staging --flavor demo`
- Build with signing config metadata: `gsxnative build all --release --signing-config .gsxnative/signing.json --artifact-manifest build/gsxnative-artifacts.json`
- Publish release artifacts locally: `gsxnative build all --release --publish-dir build/artifacts --artifact-manifest build/gsxnative-artifacts.json`
- Generate a store submission handoff plan: `gsxnative store-plan --artifact-manifest build/gsxnative-artifacts.json --output build/store-plan.json`
- Regenerate scaffolded native sources without Xcode/Gradle: `cd /tmp/MyApp && gsxnative build all --codegen-only`
- Run the dev codegen loop: `cd /tmp/MyApp && gsxnative dev all`
- Run one dev regeneration pass for scripts/CI: `cd /tmp/MyApp && gsxnative dev all --once`
- Run dev with native rebuilds: `cd /tmp/MyApp && gsxnative dev ios --build --simulator "iPhone 16"`
- Run dev against a local GoSX server: `cd /tmp/MyApp && gsxnative dev android --build --server-url http://10.0.2.2:3000`
- Trigger external reload tooling after each successful dev cycle: `cd /tmp/MyApp && gsxnative dev all --reload-stamp .gsxnative/reload.json --reload-command './scripts/reload-native.sh'`
- List Android/iOS launch targets: `gsxnative devices --json`
- Install and launch Android after each regeneration: `cd /tmp/MyApp && gsxnative dev android --launch --device emulator-5554`
- Install and launch iOS on a booted Simulator: `cd /tmp/MyApp && gsxnative dev ios --launch --ios-bundle-id com.example.MyApp`
- Target check: `go run ./cmd/gsxnative check ios testdata/corpus/go/counter.gsx`
- GoSX source to iOS: `go run ./cmd/gsxnative emit ios testdata/corpus/go/counter.gsx`
- GoSX source to Android: `go run ./cmd/gsxnative emit android testdata/corpus/go/counter.gsx`
- Scene3D static surface: `go run ./cmd/gsxnative emit ios testdata/corpus/go/scene3d.gsx`
- Scene3D instancing surface: `go run ./cmd/gsxnative emit ios testdata/corpus/go/scene3d_instancing.gsx`
- Scene3D post-fx conformance surface: `go run ./cmd/gsxnative emit ios testdata/corpus/go/scene3d_postfx.gsx`
- Scene3D compute-particle surface: `go run ./cmd/gsxnative emit ios testdata/corpus/go/scene3d_compute.gsx`
- Scene3D HTML overlay surface: `go run ./cmd/gsxnative emit ios testdata/corpus/go/scene3d_html.gsx`
- Scene3D spread-props surface: `go run ./cmd/gsxnative emit android testdata/corpus/go/scene3d_spread.gsx`
- Scene3D Canvas fallback surface: `go run ./cmd/gsxnative emit android testdata/corpus/go/scene3d_canvas.gsx`
- Scene3D conformance: `go run ./cmd/gsxnative scene-conform`
- Broader GoSX handler fixture: `go run ./cmd/gsxnative emit ios testdata/corpus/go/panel.gsx`
- GoSX text-input fixture: `go run ./cmd/gsxnative emit ios testdata/corpus/go/greeter.gsx`
- GoSX computed fixture: `go run ./cmd/gsxnative emit ios testdata/corpus/go/derived.gsx`
- GoSX conditional fixture: `go run ./cmd/gsxnative emit ios testdata/corpus/go/conditional.gsx`
- GoSX component-reference fixture: `go run ./cmd/gsxnative emit ios testdata/corpus/go/component_ref.gsx`
- GoSX loop fixture: `go run ./cmd/gsxnative emit ios testdata/corpus/go/loop.gsx`
- GoSX form-controls fixture: `go run ./cmd/gsxnative emit ios testdata/corpus/go/form_controls.gsx`
- GoSX form-controls Android fixture: `go run ./cmd/gsxnative emit android testdata/corpus/go/form_controls.gsx`
- GoSX expression-coverage fixture: `go run ./cmd/gsxnative emit ios testdata/corpus/go/expressions.gsx`
- GoSX expression-coverage Android fixture: `go run ./cmd/gsxnative emit android testdata/corpus/go/expressions.gsx`
- iOS CLI build: `make build-ios`
- Android CLI build: `make build-android`
- Combined CLI build: `make build-all`
- Scene3D iOS build: `make build-scene3d-ios`
- Scene3D Android build: `make build-scene3d-android`
- Scene3D conformance: `make scene-conform`
- iOS smoke: `make smoke`
- Android assemble smoke: `make android-smoke`
- Android emulator interaction smoke, with an emulator already booted: `make android-connected`
- Android managed-emulator interaction smoke: `make android-managed`

The Android smoke expects Gradle plus Android SDK platform 36/build-tools 36.0.0. CI pins Gradle 9.4.1, Android Gradle Plugin 9.2.0, Kotlin/Compose compiler 2.3.21, Compose BOM 2026.04.01, and Activity Compose 1.13.0. CI also builds the Android runtime as an AAR and runs the Counter UI test on an API 30 Gradle-managed ATD emulator.

## License

MIT
