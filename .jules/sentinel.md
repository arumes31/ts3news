## 2025-02-14 - Fix API Endpoint Middleware Misconfiguration
**Vulnerability:** API endpoints were using standard web page authentication middleware (`s.auth`) instead of API-specific middleware (`s.authAPI`). This caused unauthenticated API requests to redirect to an HTML error page (`/denied`) instead of returning a JSON error response, which could break clients expecting JSON or leak information about endpoint existence.
**Learning:** This codebase uses two separate auth middleware functions: `s.auth` for HTML routes and `s.authAPI` for JSON APIs. It is easy to accidentally copy-paste `s.auth` when defining new API routes.
**Prevention:** Always verify that API routes defined under `// Authenticated JSON APIs.` use `s.authAPI` and not `s.auth` to ensure consistent JSON error formats.
