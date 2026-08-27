package bot

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ts3news/internal/i18n"
)

func TestAbyssAccessibilityQoLProgramContracts(t *testing.T) {
	t.Parallel()

	root := abyssAAARepositoryRoot(t)
	checks := map[string][]string{
		"internal/bot/webassets/abyss_accessibility.html": {
			"function auditAccessibility", "function accessibleName", "function enhanceStaticEmoji",
			"function enhanceEmojiIcons", "function enhanceRarityGlyphs", "function initConsumableDialogFocus",
		},
		"internal/bot/webassets/abyss_accessibility.css": {
			"body.ab-high-contrast", "body.ab-colorblind", "min-height: 44px",
			"@media (prefers-reduced-motion: reduce)", "@media (forced-colors: active)",
		},
		"internal/bot/webassets/abyss_quality_of_life.html": {
			"function applyLogVerbosity", "function updateFavicon", "function setNotifications",
		},
		"internal/bot/webassets/abyss_player_experience.html": {
			"ab_player_motion", "ab_player_cv", "ab-user-reduce-motion", "function updateTitleAlert",
		},
		"internal/bot/webassets/abyss_longterm.html": {
			"key:'ab_logmono'", "key:'ab_logverbosity'", "key:'ab_contrast'", "key:'ab_notifications'",
		},
		"internal/bot/webassets/abyss_onboarding.html": {
			"function startAbyssTour", "ab-glossary-term", "CR · Combat Rating",
		},
		"internal/bot/webassets/abyss_insights.html": {
			"histCsvBtn", "type:'text/csv'", "histJsonBtn",
		},
		"internal/bot/webassets/abyss.html": {
			"abyssAPILatency", "abyssSessionWarning", "function setBusy", "abOrigTitle",
		},
		"internal/bot/web_abyss_setup_state.go": {
			"abyssEntrySetup", "loadAbyssEntrySetup", "saveAbyssEntrySetup",
		},
	}

	for relative, required := range checks {
		content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relative)))
		if err != nil {
			t.Fatalf("read %s: %v", relative, err)
		}
		for _, token := range required {
			if !strings.Contains(string(content), token) {
				t.Errorf("%s is missing %q", relative, token)
			}
		}
	}
}

func TestAbyssLocaleCatalogIsCompleteAcrossShippedLocales(t *testing.T) {
	if err := i18n.InitWithLocale(i18n.LocaleEnUS); err != nil {
		t.Fatalf("initialize locale catalog: %v", err)
	}

	coverage := i18n.MessageCoverage("web.abyss.")
	if len(coverage) != len(i18n.AllLocales) {
		t.Fatalf("Abyss locale coverage rows = %d, want %d", len(coverage), len(i18n.AllLocales))
	}
	for _, locale := range coverage {
		if locale.Total < 43 {
			t.Errorf("locale %s has only %d canonical Abyss messages", locale.Locale, locale.Total)
		}
		if !locale.Complete || locale.Present != locale.Total || len(locale.Missing) != 0 {
			t.Errorf(
				"locale %s coverage incomplete: present=%d missing=%v total=%d",
				locale.Locale,
				locale.Present,
				locale.Missing,
				locale.Total,
			)
		}
	}
}
