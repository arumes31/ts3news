package bot

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRiftPublicMetadataConditionalAndPrivateBoundary(t *testing.T) {
	now := time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)
	call := func(tag, method string) *httptest.ResponseRecorder {
		q := httptest.NewRequest(method, "/api/abyss/rift/metadata?uid=private-player", nil)
		q.Header.Set("Cookie", "session=private")
		q.Header.Set("If-None-Match", tag)
		w := httptest.NewRecorder()
		writeRiftMetadata(w, q, riftPublicMetadata(now))
		return w
	}
	first := call("", http.MethodGet)
	if first.Code != 200 {
		t.Fatal(first.Code)
	}
	var body map[string]json.RawMessage
	if err := json.Unmarshal(first.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	allowed := map[string]bool{"class_names": true, "class_options": true, "rooms": true, "levels": true, "bestiary": true, "rarities": true}
	if len(body) != len(allowed) {
		t.Fatal("unexpected fields", body)
	}
	for key := range body {
		if !allowed[key] {
			t.Fatal("private field", key)
		}
	}
	var levels []any
	if err := json.Unmarshal(body["levels"], &levels); err != nil {
		t.Fatal(err)
	}
	if len(levels) != 100 {
		t.Fatal("incomplete campaign")
	}
	tag := first.Header().Get("ETag")
	if tag == "" || first.Header().Get("Cache-Control") != "public, max-age=0, must-revalidate" {
		t.Fatal("missing cache contract")
	}
	for _, match := range []string{tag, "W/" + tag, `"stale", ` + tag, "*"} {
		w := call(match, http.MethodGet)
		if w.Code != 304 || w.Body.Len() != 0 || w.Header().Get("ETag") != tag {
			t.Fatal("conditional response", match, w.Code)
		}
	}
	if call(`"stale"`, http.MethodGet).Code != 200 {
		t.Fatal("stale cache accepted")
	}
	if call("", http.MethodPost).Code != 405 {
		t.Fatal("metadata mutation allowed")
	}
	q := httptest.NewRequest(http.MethodGet, "/", nil)
	q.Header.Set("If-None-Match", tag)
	w := httptest.NewRecorder()
	writeRiftMetadata(w, q, map[string]any{"levels": []int{1}})
	if w.Code != 200 || w.Header().Get("ETag") == tag {
		t.Fatal("content change did not invalidate cache")
	}
	t.Logf("public metadata body: %d bytes; revalidation: 0 body bytes", first.Body.Len())
}
