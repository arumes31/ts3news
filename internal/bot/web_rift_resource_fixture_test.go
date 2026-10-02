//go:build e2e

package bot

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"ts3news/internal/content"
	"ts3news/internal/rift"
)

func TestRiftResourceFixtureAllStylesAndCharges(t *testing.T) {
	server, err := NewWebServer(nil)
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	registerRiftFixture(mux, server)
	var styles []content.AbyssSubclass
	for _, class := range content.AbyssClasses() {
		foundation, ok := content.AbyssFoundationStyle(class.ID)
		if !ok {
			t.Fatal(class.ID)
		}
		styles = append(styles, foundation)
		styles = append(styles, class.Subclasses...)
	}
	for _, style := range styles {
		for charges := 0; charges <= 3; charges++ {
			t.Run(fmt.Sprintf("%s/%d", style.ID, charges), func(t *testing.T) {
				get := func(path string) *httptest.ResponseRecorder {
					req := httptest.NewRequest(http.MethodGet, path, nil)
					req.AddCookie(&http.Cookie{Name: "rift_fixture", Value: "resource-" + style.ID})
					w := httptest.NewRecorder()
					mux.ServeHTTP(w, req)
					if w.Code != 200 {
						t.Fatalf("%s: %d", path, w.Code)
					}
					return w
				}
				get(fmt.Sprintf("/abyss/rift?scenario=checkpoint&subclass=%s&charges=%d", style.ID, charges))
				var response struct {
					Run *rift.Run `json:"run"`
				}
				if err := json.Unmarshal(get("/api/abyss/rift").Body.Bytes(), &response); err != nil {
					t.Fatal(err)
				}
				r := response.Run
				if r == nil {
					t.Fatal("missing resource fixture")
				}
				if r.Build.Class != style.ID || r.Build.Resource != style.Resource || r.Resource != charges {
					t.Fatalf("wrong resource: class=%s label=%s charges=%d", r.Build.Class, r.Build.Resource, r.Resource)
				}
				if len(r.Build.Signatures) != 2 || r.Build.Signatures[0].Name != style.Builder || r.Build.Signatures[1].Name != style.Finisher {
					t.Fatalf("wrong signatures: %+v", r.Build.Signatures)
				}
			})
		}
	}
}
