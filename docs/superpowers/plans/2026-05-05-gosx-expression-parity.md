# GoSX expression parity

**Goal:** Prove that the native bridge handles the broader GoSX island expression subset already parsed by the shared front end.

**Scope:**
- Add a portable conditional expression node to shared `gosx/nir`.
- Lower GoSX `OpCond` into NIR conditional expressions.
- Lower string helpers and conversions into portable NIR calls: `trim`, `upper`, `lower`, `replace`, `join`, `startsWith`, `contains`, `len`, `toString`, `toInt`, and `toFloat`.
- Emit those NIR expressions as SwiftUI-compatible Swift and Compose-compatible Kotlin.
- Add `testdata/corpus/go/expressions.gsx` plus Swift and Kotlin golden output.

**Result:** GoSX fixtures now prove props, signals, handlers, event payloads, text inputs, textarea/multiline input, non-string event fields, computed values, conditional view nodes, component references, loop view nodes, ternary-style conditionals, string helpers, and conversions through the same shared NIR-to-native path.

**Remaining proof after this slice:**
- Broaden runtime smoke apps beyond Counter so non-counter generated fixtures are exercised inside real iOS and Android app shells.
