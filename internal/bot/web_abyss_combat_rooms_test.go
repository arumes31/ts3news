package bot

import (
	"slices"
	"strings"
	"testing"
	"time"

	"ts3news/internal/content"
)

func TestAbyssMirrorCloneUsesCurrentLoadout(t *testing.T) {
	skill := content.Skill{ID: "current-skill", Name: "Current Skill", Power: 2.5}
	gear := content.Gear{ID: "current-weapon", Name: "Current Weapon", Slot: content.SlotMainHand}
	stats := content.Stats{HP: 987, STR: 123, DEF: 45, SPD: 67, MNA: 89}
	user := UserInCombat{
		Nickname: "Delver",
		Level:    42,
		Stats:    stats,
		Skills:   []content.Skill{skill},
		Equipped: map[content.GearSlot]content.Gear{content.SlotMainHand: gear},
	}

	clone := abyssMirrorClone(user)
	if clone.Name != "Mirror of Delver" || clone.Level != user.Level || clone.Stats != stats {
		t.Fatalf("clone identity/loadout = %+v, want current player state", clone)
	}
	if clone.MaxHP != stats.HP || clone.CurrentHP != stats.HP {
		t.Fatalf("clone HP = %d/%d, want %d/%d", clone.CurrentHP, clone.MaxHP, stats.HP, stats.HP)
	}
	if len(clone.Spells) != 1 || clone.Spells[0].ID != skill.ID {
		t.Fatalf("clone spells = %+v, want current active skill", clone.Spells)
	}
	if len(clone.Equipped) != 1 || clone.Equipped[0].ID != gear.ID {
		t.Fatalf("clone gear = %+v, want current equipped gear", clone.Equipped)
	}
	clone.Spells[0].Name = "Changed"
	if user.Skills[0].Name != skill.Name {
		t.Fatal("mirror clone aliases the player's skill slice")
	}
}

func TestAbyssStormTelegraphsAndDamagesSelectedSide(t *testing.T) {
	if got := nextAbyssStormSide(fixedCombatRandom{intn: 0}); got != abyssStormParty {
		t.Fatalf("storm side = %q, want party", got)
	}
	if got := nextAbyssStormSide(fixedCombatRandom{intn: 1}); got != abyssStormEnemies {
		t.Fatalf("storm side = %q, want enemies", got)
	}
	if warning := abyssStormTelegraph(abyssStormParty); !strings.Contains(warning, "next round") || !strings.Contains(warning, "PARTY") {
		t.Fatalf("storm warning = %q, want side and timing", warning)
	}

	user := UserInCombat{CurrentHP: 100, Stats: content.Stats{HP: 100}}
	mob := content.Mob{Stats: content.Stats{HP: 200}, MaxHP: 200}
	partyDamage, enemyDamage := strikeAbyssStorm(
		abyssStormParty,
		[]activeUser{{u: &user}},
		[]*content.Mob{&mob},
	)
	if partyDamage != 5 || enemyDamage != 0 || user.CurrentHP != 95 || user.DamageTaken != 5 || mob.Stats.HP != 200 {
		t.Fatalf("party strike = damage %d/%d, user %+v, mob HP %d", partyDamage, enemyDamage, user, mob.Stats.HP)
	}
	partyDamage, enemyDamage = strikeAbyssStorm(
		abyssStormEnemies,
		[]activeUser{{u: &user}},
		[]*content.Mob{&mob},
	)
	if partyDamage != 0 || enemyDamage != 10 || mob.Stats.HP != 190 {
		t.Fatalf("enemy strike = damage %d/%d, mob HP %d", partyDamage, enemyDamage, mob.Stats.HP)
	}
}

func TestAbyssDarknessRedactsSerializedHealthWithoutMutatingCombat(t *testing.T) {
	originalViews := []abyssLiveCombatantView{{ID: "enemy:0", HP: 75, MaxHP: 100}}
	views := append([]abyssLiveCombatantView(nil), originalViews...)
	concealAbyssEnemyViews(views)
	if views[0].HP != -1 || views[0].MaxHP != -1 || !views[0].HPHidden {
		t.Fatalf("concealed view = %+v", views[0])
	}
	if originalViews[0].HP != 75 || originalViews[0].MaxHP != 100 {
		t.Fatalf("source combat view mutated = %+v", originalViews[0])
	}

	originalTimeline := []combatTimelineFrame{{EnemyHP: 75, EnemyMax: 100, HP: 90, MaxHP: 100}}
	concealed := concealAbyssTimeline(originalTimeline)
	if concealed[0].EnemyHP != -1 || concealed[0].EnemyMax != -1 || concealed[0].HP != 90 {
		t.Fatalf("concealed timeline = %+v", concealed[0])
	}
	if originalTimeline[0].EnemyHP != 75 || originalTimeline[0].EnemyMax != 100 {
		t.Fatalf("source timeline mutated = %+v", originalTimeline[0])
	}

	combat := &abyssLiveCombat{
		modifier:     "darkness",
		warning:      abyssEncounterWarning("darkness"),
		telegraph:    "authoritative warning",
		phase:        "resolving",
		participants: map[string]bool{"user": true},
		tactics:      map[string]string{"user": "balanced"},
		policies:     map[string]abyssLivePolicy{},
		ready:        map[string]bool{},
		queued:       map[string]abyssLiveAction{},
		options:      map[string][]abyssLiveOption{},
		timeBank:     map[string]time.Duration{},
		enemyPlans:   map[int]abyssLiveEnemyPlan{},
		enemies:      append([]abyssLiveCombatantView(nil), originalViews...),
	}
	snapshot := combat.snapshotForLocked("user")
	if snapshot.Enemies[0].HP != -1 || !snapshot.Enemies[0].HPHidden {
		t.Fatalf("darkness snapshot leaked enemy health: %+v", snapshot.Enemies[0])
	}
	if combat.enemies[0].HP != 75 {
		t.Fatalf("snapshot redaction mutated combat state: %+v", combat.enemies[0])
	}
	if snapshot.Warning == "" || snapshot.Telegraph != "authoritative warning" {
		t.Fatalf("snapshot warnings = %q / %q", snapshot.Warning, snapshot.Telegraph)
	}
}

func TestAbyssCombatRoomModifierRegistry(t *testing.T) {
	for _, modifier := range []string{"mirror_clone", "storm_floor", "darkness"} {
		if !slices.Contains(abyssCombatFloorModifiers, modifier) {
			t.Errorf("combat modifier registry missing %q", modifier)
		}
		if abyssEncounterWarning(modifier) == "" {
			t.Errorf("combat modifier %q has no warning", modifier)
		}
	}
	if hasAbyssFloorModifier("storm_flooring", "storm_floor") {
		t.Fatal("modifier matching accepted a partial token")
	}
}
