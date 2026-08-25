package bot

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAbyssCodexExplorerContracts(t *testing.T) {
	t.Parallel()

	server, err := NewWebServer(nil)
	if err != nil {
		t.Fatalf("NewWebServer: %v", err)
	}
	for _, name := range []string{"abyssCodexExplorer", "abyssCodexExplorerJS"} {
		if server.tmpl.Lookup(name) == nil {
			t.Fatalf("template %q is missing", name)
		}
	}

	assets := []string{
		"webassets/abyss.html",
		"webassets/abyss_codex_explorer.html",
		"webassets/abyss_codex_explorer.css",
	}
	var source strings.Builder
	var explorerSource string
	for _, asset := range assets {
		raw, err := webAssets.ReadFile(asset)
		if err != nil {
			t.Fatalf("read %s: %v", asset, err)
		}
		source.Write(raw)
		if asset == "webassets/abyss_codex_explorer.html" {
			explorerSource = string(raw)
		}
	}
	for _, required := range []string{
		"codexSearch", "codexFamily", "codexSort", "codexMinKills", "codexFavoritesOnly",
		"ab_codex_favorites", "ab_codex_layout", "codexFamilyProgress", "codexInspector",
		"data-first-kill", "data-last-kill", "data-milestone", "data-mastered",
		"abyss-bestiary.json", "window.print", "#bestiary=", "codexCompareDelta",
		"ArrowDown", "Copy summary", "Copy deep link", "Mastered (100+)",
	} {
		if !strings.Contains(source.String(), required) {
			t.Errorf("Codex explorer is missing %q", required)
		}
	}
	if strings.Contains(explorerSource, ".innerHTML") {
		t.Error("Codex explorer must build player-derived content with DOM text nodes")
	}
}

func TestAbyssBestiaryLastKillMigrationIsForwardCompatible(t *testing.T) {
	t.Parallel()
	root := abyssAAARepositoryRoot(t)
	checks := map[string][]string{
		"0083_abyss_bestiary_last_kill.up.sql": {
			"ADD COLUMN IF NOT EXISTS last_kill_at TIMESTAMPTZ",
			"SET last_kill_at = first_kill_at",
			"ALTER COLUMN last_kill_at SET NOT NULL",
			"abyss_bestiary_recent_idx",
		},
		"0083_abyss_bestiary_last_kill.down.sql": {
			"DROP INDEX IF EXISTS abyss_bestiary_recent_idx",
			"DROP COLUMN IF EXISTS last_kill_at",
		},
	}
	for file, required := range checks {
		raw, err := os.ReadFile(filepath.Join(root, "internal", "db", "migrations", file))
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		for _, token := range required {
			if !strings.Contains(string(raw), token) {
				t.Errorf("%s is missing %q", file, token)
			}
		}
	}
}
