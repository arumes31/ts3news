package bot

import (
	"os"
	"strings"
	"testing"
)

func TestAbyssCriticalFumblePresentationContract(t *testing.T) {
	t.Parallel()

	page, err := webAssets.ReadFile("webassets/abyss.html")
	if err != nil {
		t.Fatal(err)
	}
	server, err := os.ReadFile("web.go")
	if err != nil {
		t.Fatal(err)
	}
	live, err := webAssets.ReadFile("webassets/abyss_live.html")
	if err != nil {
		t.Fatal(err)
	}
	styles, err := webAssets.ReadFile("webassets/abyss_critical_fumble.css")
	if err != nil {
		t.Fatal(err)
	}

	for _, token := range []string{
		"/static/abyss_critical_fumble.css",
		"ab-log-critical-fumble",
	} {
		if !strings.Contains(string(page), token) {
			t.Errorf("Abyss page is missing %q", token)
		}
	}
	for _, token := range []string{
		"textContent=line",
		"ab-live-critical-fumble",
	} {
		if !strings.Contains(string(live), token) {
			t.Errorf("live fumble log is missing %q", token)
		}
	}
	for _, token := range []string{
		".ab-log-line.ab-log-critical-fumble",
		".ab-live-feed > .ab-live-critical-fumble",
		"DAMAGE INTACT",
		"prefers-reduced-motion: reduce",
		"forced-colors: active",
		"max-width: 520px",
	} {
		if !strings.Contains(string(styles), token) {
			t.Errorf("critical-fumble styles are missing %q", token)
		}
	}
	if !strings.Contains(string(server), "/static/abyss_critical_fumble.css") {
		t.Fatal("critical-fumble stylesheet has no explicit asset route")
	}
}
