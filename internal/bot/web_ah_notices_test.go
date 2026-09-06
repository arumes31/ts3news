package bot

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

const noticeHistorySQL = `SELECT id,kind,message,amount,created_at,seen FROM abyss_economy_events
	WHERE client_uid=$1 ORDER BY id DESC LIMIT 21`
const noticeOlderSQL = `SELECT id,kind,message,amount,created_at,seen FROM abyss_economy_events
	WHERE client_uid=$1 AND id<$2 ORDER BY id DESC LIMIT 21`
const noticeCountSQL = `SELECT COUNT(*) FROM abyss_economy_events WHERE client_uid=$1 AND seen=FALSE`
const noticeAckSQL = `UPDATE abyss_economy_events SET seen=TRUE WHERE client_uid=$1 AND id=ANY($2) AND seen=FALSE`
const noticeLegacySQL = `SELECT id,kind,message,amount,created_at,seen FROM abyss_economy_events
	WHERE client_uid=$1 AND seen=FALSE ORDER BY created_at,id LIMIT 20 FOR UPDATE`

func newAHNoticeTestServer(t *testing.T) (*WebServer, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Error(err)
		}
		_ = db.Close()
	})
	return &WebServer{bot: &Bot{DB: db}}, mock
}

func noticeTestRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{"id", "kind", "message", "amount", "created_at", "seen"})
}

func callAHNotices(t *testing.T, server *WebServer, method, target, body string) (*httptest.ResponseRecorder, map[string]json.RawMessage) {
	t.Helper()
	response := httptest.NewRecorder()
	server.handleAHNotices(response, httptest.NewRequest(method, target, strings.NewReader(body)), "notice-owner")
	var result map[string]json.RawMessage
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatalf("response is not JSON: %s", response.Body.String())
	}
	return response, result
}

func TestAHNoticesGetRetainsSeenHistoryWithoutMutation(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	created := time.Date(2026, 9, 5, 9, 30, 0, 0, time.FixedZone("CEST", 7200))
	mock.ExpectQuery(`SELECT id,kind,message,amount,created_at,seen FROM abyss_economy_events
		WHERE client_uid=$1 ORDER BY id DESC LIMIT 21`).WithArgs("notice-owner").WillReturnRows(
		sqlmock.NewRows([]string{"id", "kind", "message", "amount", "created_at", "seen"}).
			AddRow(32, "sale", "Your sale completed.", 90, created, false).
			AddRow(31, "refund", "Your refund arrived.", 20, created.Add(-time.Hour), true)).RowsWillBeClosed()
	mock.ExpectQuery(`SELECT COUNT(*) FROM abyss_economy_events WHERE client_uid=$1 AND seen=FALSE`).
		WithArgs("notice-owner").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	response := httptest.NewRecorder()
	(&WebServer{bot: &Bot{DB: db}}).handleAHNotices(response, httptest.NewRequest(http.MethodGet, "/api/ah/notices", nil), "notice-owner")
	var result struct {
		OK          bool `json:"ok"`
		UnseenCount int  `json:"unseen_count"`
		Notices     []struct {
			ID      int64  `json:"id"`
			Seen    bool   `json:"seen"`
			When    string `json:"when"`
			WhenISO string `json:"when_iso"`
		} `json:"notices"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatalf("response is not JSON: %s", response.Body.String())
	}
	if response.Code != http.StatusOK || !result.OK || result.UnseenCount != 1 || len(result.Notices) != 2 {
		t.Fatalf("history response = %d %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Header().Get("Cache-Control"), "no-store") {
		t.Fatalf("history cache policy = %q", response.Header().Get("Cache-Control"))
	}
	if result.Notices[0].ID != 32 || result.Notices[0].Seen || result.Notices[1].ID != 31 || !result.Notices[1].Seen || result.Notices[0].WhenISO != "2026-09-05T07:30:00Z" || result.Notices[0].When != "05 Sep 2026 · 07:30 UTC" {
		t.Fatalf("seen and unseen history = %+v", result.Notices)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAHNoticesGetPaginatesUsingBoundedLookahead(t *testing.T) {
	t.Parallel()
	server, mock := newAHNoticeTestServer(t)
	rows := noticeTestRows()
	for id := 100; id >= 80; id-- {
		rows.AddRow(id, "sale", "Sale completed.", id, time.Unix(int64(id), 0), true)
	}
	mock.ExpectQuery(noticeHistorySQL).WithArgs("notice-owner").WillReturnRows(rows).RowsWillBeClosed()
	mock.ExpectQuery(noticeCountSQL).WithArgs("notice-owner").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	response, result := callAHNotices(t, server, http.MethodGet, "/api/ah/notices", "")
	var notices []struct{ ID int64 }
	if err := json.Unmarshal(result["notices"], &notices); err != nil {
		t.Fatal(err)
	}
	if len(notices) != 20 || notices[0].ID != 100 || notices[19].ID != 81 || string(result["next_before"]) != "81" || string(result["has_more"]) != "true" {
		t.Fatalf("first page = %s", response.Body.String())
	}
	mock.ExpectQuery(noticeOlderSQL).WithArgs("notice-owner", int64(81)).WillReturnRows(
		noticeTestRows().AddRow(80, "sale", "Older sale.", 80, time.Unix(80, 0), true)).RowsWillBeClosed()
	mock.ExpectQuery(noticeCountSQL).WithArgs("notice-owner").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	response, result = callAHNotices(t, server, http.MethodGet, "/api/ah/notices?before=81", "")
	if err := json.Unmarshal(result["notices"], &notices); err != nil {
		t.Fatal(err)
	}
	if len(notices) != 1 || notices[0].ID != 80 || string(result["has_more"]) != "false" || string(result["next_before"]) != "0" {
		t.Fatalf("older page = %s", response.Body.String())
	}
}

func TestAHNoticesGetEmptyHistoryReturnsArray(t *testing.T) {
	t.Parallel()
	server, mock := newAHNoticeTestServer(t)
	mock.ExpectQuery(noticeHistorySQL).WithArgs("notice-owner").WillReturnRows(noticeTestRows())
	mock.ExpectQuery(noticeCountSQL).WithArgs("notice-owner").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	response, result := callAHNotices(t, server, http.MethodGet, "/api/ah/notices", "")
	if response.Code != http.StatusOK || string(result["notices"]) != "[]" {
		t.Fatalf("empty history = %s", response.Body.String())
	}
}

func TestAHNoticesPostAcknowledgesOnlyRequestedCurrentUserIDs(t *testing.T) {
	t.Parallel()
	server, mock := newAHNoticeTestServer(t)
	mock.ExpectBegin()
	mock.ExpectExec(noticeAckSQL).WithArgs("notice-owner", "{32,31}").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(noticeCountSQL).WithArgs("notice-owner").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(4))
	mock.ExpectCommit()
	response, result := callAHNotices(t, server, http.MethodPost, "/api/ah/notices", `{"ids":[32,31]}`)
	if response.Code != http.StatusOK || string(result["ok"]) != "true" || string(result["unseen_count"]) != "4" {
		t.Fatalf("acknowledgement = %s", response.Body.String())
	}
	if !strings.Contains(response.Header().Get("Cache-Control"), "no-store") {
		t.Fatalf("acknowledgement cache policy = %q", response.Header().Get("Cache-Control"))
	}
}

func TestAHNoticesLegacyPostConsumesOnlyReturnedNotices(t *testing.T) {
	t.Parallel()
	server, mock := newAHNoticeTestServer(t)
	mock.ExpectBegin()
	mock.ExpectQuery(noticeLegacySQL).WithArgs("notice-owner").WillReturnRows(
		noticeTestRows().AddRow(7, "refund", "Refund arrived.", 50, time.Unix(7, 0), false)).RowsWillBeClosed()
	mock.ExpectExec(noticeAckSQL).WithArgs("notice-owner", "{7}").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(noticeCountSQL).WithArgs("notice-owner").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
	mock.ExpectCommit()
	response, result := callAHNotices(t, server, http.MethodPost, "/api/ah/notices", `{}`)
	if response.Code != http.StatusOK || string(result["ok"]) != "true" || !strings.Contains(string(result["notices"]), `"id":7`) {
		t.Fatalf("legacy notices = %s", response.Body.String())
	}
}

func TestAHNoticesRejectsMalformedRequests(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		method string
		target string
		body   string
	}{
		{"zero cursor", "GET", "/api/ah/notices?before=0", ""},
		{"negative cursor", "GET", "/api/ah/notices?before=-1", ""},
		{"invalid cursor", "GET", "/api/ah/notices?before=other", ""},
		{"overflow cursor", "GET", "/api/ah/notices?before=9223372036854775808", ""},
		{"empty cursor", "GET", "/api/ah/notices?before=", ""},
		{"repeated cursor", "GET", "/api/ah/notices?before=4&before=5", ""},
		{"missing body", "POST", "/api/ah/notices", ""},
		{"invalid JSON", "POST", "/api/ah/notices", "{"},
		{"null body", "POST", "/api/ah/notices", "null"},
		{"null IDs", "POST", "/api/ah/notices", `{"ids":null}`},
		{"empty IDs", "POST", "/api/ah/notices", `{"ids":[]}`},
		{"zero ID", "POST", "/api/ah/notices", `{"ids":[0]}`},
		{"negative ID", "POST", "/api/ah/notices", `{"ids":[-1]}`},
		{"string ID", "POST", "/api/ah/notices", `{"ids":["31"]}`},
		{"too many IDs", "POST", "/api/ah/notices", `{"ids":[1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20,21]}`},
		{"trailing JSON", "POST", "/api/ah/notices", `{"ids":[31]} {}`},
		{"unknown fields", "POST", "/api/ah/notices", `{"id":31}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			server, _ := newAHNoticeTestServer(t)
			response, result := callAHNotices(t, server, test.method, test.target, test.body)
			if response.Code != http.StatusBadRequest || string(result["ok"]) != "false" || len(result["error"]) == 0 {
				t.Fatalf("invalid request = %d %s", response.Code, response.Body.String())
			}
		})
	}
}

func TestAHNoticesDatabaseErrorsDoNotReportSuccess(t *testing.T) {
	t.Parallel()
	failure := errors.New("private database details")
	tests := []struct {
		name   string
		method string
		body   string
		setup  func(sqlmock.Sqlmock)
	}{
		{"history query", "GET", "", func(mock sqlmock.Sqlmock) {
			mock.ExpectQuery(noticeHistorySQL).WithArgs("notice-owner").WillReturnError(failure)
		}},
		{"history scan", "GET", "", func(mock sqlmock.Sqlmock) {
			mock.ExpectQuery(noticeHistorySQL).WithArgs("notice-owner").WillReturnRows(
				noticeTestRows().AddRow("bad id", "sale", "Sale.", 1, time.Unix(1, 0), false)).RowsWillBeClosed()
		}},
		{"history iteration", "GET", "", func(mock sqlmock.Sqlmock) {
			mock.ExpectQuery(noticeHistorySQL).WithArgs("notice-owner").WillReturnRows(
				noticeTestRows().AddRow(1, "sale", "Sale.", 1, time.Unix(1, 0), false).RowError(0, failure)).RowsWillBeClosed()
		}},
		{"history count", "GET", "", func(mock sqlmock.Sqlmock) {
			mock.ExpectQuery(noticeHistorySQL).WithArgs("notice-owner").WillReturnRows(noticeTestRows())
			mock.ExpectQuery(noticeCountSQL).WithArgs("notice-owner").WillReturnError(failure)
		}},
		{"ack begin", "POST", `{"ids":[31]}`, func(mock sqlmock.Sqlmock) {
			mock.ExpectBegin().WillReturnError(failure)
		}},
		{"ack update", "POST", `{"ids":[31]}`, func(mock sqlmock.Sqlmock) {
			mock.ExpectBegin()
			mock.ExpectExec(noticeAckSQL).WithArgs("notice-owner", "{31}").WillReturnError(failure)
			mock.ExpectRollback()
		}},
		{"ack count rollback", "POST", `{"ids":[31]}`, func(mock sqlmock.Sqlmock) {
			mock.ExpectBegin()
			mock.ExpectExec(noticeAckSQL).WithArgs("notice-owner", "{31}").WillReturnResult(sqlmock.NewResult(0, 1))
			mock.ExpectQuery(noticeCountSQL).WithArgs("notice-owner").WillReturnError(failure)
			mock.ExpectRollback()
		}},
		{"ack commit", "POST", `{"ids":[31]}`, func(mock sqlmock.Sqlmock) {
			mock.ExpectBegin()
			mock.ExpectExec(noticeAckSQL).WithArgs("notice-owner", "{31}").WillReturnResult(sqlmock.NewResult(0, 1))
			mock.ExpectQuery(noticeCountSQL).WithArgs("notice-owner").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
			mock.ExpectCommit().WillReturnError(failure)
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			server, mock := newAHNoticeTestServer(t)
			test.setup(mock)
			response, result := callAHNotices(t, server, test.method, "/api/ah/notices", test.body)
			if response.Code != http.StatusInternalServerError || string(result["ok"]) != "false" || strings.Contains(response.Body.String(), failure.Error()) {
				t.Fatalf("database error = %d %s", response.Code, response.Body.String())
			}
		})
	}
}
