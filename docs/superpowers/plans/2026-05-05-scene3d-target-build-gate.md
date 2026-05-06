# Scene3D Target And Build Gate

## Goal

Make Scene3D visible in the native compiler path before implementing a renderer backend. The compiler should lower Scene3D syntax, target tooling should diagnose missing native support, and build commands should fail before writing unusable SwiftUI or Compose source.

## Slice

- Add a Scene3D-aware GoSX lowering path for `<Scene3D>` and composable scene children.
- Add native target validation for supported tags, component refs, handlers, and Scene3D capability gaps.
- Run validation from `check`, `emit`, and `build`.
- Add `gsxnative build ios|android|all` to regenerate native app source and invoke Xcode/Gradle.
- Wire Makefile and CI through the CLI build command.

## Remaining Backend Work

- Add native Scene3D NIR payloads instead of generic element placeholders.
- Choose iOS backend surface: SceneKit, Metal, or a portable renderer bridge.
- Choose Android backend surface: Filament, Vulkan/AGSL bridge, or a portable renderer bridge.
- Add cross-target Scene3D conformance fixtures once a backend can render.
