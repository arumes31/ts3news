package bot

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"ts3news/internal/content"
)

func TestAbyssPetGearIsSeparatedFromPlayerEquipment(t *testing.T) {
	equipped := map[content.GearSlot]content.Gear{
		content.SlotHead: {Name: "Helm", Stats: content.Stats{DEF: 5}},
		content.SlotPet1: {Name: "Collar", Stats: content.Stats{STR: 7}},
		content.SlotPet2: {Name: "Charm", Stats: content.Stats{HP: 20, SPD: 3}},
	}
	player := abyssPlayerEquipment(equipped)
	if len(player) != 1 || player[content.SlotHead].Name != "Helm" {
		t.Fatalf("player equipment = %#v", player)
	}
	bonus := abyssPetGearStats(equipped)
	if bonus.STR != 7 || bonus.HP != 20 || bonus.SPD != 3 || bonus.DEF != 0 {
		t.Fatalf("pet gear bonus = %#v", bonus)
	}
}

func TestApplyAbyssPetGearPreservesCurrentHealth(t *testing.T) {
	pet := &content.Mob{Stats: content.Stats{HP: 40, STR: 10, DEF: 4}, MaxHP: 100}
	applyAbyssPetGear(pet, content.Stats{HP: 25, STR: 6, DEF: 2, SPD: 3})
	if pet.Stats.HP != 40 || pet.MaxHP != 125 || pet.Stats.STR != 16 || pet.Stats.DEF != 6 || pet.Stats.SPD != 3 {
		t.Fatalf("pet after gear = %#v", pet)
	}
}

func TestGetPetsAppliesBothPetGearSlots(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	encode := func(gear content.Gear) string {
		data, marshalErr := json.Marshal(gear)
		if marshalErr != nil {
			t.Fatal(marshalErr)
		}
		return string(data)
	}
	mock.ExpectQuery("SELECT p.name,p.mob_type").WithArgs("keeper").WillReturnRows(
		sqlmock.NewRows([]string{"name", "mob_type", "level", "hp", "max_hp", "str", "def", "spd", "loyalty"}).
			AddRow("Mossling", string(content.MobCommon), 5, 40, 100, 10, 4, 8, 100),
	)
	mock.ExpectQuery("SELECT slot, gear_id, item_data FROM user_gear").WithArgs("keeper").WillReturnRows(
		sqlmock.NewRows([]string{"slot", "gear_id", "item_data"}).
			AddRow("Pet1", "TEST_COLLAR", encode(content.Gear{Name: "War Collar", Slot: content.SlotPet1, Stats: content.Stats{STR: 100}})).
			AddRow("Pet2", "TEST_CHARM", encode(content.Gear{Name: "Life Charm", Slot: content.SlotPet2, Stats: content.Stats{HP: 25, DEF: 10}})),
	)
	pets := (&Bot{DB: database}).getPets("keeper")
	if len(pets) != 1 {
		t.Fatalf("pets = %#v", pets)
	}
	pet := pets[0]
	if pet.Stats.HP != 40 || pet.MaxHP != 125 || pet.Stats.STR != 112 || pet.Stats.DEF != 14 {
		t.Fatalf("geared pet = %#v", pet)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAbyssPetGearLabelsBothSlotsAndUIExplainsScope(t *testing.T) {
	equipped := map[content.GearSlot]content.Gear{
		content.SlotPet1: {Name: "War Collar", Stats: content.Stats{STR: 7}},
		content.SlotPet2: {Name: "Fleet Charm", Stats: content.Stats{SPD: 4}},
	}
	label := abyssPetEquipmentLabel(equipped)
	for _, token := range []string{"Collar: War Collar", "+7 STR", "Charm: Fleet Charm", "+4 SPD"} {
		if !strings.Contains(label, token) {
			t.Errorf("pet gear label %q is missing %q", label, token)
		}
	}
	page, err := webAssets.ReadFile("webassets/abyss_social.html")
	if err != nil {
		t.Fatal(err)
	}
	for _, token := range []string{"Collar and charm stats empower active pets only", "Pet-only equipment"} {
		if !strings.Contains(string(page), token) {
			t.Errorf("companion UI is missing %q", token)
		}
	}
}
