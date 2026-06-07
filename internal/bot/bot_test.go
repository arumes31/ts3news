package bot

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"ts3news/internal/config"
	"ts3news/internal/games"
)

func TestFilterNewGames(t *testing.T) {
	allGames := []games.Game{
		{ID: 1, Title: "Game One"},
		{ID: 2, Title: "Game Two"},
		{ID: 3, Title: "Game Three"},
	}

	tests := []struct {
		name        string
		alreadySent []string // game keys
		wantTitles  []string
	}{
		{"None sent", nil, []string{"Game One", "Game Two", "Game Three"}},
		{"Some sent", []string{"gameone", "gamethree"}, []string{"Game Two"}},
		{"All sent", []string{"gameone", "gametwo", "gamethree"}, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filterNewGames(allGames, tt.alreadySent)
			if len(got) != len(tt.wantTitles) {
				t.Fatalf("got %d candidates, want %d", len(got), len(tt.wantTitles))
			}
			for i, title := range tt.wantTitles {
				if got[i].Title != title {
					t.Errorf("at index %d: got %q, want %q", i, got[i].Title, title)
				}
			}
		})
	}
}

func TestDatabasePersistence(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	b := &Bot{Cfg: &config.Config{}, DB: db}

	uid := "abc123uniqueid="
	nickname := "Daniel"
	gameKey := "gravitycircuit"
	gameTitle := "Gravity Circuit"

	// markAsSent inserts (client_uid, game_key, game_title, client_nickname).
	mock.ExpectExec("INSERT INTO sent_notifications").
		WithArgs(uid, gameKey, gameTitle, nickname).
		WillReturnResult(sqlmock.NewResult(1, 1))

	if err := b.markAsSent(uid, nickname, gameKey, gameTitle); err != nil {
		t.Errorf("markAsSent failed: %v", err)
	}

	// getSentGames (ResendAfterDays = 0 => no time filter) returns game keys.
	rows := sqlmock.NewRows([]string{"game_key"}).AddRow(gameKey)
	mock.ExpectQuery("SELECT game_key FROM sent_notifications").
		WithArgs(uid).
		WillReturnRows(rows)

	keys, err := b.getSentGames(uid)
	if err != nil {
		t.Errorf("getSentGames failed: %v", err)
	}
	if len(keys) != 1 || keys[0] != gameKey {
		t.Errorf("got keys %v, want [%s]", keys, gameKey)
	}

	// touchUser
	mock.ExpectQuery("SELECT last_session_connected_ms FROM users WHERE client_uid = \\$1").
		WithArgs(uid).
		WillReturnRows(sqlmock.NewRows([]string{"last_session_connected_ms"}).AddRow(0))
	mock.ExpectExec("INSERT INTO users").WillReturnResult(sqlmock.NewResult(1, 1))
	_ = b.touchUser(uid, nickname, 1000)

	// CleanupDeadUsers
	b.Cfg.DeadUserDays = 30
	mock.ExpectExec("DELETE FROM sent_notifications").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("DELETE FROM users").WillReturnResult(sqlmock.NewResult(1, 1))
	_, _ = b.CleanupDeadUsers()

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %s", err)
	}
}
