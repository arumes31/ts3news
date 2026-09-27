//go:build e2e

package bot

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"ts3news/internal/rift"
)

func TestRiftLongReceiptFixture(t *testing.T) {
	server, err := NewWebServer(nil)
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	registerRiftFixture(mux, server)
	get := func(path string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.AddCookie(&http.Cookie{Name: "rift_fixture", Value: "long-receipt-test"})
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		if w.Code != 200 {
			t.Fatalf("%s: %d", path, w.Code)
		}
		return w
	}
	get("/abyss/rift?scenario=long-receipt&subclass=arcanist")
	var response struct {
		Run *rift.Run `json:"run"`
	}
	if err := json.Unmarshal(get("/api/abyss/rift").Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	run := response.Run
	if run == nil {
		t.Fatal("long receipt fixture missing")
	}
	if run.Status != "complete" || run.Level.ID != 100 || run.Build.Class != "arcanist" {
		t.Fatal("incorrect completed campaign")
	}
	if len(run.BankedItems) < 1000 || len(run.BankedLoot) != len(run.BankedItems) {
		t.Fatal("missing full receipt")
	}
	if len(run.CompletedLevels) != 100 {
		t.Fatal("missing mission completion history")
	}
	seen := map[int]bool{}
	for _, item := range run.BankedLoot {
		seen[item.Mission] = true
	}
	if len(seen) != 100 {
		t.Fatal("receipt does not cover full campaign")
	}
	saved, err := encodeRift(run)
	if err != nil {
		t.Fatal(err)
	}
	restored, err := decodeRift(saved)
	if err != nil {
		t.Fatal(err)
	}
	if len(restored.BankedLoot) != rift.ReceiptHistoryLimit || restored.TotalBankedItems() != len(run.BankedItems) {
		t.Fatal("receipt compaction lost its authoritative total")
	}
}
