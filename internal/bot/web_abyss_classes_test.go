package bot

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/DATA-DOG/go-sqlmock"
	"regexp"
	"testing"
	"ts3news/internal/content"
)

func TestAbyssClassProfileValidation(t *testing.T) {
	known := []content.Skill{{ID: "a"}, {ID: "b"}}
	for _, test := range []struct {
		name    string
		profile abyssClassProfile
		valid   bool
	}{
		{"valid", abyssClassProfile{Skills: []string{"a"}, Pins: []string{"a"}}, true},
		{"unknown", abyssClassProfile{Skills: []string{"unowned"}}, false},
		{"duplicate", abyssClassProfile{Skills: []string{"a", "a"}}, false},
		{"capacity", abyssClassProfile{Skills: []string{"a", "b"}}, false},
		{"unowned pin", abyssClassProfile{Pins: []string{"unowned"}}, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			if (validateAbyssClassProfile(test.profile, known, 1) == nil) != test.valid {
				t.Fatal("invalid validation result")
			}
		})
	}
}
func TestAbyssClassLoadPreservesReadFailures(t *testing.T) {
	for _, test := range []struct {
		name, raw string
		err       error
		wantError bool
	}{
		{"missing", "", sql.ErrNoRows, false}, {"database failure", "", errors.New("unavailable"), true}, {"null", "null", nil, true}, {"missing version", "{}", nil, true}, {"corrupt", "bad", nil, true}, {"future version", `{"version":3}`, nil, true}, {"unknown subclass", `{"version":1,"selected":"bad"}`, nil, true}, {"valid", `{"version":1,"selected":"oracle"}`, nil, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			database, mock, _ := sqlmock.New()
			defer func() { _ = database.Close() }()
			query := mock.ExpectQuery("SELECT value FROM app_meta WHERE key").WithArgs("abyss_class_build:owner")
			if test.err != nil {
				query.WillReturnError(test.err)
			} else {
				query.WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(test.raw))
			}
			_, err := (&Bot{DB: database}).loadAbyssClassState(context.Background(), "owner")
			if (err != nil) != test.wantError {
				t.Fatal(err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
func TestAbyssClassSwitchRestoresSkillsWithoutChangingEquipmentOrGrants(t *testing.T) {
	original, _ := content.GetSkillByID("S0_1")
	other, _ := content.GetSkillByID("S0_2")
	state := newAbyssClassState()
	state.Selected = "oracle"
	state.Profiles["oracle"] = abyssClassProfile{Skills: []string{other.ID}, Pins: []string{other.ID}}
	user := UserInCombat{Skills: []content.Skill{original, {ID: "S_AS"}}, Equipped: map[content.GearSlot]content.Gear{content.SlotMainHand: {ID: "owned-sword"}}}
	(&Bot{}).applyAbyssClassBuild(&user, state)
	if len(user.Skills) != 4 || user.Skills[0].ID != other.ID || user.Skills[1].ID != "S_AS" || user.Equipped[content.SlotMainHand].ID != "owned-sword" {
		t.Fatal("existing investments changed", user)
	}
	if user.AbyssClass != "warden" || user.AbyssSubclass != "oracle" {
		t.Fatal("class mapping wrong")
	}
}
func TestAbyssPinnedSkillsCannotBeAutomaticallyReplaced(t *testing.T) {
	database, mock, _ := sqlmock.New()
	defer func() { _ = database.Close() }()
	state := newAbyssClassState()
	state.Profiles["oracle"] = abyssClassProfile{Pins: []string{"S0_1"}}
	raw, _ := json.Marshal(state)
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT client_uid FROM users").WithArgs("owner").WillReturnRows(sqlmock.NewRows([]string{"client_uid"}).AddRow("owner"))
	mock.ExpectQuery("SELECT value FROM app_meta WHERE key").WithArgs("abyss_class_build:owner").WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(string(raw)))
	mock.ExpectRollback()
	if err := (&Bot{DB: database}).replaceAbyssAcquiredSkill("owner", 1, "S0_1", "S0_2"); err == nil {
		t.Fatal("pin overwritten")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
func TestAbyssReplacementArchivesBothSkillsAtomically(t *testing.T) {
	database, mock, _ := sqlmock.New()
	defer func() { _ = database.Close() }()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT client_uid FROM users").WithArgs("owner").WillReturnRows(sqlmock.NewRows([]string{"client_uid"}).AddRow("owner"))
	mock.ExpectQuery("SELECT value FROM app_meta WHERE key").WillReturnError(sql.ErrNoRows)
	for _, id := range []string{"S0_1", "S0_2"} {
		mock.ExpectExec("INSERT INTO app_meta").WithArgs("abyss_learned:owner:"+id, id).WillReturnResult(sqlmock.NewResult(0, 1))
	}
	mock.ExpectExec("UPDATE user_skills").WithArgs("owner", 1, "S0_2", "S0_1").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	if err := (&Bot{DB: database}).replaceAbyssAcquiredSkill("owner", 1, "S0_1", "S0_2"); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
func TestAbyssClassEqualBudgetCombatPlaytest(t *testing.T) {
	tier, _ := abyssTierByKey("normal")
	for _, class := range content.AbyssClasses() {
		for _, sub := range class.Subclasses {
			t.Run(sub.ID, func(t *testing.T) {
				user := abyssBalancePlaytestUser(100, content.RarityRare)
				// Equal primary stat budgets isolate class behavior from gear-roll differences.
				primary := max(user.Stats.STR, user.Stats.INT, user.Stats.DEF)
				user.Stats.STR = primary
				user.Stats.INT = primary
				user.Stats.DEF = primary
				user.Stats.MNA = 200
				user.AbyssClass = class.ID
				user.AbyssSubclass = sub.ID
				user.Skills = content.AbyssClassSkills(sub.ID)
				for _, depth := range []int{1, 20, 100, 400} {
					mobs, zone, difficulty := abyssBalancePlaytestEncounter(depth, 100, tier, 17)
					result, err := (&Bot{}).simulatePreparedAbyssCombat(context.Background(), []UserInCombat{user}, mobs, 100, difficulty, zone, 12, [2]uint64{42, uint64(depth)})
					if err != nil {
						t.Fatal(err)
					}
					t.Logf("floor=%d wins=%d/%d winner_hp=%d", depth, result.Wins, result.Trials, result.MedianWinHPPct)
					if depth == 1 && result.Wins < 9 {
						t.Fatal("subclass cannot reliably clear normal entrance", result)
					}
				}
			})
		}
	}
}

func TestAbyssClassRunLockIncludesEveryPartyMemberAndPreservesQueryFailures(t *testing.T) {
	for _, fail := range []bool{false, true} {
		database, mock, _ := sqlmock.New()
		query := mock.ExpectQuery(regexp.QuoteMeta(abyssClassRunLockQuery)).WithArgs("third-helper")
		if fail {
			query.WillReturnError(errors.New("offline"))
		} else {
			query.WillReturnRows(sqlmock.NewRows([]string{"locked"}).AddRow(true))
		}
		locked, err := (&Bot{DB: database}).abyssClassRunLocked(context.Background(), "third-helper")
		if fail && err == nil || !fail && (!locked || err != nil) {
			t.Fatal(locked, err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
		_ = database.Close()
	}
}

func TestAbyssSkillGrantFailureStaysRetryable(t *testing.T) {
	database, mock, _ := sqlmock.New()
	defer func() { _ = database.Close() }()
	skill, _ := content.GetSkillByID("S0_1")
	skill.Rarity = content.RarityLegendary
	// Ownership is durable before automatic equip; failed replacement must not auction.
	mock.ExpectExec("INSERT INTO app_meta").WithArgs("abyss_learned:owner:"+skill.ID, skill.ID).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery("SELECT title FROM users").WillReturnRows(sqlmock.NewRows([]string{"title"}).AddRow(nil))
	rows := sqlmock.NewRows([]string{"slot", "skill_id"})
	for i := 1; i <= 5; i++ {
		rows.AddRow(i, "S0_2")
	}
	mock.ExpectQuery("SELECT slot, skill_id FROM user_skills").WillReturnRows(rows).RowsWillBeClosed()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT client_uid FROM users").WillReturnRows(sqlmock.NewRows([]string{"client_uid"}).AddRow("owner"))
	mock.ExpectQuery("SELECT value FROM app_meta").WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow("corrupt"))
	mock.ExpectRollback()
	if err := (&Bot{DB: database}).applyAbyssLootGrant("owner", abyssLootGrant{Type: "skill", Skill: &skill}); err == nil {
		t.Fatal("failed grant would be consumed or auctioned")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
