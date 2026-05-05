# gosx-native

The mobile counterpart to [gosx](https://github.com/odvcencio/gosx). React Native is to React what gosx-native is to gosx: same component model, same reactive primitives, same scene graph, different rendering targets.

**Status: Android and iOS counter vertical slices compile. The iOS demo builds and passes a Simulator UI smoke test; the Android demo regenerates the Compose source and assembles a debug APK in CI. GoSX `.gsx` Counter, Panel, Greeter, Derived, and Toggle fixtures now lower through the shared NIR and emit deterministic SwiftUI/Compose source.**

See [`docs/superpowers/specs/2026-05-04-gosx-native-design.md`](docs/superpowers/specs/2026-05-04-gosx-native-design.md) for the design.

## Counter Demos

- GoSX source to iOS: `go run ./cmd/gsxnative emit ios testdata/corpus/go/counter.gsx`
- GoSX source to Android: `go run ./cmd/gsxnative emit android testdata/corpus/go/counter.gsx`
- Broader GoSX handler fixture: `go run ./cmd/gsxnative emit ios testdata/corpus/go/panel.gsx`
- GoSX text-input fixture: `go run ./cmd/gsxnative emit ios testdata/corpus/go/greeter.gsx`
- GoSX computed fixture: `go run ./cmd/gsxnative emit ios testdata/corpus/go/derived.gsx`
- GoSX conditional fixture: `go run ./cmd/gsxnative emit ios testdata/corpus/go/conditional.gsx`
- iOS smoke: `make smoke`
- Android assemble smoke: `make android-smoke`
- Android emulator interaction smoke, with an emulator already booted: `make android-connected`
- Android managed-emulator interaction smoke: `make android-managed`

The Android smoke expects Gradle plus Android SDK platform 36/build-tools 36.0.0. CI pins Gradle 9.4.1, Android Gradle Plugin 9.2.0, Kotlin/Compose compiler 2.3.21, Compose BOM 2026.04.01, and Activity Compose 1.13.0. CI also builds the Android runtime as an AAR and runs the Counter UI test on an API 30 Gradle-managed ATD emulator.

## License

MIT
