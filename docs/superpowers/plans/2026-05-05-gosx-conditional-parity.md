# GoSX conditional parity

**Goal:** Prove native lowering for conditional view-tree shape, not just ordinary element/text/expression nodes.

**Scope:**
- Add `Conditional` to the shared `gosx/nir` view contract.
- Lower GoSX `<If>`, `<Show>`, and `<When>` component aliases into NIR conditionals using `when`/`if`/`cond`/`test`.
- Emit SwiftUI and Compose `if` blocks from the conditional view node.
- Add `testdata/corpus/go/conditional.gsx` plus Swift and Kotlin golden output.

**Result:** GoSX fixtures now prove props, signals, handlers, event payloads, text inputs, computed values, and conditional view nodes through the same shared NIR-to-native path.

**Remaining proof after this slice:**
- Broader expression coverage: ternary, string helpers, and conversions.
