package bot

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAbyssFeedbackJuiceProgramContracts(t *testing.T) {
	t.Parallel()

	root := abyssAAARepositoryRoot(t)
	checks := map[string][]string{
		"internal/bot/webassets/abyss.html": {
			"lastElevatorDepth", "changed&&!reduceMotion", "ab-boss-spawn-shake", "animateNumber(el", "pityFill').classList.add('pulse')",
			"achievementBannerQueue", "ab-downed", "setBiomeChip", "Slot-machine drama (#338)", "vaultOverlay", "coinPile",
		},
		"internal/bot/webassets/abyss_combat_recorder.html": {
			"playerMax*.04", "enemyMax*.04", "spawnCombatDamageFloat",
		},
		"internal/bot/webassets/abyss_combat_feedback.html": {
			"kind==='bank'", "kind==='loot'", "readLiveCombatPreference('abyssCombatAudio','off')",
		},
		"internal/bot/webassets/abyss_polish.html": {
			"playLiveCombatCue('bank'", "playLiveCombatCue('loot'", "wrapAfter('recordBurst'", "function celebrate",
		},
		"internal/bot/webassets/abyss_core_interface.html": {
			"milestoneToast", "updateParallax", "renderDropStreakFlame", "renderDepthPresentation",
		},
		"internal/bot/webassets/abyss_core_interface.css": {
			"ab-boss-hp.has-phases", "ab-depth-backdrop", "ab-escrow-danger", "ab-drop-streak-flame",
		},
		"internal/bot/webassets/abyss_loot_presentation.css": {
			"beam-rare", "beam-legendary", "beam-celestial", "beam-eternal",
		},
		"internal/bot/webassets/abyss_longterm.html": {
			"key:'ab_vignette'", "function updateVignette",
		},
		"internal/bot/webassets/abyss_ui200.css": {
			"ab-amb-fire", "ab-amb-frost", "ab-amb-void",
		},
		"internal/bot/webassets/style.css": {
			"button:active:not(:disabled)", "dial-idle", "shaft-flash", "ab-insured-tag.glow",
		},
		"internal/bot/webassets/abyss_feedback_juice.html": {
			"function trackFrame", "Damage dealt", "Damage taken", "Biggest hit", "Cache lost", "Lesson from the deep", "function decorateBankSummary", "function syncCoinPile",
		},
		"internal/bot/webassets/abyss_feedback_juice.css": {
			"ab-cache-pile-grow", "ab-run-telemetry", "ab-run-lesson", "prefers-reduced-motion", "forced-colors",
		},
	}
	for relative, required := range checks {
		raw, err := os.ReadFile(filepath.Join(root, relative))
		if err != nil {
			t.Fatalf("read %s: %v", relative, err)
		}
		content := string(raw)
		for _, token := range required {
			if !strings.Contains(content, token) {
				t.Errorf("%s is missing %q", relative, token)
			}
		}
	}
}

func TestAbyssFeedbackJuiceAssetsAreIntegratedOnce(t *testing.T) {
	t.Parallel()

	page, err := webAssets.ReadFile("webassets/abyss.html")
	if err != nil {
		t.Fatalf("read Abyss page: %v", err)
	}
	pageSource := string(page)
	for _, token := range []string{
		`{{asset "/static/abyss_feedback_juice.css"}}`,
		`{{template "abyssFeedbackJuice" .}}`,
	} {
		if strings.Count(pageSource, token) != 1 {
			t.Errorf("Abyss page must contain %q exactly once", token)
		}
	}

	routes, err := os.ReadFile(filepath.Join(abyssAAARepositoryRoot(t), "internal/bot/web.go"))
	if err != nil {
		t.Fatalf("read web routes: %v", err)
	}
	if strings.Count(string(routes), `/static/abyss_feedback_juice.css`) != 1 {
		t.Error("feedback juice stylesheet must have exactly one route")
	}
}
