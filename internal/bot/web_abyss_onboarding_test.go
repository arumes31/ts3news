package bot

import (
	"bytes"
	"strings"
	"testing"
)

func TestAbyssOnboardingTemplates(t *testing.T) {
	t.Parallel()

	server, err := NewWebServer(nil)
	if err != nil {
		t.Fatalf("NewWebServer: %v", err)
	}

	var panel bytes.Buffer
	data := map[string]any{
		"Stats":              map[string]any{"LifetimeFloors": int64(0), "Deaths": int64(0)},
		"Run":                map[string]any{"Active": false, "Escrow": int64(0), "Insured": 0},
		"FreeEntryAvailable": true,
	}
	if err := server.tmpl.ExecuteTemplate(&panel, "abyssOnboardingPanel", data); err != nil {
		t.Fatalf("render onboarding panel: %v", err)
	}
	for _, required := range []string{
		`id="abyssFieldManual"`,
		`id="abyssPrimer"`,
		`id="abyssTour"`,
		`aria-modal="true"`,
		`data-lifetime-floors="0"`,
	} {
		if !strings.Contains(panel.String(), required) {
			t.Errorf("rendered onboarding panel is missing %q", required)
		}
	}

	script := server.tmpl.Lookup("abyssOnboardingJS")
	if script == nil {
		t.Fatal("Abyss onboarding script partial is missing")
	}
	for _, required := range []string{
		"function startAbyssTour",
		"function renderAbyssTourStep",
		"function recommendAbyssTier",
		"function renderAbyssDeathCoach",
		"function showAbyssFirstDeathExplainer",
		"abyssOnboardingV1",
		"prefers-reduced-motion",
		"ArrowRight",
		"focusAbyssTourControl",
	} {
		if !strings.Contains(script.Tree.Root.String(), required) {
			t.Errorf("Abyss onboarding script is missing %q", required)
		}
	}
}

func TestAbyssOnboardingAssetsAndModalAccessibility(t *testing.T) {
	t.Parallel()

	page, err := webAssets.ReadFile("webassets/abyss.html")
	if err != nil {
		t.Fatalf("read Abyss page: %v", err)
	}
	css, err := webAssets.ReadFile("webassets/abyss_onboarding.css")
	if err != nil {
		t.Fatalf("read onboarding stylesheet: %v", err)
	}
	uiCSS, err := webAssets.ReadFile("webassets/abyss_ui200.css")
	if err != nil {
		t.Fatalf("read Abyss UI stylesheet: %v", err)
	}
	partials, err := webAssets.ReadFile("webassets/partials.html")
	if err != nil {
		t.Fatalf("read shared partials: %v", err)
	}

	checks := []struct {
		name   string
		source string
		want   string
	}{
		{name: "stylesheet", source: string(page), want: `{{asset "/static/abyss_onboarding.css"}}`},
		{name: "panel partial", source: string(page), want: `{{template "abyssOnboardingPanel" .}}`},
		{name: "script partial", source: string(page), want: `{{template "abyssOnboardingJS" .}}`},
		{name: "tour cutout", source: string(css), want: ".ab-tour-spot"},
		{name: "reduced motion", source: string(css), want: "@media (prefers-reduced-motion: reduce)"},
		{name: "forced colors", source: string(css), want: "@media (forced-colors: active)"},
		{name: "modal focus return", source: string(partials), want: "modalReturnFocus"},
		{name: "modal focus trap", source: string(partials), want: "trapSharedModalFocus"},
		{name: "dialog label", source: string(partials), want: "aria-labelledby"},
		{name: "run history empty state", source: string(page), want: "Your legend starts at floor 1"},
		{name: "bestiary empty state", source: string(page), want: "descend and slay to fill these pages"},
		{name: "lore locked state", source: string(page), want: `{{if .Unlocked}}unlocked{{else}}locked{{end}}`},
		{name: "new panel markers", source: string(page), want: "function initNewDots"},
		{name: "new marker persistence", source: string(page), want: "localStorage.getItem('ab_ui_ver')"},
		{name: "new marker dismissal", source: string(page), want: "dot.onclick=function"},
		{name: "new marker styling", source: string(uiCSS), want: ".ab-newdot"},
	}
	for _, check := range checks {
		check := check
		t.Run(check.name, func(t *testing.T) {
			t.Parallel()
			if !strings.Contains(check.source, check.want) {
				t.Errorf("%s is missing %q", check.name, check.want)
			}
		})
	}
}
