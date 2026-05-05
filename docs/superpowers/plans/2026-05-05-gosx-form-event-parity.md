# GoSX form-event parity

**Goal:** Prove that native form controls and non-string GoSX event payload fields survive the GoSX-to-NIR-to-native pipeline.

**Scope:**
- Add `testdata/corpus/go/form_controls.gsx` with textarea, checkbox, key, and selected-index handlers.
- Lower native-relevant tags and attributes for `textarea`, checkbox inputs, `select`, `option`, `checked`, and `selectedIndex`.
- Generalize event payload emission from a hardcoded `event.value` mapping to per-event-field bindings.
- Emit SwiftUI `TextEditor`, `Toggle`, key submission, and `Picker` bindings.
- Emit Compose multiline `TextField`, `Checkbox`, key-event modifier, and indexed selection controls.
- Lock the generated Swift and Kotlin with FormControls golden tests.

**Result:** GoSX fixtures now prove props, signals, handlers, event payloads, text inputs, textarea/multiline input, non-string event fields, computed values, conditional view nodes, component references, and loop view nodes through the same shared NIR-to-native path.

**Remaining proof after this slice:**
- More expression coverage: ternary, string helpers, and conversions.
