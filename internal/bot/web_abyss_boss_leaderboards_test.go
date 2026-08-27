package bot

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestTopBossSpeedBoardsGroupsPersonalBestRanks(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()

	now := time.Now()
	mock.ExpectQuery("WITH personal_best AS").
		WithArgs("hell", 3).
		WillReturnRows(sqlmock.NewRows([]string{
			"boss_rank", "client_uid", "nick", "boss_name", "depth", "kill_time_ms", "killed_at",
		}).
			AddRow(1, "fast", "Fast", "Abyssus", 50, int64(900), now).
			AddRow(2, "steady", "Steady", "Abyssus", 60, int64(1100), now).
			AddRow(1, "steady", "Steady", "Gorgoroth", 10, int64(700), now))

	boards := (&Bot{DB: database}).topBossSpeedBoards("hell", 3)
	if len(boards) != 2 || boards[0].Name != "Abyssus" || len(boards[0].Rows) != 2 || boards[0].Rows[1].Rank != 2 {
		t.Fatalf("grouped boss boards = %#v", boards)
	}
	if boards[1].Name != "Gorgoroth" || boards[1].Rows[0].KillTimeMs != 700 {
		t.Fatalf("second boss board = %#v", boards[1])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAbyssBossSpeedBoardPersistenceAndUIContracts(t *testing.T) {
	t.Parallel()

	root := abyssAAARepositoryRoot(t)
	checks := map[string][]string{
		filepath.Join(root, "internal", "db", "migrations", "0086_abyss_boss_speed_boards.up.sql"): {
			"tier, boss_name, client_uid, kill_time_ms", "depth DESC", "killed_at",
		},
		filepath.Join(root, "internal", "bot", "webassets", "abyss_boss_leaderboards.html"): {
			"Fastest Kills by Boss", "one personal best per player", "BossSpeed", "KillTimeMs",
		},
		filepath.Join(root, "internal", "bot", "webassets", "abyss.html"): {
			"abyss_boss_leaderboards.css", `template "abyssCompetition" .`,
		},
		filepath.Join(root, "internal", "bot", "webassets", "abyss_competition.html"): {
			`template "abyssBossSpeedBoards" $.Leaders`,
		},
	}
	for path, required := range checks {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		for _, token := range required {
			if !strings.Contains(string(raw), token) {
				t.Errorf("%s is missing %q", filepath.Base(path), token)
			}
		}
	}
}
