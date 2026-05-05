# GoSX handler parity

**Goal:** Push the GoSX front-end proof past Counter by covering the portable handler forms already supported by NIR and the SwiftUI/Compose emitters.

**Scope:**
- Add `testdata/corpus/go/panel.gsx` with typed props, two signals, `vstack`/`hstack` layout, named multi-statement handlers, and a static `data-on-click` inline handler.
- Assert the lowered NIR carries canonical prop refs (`props.start`, `props.label`), two signal declarations, tap handlers, multi-statement signal sets, and literal string assignment.
- Add iOS and Android golden output for the GoSX Panel fixture.

**Result:** The GoSX bridge now proves more than single-signal Counter parity without expanding the NIR contract. Both native emitters produce deterministic generated source for multi-signal, multi-handler GoSX components.

**Remaining proof after this slice:**
- Event payload expressions (`event.value`) and native text inputs.
- Computed values via a NIR `ComputedDecl`.
- Conditional, loop, and component-reference view nodes.
