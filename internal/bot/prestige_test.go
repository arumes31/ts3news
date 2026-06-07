package bot

import (
	"testing"
	"ts3news/internal/config"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestDoPrestige(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	b := &Bot{Cfg: &config.Config{}, DB: db}
	uid := "user1"

	mock.ExpectQuery(`UPDATE users SET prestige = prestige \+ 1, xp = 0, level = 1 WHERE client_uid = \$1 RETURNING prestige`).
		WithArgs(uid).
		WillReturnRows(sqlmock.NewRows([]string{"prestige"}).AddRow(1))

	newP := b.doPrestige(uid)
	if newP != 1 {
		t.Errorf("newPrestige = %d, want 1", newP)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %s", err)
	}
}
