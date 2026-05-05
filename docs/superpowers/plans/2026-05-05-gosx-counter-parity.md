# GoSX Counter parity

**Goal:** Prove the M3 front-end path by lowering a real GoSX `.gsx` Counter through the existing gosx parser/IR and into gosx-native NIR, then emitting the same SwiftUI and Compose Counter already proven by the Swift+GSX fixture.

**Scope:**
- Add a self-contained `testdata/corpus/go/counter.gsx` with typed props, one signal, and named increment/decrement handlers.
- Add a `lower/gosx` bridge that consumes `gosx.Compile` output, preserves typed props, normalizes common web tags/events to native symbolic NIR, and converts gosx island expressions into portable `RxExpr`/`RxStmt`.
- Update `gsxnative compile` and `gsxnative emit` to dispatch `.gsx` and `.swift.gsx` files.
- Keep the existing iOS and Android runtime proofs unchanged; this slice proves front-end parity into the same back ends.

**Result:** GoSX Counter now emits the same generated SwiftUI and Jetpack Compose code as the Swift fixture. Unit coverage compares GoSX NIR semantics against the existing Counter NIR snapshot after stripping source-span differences, and CLI tests cover GoSX compile plus iOS/Android emit.

**Remaining proof after this slice:**
- Preserve byte-accurate GoSX spans in NIR instead of line/column-only spans inherited from `gosx/ir`.
- Broaden the GoSX corpus beyond Counter: inline handlers, computed values, form events, conditionals, loops, and component refs.
- Decide whether the web fixture in `../gosx/examples/counter/counter.gsx` should become self-contained or remain a runtime-bound presentation fixture.
