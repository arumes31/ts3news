//go:build e2e

package bot

import (
	"mime"
	"net/http"
	"path"
	"strings"
	"testing"

	"ts3news/internal/i18n"
)

func TestAbyssE2EServer(t *testing.T) {
	if err := i18n.InitWithLocale(i18n.LocaleEnUS); err != nil {
		t.Fatal(err)
	}
	server, err := NewWebServer(nil)
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("/static/", func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/static/")
		if name == "" || path.Clean(name) != name {
			http.NotFound(w, r)
			return
		}
		ServeAsset(
			w,
			r,
			"webassets/"+name,
			mime.TypeByExtension(path.Ext(name)),
		)
	})
	mux.HandleFunc("/abyss", func(w http.ResponseWriter, r *http.Request) {
		fixture := abyssGoldenFixture(r.URL.Query().Get("active") == "1")
		if r.URL.Query().Get("room") == abyssForgeFloorType {
			run := fixture["Run"].(abyssRun)
			run.FloorType = "event"
			run.EventState = `{"type":"forge_floor"}`
			fixture["Run"] = run
		}
		if err := server.tmpl.ExecuteTemplate(w, "abyss", fixture); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
	mux.HandleFunc("/abyss/ops", func(w http.ResponseWriter, _ *http.Request) {
		if err := server.tmpl.ExecuteTemplate(w, "abyssOps", map[string]any{"Title": "Abyss Operations", "Nav": "ops"}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
	t.Log("Abyss E2E fixture listening on http://127.0.0.1:18082")
	if err := http.ListenAndServe("127.0.0.1:18082", mux); err != nil {
		t.Fatal(err)
	}
}
