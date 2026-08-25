//go:build e2e

package bot

import (
	"net/http"
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
	mux.HandleFunc("/abyss", func(w http.ResponseWriter, r *http.Request) {
		fixture := abyssGoldenFixture(r.URL.Query().Get("active") == "1")
		if err := server.tmpl.ExecuteTemplate(w, "abyss", fixture); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
	t.Log("Abyss E2E fixture listening on http://127.0.0.1:18082")
	if err := http.ListenAndServe("127.0.0.1:18082", mux); err != nil {
		t.Fatal(err)
	}
}
