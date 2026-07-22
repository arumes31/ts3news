## 2024-05-24 - Improper HTTP Status Code on Denied Page
**Vulnerability:** The access denied endpoint (`handleDenied`) was returning an HTTP 200 OK status code instead of a 403 Forbidden.
**Learning:** Returning a 200 OK for access-denied errors can lead to improper caching of sensitive or error states by browsers and CDNs, and breaks security semantics.
**Prevention:** Always use appropriate HTTP status codes like 403 (Forbidden) or 401 (Unauthorized) for access denied handlers.