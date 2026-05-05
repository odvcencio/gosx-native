# Swift Full-Grammar Spike

**Date:** 2026-05-05

**Goal:** Replace the M1 constrained Swift grammar overlay with a direct `grammargen.SwiftGrammar()` extension that adds JSX as a Swift primary expression.

**Result:** Not ready to land. The M1 subset remains the correct implementation until the scanner boundary is solved.

## Findings

1. Full Swift parsing requires the upstream Swift external scanner. Without it, common Swift syntax such as `->`, implicit semicolons, and custom operators fails.
2. Appending `jsx_text` as a normal external token makes it valid in too many Swift expression states. If GSX text scans first, it can consume host Swift like `-> Node`. If the host scanner scans first, `jsx_text` can still consume ordinary expressions such as `signal(props.start)` before JSX has actually begun.
3. Making `jsx_text` an internal token avoids host-Swift overconsumption, but nested JSX with child elements still fails around the parent closing tag.

## Required Follow-Up

Full Swift + GSX needs a JSX-context-aware scanner boundary. The likely fix is to model JSX entry/exit tokens explicitly in the scanner, or improve gotreesitter's external-token validity/per-stack scanner state so `jsx_text` is only offered inside JSX child contexts.

Until then, M1 intentionally keeps the constrained grammar that parses the portable Counter subset and feeds the lowerer/emitter deterministically.
