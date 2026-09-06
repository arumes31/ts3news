package bot

import (
	"bytes"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"path"
	"strings"
	"testing"

	"ts3news/internal/config"
)

func TestProductionStaticAssets(t *testing.T) {
	server := &WebServer{bot: &Bot{Cfg: &config.Config{EnableAbyss: true}}}
	mux := server.routes()
	types := map[string]string{
		".css": "text/css",
		".js":  "application/javascript",
		".png": "image/png",
		".svg": "image/svg+xml",
	}
	entries, err := fs.ReadDir(webAssets, "webassets")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		contentType, public := types[path.Ext(entry.Name())]
		if entry.IsDir() || !public {
			continue
		}
		t.Run(entry.Name(), func(t *testing.T) {
			url := AssetURL("/static/" + entry.Name())
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, url, nil))
			if response.Code != http.StatusOK {
				t.Fatalf("GET %s: status %d, location %q", url, response.Code, response.Header().Get("Location"))
			}
			if got := response.Header().Get("Content-Type"); !strings.HasPrefix(got, contentType) {
				t.Errorf("Content-Type = %q, want %q", got, contentType)
			}
			want, err := webAssets.ReadFile("webassets/" + entry.Name())
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(response.Body.Bytes(), want) {
				t.Error("response differs from embedded asset")
			}
			if got := response.Header().Get("Cache-Control"); got != "public, max-age=31536000, immutable" {
				t.Errorf("Cache-Control = %q", got)
			}
		})
	}
}

func TestProductionStaticAssetsRejectPrivateAndMissingFiles(t *testing.T) {
	mux := (&WebServer{bot: &Bot{Cfg: &config.Config{EnableAbyss: true}}}).routes()
	for _, name := range []string{"armory.html", "partials.html", "abyss_changelog.md", "missing.css", "", "%2e%2e/go.mod"} {
		t.Run(name, func(t *testing.T) {
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/static/"+name, nil))
			if response.Code != http.StatusNotFound {
				t.Errorf("status = %d, want 404", response.Code)
			}
			if strings.Contains(response.Header().Get("Cache-Control"), "immutable") {
				t.Error("invalid asset response must not be cached as immutable")
			}
		})
	}
}

func TestStaticAssetHandlerRejectsInvalidPaths(t *testing.T) {
	for _, url := range []string{"/static/../style.css", "/static/icons/../style.css", "/static/%2e%2e/style.css", "/static/icons%5cstyle.css", "/webassets/style.css"} {
		t.Run(url, func(t *testing.T) {
			response := httptest.NewRecorder()
			serveStaticAsset(response, httptest.NewRequest(http.MethodGet, url, nil))
			if response.Code != http.StatusNotFound {
				t.Errorf("status = %d, want 404", response.Code)
			}
		})
	}
}
