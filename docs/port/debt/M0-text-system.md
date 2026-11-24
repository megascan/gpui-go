# M0-text-system debt

- FontFeatures / FontFallbacks intern every distinct list in a process-wide sync.Map that never shrinks (upstream shares Arc lists and frees them). Feature and fallback sets are few in practice.
- D-SCHEMAKW: FontFeatures and FontWeight JSON schemas are reduced until internal/jsonschema gains patternProperties / multipleOf / default.
- TestLineLayout_BreakIndicesMatchICU pins go-text segmenter output against ICU's for two inputs only; ICU new_for_non_complex_scripts and go-text's UAX #14 may differ on other scripts.
