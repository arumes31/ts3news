package bot

import (
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestRecordAbyssBossKillAwardsTrophyAtomically(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO abyss_boss_kills").
		WithArgs("hunter", "Abyssus", 50, int64(1234), "hell").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE users SET abyss_boss_tokens").
		WithArgs("hunter").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	if !(&Bot{DB: database}).recordAbyssBossKillWithToken("hunter", "Abyssus", 50, 1234*time.Millisecond, "hell") {
		t.Fatal("boss kill and trophy transaction failed")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRecordAbyssBossKillRollsBackWhenTrophyFails(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO abyss_boss_kills").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE users SET abyss_boss_tokens").
		WillReturnError(errors.New("award failed"))
	mock.ExpectRollback()
	if (&Bot{DB: database}).recordAbyssBossKillWithToken("hunter", "Abyssus", 50, time.Second, "hell") {
		t.Fatal("boss kill committed without its trophy")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAbyssBossVendorPurchaseDebitsAndGrantsInOneTransaction(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()

	mock.ExpectBegin()
	mock.ExpectQuery("UPDATE users SET abyss_boss_tokens").
		WithArgs(int64(1), "hunter").
		WillReturnRows(sqlmock.NewRows([]string{"abyss_boss_tokens"}).AddRow(int64(2)))
	mock.ExpectExec("UPDATE users SET abyss_tokens").
		WithArgs(int64(10), "hunter").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	mock.ExpectQuery("SELECT abyss_tokens FROM users").
		WithArgs("hunter").
		WillReturnRows(sqlmock.NewRows([]string{"abyss_tokens"}).AddRow(int64(44)))
	mock.ExpectQuery("SELECT mat_id, count FROM user_materials").
		WithArgs("hunter").
		WillReturnRows(sqlmock.NewRows([]string{"mat_id", "count"}))

	server := &WebServer{bot: &Bot{DB: database}}
	request := httptest.NewRequest(http.MethodPost, "/api/abyss/boss_vendor/buy", strings.NewReader(`{"item":"token_cache"}`))
	response := httptest.NewRecorder()
	server.handleAbyssBossVendorBuy(response, request, "hunter")
	if body := response.Body.String(); !strings.Contains(body, `"ok":true`) ||
		!strings.Contains(body, `"boss_tokens":2`) || !strings.Contains(body, `"tokens":44`) {
		t.Fatalf("boss vendor response = %s", body)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAbyssBossVendorRejectsInsufficientTrophiesWithoutGrant(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()

	mock.ExpectBegin()
	mock.ExpectQuery("UPDATE users SET abyss_boss_tokens").
		WithArgs(int64(2), "hunter").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectRollback()
	server := &WebServer{bot: &Bot{DB: database}}
	request := httptest.NewRequest(http.MethodPost, "/api/abyss/boss_vendor/buy", strings.NewReader(`{"item":"prism_cache"}`))
	response := httptest.NewRecorder()
	server.handleAbyssBossVendorBuy(response, request, "hunter")
	if body := response.Body.String(); !strings.Contains(body, "not enough Boss Tokens") {
		t.Fatalf("boss vendor response = %s", body)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAbyssBossVendorPersistenceAndUIContracts(t *testing.T) {
	t.Parallel()
	root := abyssAAARepositoryRoot(t)
	checks := map[string][]string{
		filepath.Join(root, "internal", "db", "migrations", "0087_abyss_boss_tokens.up.sql"): {
			"abyss_boss_tokens", "CHECK (abyss_boss_tokens >= 0)",
		},
		filepath.Join(root, "internal", "bot", "webassets", "abyss_boss_vendor.html"): {
			"Boss Token Vendor", "BossVendor", "/api/abyss/boss_vendor/buy", "bossTokenBalance",
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
