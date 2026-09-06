package bot

import (
	"encoding/json"
	"strings"
	"testing"

	"ts3news/internal/content"
)

func TestAbyssPresentationNilCombatIsNoop(t *testing.T) {
	var live *abyssLiveCombat
	mob := &content.Mob{}
	live.present(1, "attack", "actor", "basic_attack", "Attack", content.ElementPhysical)
	if got := live.presentationEntity(mob, "enemy"); got != "" {
		t.Fatalf("nil live identity = %q", got)
	}
	live.presentMobDamage(1, "attack", "actor", "attack", "Attack", content.ElementPhysical, mob, 10, 10)
	live.presentPhase(1, mob, "enrage", "Enrage")
}

func TestAbyssPresentationWeaponUsesEquippedIdentity(t *testing.T) {
	for _, test := range []struct{ name, want string }{
		{"Lifebloom Staff", "staff"}, {"Stormcaller Longbow", "bow"},
		{"Wandering Sword", "sword"}, {"The Wanderer", ""},
	} {
		user := &UserInCombat{Equipped: map[content.GearSlot]content.Gear{content.SlotMainHand: {Name: test.name}}}
		kind, name := abyssPresentationWeapon(user)
		if kind != test.want || name != test.name {
			t.Fatalf("weapon %q = %q/%q", test.name, kind, name)
		}
	}
	user := &UserInCombat{Position: content.PositionBackline, Equipped: map[content.GearSlot]content.Gear{content.SlotRanged: {Name: "Whisper of the Dark"}}}
	if kind, _ := abyssPresentationWeapon(user); kind != "ranged" {
		t.Fatalf("backline ranged slot = %q", kind)
	}
}

func TestAbyssPresentationStableIdentityAndBoundedHistory(t *testing.T) {
	live := &abyssLiveCombat{}
	a, b := &content.Mob{}, &content.Mob{}
	aID := live.presentationEntity(a, "enemy")
	bID := live.presentationEntity(b, "enemy")
	if aID == bID || live.presentationEntity(b, "enemy") != bID {
		t.Fatal("enemy identity changed or collided")
	}
	if live.presentationEntity(a, "pet:owner") == aID {
		t.Fatal("captured ally retained hostile identity")
	}
	for i := 0; i < abyssLivePresentationLimit+10; i++ {
		live.present(1, "attack", "ally:owner", "basic_attack", "Attack", content.ElementPhysical, abyssLivePresentationOutcome{TargetID: bID, Damage: 1})
	}
	if len(live.presentationEvents) != abyssLivePresentationLimit || live.presentationEvents[0].Seq != 11 || live.presentationCursor != abyssLivePresentationLimit+10 {
		t.Fatal("presentation history is not bounded with monotonic IDs")
	}
}

func TestAbyssPresentationConcealAndClone(t *testing.T) {
	live := &abyssLiveCombat{modifier: "darkness"}
	live.present(1, "skill", "ally:owner", "fireball", "Fireball", content.ElementFire, abyssLivePresentationOutcome{TargetID: "enemy:0", Damage: 173, Healing: 19, Absorbed: 21, Critical: true})
	snapshot := live.snapshotFor("owner")
	data, err := json.Marshal(snapshot.PresentationEvents)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), `"damage":`) || strings.Contains(string(data), `"healing":`) || strings.Contains(string(data), `"absorbed":`) {
		t.Fatalf("concealed outcome leaked numeric HP: %s", data)
	}
	if !snapshot.PresentationEvents[0].Targets[0].Damaged || !snapshot.PresentationEvents[0].Targets[0].Healed {
		t.Fatalf("concealed projection discarded outcome types: %s", data)
	}
	clone := cloneAbyssLiveSnapshot(snapshot)
	clone.PresentationEvents[0].Targets[0].Status = "changed"
	if snapshot.PresentationEvents[0].Targets[0].Status != "" || live.presentationEvents[0].Targets[0].Damage != 173 {
		t.Fatal("snapshot mutation changed retained authoritative event")
	}
}

func TestAbyssPresentationInitialSnapshotUsesEmptyArray(t *testing.T) {
	live := &abyssLiveCombat{}
	data, err := json.Marshal(live.snapshotFor("owner"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"presentation_events":[]`) || !strings.Contains(string(data), `"presentation_cursor":0`) {
		t.Fatalf("initial snapshot does not identify structured presentation: %s", data)
	}
}

func TestAbyssEmptyManaIsExplicitInSnapshot(t *testing.T) {
	live := &abyssLiveCombat{allies: []abyssLiveCombatantView{{ID: "ally:owner", MaxMana: 100}}}
	data, err := json.Marshal(live.snapshotFor("owner"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"mana":0`) {
		t.Fatalf("empty mana must reach the HUD: %s", data)
	}
}

func TestAbyssManaRecoveryAndCastStaySeparate(t *testing.T) {
	live := &abyssLiveCombat{round: 2}
	user := &UserInCombat{UID: "owner", Nickname: "Delver", shadow: true, live: live,
		CurrentHP: 100, Stats: content.Stats{HP: 100, MNA: 1000, STR: 10, SPD: 10}, STRMod: 1, DEFMod: 1, SPDMod: 1,
		Skills: []content.Skill{{ID: "bolt", Name: "Bolt", Power: 1, ManaCost: 20}},
	}
	users := []activeUser{{u: user, CurrentMana: 80, MaxMana: 100, skillCooldowns: map[string]int{}}}
	mobs := []*content.Mob{{Name: "Target", Stats: content.Stats{HP: 10000, SPD: 10}, MaxHP: 10000, DEFMod: 1, SPDMod: 1}}
	var logs []string
	var loot []LootResult
	dealt, taken := 0, 0
	(&Bot{}).userTurn(users, &mobs, content.Zone{}, 1, 1, &logs, &dealt, &taken, 1, 1, nil, &loot, 2, nil,
		map[string]abyssLiveAction{"owner": {Kind: "skill", AbilityID: "bolt", TargetID: "enemy:0", Round: 2}}, false, fixedCombatRandom{float: .9, intn: 99})
	if len(live.presentationEvents) != 2 {
		t.Fatalf("expected capped recovery then cast, got %+v", live.presentationEvents)
	}
	recovery, cast := live.presentationEvents[0], live.presentationEvents[1]
	if recovery.Kind != "mana" || recovery.Mana == nil || *recovery.Mana != (combatManaChange{80, 100, 100}) {
		t.Fatalf("recovery must report only actual mana gained: %+v", recovery)
	}
	if cast.Kind != "skill" || cast.Mana == nil || *cast.Mana != (combatManaChange{100, 80, 100}) || users[0].CurrentMana != 80 {
		t.Fatalf("net-zero turn lost its spend: %+v, mana %d", cast, users[0].CurrentMana)
	}
	clone := cloneAbyssPresentationEvents(live.presentationEvents)
	clone[0].Mana.After = 0
	if recovery.Mana.After != 100 {
		t.Fatal("cloned mana event mutated retained history")
	}
}

func TestAbyssPresentationPlayerResolution(t *testing.T) {
	for _, kind := range []string{"attack", "critical", "skill", "heal", "ultimate", "defend"} {
		t.Run(kind, func(t *testing.T) {
			live := &abyssLiveCombat{round: 2}
			user := &UserInCombat{UID: "owner", Nickname: "Delver", shadow: true, live: live,
				CurrentHP: 90, Stats: content.Stats{HP: 100, STR: 100, SPD: 10}, STRMod: 1, DEFMod: 1, SPDMod: 1,
				Equipped: map[content.GearSlot]content.Gear{content.SlotMainHand: {Element: content.ElementFire}},
			}
			action := abyssLiveAction{Kind: kind, TargetID: "enemy:0", Round: 2}
			randomInt := 99
			if kind == "critical" {
				user.EscrowLoot, user.Stats.CRT, action.Kind, randomInt = true, 50, "attack", 0
			}
			if kind == "skill" || kind == "heal" {
				user.Skills = []content.Skill{{ID: "test_spell", Name: "Test Spell", Power: 2, ManaCost: 10}}
				action.Kind, action.AbilityID = "skill", "test_spell"
				if kind == "heal" {
					user.Skills[0].Power, user.Skills[0].HealPercent = 0, .5
					action.TargetID = "ally:owner"
				}
			}
			if kind == "ultimate" {
				user.Ultimates = []*content.UltimateSkill{{ID: "test_ultimate", Name: "Test Ultimate", Power: 3, CooldownRounds: 4}}
				action.AbilityID = "test_ultimate"
			}
			users := []activeUser{{u: user, CurrentMana: 100, MaxMana: 100, skillCooldowns: map[string]int{}}}
			mob := &content.Mob{Name: "Target", Stats: content.Stats{HP: 10000, SPD: 10}, MaxHP: 10000, DEFMod: 1, SPDMod: 1}
			mobs := []*content.Mob{mob}
			var logs []string
			var loot []LootResult
			dealt, taken := 0, 0
			(&Bot{}).userTurn(users, &mobs, content.Zone{}, 1, 1, &logs, &dealt, &taken, 1, 1, nil, &loot, 2, nil,
				map[string]abyssLiveAction{"owner": action}, false, fixedCombatRandom{float: .9, intn: randomInt})
			if len(live.presentationEvents) != 1 {
				t.Fatalf("resolved %s events = %+v", kind, live.presentationEvents)
			}
			event := live.presentationEvents[0]
			if kind == "skill" || kind == "heal" {
				data, err := json.Marshal(event)
				if err != nil || !strings.Contains(string(data), `"mana":{"before":100,"after":90,"max":100}`) {
					t.Fatalf("cast must preserve the exact mana debit: %s (%v)", data, err)
				}
			}
			if kind == "critical" && !event.Targets[0].Critical {
				t.Fatalf("resolved critical lost its flag: %+v", event)
			}
			if event.Kind != action.Kind || event.ActorID != "ally:owner" {
				t.Fatalf("wrong action identity: %+v", event)
			}
			if kind == "heal" {
				if len(event.Targets) != 1 || event.Targets[0].Healing != 10 || event.Targets[0].Damage != 0 || mob.Stats.HP != 10000 {
					t.Fatalf("heal was not capped actual recovery: %+v", event)
				}
			} else if kind == "defend" {
				if event.Targets[0].Status != "guard" || dealt != 0 {
					t.Fatalf("guard became damage: %+v", event)
				}
			} else if event.Targets[0].Damage != 10000-mob.Stats.HP || event.Targets[0].Damage != dealt || event.Targets[0].TargetID != "enemy:0" {
				t.Fatalf("presentation does not match actual damage: event=%+v damage=%d hp=%d", event, dealt, mob.Stats.HP)
			}
		})
	}
}

func TestAbyssPresentationEnemyShieldAndDodge(t *testing.T) {
	for _, dodge := range []bool{false, true} {
		t.Run(map[bool]string{false: "shield", true: "dodge"}[dodge], func(t *testing.T) {
			live := &abyssLiveCombat{round: 2}
			user := &UserInCombat{UID: "owner", live: live, shadow: true, EscrowLoot: true, CurrentHP: 1000, Stats: content.Stats{HP: 1000}, DEFMod: 1}
			if dodge {
				user.Stats.DGE = 25
			}
			users := []activeUser{{u: user, shield: 200, maxShield: 200}}
			mob := &content.Mob{Name: "Enemy", Stats: content.Stats{HP: 1000, STR: 100, SPD: 10}, STRMod: 1, Element: content.ElementPhysical}
			var logs []string
			dealt, taken := 0, 0
			(&Bot{}).mobTurn(users, []*content.Mob{mob}, content.Zone{}, 1, &logs, &taken, &dealt, 2, false, nil, fixedCombatRandom{float: .9})
			if len(live.presentationEvents) != 1 {
				t.Fatalf("enemy events = %+v", live.presentationEvents)
			}
			outcome := live.presentationEvents[0].Targets[0]
			if user.CurrentHP != 1000 || outcome.Damage != 0 || outcome.Dodged != dodge {
				t.Fatalf("wrong enemy outcome: %+v", outcome)
			}
			if !dodge && (outcome.Absorbed != 100 || !outcome.Blocked || users[0].shield != 100) {
				t.Fatalf("shield outcome not exact: %+v", outcome)
			}
		})
	}
}

func TestAbyssPresentationPoisonAndRegeneration(t *testing.T) {
	live := &abyssLiveCombat{round: 3}
	user := &UserInCombat{UID: "owner", live: live, shadow: true, CurrentHP: 90, Stats: content.Stats{HP: 100}, RegenStacks: 3}
	mob := &content.Mob{Stats: content.Stats{HP: 1000}, Effects: []content.MobEffect{content.EffectPoisoned}}
	var logs []string
	(&Bot{}).applyEffects([]activeUser{{u: user}}, []*content.Mob{mob}, content.Zone{}, 3, 1, 1, &logs)
	if len(live.presentationEvents) != 2 || live.presentationEvents[0].AbilityID != "poison" || live.presentationEvents[0].Targets[0].Damage != 50 || live.presentationEvents[1].Targets[0].Healing != 6 {
		t.Fatalf("tick outcomes do not match resolved effect: %+v", live.presentationEvents)
	}
}

func TestAbyssPresentationConcealedRegenerationKeepsTypeWithoutAmount(t *testing.T) {
	live := &abyssLiveCombat{round: 3, modifier: "darkness"}
	user := &UserInCombat{UID: "owner", live: live, shadow: true, CurrentHP: 100, Stats: content.Stats{HP: 100}}
	mob := &content.Mob{Stats: content.Stats{HP: 500}, MaxHP: 1000, Effects: []content.MobEffect{content.EffectRegen}}
	var logs []string
	(&Bot{}).applyEffects([]activeUser{{u: user}}, []*content.Mob{mob}, content.Zone{}, 3, 1, 1, &logs)
	snapshot := live.snapshotFor("owner")
	if len(snapshot.PresentationEvents) != 1 {
		t.Fatalf("regeneration events = %+v", snapshot.PresentationEvents)
	}
	outcome := snapshot.PresentationEvents[0].Targets[0]
	if !outcome.Healed || outcome.Damaged || outcome.Healing != 0 || outcome.Damage != 0 {
		t.Fatalf("concealed regeneration lost its type or leaked its amount: %+v", outcome)
	}
	data, err := json.Marshal(outcome)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), `"healing":`) || strings.Contains(string(data), `"damage":`) || strings.Contains(string(data), `"hp":`) {
		t.Fatalf("concealed regeneration leaked HP fields: %s", data)
	}
	if mob.Stats.HP != 525 || live.presentationEvents[0].Targets[0].Healing != 25 || live.presentationEvents[0].Targets[0].Healed {
		t.Fatal("public projection changed engine state or retained exact outcome")
	}
}

func TestAbyssPresentationPetAbilities(t *testing.T) {
	for _, healing := range []bool{false, true} {
		t.Run(map[bool]string{false: "pounce", true: "mending"}[healing], func(t *testing.T) {
			live := &abyssLiveCombat{round: 2}
			pet := &content.Mob{Name: "Companion", Loyalty: 100, Stats: content.Stats{HP: 100, STR: 100}, Element: content.ElementWater}
			if healing {
				pet.PetClass = "support"
			}
			user := &UserInCombat{UID: "owner", live: live, shadow: true, CurrentHP: 90, Stats: content.Stats{HP: 100}, Pets: []*content.Mob{pet}}
			users := []activeUser{{u: user}}
			mob := &content.Mob{Name: "Target", Stats: content.Stats{HP: 10000}}
			mobs := []*content.Mob{mob}
			var logs []string
			var loot []LootResult
			dealt, taken := 0, 0
			(&Bot{}).runAbyssPetTurns(&users[0], abyssPetTurnContext{activeUsers: users, mobs: &mobs, intensify: 1, logs: &logs, totalUserDamage: &dealt, totalMobDamage: &taken, loots: &loot, random: fixedCombatRandom{float: .9}})
			if len(live.presentationEvents) != 1 {
				t.Fatalf("pet events = %+v", live.presentationEvents)
			}
			event := live.presentationEvents[0]
			if event.Kind != "pet" || event.ActorID != "pet:owner:0" || event.Element != "Water" {
				t.Fatalf("wrong pet identity: %+v", event)
			}
			if healing && (event.AbilityID != "pet_mending_cry" || event.Targets[0].Healing != 10) {
				t.Fatalf("wrong pet recovery: %+v", event)
			}
			if !healing && (event.AbilityID != "pet_pounce" || event.Targets[0].Damage != dealt || dealt != 150) {
				t.Fatalf("wrong pet damage: %+v dealt=%d", event, dealt)
			}
		})
	}
}

func TestAbyssPresentationTerminalStateAndBossStatus(t *testing.T) {
	live := &abyssLiveCombat{round: 4}
	boss := &content.Mob{Name: "Boss", Stats: content.Stats{HP: 0}, Element: content.ElementFire}
	id := live.presentationEntity(boss, "enemy")
	live.enemies = []abyssLiveCombatantView{{ID: "enemy:0", EntityID: id, HP: 100, MaxHP: 100}}
	live.allies = []abyssLiveCombatantView{{ID: "ally:owner", EntityID: "ally:owner", HP: 100}}
	users := []activeUser{{u: &UserInCombat{UID: "owner", CurrentHP: 37}, Stunned: true, CurrentMana: 21, shield: 12}}
	live.presentPhase(4, boss, "hypnotic_pulse", "Hypnotic Pulse", abyssPresentationStunnedUsers(users)...)
	if event := live.presentationEvents[0]; len(event.Targets) != 2 || event.Targets[1].Status != "stunned" || event.Targets[1].TargetID != "ally:owner" {
		t.Fatalf("boss phase lost actual targets: %+v", event)
	}
	live.capturePresentationFinalState(users, []*content.Mob{boss})
	if live.enemies[0].HP != 0 || live.allies[0].HP != 37 || live.allies[0].Mana != 21 || live.allies[0].Shield != 12 {
		t.Fatal("terminal bars retained last planning state")
	}
	outcome := abyssPresentationDamage(id, 500, 37, -463)
	if outcome.Damage != 37 || !outcome.Defeated {
		t.Fatalf("overkill was fabricated as health loss: %+v", outcome)
	}
}

func TestAbyssPresentationMonsterSpellIdentity(t *testing.T) {
	live := &abyssLiveCombat{round: 2}
	user := &UserInCombat{UID: "owner", live: live, shadow: true, CurrentHP: 1000, Stats: content.Stats{HP: 1000}, DEFMod: 1}
	mob := &content.Mob{Name: "Caster", Stats: content.Stats{HP: 1000, STR: 100, SPD: 10}, STRMod: 1, Element: content.ElementFire,
		Spells: []content.Skill{{ID: "monster_spell", Name: "Flame Pulse", Power: 2}}}
	var logs []string
	dealt, taken := 0, 0
	(&Bot{}).mobTurn([]activeUser{{u: user}}, []*content.Mob{mob}, content.Zone{}, 1, &logs, &taken, &dealt, 2, false, nil, fixedCombatRandom{float: .1})
	if len(live.presentationEvents) != 1 {
		t.Fatalf("spell events = %+v", live.presentationEvents)
	}
	event := live.presentationEvents[0]
	if event.Kind != "skill" || event.AbilityID != "monster_spell" || event.AbilityName != "Flame Pulse" || event.Element != "Fire" || event.Targets[0].Damage != 1000-user.CurrentHP {
		t.Fatalf("monster spell identity/outcome lost: %+v", event)
	}
}
