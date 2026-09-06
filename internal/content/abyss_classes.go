package content

// AbyssClass groups two switchable combat styles. Gear is never class restricted.
type AbyssClass struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	ArtKey      string          `json:"art_key"`
	Description string          `json:"description"`
	Subclasses  []AbyssSubclass `json:"subclasses"`
}

type AbyssSubclass struct {
	ID       string   `json:"id"`
	ClassID  string   `json:"class_id"`
	Name     string   `json:"name"`
	Resource string   `json:"resource"`
	Sequence string   `json:"sequence"`
	Stats    []string `json:"stats"`
	Gear     string   `json:"gear"`
	Buffs    string   `json:"buffs"`
	Builder  string   `json:"builder"`
	Finisher string   `json:"finisher"`
	Scaling  string   `json:"scaling"`
}

// AbyssClasses returns fresh slices so callers cannot mutate the shared catalog.
func AbyssClasses() []AbyssClass {
	return []AbyssClass{
		{"warrior", "Warrior", "class:warrior", "Hold the line or turn pressure into a decisive strike.", []AbyssSubclass{
			{"vanguard", "warrior", "Vanguard", "Resolve", "Brace behind a barrier, then spend Resolve on a shield-breaking retaliation.", []string{"DEF", "STR", "HP"}, "Armor strengthens both your barrier and retaliation; STR supports your other physical skills.", "Defense and max-HP buffs support repeated guard windows.", "Iron Guard", "Resolute Bash", "DEF"},
			{"berserker", "warrior", "Berserker", "Rage", "Build Rage with Rending Strike, then spend it on a stronger execution against wounded enemies.", []string{"STR", "HP", "CRT"}, "Strength weapons power both attacks; HP gives you room to stay aggressive.", "Strength and critical buffs strengthen the payoff; bring healing for long runs.", "Rending Strike", "Rage Execution", "STR"},
		}},
		{"ranger", "Ranger", "class:ranger", "Prepare a precise shot or coordinate with your companions.", []AbyssSubclass{
			{"marksman", "ranger", "Marksman", "Focus", "Mark a target with Sighting Shot, then spend Focus to pierce that target's armor.", []string{"STR", "CRT", "MNA"}, "STR powers shots; critical chance improves burst, mana sustains the sequence.", "Strength and mana recovery support repeated aimed shots.", "Sighting Shot", "Piercing Volley", "STR"},
			{"beastmaster", "ranger", "Beastmaster", "Bond", "Build Bond with Pack Mark, then rally your living companions and strike together.", []string{"STR", "HP", "MNA"}, "Strength powers your own shot even without a companion; living pets add bounded follow-up damage.", "Pet power and defensive buffs keep the pack active.", "Pack Mark", "Pack Assault", "STR"},
		}},
		{"arcanist", "Arcanist", "class:arcanist", "Combine elements or control the timing of your strongest skills.", []AbyssSubclass{
			{"elementalist", "arcanist", "Elementalist", "Attunement", "Prime a target with Ember Seed, then detonate it with Frost Detonation for a two-element reaction.", []string{"INT", "MNA", "HP"}, "Intelligence powers spells; mana supports repeated combinations. Physical weapons remain usable.", "Intelligence and mana buffs improve spell damage and uptime.", "Ember Seed", "Frost Detonation", "INT"},
			{"chronomancer", "arcanist", "Chronomancer", "Tempo", "Store Tempo with Time Bolt, then release it to recover other skill and ultimate cooldowns.", []string{"INT", "MNA", "DEF"}, "Use cooldown skills and ultimates to benefit from recovery; INT powers both signature spells.", "Mana and defense help you reach the next burst window.", "Time Bolt", "Temporal Release", "INT"},
		}},
		{"warden", "Warden", "class:warden", "Turn restoration or durable barriers into reliable solo offense.", []AbyssSubclass{
			{"oracle", "warden", "Oracle", "Grace", "Mend yourself or an ally to store Grace, then release it as a radiant damage burst.", []string{"INT", "HP", "MNA"}, "HP increases healing, INT powers the radiant payoff. Grace works even when healing at full HP.", "Max HP and mana buffs support healing and damage in the same build.", "Mending Light", "Grace Flare", "INT"},
			{"geomancer", "warden", "Geomancer", "Stone", "Raise a Stone Aegis, then turn stored Stone into an armor-piercing quake.", []string{"DEF", "HP", "MNA"}, "Defense directly powers both stone skills. HP gives the barrier a larger safe capacity.", "Defense and mana buffs support repeated barriers and quakes.", "Stone Aegis", "Seismic Break", "DEF"},
		}},
		{"reaver", "Reaver", "class:reaver", "Recover through aggression or deliberately trade health for a void burst.", []AbyssSubclass{
			{"bloodblade", "reaver", "Bloodblade", "Sanguine", "Build Sanguine with Leech Cut, then spend it to heal and deliver a blood-red finisher.", []string{"STR", "HP", "DEF"}, "Strength powers damage; HP increases the signature healing. Defense protects your recovery windows.", "Strength and max-HP buffs improve both halves of your sequence.", "Leech Cut", "Crimson Reap", "STR"},
			{"voidwalker", "reaver", "Voidwalker", "Corruption", "Prime a target with Void Hex, then spend Corruption and up to 5% max HP for a void burst; never self-lethal.", []string{"INT", "HP", "DEF"}, "INT powers curses and bursts. HP and defense offset the explicit health cost.", "Intelligence and healing buffs support the risk and recovery cycle.", "Void Hex", "Oblivion Burst", "INT"},
		}},
		{"artificer", "Artificer", "class:artificer", "Charge equipment runes or prepare a reactive alchemical mixture.", []AbyssSubclass{
			{"runesmith", "artificer", "Runesmith", "Charge", "Inscribe a rune, then discharge it for a burst and protective barrier; an equipped relic strengthens the discharge.", []string{"INT", "DEF", "MNA"}, "INT powers the rune; DEF strengthens its barrier. Any equipped relic improves the payoff.", "Intelligence, defense and relic-power buffs suit the rune cycle.", "Rune Inscription", "Runic Discharge", "INT"},
			{"alchemist", "artificer", "Alchemist", "Catalyst", "Apply Volatile Mixture, then trigger a Catalytic Burst that pierces armor and restores health.", []string{"INT", "HP", "MNA"}, "INT powers reactions; HP increases mixture recovery. Keep mana for the second action.", "Intelligence, max HP and mana buffs support damage and sustain.", "Volatile Mixture", "Catalytic Burst", "INT"},
		}},
	}
}

func AbyssSubclassByID(id string) (AbyssSubclass, bool) {
	for _, class := range AbyssClasses() {
		for _, sub := range class.Subclasses {
			if sub.ID == id {
				return sub, true
			}
		}
	}
	return AbyssSubclass{}, false
}

// Signature actions are additive class actions, never random drops or purchased slots.
func AbyssClassSkills(id string) []Skill {
	sub, ok := AbyssSubclassByID(id)
	if !ok {
		return nil
	}
	kind := SkillPhysical
	if sub.Scaling == "INT" {
		kind = SkillMagic
	}
	result := []Skill{
		{ID: "CLASS_" + id + "_build", Name: sub.Builder, Type: kind, Rarity: RarityRare, Power: 1.2, ManaCost: 15, ScalingStat: sub.Scaling, Role: "builder", TargetMode: SkillTargetEnemy, Source: "class_signature", Mechanics: "Gain 1 " + sub.Resource + " (maximum 3). " + sub.Sequence, Tags: []string{"class", "builder"}},
		{ID: "CLASS_" + id + "_finish", Name: sub.Finisher, Type: kind, Rarity: RarityRare, Power: 1.7, ManaCost: 30, CooldownRounds: 3, ScalingStat: sub.Scaling, Role: "finisher", TargetMode: SkillTargetEnemy, Source: "class_signature", Mechanics: "Spend all " + sub.Resource + " for +20% power per charge. " + sub.Sequence, Tags: []string{"class", "finisher"}},
	}
	switch id {
	case "vanguard", "geomancer":
		result[0].Power = 0
		result[0].TargetMode = SkillTargetSelf
		result[0].Role = "defense"
		result[0].CooldownRounds = 2
	case "oracle":
		result[0].Power = 0
		result[0].HealPercent = .15
		result[0].ScalingStat = "HP"
		result[0].TargetMode = SkillTargetAlly
		result[0].Role = "healing"
		result[0].CooldownRounds = 2
	case "bloodblade":
		result[0].HealPercent = .04
	case "elementalist":
		result[0].Element = ElementFire
		result[1].Element = ElementWater
	case "alchemist":
		result[0].Element = ElementEarth
		result[1].Element = ElementFire
	}
	builderDetails := map[string]string{
		"vanguard":     "Add a DEF-sized self barrier, capped at 50% maximum HP.",
		"geomancer":    "Add a DEF-sized self barrier, capped at 50% maximum HP.",
		"oracle":       "Heal an ally for 15% of caster maximum HP.",
		"bloodblade":   "Heal yourself for 4% maximum HP.",
		"marksman":     "Mark this target for armor penetration.",
		"elementalist": "Mark this target for detonation.",
		"voidwalker":   "Mark this target for armor penetration.",
		"alchemist":    "Mark this target for armor penetration.",
	}
	finisherDetails := map[string]string{
		"vanguard":     "With resource: ignore 30% defense.",
		"berserker":    "With resource: 25% more power against targets at or below 50% HP.",
		"marksman":     "With resource: ignore 60% defense against your marked target.",
		"beastmaster":  "With resource: 10% more power per living pet, up to 3 pets.",
		"elementalist": "With resource: 20% more power against your marked target.",
		"chronomancer": "With resource: reduce other skill and ultimate cooldowns by 1 round.",
		"oracle":       "With resource: 15% more power.",
		"geomancer":    "With resource: ignore 45% defense.",
		"bloodblade":   "Heal yourself for 4% maximum HP per charge spent.",
		"voidwalker":   "With resource: spend up to 5% maximum HP, leaving at least 1 HP. Gain 25% more power if you can pay the full cost; ignore 25% defense against your marked target.",
		"runesmith":    "With resource: 15% more power with a relic equipped; add a DEF/2 self barrier, capped at 50% maximum HP.",
		"alchemist":    "Heal yourself for 3% maximum HP per charge. With resource: ignore 35% defense against your marked target.",
	}
	result[0].Mechanics += " " + builderDetails[id]
	result[1].Mechanics += " " + finisherDetails[id]
	for i := range result {
		result[i].Description = result[i].Mechanics
		result[i].UpgradeRank = 1
		result[i].StackLimit = 3
		if result[i].Element == "" {
			result[i].Element = ElementPhysical
		}
		result[i].PreviewMin = result[i].Power
		result[i].PreviewMax = result[i].Power
	}
	return result
}
