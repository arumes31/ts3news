package bot

import (
	"testing"
	"ts3news/internal/games"
	"github.com/DATA-DOG/go-sqlmock"
	"ts3news/internal/config"
)

func TestFilterNewGames(t *testing.T) {
	b := &Bot{}
	allGames := []games.Game{
		{ID: 1, Title: "Game 1"},
		{ID: 2, Title: "Game 2"},
		{ID: 3, Title: "Game 3"},
	}

	tests := []struct {
		name        string
		alreadySent []int
		wantIDs     []int
	}{
		{
			name:        "None sent",
			alreadySent: []int{},
			wantIDs:     []int{1, 2, 3},
		},
		{
			name:        "Some sent",
			alreadySent: []int{1, 3},
			wantIDs:     []int{2},
		},
		{
			name:        "All sent",
			alreadySent: []int{1, 2, 3},
			wantIDs:     []int{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := b.filterNewGames(allGames, tt.alreadySent)
			if len(got) != len(tt.wantIDs) {
				t.Errorf("got %d candidates, want %d", len(got), len(tt.wantIDs))
			}
			for i, id := range tt.wantIDs {
				if got[i].ID != id {
					t.Errorf("at index %d: got ID %d, want %d", i, got[i].ID, id)
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
	defer db.Close()

	b := &Bot{
		Cfg: &config.Config{},
		DB:  db,
	}

	nickname := "Daniel"
	gameID := 123

	// Test markAsSent
	mock.ExpectExec("INSERT INTO sent_notifications").
		WithArgs(nickname, gameID).
		WillReturnResult(sqlmock.NewResult(1, 1))

	if err := b.markAsSent(nickname, gameID); err != nil {
		t.Errorf("markAsSent failed: %v", err)
	}

	// Test getSentGames
	rows := sqlmock.NewRows([]string{"game_id"}).AddRow(gameID)
	mock.ExpectQuery("SELECT game_id FROM sent_notifications").
		WithArgs(nickname).
		WillReturnRows(rows)

	ids, err := b.getSentGames(nickname)
	if err != nil {
		t.Errorf("getSentGames failed: %v", err)
	}

	if len(ids) != 1 || ids[0] != gameID {
		t.Errorf("got ids %v, want [%d]", ids, gameID)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %s", err)
	}
}
