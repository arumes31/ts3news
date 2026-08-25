package bot

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

// TestAbyssDescend100ConcurrentDelvers is the CI load gate for the real HTTP
// descend handler. One hundred independent players arrive together and take the
// inexpensive authoritative no-active-run path; this catches lock contention,
// request leaks, and accidental global serialization before combat work begins.
func TestAbyssDescend100ConcurrentDelvers(t *testing.T) {
	const delvers = 100

	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	mock.MatchExpectationsInOrder(false)
	for range delvers {
		mock.ExpectQuery("SELECT depth, escrow, tier, insured").
			WithArgs(sqlmock.AnyArg()).
			WillReturnError(sql.ErrNoRows)
	}

	server, err := NewWebServer(&Bot{DB: database})
	if err != nil {
		t.Fatal(err)
	}
	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		server.handleAbyssDescend(w, r, r.Header.Get("X-Test-Delver"))
	}))
	t.Cleanup(httpServer.Close)

	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			MaxConnsPerHost:     delvers,
			MaxIdleConns:        delvers,
			MaxIdleConnsPerHost: delvers,
		},
	}
	t.Cleanup(client.CloseIdleConnections)

	started := time.Now()
	errors := make(chan error, delvers)
	var requests sync.WaitGroup
	requests.Add(delvers)
	for index := range delvers {
		go func() {
			defer requests.Done()
			request, err := http.NewRequest(http.MethodPost, httpServer.URL, bytes.NewReader([]byte("{}")))
			if err != nil {
				errors <- err
				return
			}
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("X-Test-Delver", fmt.Sprintf("load-delver-%03d", index))
			response, err := client.Do(request)
			if err != nil {
				errors <- err
				return
			}
			defer func() { _ = response.Body.Close() }()
			var payload struct {
				OK    bool   `json:"ok"`
				Error string `json:"error"`
			}
			if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
				errors <- err
				return
			}
			if response.StatusCode != http.StatusOK || payload.OK || payload.Error != "not in a run" {
				errors <- fmt.Errorf("status=%d payload=%+v", response.StatusCode, payload)
			}
		}()
	}
	requests.Wait()
	close(errors)
	for err := range errors {
		t.Errorf("concurrent descend: %v", err)
	}
	if elapsed := time.Since(started); elapsed > 10*time.Second {
		t.Errorf("100 concurrent descend requests took %v, want <= 10s", elapsed)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
