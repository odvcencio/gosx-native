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
- `gsxnative scene-conform` now gates static Scene3D fixtures against canonical `gosx/scene.IR`, deterministic renderer-agnostic render signatures, and iOS/Android source goldens.
- `<InstancedMesh>` now lowers to canonical `scene.IR` kind `instanced-mesh`, has source goldens for iOS and Android, and is included in the default conformance gate.
- `<PostFX.*>` now lowers into canonical `scene.IR.PostFX`, has source goldens for iOS and Android, and is included in the default conformance gate as preserved scene metadata.
- `<ComputeParticles>` now lowers into canonical `scene.IR` kind `compute-particles`, emits visible native placeholder particles, has source goldens for iOS and Android, and is included in the default conformance gate.
- `<Html>` now lowers static/literal markup into `scene.IR.Metadata["html"]`, emits simple native overlay text metadata, has source goldens for iOS and Android, and is included in the default conformance gate.
- Scene3D `map[string]any` spread props now lower into typed runtime attribute reads for native source generation.
- Native Canvas runtimes now draw placeholder post-fx visualization for bloom, vignette, color grading, and tone mapping declarations.
- Scene3D static, instancing, compute, HTML, and post-fx fixtures now have checked-in render-signature goldens covering the normalized expected draw contract.
- Runtime Scene3D now has explicit backend selection: native is the default, and `backend="canvas"` keeps the previous SwiftUI/Compose Canvas renderer as a fallback.
- The initial native runtime path is SceneKit on iOS and an OpenGL ES `GLSurfaceView` bridge on Android, so checked-in demos exercise GPU-backed native surfaces instead of only declarative Canvas placeholders.
- CI UI smoke tests now inspect simulator/emulator screenshots and require Scene3D-colored native pixels, so the gate proves the checked-in native surfaces painted instead of only existing in the accessibility tree.
- CI now emits every valid Scene3D fixture as temporary Swift/Kotlin app source and compiler-checks the fixture matrix on iOS and Android: instancing, post-fx, compute, HTML, Canvas fallback, and spread props.
- Harden the native GPU bridge into full renderer-grade `scene.IR` parity, including a final Metal/Vulkan/Filament decision if SceneKit/OpenGL ES is not sufficient.
- Replace Canvas-level post-fx placeholders with renderer-grade post-fx passes in the native backends.
- Replace compute-particle placeholders with real GPU compute in the native backends.
- `<Html>` overlays now render through WKWebView/WebView-backed native overlay views instead of plain text extraction; remaining fidelity work is layout/picking parity against the web renderer.
- Promote screenshot pixel smoke to fixture-level native render-output golden conformance.
