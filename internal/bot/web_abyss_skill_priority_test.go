package bot

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestHandleAbyssSkillPriorityPersistsValidatedOrder(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	mock.ExpectQuery(`SELECT skill_id FROM user_skills WHERE client_uid = \$1 ORDER BY slot`).
		WithArgs("user").
		WillReturnRows(sqlmock.NewRows([]string{"skill_id"}).AddRow("S0_1").AddRow("S0_2"))
	mock.ExpectQuery(`SELECT node_id FROM user_abyss_tree WHERE client_uid=\$1`).
		WithArgs("user").
		WillReturnRows(sqlmock.NewRows([]string{"node_id"}))
	mock.ExpectExec(`INSERT INTO app_meta`).
		WithArgs("abyss_skill_priority:user", `["S0_2","S0_1"]`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/abyss/combat/skill_priority",
		bytes.NewBufferString(`{"skill_priority":["S0_2","S0_1"]}`),
	)
	server := &WebServer{bot: &Bot{DB: db}}
	server.handleAbyssSkillPriority(recorder, request, "user")
	if !strings.Contains(recorder.Body.String(), `"skill_priority":["S0_2","S0_1"]`) {
		t.Fatalf("response = %s", recorder.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestHandleAbyssSkillPriorityKeepsPreviousOrderWhenPersistenceFails(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	mock.ExpectQuery(`SELECT skill_id FROM user_skills WHERE client_uid = \$1 ORDER BY slot`).
		WithArgs("user").
		WillReturnRows(sqlmock.NewRows([]string{"skill_id"}).AddRow("S0_1").AddRow("S0_2"))
	mock.ExpectQuery(`SELECT node_id FROM user_abyss_tree WHERE client_uid=\$1`).
		WithArgs("user").
		WillReturnRows(sqlmock.NewRows([]string{"node_id"}))
	mock.ExpectExec(`INSERT INTO app_meta`).
		WithArgs("abyss_skill_priority:user", `["S0_2","S0_1"]`).
		WillReturnError(errors.New("database unavailable"))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/abyss/combat/skill_priority",
		bytes.NewBufferString(`{"skill_priority":["S0_2","S0_1"]}`),
	)
	server := &WebServer{bot: &Bot{DB: db}}
	server.handleAbyssSkillPriority(recorder, request, "user")
	if !strings.Contains(recorder.Body.String(), `"ok":false`) || !strings.Contains(recorder.Body.String(), "could not be saved") {
		t.Fatalf("response = %s", recorder.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestHandleAbyssSkillPriorityRejectsDuplicate(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	mock.ExpectQuery(`SELECT skill_id FROM user_skills WHERE client_uid = \$1 ORDER BY slot`).
		WithArgs("user").
		WillReturnRows(sqlmock.NewRows([]string{"skill_id"}).AddRow("S0_1"))
	mock.ExpectQuery(`SELECT node_id FROM user_abyss_tree WHERE client_uid=\$1`).
		WithArgs("user").
		WillReturnRows(sqlmock.NewRows([]string{"node_id"}))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/abyss/combat/skill_priority",
		bytes.NewBufferString(`{"skill_priority":["S0_1","S0_1"]}`),
	)
	server := &WebServer{bot: &Bot{DB: db}}
	server.handleAbyssSkillPriority(recorder, request, "user")
	if !strings.Contains(recorder.Body.String(), `"ok":false`) || !strings.Contains(recorder.Body.String(), "duplicate") {
		t.Fatalf("response = %s", recorder.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAbyssSkillPriorityAssetsAndContracts(t *testing.T) {
	t.Parallel()

	server, err := NewWebServer(nil)
	if err != nil {
		t.Fatalf("NewWebServer: %v", err)
	}
	script := server.tmpl.Lookup("abyssSkillPriorityJS")
	if script == nil {
		t.Fatal("Abyss skill priority script partial is missing")
	}
	for _, required := range []string{
		"/api/abyss/combat/skill_priority",
		"dragstart",
		"drop",
		"event.altKey",
		"data-priority-move",
		"textContent",
	} {
		if !strings.Contains(script.Tree.Root.String(), required) {
			t.Errorf("Abyss skill priority script is missing %q", required)
		}
	}
	if strings.Contains(script.Tree.Root.String(), "innerHTML") || strings.Contains(script.Tree.Root.String(), "localStorage") {
		t.Error("Abyss skill priority must use safe DOM text and server persistence")
	}

	page, err := webAssets.ReadFile("webassets/abyss.html")
	if err != nil {
		t.Fatalf("read Abyss page: %v", err)
	}
	css, err := webAssets.ReadFile("webassets/abyss_skill_priority.css")
	if err != nil {
		t.Fatalf("read skill priority stylesheet: %v", err)
	}
	routes, err := webAssets.ReadFile("webassets/abyss_skill_priority.html")
	if err != nil {
		t.Fatalf("read skill priority component: %v", err)
	}
	for _, required := range []string{
		`{{asset "/static/abyss_skill_priority.css"}}`,
		`{{template "abyssSkillPriorityPanel" .SkillPriority}}`,
		`{{template "abyssSkillPriorityJS" .}}`,
	} {
		if !strings.Contains(string(page), required) {
			t.Errorf("Abyss page is missing %q", required)
		}
	}
	for _, required := range []string{"draggable=\"true\"", "tabindex=\"0\"", "aria-live=\"polite\""} {
		if !strings.Contains(string(routes), required) {
			t.Errorf("Abyss skill priority component is missing %q", required)
		}
	}
	for _, required := range []string{".ab-skill-priority", "@media (max-width: 520px)", "prefers-reduced-motion"} {
		if !strings.Contains(string(css), required) {
			t.Errorf("Abyss skill priority CSS is missing %q", required)
		}
	}
}
