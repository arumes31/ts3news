## 2025-02-23 - Avoid regexp recompilation in loops
**Learning:** In Go, calling `regexp.MustCompile()` inside a function recompiles the regex on every execution, creating a performance bottleneck when called in a loop.
**Action:** Use `strings.Join(strings.Fields(t), " ")` for whitespace normalization instead, which is much faster.
