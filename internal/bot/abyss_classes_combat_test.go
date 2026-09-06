package bot

import (
	"testing"
	"ts3news/internal/content"
)

func TestAbyssClassSequences(t *testing.T) {
	for _, class := range content.AbyssClasses() {
		for _, sub := range class.Subclasses {
			t.Run(sub.ID, func(t *testing.T) {
				u := &UserInCombat{AbyssClass: class.ID, AbyssSubclass: sub.ID, Stats: content.Stats{STR: 100, INT: 100, DEF: 100, HP: 1000}, CurrentHP: 700, STRMod: 1}
				au := &activeUser{u: u, skillCooldowns: map[string]int{"other": 3}}
				mob := &content.Mob{Stats: content.Stats{HP: 500}, MaxHP: 1000}
				skills := content.AbyssClassSkills(sub.ID)
				logs := []string{}
				before := previewAbyssClassSkill(au, skills[1], mob)
				for i := 0; i < 5; i++ {
					resolveAbyssClassCast(au, skills[0], mob, 1, &logs)
				}
				if au.classResource != 3 {
					t.Fatal("resource did not cap", au.classResource)
				}
				after := resolveAbyssClassCast(au, skills[1], mob, 2, &logs)
				if after.Power <= before.Power || au.classResource != 0 {
					t.Fatal("payoff failed", after, before)
				}
				if au.shield > 500 || u.CurrentHP < 1 {
					t.Fatal("unbounded defense or self kill")
				}
				if sub.ID == "chronomancer" && au.skillCooldowns["other"] != 2 {
					t.Fatal("cooldown not recovered")
				}
			})
		}
	}
}
func TestAbyssSkillScalingUsesDeclaredStat(t *testing.T) {
	u := &UserInCombat{Stats: content.Stats{STR: 20, INT: 200, DEF: 70}, STRMod: 1}
	if abyssSkillBase(u, content.Skill{ScalingStat: "INT"}) != 200 {
		t.Fatal("magic scales from STR")
	}
	if abyssSkillBase(u, content.Skill{ScalingStat: "STR"}) != 20 {
		t.Fatal("physical scales from INT")
	}
	if abyssSkillBase(u, content.Skill{ScalingStat: "DEF"}) != 70 {
		t.Fatal("defense scaling ignored")
	}
}
func TestAbyssClassPreviewAndMarksArePure(t *testing.T) {
	target := &content.Mob{}
	u := &UserInCombat{AbyssSubclass: "marksman"}
	au := &activeUser{u: u, classResource: 2, classMarkedTarget: target}
	skill := content.AbyssClassSkills("marksman")[1]
	marked := previewAbyssClassSkill(au, skill, target)
	other := previewAbyssClassSkill(au, skill, &content.Mob{})
	if marked.IgnoreDef <= other.IgnoreDef || au.classResource != 2 {
		t.Fatal("mark or pure preview broken")
	}
}
func TestAbyssClassBarrierAndVoidCostBoundaries(t *testing.T) {
	u := &UserInCombat{AbyssSubclass: "voidwalker", Stats: content.Stats{HP: 100, INT: 80}, CurrentHP: 1}
	au := &activeUser{u: u, classResource: 3}
	logs := []string{}
	resolveAbyssClassCast(au, content.AbyssClassSkills("voidwalker")[1], nil, 1, &logs)
	if u.CurrentHP != 1 {
		t.Fatal("self lethal")
	}
	for i := 0; i < 10; i++ {
		grantAbyssClassShield(au, 40)
	}
	if au.shield != 50 {
		t.Fatal("shield cap", au.shield)
	}
	au.shield = 200
	grantAbyssClassShield(au, 10)
	if au.shield != 200 {
		t.Fatal("opening shield shrank")
	}
}

func TestAbyssClassManualActionsResolveInProductionEngine(t *testing.T) {
	for _, class := range content.AbyssClasses() {
		for _, sub := range class.Subclasses {
			t.Run(sub.ID, func(t *testing.T) {
				live := &abyssLiveCombat{round: 1}
				skills := content.AbyssClassSkills(sub.ID)
				user := &UserInCombat{UID: "owner", Nickname: "Delver", AbyssClass: class.ID, AbyssSubclass: sub.ID, Skills: skills, shadow: true, EscrowLoot: true, live: live, CurrentHP: 500, Stats: content.Stats{HP: 1000, STR: 100, INT: 100, DEF: 100, SPD: 10, MNA: 200}, STRMod: 1, DEFMod: 1, SPDMod: 1}
				users := []activeUser{{u: user, CurrentMana: 100, MaxMana: 100, skillCooldowns: map[string]int{}}}
				mob := &content.Mob{Name: "Target", Stats: content.Stats{HP: 10000, SPD: 10}, MaxHP: 10000, DEFMod: 1, SPDMod: 1}
				mobs := []*content.Mob{mob}
				logs := []string{}
				loot := []LootResult{}
				dealt, taken := 0, 0
				for index, skill := range skills {
					targetID := "enemy:0"
					if skill.TargetMode == content.SkillTargetSelf || skill.TargetMode == content.SkillTargetAlly {
						targetID = "ally:owner"
					}
					action := abyssLiveAction{Kind: "skill", AbilityID: skill.ID, TargetID: targetID, Round: index + 1}
					(&Bot{}).userTurn(users, &mobs, content.Zone{}, 1, 1, &logs, &dealt, &taken, 1, 1, nil, &loot, index+1, nil, map[string]abyssLiveAction{"owner": action}, false, fixedCombatRandom{float: .9, intn: 99})
					if index == 0 && users[0].classResource != 1 {
						t.Fatal("manual builder did not prime", logs)
					}
				}
				if users[0].classResource != 0 || mob.Stats.HP >= 10000 {
					t.Fatal("manual payoff did not resolve", logs)
				}
				if len(live.presentationEvents) < 2 {
					t.Fatal("confirmed casts not presented")
				}
				if user.CurrentHP < 1 || user.CurrentHP > 1000 || users[0].CurrentMana < 0 {
					t.Fatal("invalid health or mana")
				}
			})
		}
	}
}
func TestAbyssClassAutoRecommendationUsesResourceAndMarkedTarget(t *testing.T) {
	combat := &abyssLiveCombat{round: 2, allies: []abyssLiveCombatantView{{ID: "ally:owner", HP: 100, MaxHP: 100, Mana: 50}}, enemies: []abyssLiveCombatantView{{ID: "enemy:0", HP: 100, MaxHP: 100}, {ID: "enemy:1", HP: 10, MaxHP: 100}}, options: map[string][]abyssLiveOption{"owner": {
		{Kind: "ultimate", ID: "power", Power: 100, Target: "enemy"},
		{Kind: "skill", ID: "CLASS_marksman_finish", Power: 2, Target: "enemy", Mana: 30, BuildPriority: 90, BuildReason: "Spend Focus", PreferredTarget: "enemy:0"},
	}}}
	action := combat.bestActionLocked("owner")
	if action.AbilityID != "CLASS_marksman_finish" || action.TargetID != "enemy:0" {
		t.Fatal("auto discarded the marked payoff", action)
	}
	combat.options["owner"][1].Cooldown = 1
	if action := combat.bestActionLocked("owner"); action.AbilityID != "power" {
		t.Fatal("auto cast cooling finisher", action)
	}
}
