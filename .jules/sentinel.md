## 2026-07-20 - Improper Caching of Access-Denied States
**Vulnerability:** The `handleDenied` endpoint was returning `http.StatusOK` (200) instead of a proper access-denied status code. This can lead to browsers or CDNs caching the sensitive state of being denied, or masking authorization failures from security monitoring tools.
**Learning:** Returning 200 OK for denied pages is an anti-pattern. HTTP handlers that deny access must explicitly use proper error semantics to prevent improper caching and aid in security observability.
**Prevention:** Always return `http.StatusForbidden` (403) or `http.StatusUnauthorized` (401) for access-denied HTTP handlers rather than defaulting to `http.StatusOK` (200).
