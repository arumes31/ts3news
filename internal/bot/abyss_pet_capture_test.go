package bot

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"ts3news/internal/content"
)

func TestAbyssPetCaptureLimitOffersOneFullStableDecision(t *testing.T) {
	if got := abyssPetCaptureLimit(9); got != abyssPetCaptureCap {
		t.Fatalf("capture limit = %d, want %d", got, abyssPetCaptureCap)
	}
	if !abyssCanAttemptPetCapture(2, 3, false) || !abyssCanAttemptPetCapture(3, 3, false) {
		t.Fatal("eligible normal or full-stable capture was blocked")
	}
	if abyssCanAttemptPetCapture(3, 3, true) || abyssCanAttemptPetCapture(1, 1, false) {
		t.Fatal("duplicate pending capture or underpowered full stable was allowed")
	}
}

func TestPersistAbyssPetCaptureCreatesOrPreservesDecision(t *testing.T) {
	for _, test := range []struct {
		name     string
		inserted int64
		want     abyssPetCaptureResult
	}{{"creates pending decision", 1, abyssPetCapturePending}, {"preserves existing decision", 0, abyssPetCapturePreserved}} {
		t.Run(test.name, func(t *testing.T) {
			database, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = database.Close() }()
			pet := &content.Mob{Name: "Mossling", Type: content.MobElite, Level: 8, Stats: content.Stats{HP: 1, STR: 12, DEF: 9, SPD: 11}, MaxHP: 80, Loyalty: 25}
			mock.ExpectBegin()
			mock.ExpectQuery("SELECT client_uid FROM users").WithArgs("keeper").
				WillReturnRows(sqlmock.NewRows([]string{"client_uid"}).AddRow("keeper"))
			mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM user_pets WHERE client_uid=$1")).WithArgs("keeper").
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))
			mock.ExpectExec("INSERT INTO abyss_pending_pet_captures").
				WithArgs("keeper", "Mossling", string(content.MobElite), 8, 1, 80, 12, 9, 11, 25).
				WillReturnResult(sqlmock.NewResult(0, test.inserted))
			mock.ExpectCommit()
			result, err := (&Bot{DB: database}).persistAbyssPetCapture("keeper", pet, abyssPetCaptureCap)
			if err != nil || result != test.want {
				t.Fatalf("capture result = %q, %v; want %q", result, err, test.want)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestPersistAbyssPetCaptureRecruitsBelowAuthoritativeLimit(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	pet := &content.Mob{Name: "Mossling", Type: content.MobElite, Level: 8, Stats: content.Stats{HP: 1, STR: 12, DEF: 9, SPD: 11}, MaxHP: 80, Loyalty: 25}
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT client_uid FROM users").WithArgs("keeper").
		WillReturnRows(sqlmock.NewRows([]string{"client_uid"}).AddRow("keeper"))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM user_pets WHERE client_uid=$1")).WithArgs("keeper").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
	mock.ExpectExec("INSERT INTO user_pets").
		WithArgs("keeper", "Mossling", string(content.MobElite), 8, 1, 80, 12, 9, 11, 25).
		WillReturnResult(sqlmock.NewResult(8, 1))
	mock.ExpectCommit()
	result, err := (&Bot{DB: database}).persistAbyssPetCapture("keeper", pet, abyssPetCaptureCap)
	if err != nil || result != abyssPetCaptureRecruited {
		t.Fatalf("capture result = %q, %v; want recruited", result, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestResolveAbyssPetCaptureReplacesChosenOwnedPetAtomically(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	const uid = "keeper"
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT client_uid FROM users").WithArgs(uid).
		WillReturnRows(sqlmock.NewRows([]string{"client_uid"}).AddRow(uid))
	mock.ExpectQuery("SELECT name,mob_type,level,hp,max_hp,str,def,spd,loyalty").WithArgs(uid).
		WillReturnRows(sqlmock.NewRows([]string{"name", "mob_type", "level", "hp", "max_hp", "str", "def", "spd", "loyalty"}).
			AddRow("Mossling", "Elite", 8, 1, 80, 12, 9, 11, 25))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM user_pets WHERE client_uid=$1")).WithArgs(uid).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))
	mock.ExpectQuery("SELECT name,active_slot FROM user_pets").WithArgs(int64(7), uid).
		WillReturnRows(sqlmock.NewRows([]string{"name", "active_slot"}).AddRow("Old Fang", 1))
	mock.ExpectExec("DELETE FROM user_pets").WithArgs(int64(7), uid).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO user_pets").
		WithArgs(uid, "Mossling", "Elite", 8, 1, 80, 12, 9, 11, 25, 1).
		WillReturnResult(sqlmock.NewResult(8, 1))
	mock.ExpectExec("DELETE FROM abyss_pending_pet_captures").WithArgs(uid).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	server := &WebServer{bot: &Bot{DB: database}}
	request := httptest.NewRequest(http.MethodPost, "/api/abyss/social/pet/capture/resolve", strings.NewReader(`{"release_pet_id":7,"decline":false}`))
	response := httptest.NewRecorder()
	server.handleAbyssPetCaptureResolve(response, request, uid)
	if body := response.Body.String(); !strings.Contains(body, `"ok":true`) || !strings.Contains(body, "Mossling joined") {
		t.Fatalf("capture response = %s", body)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestResolveAbyssPetCaptureKeepsDecisionWhileOwnerExplicitlyReducesLegacyOvercap(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	const uid = "legacy-keeper"
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT client_uid FROM users").WithArgs(uid).
		WillReturnRows(sqlmock.NewRows([]string{"client_uid"}).AddRow(uid))
	mock.ExpectQuery("SELECT name,mob_type,level,hp,max_hp,str,def,spd,loyalty").WithArgs(uid).
		WillReturnRows(sqlmock.NewRows([]string{"name", "mob_type", "level", "hp", "max_hp", "str", "def", "spd", "loyalty"}).
			AddRow("Mossling", "Elite", 8, 1, 80, 12, 9, 11, 25))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM user_pets WHERE client_uid=$1")).WithArgs(uid).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(4))
	mock.ExpectQuery("SELECT name,active_slot FROM user_pets").WithArgs(int64(9), uid).
		WillReturnRows(sqlmock.NewRows([]string{"name", "active_slot"}).AddRow("Legacy Fang", 0))
	mock.ExpectExec("DELETE FROM user_pets").WithArgs(int64(9), uid).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	server := &WebServer{bot: &Bot{DB: database}}
	request := httptest.NewRequest(http.MethodPost, "/api/abyss/social/pet/capture/resolve", strings.NewReader(`{"release_pet_id":9}`))
	response := httptest.NewRecorder()
	server.handleAbyssPetCaptureResolve(response, request, uid)
	if body := response.Body.String(); !strings.Contains(body, `"pending":true`) || !strings.Contains(body, "Choose another") {
		t.Fatalf("legacy overcap response = %s", body)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAbyssPendingCaptureUIAndMigrationContract(t *testing.T) {
	page, err := webAssets.ReadFile("webassets/abyss_social.html")
	if err != nil {
		t.Fatal(err)
	}
	styles, err := webAssets.ReadFile("webassets/abyss_social.css")
	if err != nil {
		t.Fatal(err)
	}
	for _, token := range []string{"STABLE DECISION", "no companion is replaced automatically", "capture/resolve", "Release new capture"} {
		if !strings.Contains(string(page), token) {
			t.Errorf("pending capture UI is missing %q", token)
		}
	}
	if !strings.Contains(string(styles), ".ab-pending-capture") || !strings.Contains(string(styles), ".ab-capture-actions") {
		t.Fatal("pending capture styles are missing")
	}
	root := abyssAAARepositoryRoot(t)
	up, err := os.ReadFile(filepath.Join(root, "internal", "db", "migrations", "0093_abyss_pending_pet_captures.up.sql"))
	if err != nil {
		t.Fatal(err)
	}
	down, err := os.ReadFile(filepath.Join(root, "internal", "db", "migrations", "0093_abyss_pending_pet_captures.down.sql"))
	if err != nil {
		t.Fatal(err)
	}
	for _, token := range []string{"PRIMARY KEY", "REFERENCES users", "loyalty", "captured_at"} {
		if !strings.Contains(string(up), token) {
			t.Errorf("pending capture migration is missing %q", token)
		}
	}
	if !strings.Contains(string(down), "DROP TABLE IF EXISTS abyss_pending_pet_captures") {
		t.Fatal("pending capture migration is not reversible")
	}
}
