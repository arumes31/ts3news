package bot

import (
	"context"
	"errors"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestAbyssPlazaCatalogContract(t *testing.T) {
	t.Parallel()

	seen := make(map[string]bool, len(abyssPlazaCatalog))
	var previousCost int64
	for _, monument := range abyssPlazaCatalog {
		if monument.Key == "" || monument.Name == "" || monument.Cost <= previousCost {
			t.Fatalf("invalid or non-escalating monument: %+v", monument)
		}
		if seen[monument.Key] {
			t.Fatalf("duplicate monument key %q", monument.Key)
		}
		seen[monument.Key] = true
		previousCost = monument.Cost
	}
	if len(seen) != 4 {
		t.Fatalf("catalog contains %d monuments, want 4", len(seen))
	}
}

func TestBuyAbyssPlazaMonumentIsAtomic(t *testing.T) {
	tests := []struct {
		name      string
		gold      int64
		owned     bool
		wantGold  int64
		wantErr   error
		wantWrite bool
	}{
		{name: "purchase", gold: 500_000, wantGold: 250_000, wantWrite: true},
		{name: "already owned", gold: 500_000, owned: true, wantErr: errAbyssPlazaOwned},
		{name: "insufficient gold", gold: 249_999, wantErr: errAbyssPlazaFunds},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			database, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = database.Close() }()

			mock.ExpectBegin()
			tx, err := database.BeginTx(context.Background(), nil)
			if err != nil {
				t.Fatal(err)
			}
			mock.ExpectQuery("SELECT gold FROM users WHERE client_uid=\\$1 FOR UPDATE").
				WithArgs("delver").WillReturnRows(sqlmock.NewRows([]string{"gold"}).AddRow(test.gold))
			mock.ExpectQuery("SELECT EXISTS").WithArgs("delver", "bronze_bust").
				WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(test.owned))
			if test.wantWrite {
				mock.ExpectExec("UPDATE users SET gold=gold-\\$1 WHERE client_uid=\\$2").
					WithArgs(int64(250_000), "delver").WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectExec("INSERT INTO abyss_plaza_monuments").
					WithArgs("delver", "bronze_bust", int64(250_000)).WillReturnResult(sqlmock.NewResult(0, 1))
			}

			gotGold, gotErr := buyAbyssPlazaMonument(context.Background(), tx, "delver", abyssPlazaCatalog[0])
			if !errors.Is(gotErr, test.wantErr) {
				t.Fatalf("error = %v, want %v", gotErr, test.wantErr)
			}
			if gotGold != test.wantGold {
				t.Fatalf("gold = %d, want %d", gotGold, test.wantGold)
			}
			if test.wantWrite {
				mock.ExpectCommit()
				if err := tx.Commit(); err != nil {
					t.Fatal(err)
				}
			} else {
				mock.ExpectRollback()
				if err := tx.Rollback(); err != nil {
					t.Fatal(err)
				}
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestAbyssPlazaEscapesPublicNicknames(t *testing.T) {
	t.Parallel()

	server, err := NewWebServer(nil)
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	err = server.tmpl.ExecuteTemplate(response, "abyss-plaza", map[string]any{
		"Title": "Hall", "Nav": "plaza", "EnableAbyss": true,
		"U": &webUser{Gold: 1_000_000},
		"Plaza": abyssPlazaView{Exhibits: []abyssPlazaExhibit{{
			Name: "Bust", Tier: "Walk", Nickname: `<img src=x onerror="alert(1)">`, AcquiredAt: "21 Aug 2026",
		}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	body := response.Body.String()
	if strings.Contains(body, `<img src=x`) || !strings.Contains(body, `&lt;img`) {
		t.Fatalf("public nickname was not escaped: %s", body)
	}
}

func TestAbyssPlazaRoutesAndMigrationContract(t *testing.T) {
	t.Parallel()

	routes, err := os.ReadFile("web.go")
	if err != nil {
		t.Fatal(err)
	}
	migration, err := os.ReadFile("../db/migrations/0095_abyss_plaza.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	page, err := webAssets.ReadFile("webassets/abyss_plaza.html")
	if err != nil {
		t.Fatal(err)
	}
	combined := string(routes) + string(migration) + string(page)
	for _, required := range []string{
		`/abyss/plaza`, `/api/abyss/plaza/buy`, `PRIMARY KEY (client_uid, monument_key)`,
		`ON DELETE CASCADE`, `Permanent vanity only`, `no stats`, `no resale`,
	} {
		if !strings.Contains(combined, required) {
			t.Errorf("plaza contract is missing %q", required)
		}
	}
}
