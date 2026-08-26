package bot

import (
	"errors"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestAbyssInsuranceCharmEligibility(t *testing.T) {
	t.Parallel()

	base := abyssRun{Active: true, Depth: 12, Escrow: 10_000}
	tests := []struct {
		name     string
		run      abyssRun
		pacts    []string
		hardcore bool
		anchor   bool
		want     bool
	}{
		{name: "uninsured defeat", run: base, want: true},
		{name: "inactive", run: abyssRun{Depth: 12, Escrow: 10_000}},
		{name: "empty cache", run: abyssRun{Active: true, Depth: 12}},
		{name: "grace", run: abyssRun{Active: true, Depth: 3, Escrow: 10_000}},
		{name: "paid cover", run: abyssRun{Active: true, Depth: 12, Escrow: 10_000, Insured: 10}},
		{name: "hardcore", run: base, hardcore: true},
		{name: "anchor", run: base, anchor: true},
		{name: "uninsured pact", run: base, pacts: []string{"uninsured"}},
		{name: "abstinence pact", run: base, pacts: []string{"abstinence"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := abyssInsuranceCharmEligible(test.run, test.pacts, test.hardcore, test.anchor); got != test.want {
				t.Fatalf("eligible = %t, want %t", got, test.want)
			}
		})
	}
	if abyssConsumableCountsTowardCarryCap(abyssInsuranceCharmID) {
		t.Fatal("passive insurance charm consumed a combat carry slot")
	}
}

func TestConsumeAbyssInsuranceCharmUsesCallerTransaction(t *testing.T) {
	t.Parallel()

	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	mock.ExpectBegin()
	tx, err := database.Begin()
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectExec("UPDATE user_consumables").
		WithArgs("delver", abyssInsuranceCharmID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("DELETE FROM user_consumables").
		WithArgs("delver", abyssInsuranceCharmID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	used, err := consumeAbyssInsuranceCharm(tx, "delver")
	if err != nil || !used {
		t.Fatalf("consume charm = (%t, %v), want (true, nil)", used, err)
	}
	mock.ExpectRollback()
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestConsumeAbyssInsuranceCharmPreservesMissingAndFailedCharges(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name string
		rows int64
		err  error
	}{
		{name: "missing"},
		{name: "database failure", err: errors.New("write failed")},
	} {
		t.Run(test.name, func(t *testing.T) {
			database, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = database.Close() }()
			mock.ExpectBegin()
			tx, err := database.Begin()
			if err != nil {
				t.Fatal(err)
			}
			expect := mock.ExpectExec("UPDATE user_consumables").WithArgs("delver", abyssInsuranceCharmID)
			if test.err != nil {
				expect.WillReturnError(test.err)
			} else {
				expect.WillReturnResult(sqlmock.NewResult(0, test.rows))
			}
			used, gotErr := consumeAbyssInsuranceCharm(tx, "delver")
			if used || (test.err != nil) != (gotErr != nil) {
				t.Fatalf("consume charm = (%t, %v)", used, gotErr)
			}
			mock.ExpectRollback()
			_ = tx.Rollback()
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestAbyssInsuranceCharmRecipeAndPresentation(t *testing.T) {
	t.Parallel()

	recipe, ok := craftRecipeByID("forge_insurance_charm")
	if !ok || recipe.Secret || recipe.ConsID != abyssInsuranceCharmID || recipe.Cost["core"] != 2 || recipe.Cost["shard"] != 5 {
		t.Fatalf("insurance charm recipe = %#v, found=%t", recipe, ok)
	}
	var source strings.Builder
	for _, name := range []string{"webassets/abyss.html", "webassets/abyss_core_risk.html", "webassets/abyss_core_risk.css"} {
		body, err := webAssets.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		source.Write(body)
	}
	for _, token := range []string{"insurance_charm_used", "insuranceCharmStatus", "ab-insurance-charm", "50% fallback"} {
		if !strings.Contains(source.String(), token) {
			t.Errorf("insurance-charm presentation missing %q", token)
		}
	}
}
