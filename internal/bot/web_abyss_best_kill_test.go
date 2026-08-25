package bot

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestAbyssBestKillUsesAuthoritativeLeaderboardOrder(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	killedAt := time.Date(2026, time.August, 25, 22, 0, 0, 0, time.FixedZone("test", 2*60*60))
	mock.ExpectQuery("SELECT boss_name,depth,kill_time_ms,tier,killed_at.*ORDER BY depth DESC,kill_time_ms ASC,killed_at DESC LIMIT 1").WithArgs("hunter").
		WillReturnRows(sqlmock.NewRows([]string{"boss_name", "depth", "kill_time_ms", "tier", "killed_at"}).
			AddRow("Abyssus", 100, int64(12_345), "insanity", killedAt))
	view := (&Bot{DB: database}).abyssBestKill("hunter")
	if !view.Available || view.Boss != "Abyssus" || view.Depth != 100 || view.KillTime != "12.3s" || view.TierName != "Insanity" || view.KilledAt != "2026-08-25" {
		t.Fatalf("best kill = %+v", view)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAbyssBestKillLookupHasMatchingIndex(t *testing.T) {
	root := abyssAAARepositoryRoot(t)
	up, err := os.ReadFile(filepath.Join(root, "internal", "db", "migrations", "0090_abyss_best_kill_index.up.sql"))
	if err != nil {
		t.Fatal(err)
	}
	down, err := os.ReadFile(filepath.Join(root, "internal", "db", "migrations", "0090_abyss_best_kill_index.down.sql"))
	if err != nil {
		t.Fatal(err)
	}
	for _, token := range []string{"client_uid", "depth DESC", "kill_time_ms", "killed_at DESC"} {
		if !strings.Contains(string(up), token) {
			t.Errorf("best-kill index is missing %q", token)
		}
	}
	if !strings.Contains(string(down), "DROP INDEX IF EXISTS abyss_boss_kills_player_best_idx") {
		t.Fatal("best-kill index migration is not reversible")
	}
}

func TestAbyssBestKillShareCardContract(t *testing.T) {
	page, err := webAssets.ReadFile("webassets/abyss_best_kill.html")
	if err != nil {
		t.Fatal(err)
	}
	styles, err := webAssets.ReadFile("webassets/abyss_best_kill.css")
	if err != nil {
		t.Fatal(err)
	}
	for _, token := range []string{"PERSONAL BOSS RECORD", "Copy PNG", "Copy text", "ClipboardItem", "abyss-best-kill.png", "generated locally"} {
		if !strings.Contains(string(page), token) {
			t.Errorf("best-kill card is missing %q", token)
		}
	}
	for _, token := range []string{".ab-best-kill", ".ab-best-kill-seal", "@media (max-width: 680px)"} {
		if !strings.Contains(string(styles), token) {
			t.Errorf("best-kill styles are missing %q", token)
		}
	}
}
