package bot

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"
	"ts3news/internal/content"
)

type abyssClassProfile struct {
	Skills []string `json:"skills"`
	Pins   []string `json:"pins"`
}
type abyssClassState struct {
	Version  int                          `json:"version"`
	Revision int64                        `json:"revision"`
	Selected string                       `json:"selected"`
	Profiles map[string]abyssClassProfile `json:"profiles"`
}

func newAbyssClassState() abyssClassState {
	return abyssClassState{Version: 1, Profiles: map[string]abyssClassProfile{}}
}
func abyssClassKey(uid string) string { return "abyss_class_build:" + uid }
func decodeAbyssClassState(raw string) (abyssClassState, error) {
	var state abyssClassState
	if err := json.Unmarshal([]byte(raw), &state); err != nil {
		return state, err
	}
	if state.Version != 1 {
		return state, errors.New("unsupported class build version")
	}
	if state.Selected != "" {
		if _, ok := content.AbyssSubclassByID(state.Selected); !ok {
			return state, errors.New("unknown saved subclass")
		}
	}
	if state.Profiles == nil {
		state.Profiles = map[string]abyssClassProfile{}
	}
	return state, nil
}
func (b *Bot) loadAbyssClassState(ctx context.Context, uid string) (abyssClassState, error) {
	var raw string
	err := b.DB.QueryRowContext(ctx, "SELECT value FROM app_meta WHERE key=$1", abyssClassKey(uid)).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return newAbyssClassState(), nil
	}
	if err != nil {
		return abyssClassState{}, err
	}
	return decodeAbyssClassState(raw)
}

func validateAbyssClassProfile(profile abyssClassProfile, available []content.Skill, capacity int) error {
	if len(profile.Skills) > capacity || len(profile.Pins) > capacity {
		return errors.New("build exceeds your acquired skill slots")
	}
	known := map[string]bool{}
	for _, skill := range available {
		known[skill.ID] = true
	}
	for _, ids := range [][]string{profile.Skills, profile.Pins} {
		seen := map[string]bool{}
		for _, id := range ids {
			if !known[id] || seen[id] {
				return errors.New("build contains an unavailable or duplicate skill")
			}
			seen[id] = true
		}
	}
	return nil
}
func (b *Bot) abyssSkillCapacity(uid string) int {
	var title sql.NullString
	_ = b.DB.QueryRow("SELECT title FROM users WHERE client_uid=$1", uid).Scan(&title)
	capacity := 5
	if t, ok := content.GetTitleByName(title.String); ok {
		capacity += t.ExtraSkills
	}
	return capacity
}

// Only already equipped/acquired skills enter this permanent collection. Tree
// grants are resolved separately, so refunding a grant never leaves it unlocked.
func archiveAbyssLearnedSkill(exec dbExecQuerier, uid, id string) error {
	if strings.HasPrefix(id, "CLASS_") {
		return nil
	}
	_, err := exec.Exec("INSERT INTO app_meta (key,value) VALUES ($1,$2) ON CONFLICT (key) DO NOTHING", "abyss_learned:"+uid+":"+id, id)
	return err
}
func (b *Bot) abyssLearnedSkills(ctx context.Context, uid string) ([]content.Skill, error) {
	rows, err := b.DB.QueryContext(ctx, `SELECT skill_id FROM user_skills WHERE client_uid=$1
 UNION SELECT value FROM app_meta WHERE starts_with(key,$2)`, uid, "abyss_learned:"+uid+":")
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []content.Skill{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		if skill, ok := content.GetSkillByID(id); ok {
			out = append(out, skill)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name || out[i].Name == out[j].Name && out[i].ID < out[j].ID
	})
	return out, rows.Err()
}
func (b *Bot) applyAbyssClassBuild(u *UserInCombat, state abyssClassState) {
	sub, ok := content.AbyssSubclassByID(state.Selected)
	if !ok {
		return
	}
	u.AbyssClass = sub.ClassID
	u.AbyssSubclass = sub.ID
	profile := state.Profiles[sub.ID]
	if profile.Skills != nil {
		skills := []content.Skill{}
		for _, id := range profile.Skills {
			if skill, ok := content.GetSkillByID(id); ok {
				skills = append(skills, skill)
			}
		}
		// Keep currently granted Skill Web actions in addition to all paid slots.
		for _, skill := range u.Skills {
			if skill.ID == "S_EQ" || skill.ID == "S_AS" {
				found := false
				for _, chosen := range skills {
					if chosen.ID == skill.ID {
						found = true
					}
				}
				if !found {
					skills = append(skills, skill)
				}
			}
		}
		u.Skills = skills
	}
	u.Skills = append(u.Skills, content.AbyssClassSkills(sub.ID)...)
}

func (s *WebServer) handleAbyssClasses(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		http.Error(w, "GET or POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()
	if r.Method == http.MethodPost {
		if s.rejectDuringLiveCombat(w, uid) {
			return
		}
		locked, lockErr := s.bot.abyssClassRunLocked(r.Context(), uid)
		if lockErr != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "Could not verify your run state. Retry."})
			return
		}
		if locked {
			writeJSON(w, map[string]any{"ok": false, "error": "Class builds can change after you bank or end this run."})
			return
		}
		var req struct {
			Selected string             `json:"selected"`
			Revision int64              `json:"revision"`
			Profile  *abyssClassProfile `json:"profile"`
		}
		if readJSON(r, &req) != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "Invalid build request."})
			return
		}
		if req.Selected != "" {
			if _, ok := content.AbyssSubclassByID(req.Selected); !ok {
				writeJSON(w, map[string]any{"ok": false, "error": "Unknown subclass."})
				return
			}
		}
		learned, err := s.bot.abyssLearnedSkills(r.Context(), uid)
		if err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "Could not load learned skills. Retry."})
			return
		}
		if req.Profile != nil {
			if err := validateAbyssClassProfile(*req.Profile, learned, s.bot.abyssSkillCapacity(uid)); err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": err.Error()})
				return
			}
		}
		tx, err := s.bot.DB.BeginTx(r.Context(), nil)
		if err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "Could not save build. Retry."})
			return
		}
		defer func() { _ = tx.Rollback() }()
		var owner string
		if err = tx.QueryRowContext(r.Context(), "SELECT client_uid FROM users WHERE client_uid=$1 FOR UPDATE", uid).Scan(&owner); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "Could not lock your character. Retry."})
			return
		}
		var runLocked bool
		if err = tx.QueryRowContext(r.Context(), abyssClassRunLockQuery, uid).Scan(&runLocked); err != nil || runLocked {
			writeJSON(w, map[string]any{"ok": false, "error": "A run is active or could not be verified. Your build was preserved."})
			return
		}
		rawDefault, _ := json.Marshal(newAbyssClassState())
		if _, err = tx.ExecContext(r.Context(), "INSERT INTO app_meta (key,value) VALUES ($1,$2) ON CONFLICT (key) DO NOTHING", abyssClassKey(uid), string(rawDefault)); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "Could not save build. Retry."})
			return
		}
		var raw string
		if err = tx.QueryRowContext(r.Context(), "SELECT value FROM app_meta WHERE key=$1 FOR UPDATE", abyssClassKey(uid)).Scan(&raw); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "Could not lock build. Retry."})
			return
		}
		state, err := decodeAbyssClassState(raw)
		if err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "Saved build could not be read; it has been preserved."})
			return
		}
		if state.Revision != req.Revision {
			writeJSON(w, map[string]any{"ok": false, "error": "This build changed in another tab. Reload before saving."})
			return
		}
		state.Selected = req.Selected
		state.Revision++
		if req.Profile != nil {
			state.Profiles[req.Selected] = *req.Profile
		}
		for _, skill := range learned {
			if err = archiveAbyssLearnedSkill(tx, uid, skill.ID); err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "Could not preserve skills. Retry."})
				return
			}
		}
		encoded, _ := json.Marshal(state)
		if _, err = tx.ExecContext(r.Context(), "UPDATE app_meta SET value=$2 WHERE key=$1", abyssClassKey(uid), string(encoded)); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "Could not save build. Retry."})
			return
		}
		if err = tx.Commit(); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "Could not save build. Retry."})
			return
		}
	}
	state, err := s.bot.loadAbyssClassState(r.Context(), uid)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "Could not read saved build. Retry."})
		return
	}
	u, _, err := s.bot.buildAbyssUser(uid)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "Could not load character. Retry."})
		return
	}
	learned, err := s.bot.abyssLearnedSkills(r.Context(), uid)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "Could not load learned skills. Retry."})
		return
	}
	locked, err := s.bot.abyssClassRunLocked(r.Context(), uid)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "Could not verify your run state. Retry."})
		return
	}
	writeJSON(w, s.bot.abyssClassBuildView(uid, state, u, learned, locked))
}

func abyssSkillFamily(skill content.Skill) string {
	name := strings.ToLower(skill.Name)
	for _, family := range []string{"shield", "heal", "mend", "curse", "sunder", "drain", "bolt", "blast", "strike", "bash", "nova"} {
		if strings.Contains(name, family) {
			return family
		}
	}
	if role := abyssClassSkillRole(skill); role != "" {
		return role
	}
	return string(skill.Type)
}
func abyssBuildSkillViews(u UserInCombat, skills []content.Skill) []map[string]any {
	out := make([]map[string]any, 0, len(skills))
	for _, skill := range skills {
		out = append(out, map[string]any{"id": skill.ID, "name": skill.Name, "family": abyssSkillFamily(skill), "role": skill.Role, "scaling": skill.ScalingStat, "mana": skill.ManaCost, "cooldown": skill.CooldownRounds, "power": skill.Power, "base_effect": int(float64(abyssSkillBase(&u, skill)) * skill.Power), "heal_percent": skill.HealPercent, "description": skill.Description, "mechanics": skill.Mechanics, "rank": skill.UpgradeRank, "source": skill.Source})
	}
	return out
}
func (b *Bot) abyssClassBuildView(uid string, state abyssClassState, u UserInCombat, learned []content.Skill, locked bool) map[string]any {
	catalog := content.AbyssClasses()
	signatures := map[string]any{}
	for _, class := range catalog {
		for _, sub := range class.Subclasses {
			signatures[sub.ID] = abyssBuildSkillViews(u, content.AbyssClassSkills(sub.ID))
		}
	}
	sub, _ := content.AbyssSubclassByID(state.Selected)
	advice := []string{"Choose a class to see your signature sequence and build priorities. Existing specializations and Skill Web investments remain active."}
	if sub.ID != "" {
		advice = []string{sub.Gear, sub.Buffs}
	}
	regen := 10 + u.Stats.MNA/20
	weakness := fmt.Sprintf("Mana recovers %d per round. A signature pair costs 45 mana before reductions.", regen)
	if regen < 23 {
		weakness += " Alternate with basic attacks or add mana sustain for longer encounters."
	} else {
		weakness += " Your base recovery supports the two-action signature cycle."
	}
	return map[string]any{"ok": true, "catalog": catalog, "state": state, "signatures": signatures, "skills": abyssBuildSkillViews(u, u.Skills), "learned": abyssBuildSkillViews(u, learned), "capacity": b.abyssSkillCapacity(uid), "stats": u.Stats, "advice": advice, "weakness": weakness, "locked": locked, "next_upgrade": b.abyssClassTreeUpgrade(uid, u, sub), "gear_comparisons": b.abyssClassGearComparisons(uid, u, sub)}
}

func abyssClassNextUpgrade(u UserInCombat, sub content.AbyssSubclass) map[string]any {
	if sub.ID == "" {
		return map[string]any{"label": "Select a class", "detail": "Inspect both subclasses before choosing; switching between runs is free."}
	}
	skills := content.AbyssClassSkills(sub.ID)
	before := int(float64(abyssSkillBase(&u, skills[1])) * skills[1].Power)
	improved := u
	switch sub.Scaling {
	case "DEF":
		improved.Stats.DEF += 10
	case "INT":
		improved.Stats.INT += 10
	default:
		improved.Stats.STR += 10
	}
	after := int(float64(abyssSkillBase(&improved, skills[1])) * skills[1].Power)
	return map[string]any{"label": "Prioritize " + sub.Scaling, "detail": fmt.Sprintf("+10 %s adds %d base damage to %s before enemy defense, resource bonuses and other modifiers. Use the Skill Web planner for exact point costs.", sub.Scaling, after-before, sub.Finisher), "href": "/abyss/tree"}
}
func (b *Bot) abyssClassGearComparisons(uid string, u UserInCombat, sub content.AbyssSubclass) []map[string]any {
	out := []map[string]any{}
	if sub.ID == "" {
		return out
	}
	rows, err := b.DB.Query("SELECT id,gear_id,item_data FROM user_inventory WHERE client_uid=$1 ORDER BY id DESC LIMIT 100", uid)
	if err != nil {
		return out
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var id int64
		var gid string
		var data sql.NullString
		if rows.Scan(&id, &gid, &data) != nil {
			continue
		}
		gear, ok := b.makeGear(gid, data)
		if !ok || gear.Unidentified {
			continue
		}
		old := u.Equipped[gear.Slot]
		delta := gear.Stats
		delta.STR -= old.Stats.STR
		delta.INT -= old.Stats.INT
		delta.DEF -= old.Stats.DEF
		delta.HP -= old.Stats.HP
		delta.MNA -= old.Stats.MNA
		primary := delta.STR
		switch sub.Scaling {
		case "INT":
			primary = delta.INT
		case "DEF":
			primary = delta.DEF
		}
		out = append(out, map[string]any{"id": id, "name": gear.Name, "slot": gear.Slot, "damage_stat": sub.Scaling, "damage_delta": primary, "hp_delta": delta.HP, "defense_delta": delta.DEF, "mana_delta": delta.MNA, "synergy": sub.Gear})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i]["damage_delta"].(int) > out[j]["damage_delta"].(int) })
	if len(out) > 6 {
		out = out[:6]
	}
	return out
}

// Replacement and collection writes commit together: a dropped skill cannot
// erase an old acquired skill or bypass a pin from a saved subclass.
var errAbyssSkillPinned = errors.New("skill is pinned by a saved build")

func (b *Bot) replaceAbyssAcquiredSkill(uid string, slot int, oldID, newID string) error {
	tx, err := b.DB.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	var owner string
	if err = tx.QueryRow("SELECT client_uid FROM users WHERE client_uid=$1 FOR UPDATE", uid).Scan(&owner); err != nil {
		return err
	}
	var raw string
	state := newAbyssClassState()
	err = tx.QueryRow("SELECT value FROM app_meta WHERE key=$1", abyssClassKey(uid)).Scan(&raw)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if err == nil {
		state, err = decodeAbyssClassState(raw)
		if err != nil {
			return err
		}
	}
	for _, profile := range state.Profiles {
		for _, pin := range profile.Pins {
			if pin == oldID {
				return errAbyssSkillPinned
			}
		}
	}

	if err = archiveAbyssLearnedSkill(tx, uid, oldID); err != nil {
		return err
	}
	if err = archiveAbyssLearnedSkill(tx, uid, newID); err != nil {
		return err
	}
	result, err := tx.Exec("UPDATE user_skills SET skill_id=$3 WHERE client_uid=$1 AND slot=$2 AND skill_id=$4", uid, slot, newID, oldID)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count != 1 {
		return errors.New("equipped skill changed")
	}
	return tx.Commit()
}

const abyssClassRunLockQuery = `SELECT EXISTS(SELECT 1 FROM abyss_active WHERE client_uid=$1 OR coop_uid=$1)
 OR EXISTS(SELECT 1 FROM abyss_party_members pm JOIN abyss_active a ON a.client_uid=pm.owner_uid WHERE pm.member_uid=$1)`

func (b *Bot) abyssClassRunLocked(ctx context.Context, uid string) (bool, error) {
	var locked bool
	err := b.DB.QueryRowContext(ctx, abyssClassRunLockQuery, uid).Scan(&locked)
	return locked, err
}

// Recommendations use the same connectivity, depth gates and discounted point
// costs as the planner. They explain a candidate without allocating anything.
func (b *Bot) abyssClassTreeUpgrade(uid string, u UserInCombat, sub content.AbyssSubclass) map[string]any {
	fallback := abyssClassNextUpgrade(u, sub)
	if sub.ID == "" {
		return fallback
	}
	active, err := b.loadTreeAllocated(uid)
	if err != nil {
		return fallback
	}
	return abyssClassTreeCandidate(content.AbyssTree(), active, b.treePointsTotal(uid), b.loadAbyssStats(uid).BestDepth, b.abyssPrestigeMemoryNode(uid), abyssNodeOfTheDay(time.Now()), u, sub, fallback)
}
func abyssClassTreeCandidate(tree *content.AbyssTreeData, active []int, points, depth, memory, day int, u UserInCombat, sub content.AbyssSubclass, fallback map[string]any) map[string]any {
	allocated := map[int]bool{0: true}
	for _, id := range active {
		allocated[id] = true
	}
	candidates := map[int]bool{}
	for id := range allocated {
		for _, next := range tree.Adj[id] {
			if !allocated[next] {
				candidates[next] = true
			}
		}
	}
	bestScore := -1.0
	bestID := 0
	var best abyssTreePlanAnalysis
	// Weight what the actual equipped skills use, retaining the signature as the primary direction.
	weights := map[string]float64{sub.Scaling: 2}
	for _, skill := range u.Skills {
		if skill.Power > 0 {
			weights[skill.ScalingStat] += .25
		}
	}
	for id := range candidates {
		plan := analyzeAbyssTreePlan(tree, active, append(append([]int(nil), active...), id), points, depth, memory, day)
		if !plan.Valid {
			continue
		}
		delta := plan.StatDelta
		score := float64(delta.STR)*weights["STR"] + float64(delta.INT)*weights["INT"] + float64(delta.DEF)*(weights["DEF"]+.3) + float64(delta.HP)*.04 + float64(delta.MNA)*.15
		if 10+u.Stats.MNA/20 < 23 {
			score += float64(delta.MNA)*.5 + plan.PctDelta["skill_mana_cost"]*200
		}
		for stat, weight := range weights {
			score += plan.PctDelta[strings.ToLower(stat)+"_pct"] * float64(abyssSkillBase(&u, content.Skill{ScalingStat: stat})) * weight
		}
		score += (plan.PctDelta["skill_damage"] + plan.PctDelta["skill_power"]) * 100
		cost := max(1, plan.PlannedCost-plan.CurrentCost)
		score /= float64(cost)
		if score > bestScore || score == bestScore && (bestID == 0 || id < bestID) {
			bestScore = score
			bestID = id
			best = plan
		}
	}
	if bestID == 0 {
		return fallback
	}
	node := tree.Node(bestID)
	delta := best.StatDelta
	parts := []string{}
	for _, stat := range []struct {
		name  string
		value int
	}{{"STR", delta.STR}, {"INT", delta.INT}, {"DEF", delta.DEF}, {"HP", delta.HP}, {"MNA", delta.MNA}} {
		if stat.value != 0 {
			parts = append(parts, fmt.Sprintf("%+d %s", stat.value, stat.name))
		}
	}
	keys := make([]string, 0, len(best.PctDelta))
	for key := range best.PctDelta {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%+.1f%% %s", best.PctDelta[key]*100, strings.ReplaceAll(key, "_", " ")))
	}
	return map[string]any{"label": node.Name, "node_id": bestID, "cost": best.PlannedCost - best.CurrentCost, "detail": fmt.Sprintf("%d skill points. %s. Reachable and affordable on your current path; selected for your %s skills and sustain.", best.PlannedCost-best.CurrentCost, strings.Join(parts, ", "), sub.Scaling), "href": "/abyss/tree"}
}
