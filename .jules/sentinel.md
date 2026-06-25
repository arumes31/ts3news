## 2024-05-24 - Missing Security Headers and API Auth Leakage
**Vulnerability:** Missing critical security headers (like X-Content-Type-Options) allowing MIME sniffing, and API endpoints returning HTML redirects instead of JSON on auth failure.
**Learning:** Web servers exposing both HTML pages and JSON APIs often misconfigure authentication middleware, using the HTML redirect middleware for API endpoints. Also, standard Go HTTP servers need explicit configuration to set security headers.
**Prevention:** Always implement a global security headers middleware. Create distinct authentication wrappers for HTML pages (redirect to `/denied`) and JSON APIs (return `{"ok": false, "error": "unauthenticated"}`).
