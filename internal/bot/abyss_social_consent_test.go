package bot

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func newConsentTestServer(t *testing.T) (*WebServer, sqlmock.Sqlmock) {
	t.Helper()
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	return &WebServer{bot: &Bot{DB: database}}, mock
}

func TestAbyssMentorOfferIsPendingUntilMenteeAccepts(t *testing.T) {
	t.Parallel()
	server, mock := newConsentTestServer(t)
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT abyss_best_depth FROM users").WithArgs("mentor").WillReturnRows(sqlmock.NewRows([]string{"d"}).AddRow(30))
	mock.ExpectQuery("SELECT abyss_best_depth FROM users").WithArgs("mentee").WillReturnRows(sqlmock.NewRows([]string{"d"}).AddRow(3))
	mock.ExpectQuery("mentee_uid=\\$1 AND accepted_at IS NOT NULL").WithArgs("mentee").WillReturnRows(sqlmock.NewRows([]string{"e"}).AddRow(false))
	// The offer row carries no accepted_at: it is pending.
	mock.ExpectExec("INSERT INTO abyss_mentor_pairs \\(mentor_uid,mentee_uid\\)\\s+VALUES \\(\\$1,\\$2\\) ON CONFLICT \\(mentor_uid,mentee_uid\\) DO NOTHING").
		WithArgs("mentor", "mentee").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO abyss_social_notifications").WithArgs("mentee").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	response := httptest.NewRecorder()
	server.handleAbyssMentor(response, httptest.NewRequest("POST", "/api/abyss/social/mentor", strings.NewReader(`{"mentee_uid":"mentee"}`)), "mentor")
	if !strings.Contains(response.Body.String(), `"ok":true`) || !strings.Contains(response.Body.String(), "once the delver accepts") {
		t.Fatalf("response = %s", response.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAbyssMentorAcceptIsScopedToTheMenteeSession(t *testing.T) {
	t.Parallel()
	server, mock := newConsentTestServer(t)
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT abyss_best_depth FROM users").WithArgs("mentee").WillReturnRows(sqlmock.NewRows([]string{"d"}).AddRow(3))
	mock.ExpectQuery("mentee_uid=\\$1 AND accepted_at IS NOT NULL").WithArgs("mentee").WillReturnRows(sqlmock.NewRows([]string{"e"}).AddRow(false))
	// The session uid is bound as mentee_uid; a caller cannot accept on another's behalf.
	mock.ExpectExec("UPDATE abyss_mentor_pairs SET accepted_at=NOW\\(\\)\\s+WHERE mentor_uid=\\$1 AND mentee_uid=\\$2 AND accepted_at IS NULL").
		WithArgs("mentor", "mentee").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("DELETE FROM abyss_mentor_pairs\\s+WHERE mentee_uid=\\$1 AND accepted_at IS NULL").WithArgs("mentee").WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectCommit()

	response := httptest.NewRecorder()
	server.handleAbyssMentor(response, httptest.NewRequest("POST", "/api/abyss/social/mentor", strings.NewReader(`{"action":"accept","mentor_uid":"mentor"}`)), "mentee")
	if !strings.Contains(response.Body.String(), `"ok":true`) {
		t.Fatalf("response = %s", response.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAbyssMentorMenteeCanDissolveBond(t *testing.T) {
	t.Parallel()
	server, mock := newConsentTestServer(t)
	mock.ExpectExec("DELETE FROM abyss_mentor_pairs\\s+WHERE mentor_uid=\\$1 AND mentee_uid=\\$2").
		WithArgs("mentor", "mentee").WillReturnResult(sqlmock.NewResult(0, 1))

	response := httptest.NewRecorder()
	server.handleAbyssMentor(response, httptest.NewRequest("POST", "/api/abyss/social/mentor", strings.NewReader(`{"action":"leave","mentor_uid":"mentor"}`)), "mentee")
	if !strings.Contains(response.Body.String(), `"ok":true`) {
		t.Fatalf("response = %s", response.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAbyssCoopInviteCreatesPendingInviteOnly(t *testing.T) {
	t.Parallel()
	server, mock := newConsentTestServer(t)
	expectEquipBestActiveRun(mock, "host")
	mock.MatchExpectationsInOrder(false)
	mock.ExpectQuery("SELECT EXISTS\\(SELECT 1 FROM users WHERE client_uid=\\$1 AND abyss_best_depth > 0\\)").
		WithArgs("helper").WillReturnRows(sqlmock.NewRows([]string{"e"}).AddRow(true))
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM abyss_party_members WHERE owner_uid=\\$1").
		WithArgs("host").WillReturnRows(sqlmock.NewRows([]string{"n"}).AddRow(0))
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO abyss_party_invites").WithArgs("host", "helper").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO abyss_social_notifications").WithArgs("helper").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	response := httptest.NewRecorder()
	server.handleAbyssCoopInvite(response, httptest.NewRequest("POST", "/api/abyss/coop/invite", strings.NewReader(`{"coop_uid":"helper"}`)), "host")
	body := response.Body.String()
	if !strings.Contains(body, `"ok":true`) || !strings.Contains(body, `"pending":true`) {
		t.Fatalf("response = %s", body)
	}
	// No membership row or coop_uid write may happen until the helper accepts;
	// sqlmock fails ExpectationsWereMet only for missing calls, so unexpected
	// party_members writes would have errored the handler instead.
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAbyssCoopRespondDeclineRemovesOnlyOwnInvite(t *testing.T) {
	t.Parallel()
	server, mock := newConsentTestServer(t)
	mock.ExpectExec("DELETE FROM abyss_party_invites WHERE owner_uid=\\$1 AND member_uid=\\$2").
		WithArgs("host", "helper").WillReturnResult(sqlmock.NewResult(0, 1))

	response := httptest.NewRecorder()
	server.handleAbyssCoopRespond(response, httptest.NewRequest("POST", "/api/abyss/coop/respond", strings.NewReader(`{"owner_uid":"host","accept":false}`)), "helper")
	if !strings.Contains(response.Body.String(), `"ok":true`) {
		t.Fatalf("response = %s", response.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAbyssCoopRespondAcceptEnrollsTheSessionUser(t *testing.T) {
	t.Parallel()
	server, mock := newConsentTestServer(t)
	started := time.Now().Add(-time.Minute)
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT client_uid FROM users WHERE client_uid=\\$1 FOR UPDATE").WithArgs("helper").
		WillReturnRows(sqlmock.NewRows([]string{"client_uid"}).AddRow("helper"))
	mock.ExpectQuery("SELECT started_at FROM abyss_active WHERE client_uid=\\$1 FOR UPDATE").WithArgs("host").
		WillReturnRows(sqlmock.NewRows([]string{"started_at"}).AddRow(started))
	mock.ExpectExec("DELETE FROM abyss_party_invites WHERE owner_uid=\\$1 AND member_uid=\\$2\\s+AND created_at>NOW\\(\\)-\\$3::interval AND created_at>=\\$4").
		WithArgs("host", "helper", abyssPartyInviteTTLInterval, started).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery("SELECT EXISTS\\(SELECT 1 FROM abyss_active WHERE client_uid=\\$1\\)").WithArgs("helper").
		WillReturnRows(sqlmock.NewRows([]string{"busy"}).AddRow(false))
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM abyss_party_members WHERE owner_uid=\\$1").WithArgs("host").
		WillReturnRows(sqlmock.NewRows([]string{"n"}).AddRow(0))
	mock.ExpectExec("INSERT INTO abyss_party_members").WithArgs("host", "helper").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE abyss_active SET coop_uid=\\$1").WithArgs("helper", "host").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("DELETE FROM abyss_party_invites WHERE member_uid=\\$1").WithArgs("helper").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()

	response := httptest.NewRecorder()
	server.handleAbyssCoopRespond(response, httptest.NewRequest("POST", "/api/abyss/coop/respond", strings.NewReader(`{"owner_uid":"host","accept":true}`)), "helper")
	if !strings.Contains(response.Body.String(), `"ok":true`) {
		t.Fatalf("response = %s", response.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAbyssCoopRespondRejectsExpiredInvite(t *testing.T) {
	t.Parallel()
	server, mock := newConsentTestServer(t)
	started := time.Now().Add(-time.Minute)
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT client_uid FROM users WHERE client_uid=\\$1 FOR UPDATE").WithArgs("helper").
		WillReturnRows(sqlmock.NewRows([]string{"client_uid"}).AddRow("helper"))
	mock.ExpectQuery("SELECT started_at FROM abyss_active").WithArgs("host").
		WillReturnRows(sqlmock.NewRows([]string{"started_at"}).AddRow(started))
	mock.ExpectExec("DELETE FROM abyss_party_invites WHERE owner_uid=\\$1 AND member_uid=\\$2").
		WithArgs("host", "helper", abyssPartyInviteTTLInterval, started).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectRollback()

	response := httptest.NewRecorder()
	server.handleAbyssCoopRespond(response, httptest.NewRequest("POST", "/api/abyss/coop/respond", strings.NewReader(`{"owner_uid":"host","accept":true}`)), "helper")
	if !strings.Contains(response.Body.String(), "expired") {
		t.Fatalf("response = %s", response.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAbyssShoutboxRelayNeutralizesBBCode(t *testing.T) {
	t.Parallel()
	got := abyssShoutboxRelayLine("[color=red]Admin[/color]", "[url=https://evil.example]free gold[/url] [b]now[/b]")
	if strings.ContainsAny(strings.TrimPrefix(got, "[Abyss] "), "[]") {
		t.Fatalf("relay line still carries BBCode: %q", got)
	}
	if !strings.HasPrefix(got, "[Abyss] ") || !strings.Contains(got, "free gold") {
		t.Fatalf("relay line lost content: %q", got)
	}
}

func TestAbyssManualBuffUseNeverRaisesRemainingFights(t *testing.T) {
	t.Parallel()
	if !strings.Contains(abyssManualBuffQuery, "LEAST(remaining_fights, $3)") {
		t.Fatalf("manual buff activation must cap, not overwrite, remaining fights: %s", abyssManualBuffQuery)
	}
	if !strings.Contains(abyssManualBuffQuery, "remaining_fights > 0") {
		t.Fatalf("manual buff activation must only touch owned charges: %s", abyssManualBuffQuery)
	}
}

func TestAbyssManualUseRejectsAffixSuppressor(t *testing.T) {
	t.Parallel()
	server, mock := newConsentTestServer(t)
	mock.MatchExpectationsInOrder(false)
	mock.ExpectQuery("SELECT remaining_fights FROM user_consumables").WithArgs("delver", abyssAffixSuppressorID).
		WillReturnRows(sqlmock.NewRows([]string{"remaining_fights"}).AddRow(1))

	response := httptest.NewRecorder()
	server.handleAbyssUseConsumable(response, httptest.NewRequest("POST", "/api/abyss/use_consumable", strings.NewReader(`{"cons_id":"abyss_affix_suppressor"}`)), "delver")
	if !strings.Contains(response.Body.String(), "consumed automatically") {
		t.Fatalf("response = %s", response.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
