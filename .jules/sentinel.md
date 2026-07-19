## 2026-07-19 - Added Global Security Headers
**Vulnerability:** The web portal lacked basic security headers (like X-Content-Type-Options, X-Frame-Options, Strict-Transport-Security), exposing users to clickjacking, MIME sniffing, and downgrade attacks.
**Learning:** Security headers should be applied globally across all routes. By wrapping the root `http.ServeMux` with a middleware right before server creation, we ensure that every HTTP response, including static assets, benefits from these protections automatically without modifying individual route handlers.
**Prevention:** In Go HTTP server setups, utilize a top-level middleware wrapper on the mux before passing it to `http.Server{Handler: ...}` to guarantee consistent application of security headers.
