package bot

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/DATA-DOG/go-sqlmock"
	"testing"
	"ts3news/internal/content"
)

func TestAbyssClassRewardsExcludeSummonsCapturesAndFlee(t *testing.T) {
	dead := &content.Mob{Stats: content.Stats{HP: 0}}
	alive := &content.Mob{Stats: content.Stats{HP: 1}}
	summon := &content.Mob{}
	captured := &content.Mob{}
	fled := &content.Mob{}
	got := abyssWaveClassKills([]*content.Mob{dead, alive, captured, fled}, []*content.Mob{dead, alive, summon, fled}, 1, 1, map[*content.Mob]bool{fled: true})
	if len(got) != 1 || got[0].ID != "1:0" || got[0].XP != 250 {
		t.Fatal(got)
	}
}
func TestAbyssClassRewardPartialReviveIsIdempotent(t *testing.T) {
	p := abyssClassProgress{}
	r := abyssClassReceipt{Version: 1, Class: "warrior"}
	first := []abyssClassKill{{"1:0", 500}}
	p, r, delta := applyAbyssClassCredit(p, r, first, false, 1)
	if delta != 500 || p.Clears != 0 {
		t.Fatal(p)
	}
	p, r, delta = applyAbyssClassCredit(p, r, first, false, 1)
	if delta != 0 {
		t.Fatal("repeat paid")
	}
	p, r, delta = applyAbyssClassCredit(p, r, append(first, abyssClassKill{"1:1", 500}), true, 1)
	if p.XP != 1000 || p.Clears != 1 || delta != 500 {
		t.Fatal(p, delta)
	}
	p, _, delta = applyAbyssClassCredit(p, r, first, true, 1)
	if delta != 0 || p.Clears != 1 {
		t.Fatal("clear repeat")
	}
}
func TestAbyssClassRewardFarmingAndCap(t *testing.T) {
	p := abyssClassProgress{XP: 75000, BestDepth: 400}
	r := abyssClassReceipt{Version: 1}
	kills := []abyssClassKill{{"1:0", 1000}}
	p, r, delta := applyAbyssClassCredit(p, r, kills, true, 5)
	if delta != 250 || r.Efficiency != 25 {
		t.Fatal(p, r)
	}
	p.XP = content.AbyssClassPointFloors[14] * 1000
	p, _, _ = applyAbyssClassCredit(p, abyssClassReceipt{}, kills, true, 500)
	if content.AbyssClassPoints(p.XP) != 15 {
		t.Fatal(p)
	}
}
func TestAbyssClassRewardReceiptFailureRollsBackProgress(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()
	state := newAbyssClassState()
	state.Class = "warrior"
	raw, _ := json.Marshal(state)
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT client_uid FROM users").WithArgs("owner").WillReturnRows(sqlmock.NewRows([]string{"uid"}).AddRow("owner"))
	mock.ExpectQuery("SELECT value FROM app_meta").WithArgs("abyss_class_build:owner").WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(string(raw)))
	mock.ExpectQuery("SELECT value FROM app_meta").WithArgs("abyss_class_reward:owner:host:00000000000000010000000000000002").WillReturnRows(sqlmock.NewRows([]string{"value"}))
	mock.ExpectExec("INSERT INTO app_meta").WithArgs("abyss_class_build:owner", sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO app_meta").WillReturnError(errors.New("receipt write unavailable"))
	mock.ExpectRollback()
	_, err := (&Bot{DB: db}).grantAbyssClassCredit(context.Background(), "owner", "warrior", "host", [2]uint64{1, 2}, 1, []abyssClassKill{{"1:0", 1000}}, true)
	if err == nil {
		t.Fatal("failed receipt committed")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
func TestAbyssTalentEffectsApplyAndRemainBounded(t *testing.T) {
	state := newAbyssClassState()
	state.Class = "warrior"
	state.Progress["warrior"] = abyssClassProgress{XP: 75000, Foundation: []string{"warrior_t1_1", "warrior_t2_2", "warrior_t3_1", "warrior_t4_2", "warrior_t5_3"}}
	u := UserInCombat{AbyssClass: "warrior", Stats: content.Stats{STR: 100, DEF: 100, HP: 1000, MNA: 100}, CurrentHP: 1000, Skills: content.AbyssClassSkills("warrior")}
	applyAbyssTalentStats(&u, state)
	au := activeUser{u: &u}
	if u.Stats.STR != 105 || u.Stats.DEF != 108 {
		t.Fatal(u.Stats)
	}
	if abyssClassShield(&au, u.Skills[0]) != 27 {
		t.Fatal("builder barrier talent inactive")
	}
	p := previewAbyssClassSkill(&au, u.Skills[1], nil)
	if p.Power <= u.Skills[1].Power || p.HealPercent != .03 {
		t.Fatal(p)
	}
}

func TestAbyssClassReviveWithDifferentWavesCannotInflateXP(t *testing.T) {
	p, r, _ := applyAbyssClassCredit(abyssClassProgress{}, abyssClassReceipt{}, []abyssClassKill{{"1:0", 250}}, false, 10)
	p, r, _ = applyAbyssClassCredit(p, r, []abyssClassKill{{"1:0", 166}, {"1:1", 166}, {"1:2", 168}}, false, 10)
	if p.XP != 500 {
		t.Fatal("changed wave size inflated partial progress", p.XP)
	}
	p, r, _ = applyAbyssClassCredit(p, r, []abyssClassKill{{"1:0", 500}, {"2:0", 500}}, true, 10)
	p, _, _ = applyAbyssClassCredit(p, r, []abyssClassKill{{"1:0", 500}, {"2:0", 500}}, true, 10)
	if p.XP != 1000 || p.Clears != 1 {
		t.Fatal(p)
	}
}
func TestAbyssTalentHealthDoesNotHealOnEveryBuild(t *testing.T) {
	state := newAbyssClassState()
	state.Class = "warrior"
	state.Progress["warrior"] = abyssClassProgress{XP: 1000, Foundation: []string{"warrior_t1_2"}}
	for i := 0; i < 3; i++ {
		u := UserInCombat{AbyssClass: "warrior", Stats: content.Stats{HP: 100}, CurrentHP: 50}
		applyAbyssTalentStats(&u, state)
		if u.CurrentHP != 50 || u.Stats.HP != 108 {
			t.Fatal(u)
		}
	}
}

func TestAbyssClassPartialFarmingRoundingPreservesTotal(t *testing.T) {
	p := abyssClassProgress{XP: 75000, BestDepth: 400}
	r := abyssClassReceipt{}
	p, r, _ = applyAbyssClassCredit(p, r, []abyssClassKill{{"1:0", 333}}, false, 1)
	p, _, _ = applyAbyssClassCredit(p, r, []abyssClassKill{{"1:0", 1000}}, true, 1)
	if p.XP != 75250 {
		t.Fatal("partial rounding lost XP", p.XP)
	}
}
func TestAbyssClassCorruptReceiptsCannotBeRecredited(t *testing.T) {
	for _, bad := range []string{"null", "{}", `{"version":2}`, `{"version":1,"class":"warrior","efficiency":100,"maximum_xp":9999}`} {
		db, mock, _ := sqlmock.New()
		state := newAbyssClassState()
		state.Class = "warrior"
		raw, _ := json.Marshal(state)
		mock.ExpectBegin()
		mock.ExpectQuery("SELECT client_uid").WillReturnRows(sqlmock.NewRows([]string{"uid"}).AddRow("owner"))
		mock.ExpectQuery("SELECT value").WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(string(raw)))
		mock.ExpectQuery("SELECT value").WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(bad))
		mock.ExpectRollback()
		_, err := (&Bot{DB: db}).grantAbyssClassCredit(context.Background(), "owner", "warrior", "host", [2]uint64{1, 2}, 1, []abyssClassKill{{"1:0", 1000}}, true)
		if err == nil {
			t.Fatal("corrupt receipt accepted", bad)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
		db.Close()
	}
}

func TestAbyssClassPendingJournalSurvivesCreditFailure(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	p := abyssClassPending{UID: "owner", Class: "warrior", Owner: "host", Seed: [2]uint64{1, 2}, Depth: 1, Kills: []abyssClassKill{{"1:0", 1000}}, Cleared: true}
	raw, _ := json.Marshal(p)
	key := abyssClassPendingKey(p, raw)
	mock.ExpectExec("INSERT INTO app_meta").WithArgs(key, string(raw)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectBegin().WillReturnError(errors.New("credit transaction unavailable"))
	if _, err := (&Bot{DB: db}).deliverAbyssClassPending(context.Background(), p); err == nil {
		t.Fatal("credit failure hidden")
	}
	// No DELETE is permitted after a failed transaction: restart replay must keep the journal.
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
