//go:build e2e

package bot

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"ts3news/internal/rift"
)

func TestRiftTerminalFixtureRoutes(t *testing.T) {
	server, err := NewWebServer(nil)
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	registerRiftFixture(mux, server)
	for _, state := range []string{"complete", "banked", "defeated", "expired"} {
		t.Run(state, func(t *testing.T) {
			get := func(path string) *httptest.ResponseRecorder {
				t.Helper()
				req := httptest.NewRequest(http.MethodGet, path, nil)
				req.AddCookie(&http.Cookie{Name: "rift_fixture", Value: "terminal-test-" + state})
				w := httptest.NewRecorder()
				mux.ServeHTTP(w, req)
				if w.Code != http.StatusOK {
					t.Fatalf("%s: %d %s", path, w.Code, w.Body.String())
				}
				return w
			}
			get("/abyss/rift?scenario=terminal-" + state)
			read := func() *rift.Run {
				t.Helper()
				var response struct {
					Run *rift.Run `json:"run"`
				}
				if err := json.Unmarshal(get("/api/abyss/rift").Body.Bytes(), &response); err != nil {
					t.Fatal(err)
				}
				if response.Run == nil {
					t.Fatal("terminal fixture did not create a run")
				}
				return response.Run
			}
			run := read()
			if run.Status != state || run.Gold != 0 {
				t.Fatalf("unexpected terminal status/rewards: %s / %d", run.Status, run.Gold)
			}
			if state == "expired" {
				if run.BankedGold != 0 || run.PastExpeditions.Gold != 50 {
					t.Fatal("expired rewards not historical")
				}
			} else if run.BankedGold != 50 {
				t.Fatal("secured rewards lost")
			}
			if state == "complete" && (run.Room != 2 || len(run.CompletedLevels) != 1 || run.Player.Pose != "victory") {
				t.Fatal("missing completion state")
			}
			if state == "defeated" && (run.Player.HP != 0 || len(run.Drops) != 0) {
				t.Fatal("missing defeat state")
			}
			get("/abyss/rift")
			if saved := read(); saved.ID != run.ID || saved.Status != state {
				t.Fatal("plain reload replaced terminal fixture")
			}
		})
	}
}
