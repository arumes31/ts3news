## 2024-05-23 - Avoid inline regexp.MustCompile for simple whitespace tasks
**Learning:** Using inline `regexp.MustCompile` inside frequently called functions (like data parsing/cleaning) causes unnecessary allocations and compilation overhead on every call.
**Action:** For simple whitespace normalization (trimming and collapsing multiple spaces), use `strings.Join(strings.Fields(s), " ")` instead. It's significantly faster and avoids regex overhead completely.
