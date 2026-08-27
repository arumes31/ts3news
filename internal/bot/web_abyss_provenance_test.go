package bot

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestAbyssRunFloorSeedIsStableAndDepthScoped(t *testing.T) {
	t.Parallel()

	runSeed := [2]uint64{91, 117}
	first := abyssRunFloorSeed(runSeed, 12)
	if first == [2]uint64{} {
		t.Fatal("derived floor seed is empty")
	}
	if first != abyssRunFloorSeed(runSeed, 12) {
		t.Fatal("same run seed and depth produced different floor seeds")
	}
	if first == abyssRunFloorSeed(runSeed, 13) {
		t.Fatal("adjacent depths produced the same floor seed")
	}
}

func TestAbyssCompetitionAuditRejectsCorruptRunProvenance(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	mock.ExpectQuery("SELECT COALESCE").WithArgs("user").
		WillReturnRows(sqlmock.NewRows([]string{"active_specialization"}).AddRow("warden"))
	mock.ExpectQuery("SELECT channel_id").WithArgs("user").
		WillReturnRows(sqlmock.NewRows([]string{"channel_id"}))
	mock.ExpectQuery("SELECT audit_hash").WithArgs("user").
		WillReturnRows(sqlmock.NewRows([]string{"audit_hash"}))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT value FROM app_meta WHERE key=$1")).
		WithArgs(abyssRunProvenanceKey("user")).
		WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow("{"))

	_, err = (&Bot{DB: database}).newAbyssCompetitionRunRecord(
		"user",
		abyssRun{Depth: 2, Tier: "normal", StartedAt: time.Now().Add(-time.Minute)},
		120,
		true,
		false,
		"banked",
		1,
	)
	if err == nil || !strings.Contains(err.Error(), "decoding run provenance") {
		t.Fatalf("error = %v, want corrupt provenance rejection", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAbyssCompetitionAuditIncludesSignedRunProvenance(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	provenance := abyssRunProvenance{
		Version: abyssRunProvenanceVersion,
		Seed:    [2]uint64{101, 202},
		Choices: []abyssRunChoice{{Depth: 2, Kind: "floor", Value: "combat"}},
		Floors: []abyssRunFloorRecord{{
			Depth: 2, Victory: true, Seed: [2]uint64{303, 404}, Logs: []string{"victory"},
		}},
	}
	raw, err := json.Marshal(provenance)
	if err != nil {
		t.Fatalf("marshal provenance: %v", err)
	}
	mock.ExpectQuery("SELECT COALESCE").WithArgs("user").
		WillReturnRows(sqlmock.NewRows([]string{"active_specialization"}).AddRow("warden"))
	mock.ExpectQuery("SELECT channel_id").WithArgs("user").
		WillReturnError(sqlmock.ErrCancelled)
	mock.ExpectQuery("SELECT audit_hash").WithArgs("user").
		WillReturnRows(sqlmock.NewRows([]string{"audit_hash"}).AddRow("previous"))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT value FROM app_meta WHERE key=$1")).
		WithArgs(abyssRunProvenanceKey("user")).
		WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(string(raw)))

	bot := &Bot{DB: database}
	record, err := bot.newAbyssCompetitionRunRecord(
		"user",
		abyssRun{Depth: 2, Tier: "normal", StartedAt: time.Now().Add(-time.Minute)},
		120,
		true,
		false,
		"banked",
		1,
	)
	if err != nil {
		t.Fatalf("newAbyssCompetitionRunRecord: %v", err)
	}
	var audit abyssCompetitionAudit
	if err := json.Unmarshal([]byte(record.AuditJSON), &audit); err != nil {
		t.Fatalf("decode audit: %v", err)
	}
	if audit.RunSeed == nil || *audit.RunSeed != provenance.Seed ||
		len(audit.Choices) != 1 || len(audit.Floors) != 1 {
		t.Fatalf("audit provenance = %+v", audit)
	}
	digest := sha256.Sum256([]byte(record.AuditJSON))
	if record.AuditHash != hex.EncodeToString(digest[:]) {
		t.Fatalf("audit hash = %q, want canonical digest", record.AuditHash)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
