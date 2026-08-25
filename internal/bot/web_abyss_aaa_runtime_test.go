package bot

import (
	"reflect"
	"strings"
	"testing"
)

func TestAbyssAAARuntimeKeepsClientReportsMetadataOnly(t *testing.T) {
	t.Parallel()

	reportType := reflect.TypeOf(abyssClientErrorReport{})
	fields := make(map[string]struct{}, reportType.NumField())
	for index := range reportType.NumField() {
		fields[reportType.Field(index).Tag.Get("json")] = struct{}{}
	}
	for _, forbidden := range []string{"message", "stack", "nickname", "user_agent", "url"} {
		if _, exists := fields[forbidden]; exists {
			t.Errorf("client report payload exposes forbidden field %q", forbidden)
		}
	}
	for _, required := range []string{"kind", "source", "path", "line", "column"} {
		if _, exists := fields[required]; !exists {
			t.Errorf("client report payload is missing bounded field %q", required)
		}
	}

	server, err := NewWebServer(nil)
	if err != nil {
		t.Fatalf("NewWebServer: %v", err)
	}
	partial := server.tmpl.Lookup("abyssAAARuntime")
	if partial == nil {
		t.Fatal("Abyss AAA runtime partial is missing")
	}
	source := partial.Tree.Root.String()
	for _, required := range []string{
		"sent.length>=3",
		"now-recent[signature]<10000",
		"url.origin!==location.origin",
		"credentials:'same-origin'",
		"keepalive:true",
	} {
		if !strings.Contains(source, required) {
			t.Errorf("Abyss client reporting is missing %q", required)
		}
	}
}

func TestAbyssAAAClientPerformanceGuardsAreWired(t *testing.T) {
	t.Parallel()

	page, err := webAssets.ReadFile("webassets/abyss.html")
	if err != nil {
		t.Fatalf("read Abyss page: %v", err)
	}
	source := string(page)
	for _, required := range []string{
		"AB_LOG_VIRTUAL_THRESHOLD=500",
		"function virtualizeCombatLog(log)",
		"window.__abyssLogDOMCount",
		"requestAnimationFrame(function(){hudChipFrame=0;renderHudChipsNow();})",
		"window.__abyssHUDRenderCount",
		"var AB_RARITY_META=Object.freeze([",
		"var AB_RAR_ORDER=AB_RARITY_META.slice().reverse().map",
	} {
		if !strings.Contains(source, required) {
			t.Errorf("Abyss performance guard is missing %q", required)
		}
	}

	runtimeIndex := strings.Index(source, `{{template "abyssAAARuntime" .}}`)
	tabBuildIndex := strings.Index(source, "window.buildAbyssTabs()")
	if runtimeIndex < 0 || tabBuildIndex < 0 || runtimeIndex > tabBuildIndex {
		t.Fatal("Abyss changelog must be present before workspace tabs are built")
	}
}
