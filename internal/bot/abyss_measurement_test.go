package bot

import (
	"encoding/json"
	"github.com/DATA-DOG/go-sqlmock"
	"testing"
	"time"
)

func TestAbyssMeasuredResolutionExcludesPlanningAndDaysBetweenTurns(t *testing.T) {
	start := time.Date(2026, 9, 13, 1, 0, 0, 0, time.UTC)
	combat := &abyssLiveCombat{resolutionStarted: start}
	combat.stopResolutionLocked(start.Add(15 * time.Millisecond))
	combat.stopResolutionLocked(start.Add(48 * time.Hour))
	combat.resolutionStarted = start.Add(48 * time.Hour)
	combat.stopResolutionLocked(start.Add(48*time.Hour + 25*time.Millisecond))
	if combat.timing.ResolutionNS != int64(40*time.Millisecond) {
		t.Fatalf("included idle time: %+v", combat.timing)
	}
	timing := combat.measurement()
	if timing.MeasuredFloors != 1 || timing.ActionWindowNS != 0 {
		t.Fatalf("timing=%+v", timing)
	}
	var absent *abyssLiveCombat
	if absent.measurement().UntimedFloors != 1 {
		t.Fatal("legacy execution was falsely timed")
	}
}
func TestAbyssHistoryLabelsCurrentPriorAndUnknownEconomies(t *testing.T) {
	for _, test := range []struct{ epoch, current, want string }{{"2", "2", "Current economy"}, {"1", "2", "Prior economy"}, {"", "2", "Economy unknown"}, {"2", "", "Economy unknown"}, {"unknown", "2", "Economy unknown"}} {
		if got := abyssEconomyLabel(test.epoch, test.current); got != test.want {
			t.Fatalf("label=%s want%s", got, test.want)
		}
	}
}
func TestAbyssBankAndDeathAuditRetainCohortAndMeasuredCoverage(t *testing.T) {
	for _, victory := range []bool{false, true} {
		database, mock, err := sqlmock.New()
		if err != nil {
			t.Fatal(err)
		}
		provenance := abyssRunProvenance{Version: 1, Seed: [2]uint64{1, 2}, Cohort: &abyssMeasurementCohort{Schema: 1, EconomyEpoch: "2", BuildRevision: "commit-verified"}, Timing: abyssCombatTiming{ResolutionNS: 40_000_000, ActionWindowNS: 3_000_000_000, MeasuredFloors: 1, UntimedFloors: 2}}
		encoded, err := json.Marshal(provenance)
		if err != nil {
			t.Fatal(err)
		}
		mock.ExpectQuery("SELECT COALESCE").WillReturnRows(sqlmock.NewRows([]string{"build"}).AddRow("warden"))
		mock.ExpectQuery("SELECT channel_id").WillReturnRows(sqlmock.NewRows([]string{"channel"}))
		mock.ExpectQuery("SELECT audit_hash").WillReturnRows(sqlmock.NewRows([]string{"hash"}))
		mock.ExpectQuery("SELECT value FROM app_meta").WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(string(encoded)))
		record, err := (&Bot{DB: database}).newAbyssCompetitionRunRecord("player", abyssRun{Depth: 10, Tier: "normal", StartedAt: time.Now().Add(-48 * time.Hour)}, 500, victory, false, "test", 1)
		if err != nil {
			t.Fatal(err)
		}
		var audit abyssCompetitionAudit
		if err := json.Unmarshal([]byte(record.AuditJSON), &audit); err != nil {
			t.Fatal(err)
		}
		if audit.Cohort == nil || *audit.Cohort != *provenance.Cohort || audit.Timing != provenance.Timing || audit.Victory != victory {
			t.Fatalf("lost audit measurement: %+v", audit)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
		_ = database.Close()
	}
}
