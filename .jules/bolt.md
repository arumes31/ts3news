## 2024-05-24 - Avoid Inline Regexp Compilation in Hot Paths
**Learning:** Compiling regular expressions inline using `regexp.MustCompile` inside frequently called functions causes unnecessary allocations and CPU overhead, especially for simple whitespace normalizations.
**Action:** Use `strings.Fields` combined with `strings.Join` instead of regular expressions for simple whitespace normalization in Go.
