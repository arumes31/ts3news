package bot

import (
	"bytes"
	"strings"
	"testing"
)

func TestLoadAbyssChangelog(t *testing.T) {
	t.Parallel()

	changelog, err := loadAbyssChangelog()
	if err != nil {
		t.Fatalf("loadAbyssChangelog: %v", err)
	}
	if len(changelog) == 0 {
		t.Fatal("changelog has no releases")
	}
	latest := changelog[0]
	if latest.Date != "2026-08-26" || latest.Title == "" {
		t.Fatalf("latest release = %+v", latest)
	}
	if len(latest.Items) < 3 {
		t.Fatalf("latest release has %d items, want at least 3", len(latest.Items))
	}

	server, err := NewWebServer(nil)
	if err != nil {
		t.Fatalf("NewWebServer: %v", err)
	}
	var rendered bytes.Buffer
	if err := server.tmpl.ExecuteTemplate(&rendered, "abyssAAARuntime", nil); err != nil {
		t.Fatalf("render Abyss AAA runtime: %v", err)
	}
	for _, required := range []string{latest.Title, latest.Items[0], "/api/abyss/client-error"} {
		if !strings.Contains(rendered.String(), required) {
			t.Errorf("rendered runtime is missing %q", required)
		}
	}
}
