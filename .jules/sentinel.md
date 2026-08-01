## 2026-08-01 - Improper Status Code for Access Denied Endpoints
**Vulnerability:** The `/denied` endpoint returned an `http.StatusOK` (200) instead of `http.StatusForbidden` (403).
**Learning:** Returning 200 for access denied endpoints can lead to improper caching of sensitive states by browsers or CDNs, potentially obscuring access control failures.
**Prevention:** Always use the correct semantic HTTP status code (e.g., 401 or 403) for access control failures, rather than relying solely on the rendered HTML to convey the error.
