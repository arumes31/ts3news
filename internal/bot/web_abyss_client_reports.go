package bot

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	abyssClientReportBodyLimit = 8 << 10
	abyssClientReportMaxGroups = 100
	abyssClientReportMaxUsers  = 5000
	abyssClientReportRate      = 2 * time.Second
)

type abyssClientErrorReport struct {
	Kind   string `json:"kind"`
	Source string `json:"source"`
	Path   string `json:"path"`
	Line   int    `json:"line"`
	Column int    `json:"column"`
}

type abyssClientErrorSummary struct {
	Signature string    `json:"signature"`
	Kind      string    `json:"kind"`
	Source    string    `json:"source"`
	Path      string    `json:"path"`
	Line      int       `json:"line"`
	Column    int       `json:"column"`
	Count     int64     `json:"count"`
	LastSeen  time.Time `json:"last_seen"`
}

type abyssClientReportStore struct {
	mu        sync.Mutex
	lastByUID map[string]time.Time
	reports   map[string]abyssClientErrorSummary
	received  atomic.Int64
	dropped   atomic.Int64
}

func (s *abyssClientReportStore) record(
	uid string,
	report abyssClientErrorReport,
	now time.Time,
) (abyssClientErrorSummary, bool) {
	s.received.Add(1)
	report = sanitizeAbyssClientReport(report)
	if report.Kind == "" {
		s.dropped.Add(1)
		return abyssClientErrorSummary{}, false
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.lastByUID == nil {
		s.lastByUID = make(map[string]time.Time)
	}
	if last := s.lastByUID[uid]; !last.IsZero() && now.Sub(last) < abyssClientReportRate {
		s.dropped.Add(1)
		return abyssClientErrorSummary{}, false
	}
	if _, exists := s.lastByUID[uid]; !exists && len(s.lastByUID) >= abyssClientReportMaxUsers {
		s.evictOldestUIDLocked()
	}
	s.lastByUID[uid] = now
	if s.reports == nil {
		s.reports = make(map[string]abyssClientErrorSummary)
	}

	signature := abyssClientReportSignature(report)
	if _, exists := s.reports[signature]; !exists && len(s.reports) >= abyssClientReportMaxGroups {
		s.evictOldestLocked()
	}
	summary := s.reports[signature]
	summary.Signature = signature
	summary.Kind = report.Kind
	summary.Source = report.Source
	summary.Path = report.Path
	summary.Line = report.Line
	summary.Column = report.Column
	summary.Count++
	summary.LastSeen = now.UTC()
	s.reports[signature] = summary
	return summary, true
}

func (s *abyssClientReportStore) evictOldestUIDLocked() {
	var oldestUID string
	var oldest time.Time
	for uid, seen := range s.lastByUID {
		if oldestUID == "" || seen.Before(oldest) {
			oldestUID = uid
			oldest = seen
		}
	}
	delete(s.lastByUID, oldestUID)
}

func (s *abyssClientReportStore) evictOldestLocked() {
	var oldestKey string
	var oldest time.Time
	for key, summary := range s.reports {
		if oldestKey == "" || summary.LastSeen.Before(oldest) {
			oldestKey = key
			oldest = summary.LastSeen
		}
	}
	delete(s.reports, oldestKey)
}

func (s *abyssClientReportStore) snapshot() map[string]any {
	s.mu.Lock()
	reports := make([]abyssClientErrorSummary, 0, len(s.reports))
	for _, summary := range s.reports {
		reports = append(reports, summary)
	}
	s.mu.Unlock()
	sort.Slice(reports, func(i, j int) bool {
		if reports[i].Count == reports[j].Count {
			return reports[i].LastSeen.After(reports[j].LastSeen)
		}
		return reports[i].Count > reports[j].Count
	})
	if len(reports) > 20 {
		reports = reports[:20]
	}
	return map[string]any{
		"received": s.received.Load(),
		"dropped":  s.dropped.Load(),
		"top":      reports,
	}
}

func sanitizeAbyssClientReport(report abyssClientErrorReport) abyssClientErrorReport {
	switch report.Kind {
	case "script_error", "promise_rejection", "resource_error":
	default:
		report.Kind = ""
	}
	report.Source = boundedClientReportText(report.Source, 256)
	report.Path = boundedClientReportText(report.Path, 256)
	if report.Source != "inline" && report.Source != "cross-origin" &&
		!strings.HasPrefix(report.Source, "/static/") {
		report.Source = "other"
	}
	if report.Path != "/abyss" && report.Path != "/abyss/tree" {
		report.Path = "/abyss"
	}
	report.Line = max(0, min(report.Line, 10_000_000))
	report.Column = max(0, min(report.Column, 10_000_000))
	return report
}

func boundedClientReportText(value string, limit int) string {
	value = strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return ' '
		}
		return r
	}, strings.TrimSpace(value))
	value = strings.Join(strings.Fields(value), " ")
	runes := []rune(value)
	if len(runes) > limit {
		value = string(runes[:limit])
	}
	return value
}

func abyssClientReportSignature(report abyssClientErrorReport) string {
	key := report.Kind + "\x00" + report.Source + "\x00" + report.Path + "\x00" +
		strconv.Itoa(report.Line) + "\x00" + strconv.Itoa(report.Column)
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:8])
}

func (s *WebServer) handleAbyssClientError(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, abyssClientReportBodyLimit)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var report abyssClientErrorReport
	if err := decoder.Decode(&report); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, map[string]any{"ok": false, "error": "invalid client report"})
		return
	}
	summary, accepted := s.abyssClientReports.record(uid, report, time.Now())
	if accepted {
		slog.Info(
			"abyss client error received",
			"signature", summary.Signature,
			"count", summary.Count,
			"kind", summary.Kind,
			"source", summary.Source,
		)
	}
	w.WriteHeader(http.StatusAccepted)
	writeJSON(w, map[string]any{"ok": true, "accepted": accepted})
}
