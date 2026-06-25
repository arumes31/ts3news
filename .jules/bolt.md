## 2024-05-24 - Inline Regexp Compilation
**Learning:** Found an inline `regexp.MustCompile` in `cleanRedditTitle` that was compiled on every call. In benchmarks, `strings.Join(strings.Fields(t), " ")` is ~14x faster and creates fewer allocations than an inline regex for normalizing whitespace.
**Action:** Always prefer `strings.Fields` + `strings.Join` for simple whitespace normalization over regex, and never compile regex inside a hot loop or frequently called function.
