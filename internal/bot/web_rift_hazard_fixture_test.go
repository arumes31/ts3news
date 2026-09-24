//go:build e2e

package bot

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"ts3news/internal/rift"
)

func TestRiftHazardFixtureRoutes(t *testing.T) {
	server, err := NewWebServer(nil)
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	registerRiftFixture(mux, server)
	kinds := map[string]bool{}
	for _, level := range rift.Campaign() {
		for _, room := range level.Rooms {
			for _, h := range room.Hazards {
				kinds[h.Kind] = true
			}
		}
	}
	if len(kinds) != 8 {
		t.Fatalf("review hazard coverage: %d kinds", len(kinds))
	}
	for kind := range kinds {
		t.Run(kind, func(t *testing.T) {
			get := func(path string) *httptest.ResponseRecorder {
				req := httptest.NewRequest(http.MethodGet, path, nil)
				req.AddCookie(&http.Cookie{Name: "rift_fixture", Value: "hazard-fixture-" + kind})
				w := httptest.NewRecorder()
				mux.ServeHTTP(w, req)
				if w.Code != 200 {
					t.Fatalf("%s: %d", path, w.Code)
				}
				return w
			}
			get("/abyss/rift?scenario=hazard-" + kind)
			var response struct {
				Run *rift.Run `json:"run"`
			}
			if err := json.Unmarshal(get("/api/abyss/rift").Body.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			run := response.Run
			if run == nil {
				t.Fatal("hazard fixture not created")
			}
			hazards := run.Arena().Hazards
			if !run.Paused || run.Status != "fighting" || len(hazards) != 1 || hazards[0].Kind != kind {
				t.Fatalf("invalid hazard scene: %+v", run)
			}
			phase := hazards[0].Phase(run.Clock)
			if phase < 1.2 || phase >= 1.2+hazards[0].Duration {
				t.Fatalf("hazard is not active: %f", phase)
			}
			source := rift.Campaign()[run.Level.ID-1].Rooms[run.Room].Hazards
			matched := false
			for _, h := range source {
				if h == hazards[0] {
					matched = true
				}
			}
			if !matched {
				t.Fatal("fixture changed authored hazard behavior")
			}
		})
	}
}
