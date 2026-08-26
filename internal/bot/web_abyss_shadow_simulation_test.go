package bot

import (
	"context"
	"database/sql"
	"errors"
	"reflect"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"ts3news/internal/content"
)

func TestAbyssWilsonIntervalBounds(t *testing.T) {
	tests := []struct {
		wins     int
		trials   int
		wantLow  int
		wantHigh int
	}{
		{wins: 0, trials: 0, wantLow: 0, wantHigh: 0},
		{wins: 0, trials: 100, wantLow: 0, wantHigh: 4},
		{wins: 50, trials: 100, wantLow: 40, wantHigh: 60},
		{wins: 100, trials: 100, wantLow: 96, wantHigh: 100},
	}
	for _, test := range tests {
		low, high := abyssWilsonInterval(test.wins, test.trials)
		if low != test.wantLow || high != test.wantHigh {
			t.Fatalf("interval(%d, %d) = %d-%d, want %d-%d", test.wins, test.trials, low, high, test.wantLow, test.wantHigh)
		}
	}
}

func TestMedianInt(t *testing.T) {
	for _, test := range []struct {
		values []int
		want   int
	}{
		{want: 0},
		{values: []int{7}, want: 7},
		{values: []int{9, 1, 5}, want: 5},
		{values: []int{8, 2, 6, 4}, want: 5},
	} {
		if got := medianInt(test.values); got != test.want {
			t.Fatalf("medianInt(%v) = %d, want %d", test.values, got, test.want)
		}
	}
}

func TestCloneAbyssShadowUsersOwnsMutableCombatState(t *testing.T) {
	users := []UserInCombat{{
		Skills: []content.Skill{{ID: "skill"}},
		Pets: []*content.Mob{{
			Name:      "Companion",
			CurrentHP: 20,
			Effects:   []content.MobEffect{content.EffectEnraged},
		}},
		Ultimates:         []*content.UltimateSkill{{ID: "ultimate", CurrentCooldown: 2}},
		Equipped:          map[content.GearSlot]content.Gear{content.SlotMainHand: {ID: "weapon"}},
		shadowConsumables: []content.Consumable{{ID: "potion"}},
		shadowEffects:     []content.ItemEffect{content.EffectPhoenix},
		abyssSkillsUsed:   map[string]struct{}{"skill": {}},
	}}

	clones := cloneAbyssShadowUsers(users)
	clones[0].Skills[0].ID = "changed"
	clones[0].Pets[0].CurrentHP = 0
	clones[0].Pets[0].Effects[0] = content.EffectPoisoned
	clones[0].Ultimates[0].CurrentCooldown = 0
	clones[0].Equipped[content.SlotMainHand] = content.Gear{ID: "changed"}
	clones[0].shadowConsumables[0].ID = "used"
	clones[0].shadowEffects[0] = content.EffectVampiric
	clones[0].abyssSkillsUsed["other"] = struct{}{}

	if users[0].Skills[0].ID != "skill" || users[0].Pets[0].CurrentHP != 20 ||
		users[0].Pets[0].Effects[0] != content.EffectEnraged ||
		users[0].Ultimates[0].CurrentCooldown != 2 ||
		users[0].Equipped[content.SlotMainHand].ID != "weapon" ||
		users[0].shadowConsumables[0].ID != "potion" ||
		users[0].shadowEffects[0] != content.EffectPhoenix ||
		len(users[0].abyssSkillsUsed) != 1 {
		t.Fatal("mutating a shadow trial changed the prepared combat snapshot")
	}
}
func TestAbyssShadowSimulationIsDeterministicAndCancellable(t *testing.T) {
	users := []UserInCombat{{
		UID:        "shadow",
		Nickname:   "Scout",
		Level:      10,
		Stats:      content.Stats{HP: 500, STR: 80, DEF: 40, SPD: 30},
		CurrentHP:  500,
		EscrowLoot: true,
		IsClone:    true,
		shadow:     true,
	}}
	mobs := []*content.Mob{{
		Name:      "Target",
		Type:      content.MobCommon,
		Level:     10,
		Stats:     content.Stats{HP: 300, STR: 30, DEF: 15, SPD: 20},
		CurrentHP: 300,
		MaxHP:     300,
	}}
	zone := content.Zone{Name: "Projection", Difficulty: 1}
	seed := [2]uint64{17, 29}

	first, err := (&Bot{}).simulatePreparedAbyssCombat(context.Background(), users, mobs, 10, 1, zone, 4, seed)
	if err != nil {
		t.Fatalf("first simulation: %v", err)
	}
	second, err := (&Bot{}).simulatePreparedAbyssCombat(context.Background(), users, mobs, 10, 1, zone, 4, seed)
	if err != nil {
		t.Fatalf("second simulation: %v", err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("same snapshot and seed produced different results: %+v != %+v", first, second)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := (&Bot{}).simulatePreparedAbyssCombat(ctx, users, mobs, 10, 1, zone, 4, seed); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled simulation error = %v, want context.Canceled", err)
	}
}

func TestShadowCombatConsumesOnlyItsLocalPotion(t *testing.T) {
	user := &UserInCombat{
		shadow: true,
		shadowConsumables: []content.Consumable{
			{ID: "heal"},
			{ID: "heal"},
			{ID: "revive"},
			{ID: "revive"},
		},
	}
	(&Bot{}).consumeCombatConsumable(user, "heal", false)
	if got := user.shadowConsumables; len(got) != 3 || got[0].ID != "heal" {
		t.Fatalf("single local consumption = %+v, want one heal and two revives", got)
	}
	(&Bot{}).consumeCombatConsumable(user, "revive", true)
	if got := user.shadowConsumables; len(got) != 1 || got[0].ID != "heal" {
		t.Fatalf("consume-all local result = %+v, want only one heal", got)
	}
}

func TestChargeAbyssShadowSimulationIsAtomic(t *testing.T) {
	tests := []struct {
		name       string
		row        *sqlmock.Rows
		queryErr   error
		commitErr  error
		wantTokens int
		wantErr    error
	}{
		{name: "success", row: sqlmock.NewRows([]string{"abyss_tokens"}).AddRow(8), wantTokens: 8},
		{name: "insufficient", row: sqlmock.NewRows([]string{"abyss_tokens"}), wantErr: errAbyssShadowInsufficientTokens},
		{name: "update failure", queryErr: sql.ErrConnDone, wantErr: sql.ErrConnDone},
		{name: "commit failure", row: sqlmock.NewRows([]string{"abyss_tokens"}).AddRow(8), commitErr: sql.ErrTxDone, wantErr: sql.ErrTxDone},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = db.Close() }()
			mock.ExpectBegin()
			query := mock.ExpectQuery("UPDATE users SET abyss_tokens").
				WithArgs(abyssShadowSimulationCost, "scout")
			if test.queryErr != nil {
				query.WillReturnError(test.queryErr)
			} else {
				query.WillReturnRows(test.row)
			}
			if test.queryErr != nil || test.wantErr == errAbyssShadowInsufficientTokens {
				mock.ExpectRollback()
			} else if test.commitErr != nil {
				mock.ExpectCommit().WillReturnError(test.commitErr)
			} else {
				mock.ExpectCommit()
			}

			tokens, err := chargeAbyssShadowSimulation(context.Background(), db, "scout")
			if tokens != test.wantTokens || !errors.Is(err, test.wantErr) {
				t.Fatalf("charge = %d, %v; want %d, %v", tokens, err, test.wantTokens, test.wantErr)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatalf("database expectations: %v", err)
			}
		})
	}
}
