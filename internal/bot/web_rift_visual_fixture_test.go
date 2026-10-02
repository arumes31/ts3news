//go:build e2e

package bot

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"ts3news/internal/rift"
)

func TestRiftVisualFixtureReproducesSeededSnapshots(t *testing.T) {
	server, err := NewWebServer(nil)
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	registerRiftFixture(mux, server)
	get := func(path, cookie string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.AddCookie(&http.Cookie{Name: "rift_fixture", Value: cookie})
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		return w
	}
	read := func(query, cookie string) *rift.Run {
		t.Helper()
		w := get("/abyss/rift?scenario=visual&"+query, cookie)
		if w.Code != 200 {
			t.Fatal(w.Code, w.Body.String())
		}
		var response struct {
			Run *rift.Run `json:"run"`
		}
		if err := json.Unmarshal(get("/api/abyss/rift", cookie).Body.Bytes(), &response); err != nil {
			t.Fatal(err)
		}
		if response.Run == nil {
			t.Fatal("visual fixture missing")
		}
		return response.Run
	}
	for _, query := range []string{"seed=ruins-v1&level=1&room=0", "seed=forge-v1&level=15&room=1", "seed=boss-v1&level=100&room=2"} {
		a, b := read(query, "visual-a"), read(query, "visual-b")
		if !reflect.DeepEqual(a, b) {
			t.Fatal("same seed differs across sessions")
		}
		if !a.Paused || a.Status != "fighting" {
			t.Fatal("visual scene is not paused combat")
		}
	}
	crowded := read("seed=crowded-v1&crowd=120", "crowded-a")
	if len(crowded.Enemies) != 120 {
		t.Fatalf("crowded scene has %d enemies, want 120", len(crowded.Enemies))
	}
	if !reflect.DeepEqual(crowded, read("seed=crowded-v1&crowd=120", "crowded-b")) {
		t.Fatal("crowded scene differs across sessions")
	}
	ids := make(map[string]bool)
	for _, actor := range crowded.Enemies {
		if ids[actor.ID] || actor.HP <= 0 || actor.X < 35 || actor.X > 1565 || actor.Y < 315 || actor.Y > 490 {
			t.Fatalf("invalid crowded actor: %+v", actor)
		}
		ids[actor.ID] = true
	}
	a, b := read("seed=first", "visual-a"), read("seed=second", "visual-b")
	if reflect.DeepEqual(a.EncounterPlan, b.EncounterPlan) {
		t.Fatal("different seeds did not select different encounters")
	}
	for _, query := range []string{"seed=bad!", "level=0", "level=101", "level=abc", "room=-1", "room=3", "crowd=0", "crowd=201", "crowd=no"} {
		if w := get("/abyss/rift?scenario=visual&"+query, "invalid-visual"); w.Code != 400 {
			t.Fatalf("invalid %s: %d", query, w.Code)
		}
	}
}
