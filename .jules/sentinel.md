
## 2024-06-24 - Missing Security Headers
**Vulnerability:** The web portal application lacked standard HTTP security headers (such as Content-Security-Policy, X-Frame-Options).
**Learning:** Default net/http handlers do not automatically apply fundamental security constraints, leaving them vulnerable to typical client-side attacks like clickjacking and XSS if a vulnerability exists.
**Prevention:** Implement a global middleware function (e.g., `withSecurityHeaders`) that wraps the main multiplexer and consistently injects required security headers across all endpoints.
