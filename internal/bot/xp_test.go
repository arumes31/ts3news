package bot

import (
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"ts3news/internal/config"
	"ts3news/internal/content"
)

func mockUserState(mock sqlmock.Sqlmock, uid string) {
	// activeLootMult calls:
	// 1. Title
	mock.ExpectQuery(`SELECT title, title_mult, title_expires FROM users WHERE client_uid=\$1`).
		WithArgs(uid).
		WillReturnRows(sqlmock.NewRows([]string{"title", "title_mult", "title_expires"}).AddRow(nil, 1.0, nil))
	// 2. Artifact
	mock.ExpectQuery(`SELECT artifact_mult, artifact_name, artifact_durability FROM users WHERE client_uid=\$1`).
		WithArgs(uid).
		WillReturnRows(sqlmock.NewRows([]string{"artifact_mult", "artifact_name", "artifact_durability"}).AddRow(1.0, nil, 0))
	// 3. Gear
	mock.ExpectQuery(`SELECT gear_id, durability, enchantment_id FROM user_gear WHERE client_uid = \$1`).
		WithArgs(uid).
		WillReturnRows(sqlmock.NewRows([]string{"gear_id", "durability", "enchantment_id"}))
	// 4. Skills
	mock.ExpectQuery(`SELECT skill_id FROM user_skills WHERE client_uid = \$1`).
		WithArgs(uid).
		WillReturnRows(sqlmock.NewRows([]string{"skill_id"}))
	// 5. Active ultimates
	mock.ExpectQuery(`SELECT ultimate_id, current_cooldown FROM user_ultimate_skills WHERE client_uid=\$1 AND active`).
		WithArgs(uid, maxActiveUltimates).
		WillReturnRows(sqlmock.NewRows([]string{"ultimate_id", "current_cooldown"}))
}

func TestComputeMiscMult(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer func() { _ = db.Close() }()
	b := &Bot{Cfg: &config.Config{EnableXPModifiers: true}, DB: db}

	ctx := cycleContext{
		onlineNormal:       1,
		channelNormalCount: map[int]int{1: 1},
		today:              time.Now(),
	}

	// Mock updateStreak
	mock.ExpectQuery(`SELECT last_poke_date, streak_days FROM users`).
		WithArgs("user1").
		WillReturnRows(sqlmock.NewRows([]string{"last_poke_date", "streak_days"}).AddRow(time.Now(), 1))

	m := b.computeMiscMult("user1", "Hero", 1, ctx)
	if m < 1.0 {
		t.Errorf("expected multiplier >= 1.0, got %f", m)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %s", err)
	}
}

func TestAwardXP(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer func() { _ = db.Close() }()
	b := &Bot{Cfg: &config.Config{}, DB: db}

	uid := "user1"
	nickname := "Hero"

	// Existing user
	mock.ExpectQuery(`SELECT xp, level FROM users`).
		WithArgs(uid).
		WillReturnRows(sqlmock.NewRows([]string{"xp", "level"}).AddRow(100, 5))
	mock.ExpectExec(`INSERT INTO users`).
		WithArgs(uid, nickname, 150, 8). // 100 + 50 = 150 XP which is level 8
		WillReturnResult(sqlmock.NewResult(1, 1))

	lr, err := b.awardXP(uid, nickname, 50)
	if err != nil || lr.NewLevel != 8 || lr.TotalXP != 150 {
		t.Errorf("awardXP failed: %v, %+v", err, lr)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %s", err)
	}
}

func TestUpdateStreak(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer func() { _ = db.Close() }()
	b := &Bot{Cfg: &config.Config{}, DB: db}

	uid := "user1"
	today := time.Now()
	yesterday := today.AddDate(0, 0, -1)

	// 1. New streak
	mock.ExpectQuery(`SELECT last_poke_date, streak_days FROM users`).
		WithArgs(uid).
		WillReturnRows(sqlmock.NewRows([]string{"last_poke_date", "streak_days"}).AddRow(yesterday, 5))
	mock.ExpectExec(`UPDATE users SET streak_days=\$2, last_poke_date=\$3 WHERE client_uid=\$1`).
		WithArgs(uid, 6, today).
		WillReturnResult(sqlmock.NewResult(1, 1))

	s := b.updateStreak(uid, today)
	if s != 6 {
		t.Errorf("streak = %d, want 6", s)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %s", err)
	}
}

func TestResolveChannelCombat_Comprehensive(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	b := &Bot{
		Cfg: &config.Config{EnableXPModifiers: true},
		DB:  db,
	}

	zone := content.Zone{Name: "Test Zone", Difficulty: 1.0}

	t.Run("Solo Victory", func(t *testing.T) {
		users := []UserInCombat{
			{
				UID:      "user1",
				Nickname: "Hero",
				Level:    10,
				// Overwhelming bulk so the hero deterministically survives every
				// randomly-spawned follow-up wave (combat uses unseeded RNG); the
				// test's intent is "strong hero beats trivial content".
				Stats:     content.Stats{HP: 1000000, STR: 1000000, DEF: 100000, SPD: 50},
				CurrentHP: 1000000,
			},
		}
		mobs := []*content.Mob{
			{
				Name: "Weak Mob",
				Level: 1,
				Stats: content.Stats{HP: 10, STR: 5, DEF: 5, SPD: 5},
				RewardXP: 50,
			},
		}

		// initializeCombat
		mockUserState(mock, "user1")
		mock.ExpectQuery(`SELECT cons_id, remaining_fights FROM user_consumables WHERE client_uid = \$1`).
			WithArgs("user1").
			WillReturnRows(sqlmock.NewRows([]string{"cons_id", "remaining_fights"}))

		// userTurn: SELECT title
		mock.ExpectQuery(`SELECT title FROM users WHERE client_uid=\$1`).
			WithArgs("user1").
			WillReturnRows(sqlmock.NewRows([]string{"title"}).AddRow(nil))
		// userTurn: SELECT gear_id (Mind Control check)
		mock.ExpectQuery(`SELECT gear_id FROM user_gear WHERE client_uid = \$1`).
			WithArgs("user1").
			WillReturnRows(sqlmock.NewRows([]string{"gear_id"}))

		// distributeRewards
		// updateQuest
		mock.ExpectExec("INSERT INTO user_quests").WillReturnResult(sqlmock.NewResult(1, 1))
		// consecutive_losses = 0
		mock.ExpectExec(`UPDATE users SET consecutive_losses = 0 WHERE client_uid = \$1`).
			WithArgs("user1").
			WillReturnResult(sqlmock.NewResult(1, 1))
		// Regen stacks check
		mockUserState(mock, "user1")
		// Gold drop - inflation check
		mock.ExpectQuery(`SELECT SUM\(gold\) FROM users`).
			WillReturnRows(sqlmock.NewRows([]string{"sum"}).AddRow(0))
		// First Win of the Day
		mock.ExpectQuery(`SELECT last_win FROM users WHERE client_uid=\$1`).
			WithArgs("user1").
			WillReturnRows(sqlmock.NewRows([]string{"last_win"}).AddRow(nil))
		mock.ExpectExec(`UPDATE users SET last_win=NOW\(\) WHERE client_uid=\$1`).
			WithArgs("user1").
			WillReturnResult(sqlmock.NewResult(1, 1))
		// VIP Gold Bonus
		mock.ExpectQuery(`SELECT vip_points FROM users WHERE client_uid=\$1`).
			WithArgs("user1").
			WillReturnRows(sqlmock.NewRows([]string{"vip_points"}).AddRow(0))
		// Update persistent state
		mock.ExpectExec(`UPDATE users SET current_hp = \$2, regen_stacks = \$3, gold = users.gold \+ \$4 WHERE client_uid = \$1`).
			WillReturnResult(sqlmock.NewResult(1, 1))
		// Consumables update
		mock.ExpectExec(`UPDATE user_consumables SET remaining_fights = remaining_fights - 1 WHERE client_uid = \$1`).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectExec(`DELETE FROM user_consumables WHERE client_uid = \$1 AND remaining_fights < 0`).
			WillReturnResult(sqlmock.NewResult(1, 1))


		logs, xp, victory, loots := b.resolveChannelCombat(users, mobs, 10, 1.0, zone)
		_ = loots

		if !victory {
			t.Errorf("expected victory")
		}
		if xp <= 0 {
			t.Errorf("expected positive XP, got %d", xp)
		}
		if len(logs) == 0 {
			t.Errorf("expected logs")
		}
	})

	t.Run("Solo Defeat", func(t *testing.T) {
		users := []UserInCombat{
			{
				UID:      "user2",
				Nickname: "Weakling",
				Level:    1,
				Stats:    content.Stats{HP: 1, STR: 1, DEF: 1, SPD: 1},
				CurrentHP: 1,
			},
		}
		mobs := []*content.Mob{
			{
				Name: "Strong Mob",
				Level: 50,
				Stats: content.Stats{HP: 1000, STR: 100, DEF: 100, SPD: 100},
				RewardXP: 1000,
			},
		}

		// initializeCombat
		mockUserState(mock, "user2")
		mock.ExpectQuery(`SELECT cons_id, remaining_fights FROM user_consumables WHERE client_uid = \$1`).
			WithArgs("user2").
			WillReturnRows(sqlmock.NewRows([]string{"cons_id", "remaining_fights"}))

		// Combat happens... user dies.
		// checkUserRevive: 1. getConsumables
		mock.ExpectQuery(`SELECT cons_id, remaining_fights FROM user_consumables WHERE client_uid = \$1`).
			WithArgs("user2").
			WillReturnRows(sqlmock.NewRows([]string{"cons_id", "remaining_fights"}))
		// checkUserRevive: 2. activeLootMult
		mockUserState(mock, "user2")

		// distributeRewards
		mock.ExpectExec(`UPDATE users SET consecutive_losses = consecutive_losses \+ 1 WHERE client_uid = \$1`).
			WillReturnResult(sqlmock.NewResult(1, 1))
		// Death penalty: SELECT xp
		mock.ExpectQuery(`SELECT xp FROM users WHERE client_uid=\$1`).
			WithArgs("user2").
			WillReturnRows(sqlmock.NewRows([]string{"xp"}).AddRow(1000))
		// Update persistent state
		mock.ExpectExec(`UPDATE users SET current_hp = \$2, regen_stacks = \$3, gold = users.gold \+ \$4 WHERE client_uid = \$1`).
			WillReturnResult(sqlmock.NewResult(1, 1))
		// Consumables update
		mock.ExpectExec(`UPDATE user_consumables SET remaining_fights = remaining_fights - 1 WHERE client_uid = \$1`).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectExec(`DELETE FROM user_consumables WHERE client_uid = \$1 AND remaining_fights < 0`).
			WillReturnResult(sqlmock.NewResult(1, 1))
		
		// awardXP (penalty)
		mock.ExpectQuery(`SELECT xp, level FROM users WHERE client_uid = \$1`).
			WithArgs("user2").
			WillReturnRows(sqlmock.NewRows([]string{"xp", "level"}).AddRow(1000, 1))
		mock.ExpectExec(`UPDATE users SET xp = \$2, level = \$3, last_seen = NOW\(\) WHERE client_uid = \$1`).WillReturnResult(sqlmock.NewResult(1, 1))

		_, xp, victory, loots := b.resolveChannelCombat(users, mobs, 5, 1.0, zone)
		_ = loots

		if victory {
			t.Errorf("expected defeat")
		}
		// Per-kill model: a lost fight banks 25% of the killed mobs' XP. This mob
		// is never killed, so killedXP is 0 and the combat reward is 0 (the level
		// death penalty is applied separately inside distributeRewards).
		if xp != 0 {
			t.Errorf("expected 0 combat XP on a defeat with no kills, got %d", xp)
		}
	})
}
