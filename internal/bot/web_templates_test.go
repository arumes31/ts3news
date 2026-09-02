package bot

import (
	"bytes"
	"strings"
	"testing"
)

// TestTemplatesParse ensures every embedded web template parses (catches HTML
// template syntax errors before they reach runtime). NewWebServer only parses
// templates and does not dereference the bot during construction.
func TestTemplatesParse(t *testing.T) {
	if _, err := NewWebServer(nil); err != nil {
		t.Fatalf("web templates failed to parse: %v", err)
	}
}

func TestAbyssCommandTheme(t *testing.T) {
	t.Parallel()

	server, err := NewWebServer(nil)
	if err != nil {
		t.Fatalf("NewWebServer: %v", err)
	}
	abyss := server.tmpl.Lookup("abyss")
	if abyss == nil {
		t.Fatal("abyss template is missing")
	}
	source := abyss.Tree.Root.String()
	for _, required := range []string{"abyss_command.css", `class="abyss-command-page"`} {
		if !strings.Contains(source, required) {
			t.Errorf("abyss template is missing command theme marker %q", required)
		}
	}
}

func TestAbyssLivePartials(t *testing.T) {
	server, err := NewWebServer(nil)
	if err != nil {
		t.Fatalf("NewWebServer: %v", err)
	}

	abyss := server.tmpl.Lookup("abyss")
	if abyss == nil {
		t.Fatal("abyss template is missing")
	}
	abyssSource := abyss.Tree.Root.String()
	for _, partialName := range []string{
		"abyssLiveControls",
		"abyssLiveActionBarJS",
		"abyssCombatFeedbackJS",
	} {
		if !strings.Contains(abyssSource, `{{template "`+partialName+`" .}}`) {
			t.Errorf("abyss template does not invoke %q at its extraction point", partialName)
		}
	}

	tests := []struct {
		name         string
		templateName string
		contains     string
	}{
		{
			name:         "controls markup",
			templateName: "abyssLiveControls",
			contains:     `id="liveCombat"`,
		},
		{
			name:         "action bar script",
			templateName: "abyssLiveActionBarJS",
			contains:     "function startLiveCombat",
		},
		{
			name:         "combat feedback controls",
			templateName: "abyssCombatFeedbackControls",
			contains:     `id="liveAudioToggle"`,
		},
		{
			name:         "combat feedback script",
			templateName: "abyssCombatFeedbackJS",
			contains:     "function emitLiveCombatFeedback",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var rendered bytes.Buffer
			if err := server.tmpl.ExecuteTemplate(&rendered, tt.templateName, nil); err != nil {
				t.Fatalf("ExecuteTemplate(%q): %v", tt.templateName, err)
			}
			if !strings.Contains(rendered.String(), tt.contains) {
				t.Errorf("rendered %q does not contain %q", tt.templateName, tt.contains)
			}
		})
	}
	if !strings.Contains(server.tmpl.Lookup("abyssLiveActionBarJS").Tree.Root.String(), "ev.lastEventId") {
		t.Fatal("live action bar script does not preserve SSE event sequence IDs")
	}
	liveScripts := server.tmpl.Lookup("abyssLiveActionBarJS").Tree.Root.String() +
		server.tmpl.Lookup("abyssPixelCombatJS").Tree.Root.String() +
		server.tmpl.Lookup("abyssCombatFeedbackJS").Tree.Root.String()
	for _, required := range []string{
		"scheduleLiveReconnect",
		"RECEIVED · ",
		"REPLACED · ",
		"liveEffectEstimate",
		"state.recommended",
		"liveInitiative",
		"state.enemy_intents",
		"/api/abyss/combat/ready",
		"setLiveReady",
		"/api/abyss/combat/time",
		"spendLiveTimeBank",
		"/api/abyss/combat/pause",
		"setLivePauseMode",
		"/api/abyss/combat/policy",
		"setLivePolicy",
		"renderLiveLoadouts",
		"liveLoadoutPage",
		"liveRecap",
		"THREAT ",
		"entry.textContent=text",
		"if(sessionID!==liveCombatSessionID)return",
		"setTimeout(function(){dismissFinishedLiveCombat(completedSession);},900)",
		"!resumeLiveCombatPhase(state.phase)",
		"phase==='starting'||phase==='planning'||phase==='resolving'",
		"reorderLiveAction",
		"toggleLivePin",
		"--cooldown-angle",
		"Mana after:",
		"low-count",
		" unavailable",
		"TIMEOUT →",
		"Use your last ",
		"toggleLiveTargetLock",
		"configureLiveShortcuts",
		"pollLiveGamepad",
		"toggleLiveCompact",
		"toggleLiveLog",
		"setLiveLogFilter",
		"title=\"'+consEsc",
		"renderLiveRoundTimeline",
		"option.kind==='relic'",
		"option.kind==='companion'",
		"Combo tags:",
		"liveSocialStatus",
	} {
		if !strings.Contains(liveScripts, required) {
			t.Errorf("live combat scripts are missing %q feedback", required)
		}
	}
	liveControls := server.tmpl.Lookup("abyssLiveControls").Tree.Root.String()
	for _, required := range []string{"liveTacticVote", "liveSocial('revive_vote'", "liveSocial('abandon_vote'"} {
		if !strings.Contains(liveControls, required) {
			t.Errorf("live controls are missing %q social control", required)
		}
	}
	abyssMain := server.tmpl.Lookup("abyss").Tree.Root.String()
	for _, required := range []string{
		"scheduleLootSort",
		"saveAbyssLootSettings",
		"toggleAbyssLootReserve",
		"showAbyssMaterialFlow",
		"abyssPreset",
		"vault_cache",
		"vault_tokens",
		"vault_materials",
		"veteranTrack",
		"ab-progression-clarity",
		"ab-sanctuary-stage-",
		"partyLootRule",
		"coopPaceFilter",
		"shareLastAbyssReplay",
		"armAbyssGhostReplay",
		"abyssAAARuntime",
		"AB_LOG_VIRTUAL_THRESHOLD",
	} {
		if !strings.Contains(abyssMain, required) {
			t.Errorf("abyss template is missing %q loot control", required)
		}
	}
	runtime := server.tmpl.Lookup("abyssAAARuntime")
	if runtime == nil {
		t.Fatal("Abyss AAA runtime partial is missing")
	}
	runtimeSource := runtime.Tree.Root.String()
	for _, required := range []string{
		`range abyssChangelog`,
		`kind:kind`,
		`/api/abyss/client-error`,
		`unhandledrejection`,
	} {
		if !strings.Contains(runtimeSource, required) {
			t.Errorf("Abyss AAA runtime is missing %q", required)
		}
	}
	abyssTree := server.tmpl.Lookup("abysstree").Tree.Root.String()
	for _, required := range []string{
		"/api/abyss/tree/batch_allocate",
		"confirmQueuedAllocations",
		"LOADOUT_NAMES",
		"SEASONAL_SECTOR",
		"abysstree-progression",
		"abysstree-accessibility",
	} {
		if !strings.Contains(abyssTree, required) {
			t.Errorf("Abyss tree template is missing %q progression control", required)
		}
	}
	accessibility := server.tmpl.Lookup("abysstree-accessibility").Tree.Root.String()
	for _, required := range []string{
		"treeContrastToggle", "treeAnimationToggle", "treeMutationLive",
		"ArrowLeft", "tryAllocate(node.id)", "prefers-reduced-motion",
		"tree-touch-hit", "touchDistance", "parsedIconGeometry",
		"virtualizeDecorations", "requestAnimationFrame", "treePerformanceMeter",
	} {
		if !strings.Contains(accessibility, required) {
			t.Errorf("Abyss tree accessibility partial is missing %q", required)
		}
	}
	progression := server.tmpl.Lookup("abysstree-progression").Tree.Root.String()
	for _, required := range []string{
		"treePointSources", "treeSectorMastery", "treeArchetypeScores",
		"treeRecommendations", "treeArchetypePin", "treeCompletionGoal",
		"abyssTreeDismissedRecommendations", "treeAchievementProgress",
	} {
		if !strings.Contains(progression, required) {
			t.Errorf("Abyss tree progression partial is missing %q", required)
		}
	}

	scriptServer, err := NewWebServer(nil)
	if err != nil {
		t.Fatalf("NewWebServer for script context: %v", err)
	}
	scriptTemplates := scriptServer.tmpl
	if _, err := scriptTemplates.Parse(
		`{{define "abyssLiveScriptHost"}}<script>{{template "abyssLiveActionBarJS" .}}</script>{{end}}`,
	); err != nil {
		t.Fatalf("parse live script host: %v", err)
	}
	var renderedScript bytes.Buffer
	if err := scriptTemplates.ExecuteTemplate(&renderedScript, "abyssLiveScriptHost", nil); err != nil {
		t.Fatalf("execute live script host: %v", err)
	}
	if strings.Contains(renderedScript.String(), "ZgotmplZ") {
		t.Fatal("live action bar script is unsafe in its script context")
	}
}

func TestAbyssLiveCombatResetsEventCursorForNewSession(t *testing.T) {
	t.Parallel()

	server, err := NewWebServer(nil)
	if err != nil {
		t.Fatalf("NewWebServer: %v", err)
	}
	partial := server.tmpl.Lookup("abyssLiveActionBarJS")
	if partial == nil {
		t.Fatal("live action bar script is missing")
	}
	source := partial.Tree.Root.String()
	reset := strings.Index(source, "liveLastEventID=liveEventCursor(payload&&payload.state)")
	render := strings.Index(source, "renderLiveCombat(payload.state)")
	if !strings.Contains(source, "function liveEventCursor(state)") {
		t.Fatal("live action bar script has no session event cursor parser")
	}
	if reset < 0 || render < 0 || reset > render {
		t.Fatal("new live combat must reset its SSE cursor before rendering the initial snapshot")
	}
}

func TestAbyssTreeAndForgePartials(t *testing.T) {
	t.Parallel()

	server, err := NewWebServer(nil)
	if err != nil {
		t.Fatalf("NewWebServer: %v", err)
	}
	tests := []struct {
		host     string
		partial  string
		required []string
	}{
		{
			host: "abysstree", partial: "abysstree-navigation",
			required: []string{"treeSectorFilter", "treeEffectFilter", "abyssTreeBookmarks", "treeMinimapSvg"},
		},
		{
			host: "abysstree", partial: "abysstree-planner",
			required: []string{"treePlanToggle", "plan_preview", "treePlanCommit", "abyssTreeDraft"},
		},
		{
			host: "abysstree", partial: "abysstree-inspector",
			required: []string{"treeInspectorTitle", "shortestRouteData", "abyssTreeDiscoveredNodes", "treeInspectorLink"},
		},
		{
			host: "abyss", partial: "abyss-forge-workstation",
			required: []string{"forge-discipline-tabs", "forgeEligibilityFilter", "abyssForgeFavorites", "recordForgeRecent"},
		},
		{
			host: "abyss", partial: "abyss-forge-planner",
			required: []string{"color-scheme:dark", ".forge-plan select", "forgePlanOperation", "forgeRecipeSearch"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.partial, func(t *testing.T) {
			host := server.tmpl.Lookup(tt.host)
			if host == nil || !strings.Contains(host.Tree.Root.String(), `{{template "`+tt.partial+`" .}}`) {
				t.Fatalf("%s does not invoke %s", tt.host, tt.partial)
			}
			partial := server.tmpl.Lookup(tt.partial)
			if partial == nil {
				t.Fatalf("partial %s is missing", tt.partial)
			}
			source := partial.Tree.Root.String()
			for _, required := range tt.required {
				if !strings.Contains(source, required) {
					t.Errorf("%s is missing %q", tt.partial, required)
				}
			}
		})
	}
}
