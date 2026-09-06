package bot

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/lib/pq"
)

const ahNoticePageSize = 20

func (s *WebServer) handleAHNotices(w http.ResponseWriter, r *http.Request, uid string) {
	switch r.Method {
	case http.MethodGet:
		s.handleAHNoticeHistory(w, r, uid)
	case http.MethodPost:
		s.handleAHNoticeAcknowledgement(w, r, uid)
	default:
		w.Header().Set("Allow", "GET, POST")
		writeJSONStatus(w, http.StatusMethodNotAllowed, map[string]any{"ok": false, "error": "Use GET to read notices or POST to mark them as read."})
	}
}

func (s *WebServer) handleAHNoticeHistory(w http.ResponseWriter, r *http.Request, uid string) {
	query := `SELECT id,kind,message,amount,created_at,seen FROM abyss_economy_events
		WHERE client_uid=$1 ORDER BY id DESC LIMIT 21`
	args := []any{uid}
	if values, present := r.URL.Query()["before"]; present {
		var before int64
		var err error
		if len(values) == 1 {
			before, err = strconv.ParseInt(values[0], 10, 64)
		}
		if err != nil || before <= 0 {
			writeJSONStatus(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "Choose a valid notice page and try again."})
			return
		}
		query = `SELECT id,kind,message,amount,created_at,seen FROM abyss_economy_events
			WHERE client_uid=$1 AND id<$2 ORDER BY id DESC LIMIT 21`
		args = append(args, before)
	}
	rows, err := s.bot.DB.QueryContext(r.Context(), query, args...)
	if err != nil {
		writeAHNoticeDBError(w, "load notice history", err)
		return
	}
	notices, err := readAHNoticeRows(rows)
	if err != nil {
		writeAHNoticeDBError(w, "read notice history", err)
		return
	}
	var unseenCount int
	if err := s.bot.DB.QueryRowContext(r.Context(), `SELECT COUNT(*) FROM abyss_economy_events WHERE client_uid=$1 AND seen=FALSE`, uid).Scan(&unseenCount); err != nil {
		writeAHNoticeDBError(w, "count unread notices", err)
		return
	}
	hasMore := len(notices) > ahNoticePageSize
	var nextBefore int64
	if hasMore {
		notices = notices[:ahNoticePageSize]
		nextBefore = notices[len(notices)-1].ID
	}
	writeJSON(w, map[string]any{"ok": true, "notices": notices, "unseen_count": unseenCount, "has_more": hasMore, "next_before": nextBefore})
}

func (s *WebServer) handleAHNoticeAcknowledgement(w http.ResponseWriter, r *http.Request, uid string) {
	var req *struct {
		IDs json.RawMessage `json:"ids"`
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 2048))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil || req == nil {
		writeJSONStatus(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "Choose up to 20 valid notices to mark as read."})
		return
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeJSONStatus(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "Choose up to 20 valid notices to mark as read."})
		return
	}
	legacy := req.IDs == nil
	var ids []int64
	if !legacy {
		if err := json.Unmarshal(req.IDs, &ids); err != nil || len(ids) == 0 || len(ids) > ahNoticePageSize {
			writeJSONStatus(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "Choose between 1 and 20 valid notices to mark as read."})
			return
		}
		for _, id := range ids {
			if id <= 0 {
				writeJSONStatus(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "Choose valid notices to mark as read."})
				return
			}
		}
	}
	tx, err := s.bot.DB.BeginTx(r.Context(), nil)
	if err != nil {
		writeAHNoticeDBError(w, "begin notice acknowledgement", err)
		return
	}
	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			log.Printf("web: rollback notice acknowledgement: %v", err)
		}
	}()
	notices := []abyssEconomyNotice{}
	if legacy {
		// Cached pages send {} and expect the oldest unread batch to be consumed.
		rows, err := tx.QueryContext(r.Context(), `SELECT id,kind,message,amount,created_at,seen FROM abyss_economy_events
			WHERE client_uid=$1 AND seen=FALSE ORDER BY created_at,id LIMIT 20 FOR UPDATE`, uid)
		if err != nil {
			writeAHNoticeDBError(w, "load legacy notices", err)
			return
		}
		notices, err = readAHNoticeRows(rows)
		if err != nil {
			writeAHNoticeDBError(w, "read legacy notices", err)
			return
		}
		for i := range notices {
			ids = append(ids, notices[i].ID)
			notices[i].Seen = true
		}
	}
	if len(ids) > 0 {
		if _, err := tx.ExecContext(r.Context(), `UPDATE abyss_economy_events SET seen=TRUE WHERE client_uid=$1 AND id=ANY($2) AND seen=FALSE`, uid, pq.Array(ids)); err != nil {
			writeAHNoticeDBError(w, "acknowledge notices", err)
			return
		}
	}
	var unseenCount int
	if err := tx.QueryRowContext(r.Context(), `SELECT COUNT(*) FROM abyss_economy_events WHERE client_uid=$1 AND seen=FALSE`, uid).Scan(&unseenCount); err != nil {
		writeAHNoticeDBError(w, "count unread notices after acknowledgement", err)
		return
	}
	if err := tx.Commit(); err != nil {
		writeAHNoticeDBError(w, "commit notice acknowledgement", err)
		return
	}
	result := map[string]any{"ok": true, "unseen_count": unseenCount}
	if legacy {
		result["notices"] = notices
	}
	writeJSON(w, result)
}

func readAHNoticeRows(rows *sql.Rows) (notices []abyssEconomyNotice, err error) {
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("close notice rows: %w", closeErr))
		}
	}()
	notices = []abyssEconomyNotice{}
	for rows.Next() {
		var notice abyssEconomyNotice
		var created time.Time
		if err := rows.Scan(&notice.ID, &notice.Kind, &notice.Message, &notice.Amount, &created, &notice.Seen); err != nil {
			return nil, fmt.Errorf("scan notice: %w", err)
		}
		notice.When = created.UTC().Format("02 Jan 2006 · 15:04 UTC")
		notice.WhenISO = created.UTC().Format(time.RFC3339)
		notices = append(notices, notice)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate notices: %w", err)
	}
	return notices, nil
}

func writeAHNoticeDBError(w http.ResponseWriter, operation string, err error) {
	log.Printf("web: %s: %v", operation, err)
	writeJSONStatus(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "Notices could not be updated or loaded. Please try again."})
}
