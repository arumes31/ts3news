package bot

import (
	"errors"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestAbyssTalentEffectiveIntSoftCap(t *testing.T) {
	t.Parallel()

	tests := []struct {
		level int
		want  int
	}{
		{level: 5, want: 5},
		{level: 6, want: 6},
		{level: 7, want: 6},
		{level: 8, want: 7},
		{level: 9, want: 7},
		{level: 10, want: 8},
	}
	for _, test := range tests {
		if got := abyssTalentEffectiveInt(test.level); got != test.want {
			t.Errorf("abyssTalentEffectiveInt(%d) = %d, want %d", test.level, got, test.want)
		}
	}
}

func TestAbyssTalentRefundIncludesSoftCapRanks(t *testing.T) {
	t.Parallel()

	if got := abyssTalentRefund(map[string]int{"dd_0_0": 10}); got != 550 {
		t.Fatalf("refund = %d, want 550", got)
	}
}

func TestAbyssTalentUpgradeIsAtomic(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE users SET abyss_tokens").
		WithArgs(int64(60), "talent-player").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO app_meta").
		WithArgs(abyssTalentKey("talent-player"), `{"dd_0_0":6}`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	tx, err := database.Begin()
	if err != nil {
		t.Fatal(err)
	}
	spent, err := persistAbyssTalentUpgrade(tx, "talent-player", 60, map[string]int{"dd_0_0": 6})
	if err != nil || !spent {
		t.Fatalf("persistAbyssTalentUpgrade() = %t, %v", spent, err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAbyssTalentUpgradeRollsBackFailedLevelSave(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE users SET abyss_tokens").
		WithArgs(int64(60), "talent-player").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO app_meta").
		WithArgs(abyssTalentKey("talent-player"), `{"dd_0_0":6}`).
		WillReturnError(errors.New("persistence failed"))
	mock.ExpectRollback()

	tx, err := database.Begin()
	if err != nil {
		t.Fatal(err)
	}
	if spent, err := persistAbyssTalentUpgrade(tx, "talent-player", 60, map[string]int{"dd_0_0": 6}); err == nil || spent {
		t.Fatalf("persistAbyssTalentUpgrade() = %t, %v; want failure", spent, err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAbyssTalentTreeShowsSharedCap(t *testing.T) {
	t.Parallel()

	page, err := webAssets.ReadFile("webassets/abysstree.html")
	if err != nil {
		t.Fatal(err)
	}
	text := string(page)
	for _, required := range []string{
		"TALENT_MAX_LEVEL = {{.TalentMaxLevel}}",
		"Soft cap active",
		"ranks 6–10 grant 50% effect",
	} {
		if !strings.Contains(text, required) {
			t.Errorf("talent tree is missing %q", required)
		}
	}
}
