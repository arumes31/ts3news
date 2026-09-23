package rift

// configureClassTarget prepares repeatable conditions for the equipped subclass.
// Equipment, pets and relics remain the player's own build.
func (r *Run) configureClassTarget() {
	e := &r.Enemies[0]
	switch r.Build.Class {
	case "vanguard":
		e.Name, e.Armor = "Bulwark dummy", .4
		r.Practice.TargetHint = "Armored target: compare basic hits with your charged finisher's armor piercing."
	case "berserker":
		e.Name = "Wounded execution dummy"
		e.HP = e.MaxHP * .5
		r.Practice.TargetHint = "Target stays at half health between hits, enabling your charged finisher's execution bonus."
	case "marksman":
		e.Name, e.Armor = "Armored precision dummy", .6
		r.Practice.TargetHint = "Armored target: mark it with your builder, then spend charges on that same target for precision piercing."
	case "beastmaster":
		e.Name = "Pack command dummy"
		r.Practice.TargetHint = "Mark this target and follow with a charged pack attack. The bonus uses your actual equipped pet count."
	case "elementalist":
		e.Name = "Reaction dummy"
		r.Practice.TargetHint = "Mark this target with your builder, then land a charged finisher on it to trigger the elemental reaction."
	case "chronomancer":
		e.Name = "Rewind dummy"
		r.Practice.TargetHint = "Use another equipped ability or jump before your charged finisher; watch its remaining cooldown rewind."
	case "oracle":
		e.Name = "Grace practice dummy"
		r.Player.HP = r.Player.MaxHP * .6
		r.Practice.TargetHint = "You start at 60% health so your healing builder can restore health before you spend Grace on this target."
	case "geomancer":
		e.Name, e.Armor = "Stone armor dummy", .8
		r.Practice.TargetHint = "Heavily armored target: build charges, then compare your charged finisher's piercing with ordinary hits."
	case "bloodblade":
		e.Name = "Blood recovery dummy"
		r.Player.HP = r.Player.MaxHP * .6
		r.Practice.TargetHint = "You start at 60% health. Strike this target and watch your builder and charged finisher restore health."
	case "voidwalker":
		e.Name, e.Armor = "Void mark dummy", .4
		r.Practice.TargetHint = "Armored target: mark it before spending charges. Watch the health cost and marked-target piercing together."
	case "runesmith":
		e.Name = "Rune barrier dummy"
		r.Practice.TargetHint = "Spend charges on this target and watch your barrier appear. Relic damage uses your actual equipped relic."
	case "alchemist":
		e.Name, e.Armor = "Mixture test dummy", .4
		r.Player.HP = r.Player.MaxHP * .6
		r.Practice.TargetHint = "You start at 60% health against an armored target. Mark it, then spend mixture charges to heal and pierce armor."
	default:
		r.Practice.TargetHint = "Build charges and land a charged finisher on this target."
	}
}

func (r *Run) restorePracticeTarget(e *Actor) {
	e.HP = e.MaxHP
	if r.Practice.Mode == "class" && r.Build.Class == "berserker" {
		e.HP = e.MaxHP * .5
	}
}
