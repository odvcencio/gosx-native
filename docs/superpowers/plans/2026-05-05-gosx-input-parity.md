# GoSX input parity

**Goal:** Prove that GoSX event payloads can drive native text input state on both SwiftUI and Jetpack Compose.

**Scope:**
- Add `testdata/corpus/go/greeter.gsx` with a string prop, a string signal, a native text input, and an `onInput` handler that reads GoSX's event `value`.
- Lower GoSX event payload reads into the portable NIR ref `event.value`.
- Preserve native-relevant input attributes (`value`, `placeholder`, `type`) without pulling web-only attrs into NIR.
- Emit SwiftUI `TextField` and Compose `TextField` bindings that map `event.value` to each platform's setter callback value.
- Lock the generated Swift and Kotlin with Greeter golden tests.

**Result:** GoSX Counter and Panel parity now extends to form input: a `.gsx` component can bind a text input to a signal and produce deterministic native source for both targets.

**Remaining proof after this slice:**
- Computed values via a NIR `ComputedDecl`.
- Broader expression coverage: ternary, string helpers, and conversions.
