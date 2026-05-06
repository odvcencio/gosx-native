# Scene3D Target And Static Surface

## Goal

Make Scene3D visible in the native compiler path before implementing a renderer backend. The compiler should lower Scene3D syntax, target tooling should diagnose missing native support, and build commands should fail before writing unusable SwiftUI or Compose source.

## Slice

- Add a Scene3D-aware GoSX lowering path for `<Scene3D>` and composable scene children.
- Add native target validation for supported tags, component refs, handlers, and Scene3D capability gaps.
- Run validation from `check`, `emit`, and `build`.
- Add `gsxnative build ios|android|all` to regenerate native app source and invoke Xcode/Gradle.
- Wire Makefile and CI through the CLI build command.

## Remaining Backend Work

- Static `<Mesh>`, `<Model>`, and `<Points>` now emit into native runtime Scene3D views.
- Checked-in iOS and Android demo app shells now include generated Scene3D sources, and CI regenerates, diffs, and compiles those sources.
- Add native Scene3D NIR payloads instead of generic element placeholders.
- Choose the durable iOS backend beyond the static SwiftUI canvas surface: SceneKit, Metal, or a portable renderer bridge.
- Choose the durable Android backend beyond the static Compose canvas surface: Filament, Vulkan/AGSL bridge, or a portable renderer bridge.
- Add native support for `<Html>`, `<InstancedMesh>`, `<ComputeParticles>`, and `<PostFX.*>` after the durable backend is selected.
- Add cross-target Scene3D conformance fixtures for native render output.
