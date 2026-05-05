# GoSX loop parity

**Goal:** Prove native lowering for repeated view subtrees using GoSX's existing `<Each>`/`<For>` island shape.

**Scope:**
- Add `Loop` to the shared `gosx/nir` view contract.
- Lower GoSX `<Each of={...} as="...">` and `<For>` component aliases into NIR loops with a child expression scope for the loop item.
- Map Go slice props such as `[]string` to Swift `[String]` and Compose `List<String>`.
- Emit SwiftUI `ForEach` and Compose `forEach` blocks from the loop view node.
- Add `testdata/corpus/go/loop.gsx` plus Swift and Kotlin golden output.

**Result:** GoSX fixtures now prove props, signals, handlers, event payloads, text inputs, computed values, conditional view nodes, component references, and loop view nodes through the same shared NIR-to-native path.

**Remaining proof after this slice:**
- Native textarea/multiline input and non-string event fields (`checked`, `key`, `selectedIndex`).
- More expression coverage: ternary, string helpers, and conversions.
