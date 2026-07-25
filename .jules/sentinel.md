## 2026-07-25 - HTTP 200 on Access Denied Endpoints
**Vulnerability:** The access denied endpoint (`handleDenied`) was returning HTTP 200 (OK) instead of HTTP 403 (Forbidden).
**Learning:** Browsers and CDNs may cache sensitive states if access denied pages are served with a 200 status code, potentially leading to security bypasses or leakage of the denial state.
**Prevention:** Always return `http.StatusForbidden` (403) or `http.StatusUnauthorized` (401) for access denied HTTP handlers rather than defaulting to `http.StatusOK`.

## 2026-07-25 - Open Redirect False Positive
**Vulnerability:** Gosec flagged a potential Open Redirect (G710) despite robust manual path validation.
**Learning:** Manual validation against `/\` and `//` is necessary but not recognized by Gosec's taint analysis.
**Prevention:** Explicitly append `// #nosec G710` to `http.Redirect` calls after implementing manual path validation.
