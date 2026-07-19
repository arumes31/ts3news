package bot

import (
	"net/http"
)

// secureHeaders is a middleware that applies standard security headers
// to all HTTP responses, mitigating risks like XSS, clickjacking, and MIME sniffing.
func secureHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Prevent browsers from MIME-sniffing a response away from the declared content-type.
		w.Header().Set("X-Content-Type-Options", "nosniff")

		// Prevent clickjacking by forbidding the rendering of the page in a frame/iframe.
		w.Header().Set("X-Frame-Options", "DENY")

		// Force the browser to use HTTPS (HSTS). We set this to 1 year (31536000 seconds).
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")

		// Basic XSS protection for older browsers.
		w.Header().Set("X-XSS-Protection", "1; mode=block")

		next.ServeHTTP(w, r)
	})
}
