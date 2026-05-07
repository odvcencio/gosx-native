# gosx-native

The mobile counterpart to [gosx](https://github.com/odvcencio/gosx). React Native is to React what gosx-native is to gosx: same component model, same reactive primitives, same scene graph, different rendering targets.

**Status: Android and iOS counter vertical slices compile. The iOS demo builds and passes a Simulator UI smoke test; the Android demo regenerates Compose source and assembles a debug APK in CI. GoSX `.gsx` Counter, Panel, Greeter, Derived, Toggle, Profile, Roster, FormControls, Expressions, and Scene3D fixtures now lower through the shared NIR and emit deterministic SwiftUI/Compose source. Scene3D lowers into a typed NIR payload, maps into the canonical `gosx/scene.IR` conformance contract, renders static meshes/models/points, instanced mesh batches, compute-particle placeholders, simple native HTML overlay text, map-backed Scene3D spread props, and Canvas-level post-fx visualization through runtime views. Runtime Scene3D now defaults to native GPU-backed surfaces: SceneKit on iOS and an OpenGL ES `GLSurfaceView` bridge on Android, with the old Canvas renderer available via `backend="canvas"`. `scene-conform` gates canonical IR, deterministic renderer-agnostic render signatures, and iOS/Android source goldens for the Scene3D static, instancing, compute, HTML, and post-fx fixtures. CI regenerates, diffs, compiles, hosts, UI-asserts, and screenshot pixel-smokes checked-in Scene3D demo sources for both native app shells; it also emits every valid Scene3D fixture into temporary native app sources and compiler-checks them on iOS and Android. No currently enumerated Scene3D native target tags fail validation, but renderer-grade post-fx passes, real GPU compute, renderer-backed DOM/WebView HTML overlays, golden native render-output conformance, and full Metal/Vulkan-grade `scene.IR` parity remain open.**

See [`docs/superpowers/specs/2026-05-04-gosx-native-design.md`](docs/superpowers/specs/2026-05-04-gosx-native-design.md) for the design.

## Counter Demos

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
