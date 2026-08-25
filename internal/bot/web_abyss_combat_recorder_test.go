package bot

import (
	"strings"
	"testing"
)

func TestAbyssCombatRecorderContracts(t *testing.T) {
	t.Parallel()

	server, err := NewWebServer(nil)
	if err != nil {
		t.Fatalf("NewWebServer: %v", err)
	}

	var panel strings.Builder
	if err := server.tmpl.ExecuteTemplate(&panel, "abyssCombatRecorderPanel", nil); err != nil {
		t.Fatalf("render combat recorder panel: %v", err)
	}
	for _, required := range []string{
		`id="logCategoryFilters"`,
		`id="logPlaybackRate"`,
		`id="logSkipButton"`,
		`id="logAutoScroll"`,
		`id="logJumpLatest"`,
		`id="fightStatus"`,
		`id="fightResultDelta"`,
		`id="damageFloatLayer"`,
		`aria-live="polite"`,
	} {
		if !strings.Contains(panel.String(), required) {
			t.Errorf("combat recorder panel is missing %q", required)
		}
	}

	script := server.tmpl.Lookup("abyssCombatRecorderJS")
	if script == nil {
		t.Fatal("Abyss combat recorder script partial is missing")
	}
	for _, required := range []string{
		"function startCombatRecorder",
		"function classifyCombatLogLine",
		"function combatRecorderDelay",
		"function skipCombatPlayback",
		"function updateCombatRecorderFrame",
		"function finishCombatRecorder",
		"function renderFightResultDelta",
		"function setCombatAutoScroll",
		"abyssCombatRecorderV1",
		"AB_RECORDER_MAX_FLOATS",
	} {
		if !strings.Contains(script.Tree.Root.String(), required) {
			t.Errorf("combat recorder script is missing %q", required)
		}
	}
}

func TestAbyssCombatRecorderAssetsAndHooks(t *testing.T) {
	t.Parallel()

	page, err := webAssets.ReadFile("webassets/abyss.html")
	if err != nil {
		t.Fatalf("read Abyss page: %v", err)
	}
	css, err := webAssets.ReadFile("webassets/abyss_combat_recorder.css")
	if err != nil {
		t.Fatalf("read combat recorder stylesheet: %v", err)
	}
	uiCSS, err := webAssets.ReadFile("webassets/abyss_ui200.css")
	if err != nil {
		t.Fatalf("read Abyss UI stylesheet: %v", err)
	}

	for _, required := range []string{
		`{{asset "/static/abyss_combat_recorder.css"}}`,
		`{{template "abyssCombatRecorderPanel" .}}`,
		`{{template "abyssCombatRecorderJS" .}}`,
		"startCombatRecorder(",
		"combatRecorderDelay(",
		"finishCombatRecorder(",
		"recordCombatLogLine(",
		"updateCombatRecorderFrame(",
		"middle combat rounds",
		"row.onkeydown=",
		"function maybeBossCard",
		"ab-log-critical-event",
		"navigator.clipboard.writeText(lastFightText)",
	} {
		if !strings.Contains(string(page), required) {
			t.Errorf("Abyss page is missing recorder hook %q", required)
		}
	}
	for _, required := range []string{
		"--recorder-you",
		".ab-recorder-status",
		".ab-damage-float",
		".ab-fight-delta",
		"@media (prefers-reduced-motion: reduce)",
		"@media (forced-colors: active)",
	} {
		if !strings.Contains(string(css), required) {
			t.Errorf("combat recorder CSS is missing %q", required)
		}
	}
	for _, required := range []string{".ab-log-critical-event::before", "ab-critical-event-flash"} {
		if !strings.Contains(string(uiCSS), required) {
			t.Errorf("Abyss UI CSS is missing %q", required)
		}
	}
}
