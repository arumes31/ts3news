## 2026-07-27 - Proper Status Codes for Access Denied Pages
**Vulnerability:** Access-denied HTTP handler (`handleDenied`) was defaulting to `http.StatusOK` (200), which can cause CDNs or browsers to improperly cache access-denied states as normal successful pages.
**Learning:** Returning 200 OK for access denied pages undermines HTTP semantics and cache control, potentially leading to caching of sensitive state logic or false positives in uptime monitoring.
**Prevention:** Always explicitly return `http.StatusForbidden` (403) or `http.StatusUnauthorized` (401) for access denied routes.
