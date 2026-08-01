## YYYY-MM-DD - Dynamic Regex Compilation in Hot Paths
**Learning:** `regexp.MustCompile` inside a function recompiles the regex on every execution, causing severe performance degradation.
**Action:** Always declare `regexp.MustCompile` at the package level, or for simple whitespace normalization, use `strings.Join(strings.Fields(s), " ")` which is ~12x faster.
