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
- The iOS and Android smoke apps mount the generated SceneDemo and assert the native Scene3D accessibility/test-tag surface exists.
- Scene3D now lowers into native NIR payloads instead of generic element children.
- `gsxnative scene-conform` now gates static Scene3D fixtures against canonical `gosx/scene.IR` plus iOS and Android source goldens.
- `<InstancedMesh>` now lowers to canonical `scene.IR` kind `instanced-mesh`, has source goldens for iOS and Android, and is included in the default conformance gate.
- `<PostFX.*>` now lowers into canonical `scene.IR.PostFX`, has source goldens for iOS and Android, and is included in the default conformance gate as preserved scene metadata.
- `<ComputeParticles>` now lowers into canonical `scene.IR` kind `compute-particles`, emits visible native placeholder particles, has source goldens for iOS and Android, and is included in the default conformance gate.
- `<Html>` now lowers static/literal markup into `scene.IR.Metadata["html"]`, emits simple native overlay text metadata, has source goldens for iOS and Android, and is included in the default conformance gate.
- Scene3D `map[string]any` spread props now lower into typed runtime attribute reads for native source generation.
- Choose the durable iOS backend beyond the static SwiftUI canvas surface: SceneKit, Metal, or a portable renderer bridge.
- Choose the durable Android backend beyond the static Compose canvas surface: Filament, Vulkan/AGSL bridge, or a portable renderer bridge.
- Add native visual post-fx passes after the durable backend is selected.
- Replace compute-particle placeholders with real GPU compute after the durable backend is selected.
- Replace native text extraction for `<Html>` with renderer-backed DOM/WebView overlay semantics after the durable backend is selected.
- Add cross-target Scene3D conformance fixtures for native render output.
