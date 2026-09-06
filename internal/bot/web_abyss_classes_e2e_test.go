//go:build e2e

package bot

import (
	"fmt"
	"net/http"
	"ts3news/internal/content"
)

func registerAbyssClassFixture(mux *http.ServeMux) {
	mux.HandleFunc("/api/abyss/classes", func(w http.ResponseWriter, r *http.Request) {
		u := UserInCombat{Stats: content.Stats{STR: 120, INT: 160, DEF: 100, HP: 1000, MNA: 200}, STRMod: 1}
		learned := []content.Skill{}
		for _, id := range []string{"S0_1", "S0_2", "S1_2"} {
			if skill, ok := content.GetSkillByID(id); ok {
				learned = append(learned, skill)
			}
		}
		u.Skills = learned
		signatures := map[string]any{}
		trees := map[string]content.AbyssTalentTree{}
		foundations := map[string]content.AbyssSubclass{}
		progress := map[string]any{}
		state := newAbyssClassState()
		for _, class := range content.AbyssClasses() {
			trees[class.ID] = content.AbyssTalents(class.ID)
			foundations[class.ID], _ = content.AbyssFoundationStyle(class.ID)
			base := []string{}
			for i := 1; i <= 5; i++ {
				base = append(base, fmt.Sprintf("%s_t%d_1", class.ID, i))
			}
			state.Progress[class.ID] = abyssClassProgress{XP: 23825000, Foundation: base}
			progress[class.ID] = map[string]any{"xp": 23825000, "clears": 23825, "points": 15, "next_xp": 0, "foundation": base, "subclass_unlocked": true, "best_depth": 400}
			signatures[class.ID] = abyssBuildSkillViews(u, content.AbyssClassSkills(class.ID))
			for _, sub := range class.Subclasses {
				trees[sub.ID] = content.AbyssTalents(sub.ID)
				signatures[sub.ID] = abyssBuildSkillViews(u, content.AbyssClassSkills(sub.ID))
			}
		}
		writeJSON(w, map[string]any{"ok": true, "catalog": content.AbyssClasses(), "state": state, "talent_catalog": trees, "foundation_styles": foundations, "class_progress": progress, "signatures": signatures, "skills": abyssBuildSkillViews(u, learned), "learned": abyssBuildSkillViews(u, learned), "capacity": 5, "stats": u.Stats, "advice": []string{}, "weakness": "Mana recovers 20 per round; weave basic attacks for long fights.", "locked": false, "next_upgrade": map[string]any{"label": "Arcane Studies", "detail": "1 skill point. +5 INT. Reachable on your path."}, "gear_comparisons": []map[string]any{{"id": 1, "name": "Runic Staff", "slot": "MainHand", "damage_stat": "INT", "damage_delta": 12, "hp_delta": -10, "defense_delta": 2, "mana_delta": 20}}})
	})
}
