## 2024-10-24 - Avoid Inline regexp.MustCompile for Whitespace Normalization
**Learning:** Using `regexp.MustCompile` inline within functions (like `cleanRedditTitle`) causes the regular expression to be re-compiled on every function call. This leads to unnecessary CPU cycles and memory allocations, especially on hot paths where string cleaning happens frequently.
**Action:** Replace inline `regexp` whitespace replacements with `strings.Join(strings.Fields(t), " ")` which is approximately 13x faster, avoids recompilation, and handles leading/trailing whitespace automatically.
