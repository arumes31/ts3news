## 2024-06-23 - Pre-compiling Regex in Go
**Learning:** Compiling regular expressions using `regexp.MustCompile` inside frequently called functions (like `cleanRedditTitle`) creates a hidden performance bottleneck.
**Action:** Always extract static regular expressions to pre-compiled package-level variables to optimize execution speed.
