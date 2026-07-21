## 2026-07-21 - [Performance] Inline regexp.MustCompile is inefficient
**Learning:** Using `regexp.MustCompile` dynamically inside functions causes unnecessary runtime overhead and recompilation in Go. It can be significantly slower than simpler string operations.
**Action:** Use string functions like `strings.Fields` combined with `strings.Join` for whitespace normalization instead of regular expressions to improve performance and avoid re-compilations in hot paths.
