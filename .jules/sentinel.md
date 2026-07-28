## 2024-05-24 - Improper HTTP Status Code on Denied Page
**Vulnerability:** The access-denied page was returning HTTP 200 OK.
**Learning:** Returning 200 OK for an access-denied state can cause browsers and CDNs to cache the response improperly, potentially leading to security semantic issues.
**Prevention:** Ensure access-denied HTTP handlers explicitly return `http.StatusForbidden` (403) or `http.StatusUnauthorized` (401).
