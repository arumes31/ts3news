package bot

import (
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestAbyssTreasureGoblinSignaturesAreDistinct(t *testing.T) {
	tests := []struct {
		name   string
		kind   string
		runKey bool
		mat    string
		tokens int64
	}{
		{name: "Gem Goblin", kind: "mat", mat: "prism"},
		{name: "Token Goblin", kind: "tokens", tokens: 5},
		{name: "Key Goblin", runKey: true},
	}
	labels := make(map[string]bool, len(tests))
	for _, test := range tests {
		reward, ok := abyssTreasureGoblinSignature(test.name)
		if !ok || reward.Label == "" || labels[reward.Label] {
			t.Fatalf("%s reward = %+v, ok=%v", test.name, reward, ok)
		}
		labels[reward.Label] = true
		if reward.Grant.Type != test.kind || reward.Grant.MatID != test.mat || reward.Grant.Tokens != test.tokens || reward.RunKey != test.runKey {
			t.Errorf("%s reward = %+v", test.name, reward)
		}
	}
	if _, ok := abyssTreasureGoblinSignature("Treasure Goblin"); ok {
		t.Fatal("legacy goblin received a variant signature")
	}
}

func TestGrantAbyssRunVaultKeyCommitsWithRunFlags(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT value FROM app_meta").WithArgs("abyss_run_flags_delver").
		WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(`{"vault_keys":2}`))
	mock.ExpectExec("INSERT INTO app_meta").WithArgs("abyss_run_flags_delver", `{"vault_keys":3}`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	if err := (&Bot{DB: database}).grantAbyssRunVaultKey("delver"); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestGrantAbyssRunVaultKeyRollsBackOnFlagWriteFailure(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT value FROM app_meta").WithArgs("abyss_run_flags_delver").
		WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(`{"vault_keys":2}`))
	mock.ExpectExec("INSERT INTO app_meta").WithArgs("abyss_run_flags_delver", `{"vault_keys":3}`).
		WillReturnError(errors.New("flag write failed"))
	mock.ExpectRollback()
	if err := (&Bot{DB: database}).grantAbyssRunVaultKey("delver"); err == nil {
		t.Fatal("failed flag write returned nil")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
