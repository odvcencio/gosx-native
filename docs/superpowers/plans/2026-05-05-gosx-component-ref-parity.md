# GoSX component-reference parity

**Goal:** Prove native lowering for one component invoking another through the shared view-tree contract.

**Scope:**
- Add `ComponentRef` to the shared `gosx/nir` view contract.
- Lower ordinary capitalized GoSX components into NIR component references with portable prop expressions.
- Emit SwiftUI nested component initializers and Compose composable calls from the component-reference view node.
- Add `testdata/corpus/go/component_ref.gsx` plus Swift and Kotlin golden output.

**Result:** GoSX fixtures now prove props, signals, handlers, event payloads, text inputs, computed values, conditional view nodes, and component references through the same shared NIR-to-native path.

**Remaining proof after this slice:**
- Broader expression coverage: ternary, string helpers, and conversions.
