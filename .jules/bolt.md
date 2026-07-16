## 2024-05-24 - Whitespace Normalization Optimization
**Learning:** Using `strings.Fields` combined with `strings.Join` is an order of magnitude faster than `regexp.MustCompile("\\s+").ReplaceAllString` for simple whitespace normalization and avoids inline regex recompilation in hot paths.
**Action:** Always prefer `strings.Fields` and `strings.Join` for normalizing spaces, and avoid inline `regexp.MustCompile` inside functions to prevent unnecessary allocations.
