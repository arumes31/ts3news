package bot

import (
	"strings"
	"testing"
)

func TestAbyssQualityOfLifeContracts(t *testing.T) {
	t.Parallel()

	server, err := NewWebServer(nil)
	if err != nil {
		t.Fatalf("NewWebServer: %v", err)
	}
	partial := server.tmpl.Lookup("abyss-quality-of-life")
	if partial == nil {
		t.Fatal("Abyss quality-of-life template is missing")
	}
	source := partial.Tree.Root.String()
	for _, required := range []string{
		"function setNotifications",
		"Notification.requestPermission()",
		"function notify",
		"sessionStorage.getItem(key)",
		"function summaryLine",
		"function applyLogVerbosity",
		"function updateFavicon",
		"data-sold-total",
	} {
		if !strings.Contains(source, required) {
			t.Errorf("quality-of-life module is missing %q", required)
		}
	}
}

func TestAbyssQualityOfLifeIntegrationHooks(t *testing.T) {
	t.Parallel()

	files := []string{
		"webassets/partials.html",
		"webassets/abyss.html",
		"webassets/abyss_live.html",
		"webassets/abyss_longterm.html",
		"webassets/abyss_combat_recorder.html",
		"webassets/abyss_item_numbers.html",
		"webassets/abyss_insights.html",
		"webassets/ah.html",
	}
	var source strings.Builder
	for _, name := range files {
		content, err := webAssets.ReadFile(name)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		source.Write(content)
	}
	for _, required := range []string{
		`template "abyss-quality-of-life"`,
		"key:'ab_logverbosity'",
		"key:'ab_logmono'",
		"key:'ab_notifications'",
		"AbyssQoL.logVerbosity()==='summary'",
		"AbyssQoL.summaryLine(line)",
		"AbyssQoL.notify('bounty'",
		"AbyssQoL.notify('revive'",
		`data-sold-total="{{.Economy.SoldTotal}}"`,
		"toLocaleString(undefined",
		"Intl.NumberFormat(undefined",
		"histCsvBtn",
		"histJsonBtn",
		"type:'text/csv'",
		"navigator.clipboard.writeText",
	} {
		if !strings.Contains(source.String(), required) {
			t.Errorf("quality-of-life integration is missing %q", required)
		}
	}
}
