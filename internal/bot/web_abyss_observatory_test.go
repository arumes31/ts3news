package bot

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestAbyssObservatoryAssetsAndRoutesAreWired(t *testing.T) {
	t.Parallel()

	page, err := webAssets.ReadFile("webassets/abyss.html")
	if err != nil {
		t.Fatalf("read abyss page: %v", err)
	}
	panel, err := webAssets.ReadFile("webassets/abyss_observatory.html")
	if err != nil {
		t.Fatalf("read observatory panel: %v", err)
	}
	styles, err := webAssets.ReadFile("webassets/abyss_observatory.css")
	if err != nil {
		t.Fatalf("read observatory styles: %v", err)
	}
	server, err := os.ReadFile(filepath.Join(abyssAAARepositoryRoot(t), "internal", "bot", "web.go"))
	if err != nil {
		t.Fatalf("read web server: %v", err)
	}
	for label, contract := range map[string][]byte{
		"stylesheet":      []byte(`{{asset "/static/abyss_observatory.css"}}`),
		"panel":           []byte(`{{template "abyssObservatory" .}}`),
		"script":          []byte(`{{template "abyssObservatoryJS" .}}`),
		"escaped logs":    []byte(`consEsc(String(line))`),
		"opaque panel":    []byte(`background:#090f18!important`),
		"token route":     []byte(`"/api/abyss/api-token"`),
		"run replay":      []byte(`"/api/abyss/run/replay"`),
		"token stats API": []byte(`"/api/abyss/stats"`),
	} {
		haystack := panel
		switch label {
		case "stylesheet", "panel", "script":
			haystack = page
		case "opaque panel":
			haystack = styles
		case "token route", "run replay", "token stats API":
			haystack = server
		}
		if !bytes.Contains(haystack, contract) {
			t.Errorf("missing %s contract %q", label, contract)
		}
	}
}
