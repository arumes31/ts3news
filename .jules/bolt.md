## 2025-02-20 - Go Performance Anti-pattern: Inline Regex Compilation
**Learning:** Compiling regular expressions dynamically in the hot path function (e.g. `regexp.MustCompile` inside a function called frequently, like processing RSS entries) causes a significant performance hit and unnecessary allocations.
**Action:** Always declare `regexp.MustCompile` at the package level as a global variable. Alternatively, if performing simple whitespace normalization, avoid regex entirely and use `strings.Join(strings.Fields(t), " ")` which is typically an order of magnitude faster.
