package bot

import (
	"database/sql/driver"
	"encoding/json"
	"github.com/DATA-DOG/go-sqlmock"
	"testing"
	"ts3news/internal/content"
)

func TestPetRegenerationCannotExceedMaximum(t *testing.T) {
	pet := &content.Mob{Level: 100, MaxHP: 100, Stats: content.Stats{HP: 90}}
	user := &UserInCombat{UID: "keeper", CurrentHP: 100, Stats: content.Stats{HP: 100}, Pets: []*content.Mob{pet}}
	var logs []string
	for round := 1; round <= 1000; round++ {
		(&Bot{}).applyEffects([]activeUser{{u: user}}, nil, content.Zone{}, round, 1, 1, &logs)
	}
	if pet.Stats.HP != 100 {
		t.Fatalf("pet HP=%d, want maximum100", pet.Stats.HP)
	}
}

func TestPetHealingReportsActualRestoredHealth(t *testing.T) {
	pet := &content.Mob{MaxHP: 100, Stats: content.Stats{HP: 97}}
	if got := healAbyssPet(pet, 200); got != 3 {
		t.Fatalf("healing=%d", got)
	}
	if got := healAbyssPet(pet, 200); got != 0 {
		t.Fatalf("full-health healing=%d", got)
	}
	pet.Stats.HP = 0
	if got := healAbyssPet(pet, 200); got != 0 || pet.Stats.HP != 0 {
		t.Fatal("regen resurrected a dead pet")
	}
}

func TestPetHealthSaveLoadDoesNotRatchetWithGearAndClass(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	bot := &Bot{DB: database}
	for _, hp := range []int{1, 99, 120, 137} {
		exact := hp
		for cycle := 0; cycle < 10; cycle++ {
			pet := &content.Mob{Name: "Tank", Type: content.MobBoss, MaxHP: 100, Stats: content.Stats{HP: min(100, exact)}, Loyalty: 100}
			profile := abyssPetProfile{CombatHealth: &abyssPetHealthState{HP: exact, BaseHP: min(100, exact), BaseMaxHP: 100}}
			encoded, _ := json.Marshal(profile)
			mock.ExpectQuery("SELECT p.name,p.mob_type").WithArgs("keeper").WillReturnRows(sqlmock.NewRows([]string{"name", "mob_type", "level", "hp", "max_hp", "str", "def", "spd", "loyalty", "autoskills"}).AddRow(pet.Name, string(pet.Type), 5, pet.Stats.HP, 100, 10, 4, 8, 100, string(encoded)))
			mock.ExpectQuery("SELECT slot, gear_id, item_data FROM user_gear").WithArgs("keeper").WillReturnRows(sqlmock.NewRows([]string{"slot", "gear_id", "item_data"}).AddRow("Pet1", "TEST_COLLAR", `{"Stats":{"HP":25}}`))
			loaded := bot.getPets("keeper")
			if len(loaded) != 1 || loaded[0].Stats.HP != exact || loaded[0].MaxHP != 137 {
				t.Fatalf("cycle%d hp%d pets=%+v", cycle, hp, loaded)
			}
			mock.ExpectExec("UPDATE user_pets SET hp=LEAST").WithArgs(exact, 100, "keeper", "Tank").WillReturnResult(sqlmock.NewResult(0, 1))
			bot.updatePetState("keeper", loaded[0])
		}
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
func TestPetHealthInvalidatesSnapshotAfterBaseChange(t *testing.T) {
	pet := &content.Mob{MaxHP: 150}
	restoreAbyssPetHealth(pet, 80, 100, &abyssPetHealthState{HP: 120, BaseHP: 100, BaseMaxHP: 100})
	if pet.Stats.HP != 80 {
		t.Fatalf("stale health survived: %d", pet.Stats.HP)
	}
	restoreAbyssPetHealth(pet, 10000, 100, nil)
	if pet.Stats.HP != 100 {
		t.Fatalf("legacy overheal survived: %d", pet.Stats.HP)
	}
	restoreAbyssPetHealth(pet, 100, 100, &abyssPetHealthState{HP: 9999, BaseHP: 100, BaseMaxHP: 100})
	if pet.Stats.HP != 150 {
		t.Fatalf("effective cap not applied: %d", pet.Stats.HP)
	}
}

type petProfileWithoutCombatHealth struct{}

func (petProfileWithoutCombatHealth) Match(value driver.Value) bool {
	raw, ok := value.(string)
	if !ok {
		return false
	}
	var profile abyssPetProfile
	return json.Unmarshal([]byte(raw), &profile) == nil && profile.CombatHealth == nil
}
