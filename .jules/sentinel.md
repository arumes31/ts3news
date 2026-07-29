## 2025-05-24 - Fix Access-Denied HTTP Status
**Vulnerability:** Access-denied handlers were returning 200 OK instead of proper error codes like 401 or 403.
**Learning:** Returning 200 OK for denied states can cause browsers or CDNs to improperly cache the denied page as a valid response, leading to unexpected behavior.
**Prevention:** Always use proper HTTP status codes (e.g. `http.StatusForbidden`) for access-denied or unauthenticated responses.
