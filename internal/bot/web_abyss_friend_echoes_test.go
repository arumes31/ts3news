package bot

import (
	"database/sql"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestSelectAbyssEchoIdentityPrefersConsentingBond(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	mock.ExpectQuery("FROM abyss_social_profiles owner").WithArgs("delver").
		WillReturnRows(sqlmock.NewRows([]string{"client_uid", "nickname", "depth"}).AddRow("friend", "Nyra", 42))

	echo, err := (&Bot{DB: db}).selectAbyssEchoIdentity("delver")
	if err != nil {
		t.Fatal(err)
	}
	if echo.UID != "friend" || echo.Nick != "Nyra" || echo.Depth != 42 {
		t.Fatalf("selected echo = %+v", echo)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestSelectAbyssEchoIdentityFallsBackToLegacyPool(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	mock.ExpectQuery("FROM abyss_social_profiles owner").WithArgs("delver").WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery("FROM abyss_runs r JOIN users").WithArgs("delver").
		WillReturnRows(sqlmock.NewRows([]string{"client_uid", "nickname", "depth"}).AddRow("random", "Sable", 30))

	echo, err := (&Bot{DB: db}).selectAbyssEchoIdentity("delver")
	if err != nil {
		t.Fatal(err)
	}
	if echo.UID != "random" || echo.Nick != "Sable" || echo.Depth != 30 {
		t.Fatalf("fallback echo = %+v", echo)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAbyssFriendEchoSettingsRejectsUnconsentingTarget(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	mock.ExpectQuery("SELECT EXISTS").WithArgs("delver", "friend").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	server := &WebServer{bot: &Bot{DB: db}}
	req := httptest.NewRequest("POST", "/api/abyss/social/friend_echo", strings.NewReader(`{"share_enabled":true,"echo_uid":"friend"}`))
	res := httptest.NewRecorder()

	server.handleAbyssFriendEchoSettings(res, req, "delver")
	if !strings.Contains(res.Body.String(), "not available") {
		t.Fatalf("response = %s", res.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAbyssFriendEchoSettingsPersistsSharingOptIn(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	mock.ExpectExec("INSERT INTO abyss_social_profiles").WithArgs("delver", true, "").
		WillReturnResult(sqlmock.NewResult(1, 1))
	server := &WebServer{bot: &Bot{DB: db}}
	req := httptest.NewRequest("POST", "/api/abyss/social/friend_echo", strings.NewReader(`{"share_enabled":true,"echo_uid":""}`))
	res := httptest.NewRecorder()

	server.handleAbyssFriendEchoSettings(res, req, "delver")
	if !strings.Contains(res.Body.String(), `"ok":true`) {
		t.Fatalf("response = %s", res.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAbyssFriendEchoEvidence(t *testing.T) {
	page, err := webAssets.ReadFile("webassets/abyss_social.html")
	if err != nil {
		t.Fatal(err)
	}
	for _, token := range []string{"Bonded echoes", "let bonded friends encounter my echo", "friendEchoUID", "/api/abyss/social/friend_echo"} {
		if !strings.Contains(string(page), token) {
			t.Errorf("friend echo UI is missing %q", token)
		}
	}
	migration, err := os.ReadFile(filepath.Join(abyssAAARepositoryRoot(t), "internal", "db", "migrations", "0088_abyss_friend_echoes.up.sql"))
	if err != nil {
		t.Fatal(err)
	}
	for _, token := range []string{"ghost_echo_opt_in", "preferred_echo_uid", "ON DELETE SET NULL"} {
		if !strings.Contains(string(migration), token) {
			t.Errorf("friend echo migration is missing %q", token)
		}
	}
}
