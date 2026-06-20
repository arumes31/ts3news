## 2025-02-27 - [Missing Security Headers on Web Portal]
**Vulnerability:** [The Go HTTP server for the web portal lacked basic security headers like `X-Frame-Options`, `X-Content-Type-Options`, `Content-Security-Policy`, etc.]
**Learning:** [A common gap in standard `net/http` servers is that they don't provide security headers by default. This application relies heavily on `mux.HandleFunc` which needs to be wrapped via a middleware before passing to the `http.Server` to inject headers.]
**Prevention:** [Always wrap the main mux or individual authenticated routes with a middleware that sets strict security headers, specifically `X-Frame-Options` and `Strict-Transport-Security` for portal-like UIs.]
