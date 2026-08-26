package bot

import (
	"context"
	"encoding/json"
	"regexp"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"ts3news/internal/content"
)

func TestRerollAbyssRingSocketsPreservesGemsAndCapsLayout(t *testing.T) {
	t.Parallel()

	rolls := []int{1, 0}
	next := func(limit int) int {
		value := rolls[0]
		rolls = rolls[1:]
		if value >= limit {
			t.Fatalf("test roll %d exceeds limit %d", value, limit)
		}
		return value
	}
	gear := content.Gear{
		Slot: content.SlotFinger1, Sockets: 2,
		Gemstones: []string{"Ruby II", "Sapphire"},
	}
	rerolled, err := rerollAbyssRingSockets(gear, next)
	if err != nil {
		t.Fatalf("rerollAbyssRingSockets: %v", err)
	}
	if rerolled.Sockets != 3 {
		t.Fatalf("sockets = %d, want 3", rerolled.Sockets)
	}
	if got := strings.Join(rerolled.Gemstones, ","); got != "Sapphire,Ruby II" {
		t.Fatalf("rerolled gems = %q", got)
	}
	if got := strings.Join(gear.Gemstones, ","); got != "Ruby II,Sapphire" {
		t.Fatalf("source ring was mutated: %q", got)
	}
}

func TestRerollAbyssRingSocketsRejectsInvalidTargets(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		gear content.Gear
		want string
	}{
		{name: "non-ring", gear: content.Gear{Slot: content.SlotNeck}, want: "limited to rings"},
		{name: "unidentified", gear: content.Gear{Slot: content.SlotFinger2, Unidentified: true}, want: "identify"},
		{name: "too many fitted gems", gear: content.Gear{Slot: content.SlotFinger1, Gemstones: []string{"a", "b", "c", "d"}}, want: "at most 3"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := rerollAbyssRingSockets(test.gear, func(int) int { return 0 })
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestAbyssRingSocketForgeCatalogIsRingOnly(t *testing.T) {
	t.Parallel()

	operation, found := abyssForgeOperationByID("reroll_ring_sockets")
	if !found {
		t.Fatal("reroll_ring_sockets is missing from the forge catalog")
	}
	if operation.Discipline != forgeDisciplineGemcraft || !operation.RequiresGear || !operation.Reversible {
		t.Fatalf("ring socket operation = %+v", operation)
	}
	want := []content.GearSlot{content.SlotFinger1, content.SlotFinger2}
	if len(operation.CompatibleSlots) != len(want) {
		t.Fatalf("compatible slots = %v", operation.CompatibleSlots)
	}
	for index := range want {
		if operation.CompatibleSlots[index] != want[index] {
			t.Fatalf("compatible slots = %v, want %v", operation.CompatibleSlots, want)
		}
	}
	if policy := forgeQuoteCostCoverage[operation.ID]; policy != "fixed" {
		t.Fatalf("quote policy = %q, want fixed", policy)
	}
}

func TestAbyssRingSocketQuoteRejectsOverfilledRing(t *testing.T) {
	t.Parallel()

	gear := content.Gear{
		Slot:      content.SlotFinger1,
		Gemstones: []string{"Ruby", "Sapphire", "Topaz", "Diamond"},
	}
	server := &WebServer{}
	_, _, _, err := server.resolveAbyssForgeQuoteCost(
		context.Background(), "user", "reroll_ring_sockets", &gear, nil,
	)
	if err == nil || !strings.Contains(err.Error(), "at most 3") {
		t.Fatalf("quote error = %v", err)
	}
}

func TestHandleAbyssRerollRingSocketsCommitsAtomically(t *testing.T) {
	server, mock, done := newForge2TestServer(t)
	defer done()
	uid := "ring-reroll-user"

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT gear_id, item_data FROM user_inventory").
		WithArgs(int64(97), uid).
		WillReturnRows(sqlmock.NewRows([]string{"gear_id", "item_data"}).AddRow(
			"U_LEG_2", `{"Slot":"Finger1","Name":"Prism Band","sockets":2,"gemstones":["Ruby","Topaz II"]}`,
		))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE user_materials SET count = count - $1 WHERE client_uid=$2 AND mat_id=$3 AND count >= $1")).
		WithArgs(abyssRingSocketCost, uid, "shard").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO abyss_forge_material_flow").
		WithArgs(uid, "shard", abyssRingSocketCost).
		WillReturnResult(sqlmock.NewResult(1, 1))
	expectNoSecondForgeUndo(mock, uid)
	mock.ExpectExec("UPDATE users SET forge_undo=").
		WithArgs(uid, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	var saved string
	mock.ExpectExec("UPDATE user_inventory SET item_data=").
		WithArgs(captureJSON{&saved}, int64(97), uid).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	mock.ExpectExec("INSERT INTO forge_history").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("UPDATE users SET forge_rep").WillReturnResult(sqlmock.NewResult(0, 1))

	recorder := postForge2(t, server.handleAbyssRerollRingSockets, `{"inv_id":97}`, uid)
	if !strings.Contains(recorder.Body.String(), `"ok":true`) {
		t.Fatalf("ring socket reroll failed: %s", recorder.Body.String())
	}
	var savedGear content.Gear
	if err := json.Unmarshal([]byte(saved), &savedGear); err != nil {
		t.Fatalf("saved item_data: %v", err)
	}
	if savedGear.Sockets < 2 || savedGear.Sockets > abyssRingSocketMaximum {
		t.Fatalf("saved socket count = %d", savedGear.Sockets)
	}
	if len(savedGear.Gemstones) != 2 {
		t.Fatalf("saved gems = %v", savedGear.Gemstones)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("database expectations: %v", err)
	}
}
