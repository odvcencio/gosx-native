# GoSX computed parity

**Goal:** Move the GoSX native proof from direct signal reads into derived reactive state.

**Scope:**
- Add `ComputedDecl` to the shared `gosx/nir` component contract.
- Lower GoSX `signal.Derive(func() T { return expr })` declarations into NIR computed declarations.
- Infer computed native types from portable Rx expressions and known signal/prop refs.
- Emit SwiftUI computed properties and Compose local derived values.
- Add `testdata/corpus/go/derived.gsx` plus Swift and Kotlin golden output.

**Result:** GoSX fixtures now prove props, signals, event payloads, handlers, text inputs, and computed values through the same NIR-to-native path.

**Remaining proof after this slice:**
- Broader expression coverage: ternary, string helpers, and conversions.
