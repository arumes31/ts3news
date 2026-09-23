package rift

// ObjectiveProgress is a server-confirmed optional mission challenge.
type ObjectiveProgress struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Current     float64 `json:"current"`
	Target      float64 `json:"target"`
	Status      string  `json:"status"`
	Reason      string  `json:"reason,omitempty"`
}

type MissionObjectives struct {
	StartDodges    int                 `json:"start_dodges"`
	StartNonMelee  int                 `json:"start_non_melee"`
	StartAerial    int                 `json:"start_aerial"`
	StartTreasure  int                 `json:"start_treasure"`
	StartHazards   int                 `json:"start_hazards"`
	StartUltimates int                 `json:"start_ultimates"`
	StartFinishers int                 `json:"start_finishers"`
	Banked         bool                `json:"banked,omitempty"`
	Difficulty     string              `json:"difficulty,omitempty"`
	Mission        int                 `json:"mission"`
	Finished       bool                `json:"finished"`
	StartSeconds   float64             `json:"start_seconds"`
	StartDamage    float64             `json:"start_damage"`
	StartSkills    int                 `json:"start_skills"`
	StartGuards    int                 `json:"start_guards"`
	Entries        []ObjectiveProgress `json:"entries"`
}

// ObjectiveOptions supplies both mission tracking and the pre-run preview.
func ObjectiveOptions(build Build) []ObjectiveProgress {
	entries := []ObjectiveProgress{
		{ID: "timed", Name: "Swift clear", Description: "Clear all three tiers within 180 combat seconds. Pauses do not count.", Target: 180, Status: "active"},
		{ID: "no_damage", Name: "Untouched", Description: "Clear all three tiers without taking health damage.", Status: "active"},
		{ID: "basic_only", Name: "Basic attacks only", Description: "Clear all three tiers without using abilities. Movement, jumping and guarding are allowed.", Status: "active"},
		{ID: "guard", Name: "Guard mastery", Description: "Block at least five attacks and clear all three tiers.", Target: 5, Status: "active"},
		{ID: "hazard_avoidance", Name: "Safe footing", Description: "Clear all three tiers without triggering an active floor hazard. Jumping avoids hazards; shields do not.", Status: "active"},
		{ID: "treasure_capture", Name: "Treasure hunter", Description: "Defeat at least one treasure goblin and clear all three tiers. Offered when the mission contains a goblin; escapes do not count.", Target: 1, Status: "active"},
		{ID: "aerial_finish", Name: "Aerial finish", Description: "Defeat at least one enemy with a basic attack while airborne, then clear all three tiers. Spell and pet kills do not count.", Target: 1, Status: "active"},
		{ID: "melee_only", Name: "Close combat", Description: "Clear all three tiers using only basic attacks, slash abilities, heals and shields. Projectile, quake and area-ultimate casts fail this goal, even on a miss.", Status: "active"},
		{ID: "ranged_priority", Name: "Ranged enemies first", Description: "Clear all three tiers, defeating active archers and spellcasters before other active enemies. Offered when the mission contains ranged enemies.", Status: "active"},
		{ID: "elite_priority", Name: "Elites first", Description: "Clear all three tiers, defeating active elite minions, elites and minibosses before other active enemies. Bosses are excluded. Offered when the mission contains elites.", Status: "active"},
		{ID: "limited_dodge", Name: "Measured evasion", Description: "Clear all three tiers with at most three airborne dodges of enemy attacks. Empty jumps and floor hazards do not count.", Target: 3, Status: "active"},
	}
	builder, finisher := false, false
	for _, skill := range build.Signatures {
		builder = builder || skill.Role == "builder"
		finisher = finisher || skill.Role == "finisher"
	}
	if builder && finisher {
		entries = append(entries, ObjectiveProgress{ID: "finisher", Name: "Class finisher", Description: "Use at least one charged class finisher and clear all three tiers. Empty finishers do not count.", Target: 1, Status: "active"})
	}
	if build.Ultimate != nil {
		entries = append(entries, ObjectiveProgress{ID: "save_ultimate", Name: "Ultimate in reserve", Description: "Clear all three tiers without casting an ultimate. Other abilities are allowed.", Status: "active"})
	}
	return entries
}

func (r *Run) beginObjectives() {
	if r.Objectives != nil && r.Objectives.Finished {
		r.LastObjectives = r.Objectives
	}
	entries := ObjectiveOptions(r.Build)
	hasTreasure, hasRanged, hasElite := false, false, false
	for _, room := range r.EncounterPlan {
		for _, enemy := range room {
			hasTreasure = hasTreasure || enemy.Kind == "treasure"
			hasRanged = hasRanged || enemy.Kind == "archer"
			hasElite = hasElite || isPriorityElite(enemy)
		}
	}
	if !hasTreasure || !hasRanged || !hasElite {
		filtered := entries[:0]
		for _, entry := range entries {
			if (entry.ID != "treasure_capture" || hasTreasure) && (entry.ID != "ranged_priority" || hasRanged) && (entry.ID != "elite_priority" || hasElite) {
				filtered = append(filtered, entry)
			}
		}
		entries = filtered
	}
	r.Objectives = &MissionObjectives{Mission: r.Level.ID, Difficulty: r.Level.Difficulty, StartSeconds: r.Stats.Seconds, StartDamage: r.Stats.DamageTaken, StartSkills: r.Stats.SkillsCast, StartGuards: r.Stats.Guards, StartFinishers: r.Stats.ChargedFinishers, StartUltimates: r.Stats.UltimateCasts, StartHazards: r.Stats.HazardContacts, StartTreasure: r.Stats.TreasureGoblins, StartAerial: r.Stats.AerialFinishes, StartNonMelee: r.Stats.NonMeleeCasts, StartDodges: r.Stats.Dodges, Entries: entries}
}

// UpdateObjectives refreshes progress from confirmed combat and finalizes ended runs.
func (r *Run) UpdateObjectives() {
	o := r.Objectives
	if o == nil || o.Finished || r.Practice != nil {
		return
	}
	cleared := r.Room == len(Rooms)-1 && (r.Status == "cleared" || r.Status == "complete" || r.Status == "banked")
	ended := cleared || r.Status == "defeated" || r.Status == "banked" || r.Status == "expired"
	for i := range o.Entries {
		e := &o.Entries[i]
		if e.Status == "failed" {
			continue
		}
		reason := ""
		switch e.ID {
		case "timed":
			e.Current = max(0, r.Stats.Seconds-o.StartSeconds)
			if e.Current > e.Target {
				reason = "Exceeded 180 combat seconds."
			}
		case "no_damage":
			e.Current = max(0, r.Stats.DamageTaken-o.StartDamage)
			if e.Current > 0 {
				reason = "Took health damage."
			}
		case "basic_only":
			e.Current = float64(max(0, r.Stats.SkillsCast-o.StartSkills))
			if e.Current > 0 {
				reason = "Used an ability."
			}
		case "limited_dodge":
			e.Current = float64(max(0, r.Stats.Dodges-o.StartDodges))
			if e.Current > e.Target {
				reason = "Exceeded three airborne dodges of enemy attacks."
			}
		case "melee_only":
			e.Current = float64(max(0, r.Stats.NonMeleeCasts-o.StartNonMelee))
			if e.Current > 0 {
				reason = "Cast an ability outside the melee/support allowance."
			}
		case "aerial_finish":
			e.Current = float64(max(0, r.Stats.AerialFinishes-o.StartAerial))
			if cleared && e.Current < e.Target {
				reason = "Finished without an airborne basic-attack defeat."
			}
		case "treasure_capture":
			e.Current = float64(max(0, r.Stats.TreasureGoblins-o.StartTreasure))
			if cleared && e.Current < e.Target {
				reason = "Finished without defeating a treasure goblin."
			}
		case "hazard_avoidance":
			e.Current = float64(max(0, r.Stats.HazardContacts-o.StartHazards))
			if e.Current > 0 {
				reason = "Triggered an active floor hazard."
			}
		case "save_ultimate":
			e.Current = float64(max(0, r.Stats.UltimateCasts-o.StartUltimates))
			if e.Current > 0 {
				reason = "Cast an ultimate."
			}
		case "finisher":
			e.Current = float64(max(0, r.Stats.ChargedFinishers-o.StartFinishers))
			if cleared && e.Current < e.Target {
				reason = "Finished without using a charged class finisher."
			}
		case "guard":
			e.Current = float64(max(0, r.Stats.Guards-o.StartGuards))
			if cleared && e.Current < e.Target {
				reason = "Finished with fewer than five blocks."
			}
		}
		if reason != "" {
			e.Status = "failed"
			e.Reason = reason
		} else if cleared {
			e.Status = "complete"
		} else if ended {
			e.Status = "failed"
			e.Reason = "Mission ended before all three tiers were cleared."
		}
	}
	o.Finished = ended
}
