package bot

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ts3news/internal/content"
)

func TestRequireAbyssTreeLayout(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		layoutHash string
		wantCalled bool
	}{
		{name: "legacy client without header", wantCalled: true},
		{name: "current layout", layoutHash: content.AbyssTree().TopologyHash(), wantCalled: true},
		{name: "stale layout", layoutHash: "stale-layout", wantCalled: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			called := false
			next := func(w http.ResponseWriter, _ *http.Request, uid string) {
				called = true
				if uid != "user-1" {
					t.Fatalf("uid = %q, want user-1", uid)
				}
				writeJSON(w, map[string]any{"ok": true})
			}

			req := httptest.NewRequest(http.MethodPost, "/api/abyss/tree/allocate", nil)
			if tt.layoutHash != "" {
				req.Header.Set(abyssTreeLayoutHeader, tt.layoutHash)
			}
			recorder := httptest.NewRecorder()
			(&WebServer{}).requireAbyssTreeLayout(next)(recorder, req, "user-1")

			if called != tt.wantCalled {
				t.Fatalf("next called = %v, want %v", called, tt.wantCalled)
			}
			var response struct {
				OK         bool   `json:"ok"`
				Error      string `json:"error"`
				LayoutHash string `json:"layout_hash"`
			}
			if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if tt.wantCalled {
				if !response.OK {
					t.Fatalf("response ok = false, want true")
				}
				return
			}
			if response.OK || response.Error == "" {
				t.Fatalf("stale response = %+v", response)
			}
			if response.LayoutHash != content.AbyssTree().TopologyHash() {
				t.Fatalf("layout hash = %q, want current topology hash", response.LayoutHash)
			}
		})
	}
}
