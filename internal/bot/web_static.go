package bot

import (
	"io/fs"
	"net/http"
	"path"
	"strings"
)

// serveStaticAsset exposes embedded browser assets, never page templates or docs.
// Keep this handler shared with browser fixtures so new assets work in production.
func serveStaticAsset(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/static/")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if !strings.HasPrefix(r.URL.Path, "/static/") || !fs.ValidPath(name) || strings.Contains(name, "\\") {
		http.NotFound(w, r)
		return
	}

	var contentType string
	switch path.Ext(name) {
	case ".css":
		contentType = "text/css; charset=utf-8"
	case ".js":
		contentType = "application/javascript; charset=utf-8"
	case ".png":
		contentType = "image/png"
	case ".svg":
		contentType = "image/svg+xml"
	default:
		http.NotFound(w, r)
		return
	}
	ServeAsset(w, r, "webassets/"+name, contentType)
}
