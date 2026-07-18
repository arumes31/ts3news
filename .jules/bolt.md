## 2024-10-24 - Avoid Inline Regex Compilation for Whitespace Normalization
**Learning:** Using `regexp.MustCompile` inline within frequently called functions forces the regex to be recompiled on every invocation, causing significant performance degradation. For simple whitespace normalization, `strings.Join(strings.Fields(t), " ")` is significantly faster.
**Action:** Avoid inline regex compilation, especially in loops or frequently called functions. Use `strings.Fields` and `strings.Join` for simple whitespace normalization.
