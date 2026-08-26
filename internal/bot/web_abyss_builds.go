package bot

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"ts3news/internal/content"
)

const (
	abyssRunFlagBuildKit      = "build_kit"
	abyssRunFlagSkillMutation = "skill_mutation"
	abyssRunFlagBuildRespec   = "build_respec_used"
	abyssRunFlagRelicCharges  = "active_relic_charges"
)

var abyssBuildKits = map[string]int64{
	"vanguard": 1,
	"arcanist": 2,
	"survival": 3,
}

var abyssSkillMutations = map[string]int64{
	"empowered":   1,
	"piercing":    2,
	"restorative": 3,
}

func normalizeAbyssBuildKit(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if _, ok := abyssBuildKits[value]; ok {
		return value
	}
	return "vanguard"
}

func normalizeAbyssSkillMutation(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if _, ok := abyssSkillMutations[value]; ok {
		return value
	}
	return "empowered"
}

func abyssBuildNameByValue(values map[string]int64, value int64) string {
	for name, candidate := range values {
		if candidate == value {
			return name
		}
	}
	return ""
}

func applyAbyssRunBuild(u *UserInCombat, flags map[string]int64, mastery map[string]int) {
	applyAbyssCombatPosition(u, flags)

	switch abyssBuildNameByValue(abyssBuildKits, flags[abyssRunFlagBuildKit]) {
	case "arcanist":
		u.Stats.INT += u.Stats.INT / 10
		u.Stats.MNA += 30
	case "survival":
		u.Stats.HP += u.Stats.HP / 10
		u.Stats.DEF += u.Stats.DEF / 20
	default:
		u.Stats.STR += u.Stats.STR / 10
		u.Stats.CRT += 5
	}

	mutation := abyssBuildNameByValue(abyssSkillMutations, flags[abyssRunFlagSkillMutation])
	for i := range u.Skills {
		skill := &u.Skills[i]
		switch mutation {
		case "piercing":
			skill.IgnoreDef += 0.15
			if skill.IgnoreDef > 1 {
				skill.IgnoreDef = 1
			}
			skill.Description = strings.TrimSpace(skill.Description + " [Piercing: +15% armor penetration]")
		case "restorative":
			skill.HealPercent += 0.05
			skill.Description = strings.TrimSpace(skill.Description + " [Restorative: heals 5% max HP]")
		default:
			skill.Power *= 1.20
			skill.Description = strings.TrimSpace(skill.Description + " [Empowered: +20% power]")
		}
		// Every 25 casts grants +5% mastery power, capped at four milestones.
		milestones := mastery[skill.ID] / 25
		if milestones > 4 {
			milestones = 4
		}
		if milestones > 0 {
			skill.Power *= 1 + float64(milestones)*0.05
			skill.Description = strings.TrimSpace(fmt.Sprintf("%s [Mastery %d: +%d%% power]", skill.Description, milestones, milestones*5))
		}
	}
}

func abyssBuildIdentity(u UserInCombat, flags map[string]int64) string {
	if kit := abyssBuildNameByValue(abyssBuildKits, flags[abyssRunFlagBuildKit]); kit != "" {
		return kit
	}
	switch {
	case u.Stats.INT >= u.Stats.STR && u.Stats.INT >= u.Stats.DEF:
		return "arcanist"
	case u.Stats.DEF >= u.Stats.STR:
		return "survival"
	default:
		return "vanguard"
	}
}

func abyssBuildSummary(u UserInCombat, flags map[string]int64) string {
	identity := abyssBuildIdentity(u, flags)
	mutation := abyssBuildNameByValue(abyssSkillMutations, flags[abyssRunFlagSkillMutation])
	parts := []string{strings.ToUpper(identity[:1]) + identity[1:] + " kit"}
	if mutation != "" {
		parts = append(parts, mutation+" skills")
	}
	if relic, ok := u.Equipped[content.SlotRelic]; ok {
		parts = append(parts, relic.Name+" active")
	}
	setCounts := map[string]int{}
	for _, gear := range u.Equipped {
		if set := gear.EffectiveSetID(); set != "" {
			setCounts[set]++
		}
	}
	sets := make([]string, 0, len(setCounts))
	for set := range setCounts {
		sets = append(sets, set)
	}
	sort.Strings(sets)
	for _, set := range sets {
		count := setCounts[set]
		if count >= 2 {
			parts = append(parts, fmt.Sprintf("%s %d-piece synergy", set, count))
		}
	}
	return strings.Join(parts, " · ")
}

func applyAbyssPartyBuildSynergy(users []UserInCombat, flagsByUID map[string]map[string]int64) (string, bool) {
	if len(users) < 2 {
		return "", false
	}
	identities := map[string]bool{}
	for i := range users {
		identities[abyssBuildIdentity(users[i], flagsByUID[users[i].UID])] = true
	}
	if len(identities) < 2 {
		return "", false
	}
	for i := range users {
		users[i].Stats = users[i].Stats.Scaled(1.05)
	}
	return "🤝 Cross-class resonance: distinct party roles grant everyone +5% stats.", true
}

func abyssSkillComboTags(skill content.Skill) []string {
	tags := append([]string(nil), skill.Tags...)
	if len(tags) == 0 {
		tags = []string{"spender"}
	}
	if element := abyssElementForSkillName(skill.Name); element != content.ElementPhysical {
		tags = append(tags, strings.ToLower(string(element)))
	}
	if skill.HealPercent > 0 {
		tags = append(tags, "heal", "sustain")
	}
	if skill.StunChance > 0 {
		tags = append(tags, "control", "setup")
	}
	if skill.IgnoreDef > 0 {
		tags = append(tags, "armor-break", "finisher")
	}
	if skill.Special != content.EffectNone {
		tags = append(tags, strings.ToLower(string(skill.Special)))
	}
	return tags
}

func abyssElementForSkillName(name string) content.Element {
	lower := strings.ToLower(name)
	switch {
	case strings.Contains(lower, "fire"), strings.Contains(lower, "flame"), strings.Contains(lower, "ember"), strings.Contains(lower, "inferno"):
		return content.ElementFire
	case strings.Contains(lower, "water"), strings.Contains(lower, "ice"), strings.Contains(lower, "frost"), strings.Contains(lower, "tidal"):
		return content.ElementWater
	case strings.Contains(lower, "earth"), strings.Contains(lower, "stone"), strings.Contains(lower, "quake"):
		return content.ElementEarth
	case strings.Contains(lower, "air"), strings.Contains(lower, "wind"), strings.Contains(lower, "storm"), strings.Contains(lower, "lightning"):
		return content.ElementAir
	default:
		return content.ElementPhysical
	}
}

func abyssSkillElement(skill content.Skill, equipped map[content.GearSlot]content.Gear) content.Element {
	if skill.Element != "" {
		return skill.Element
	}
	if element := abyssElementForSkillName(skill.Name); element != content.ElementPhysical {
		return element
	}
	if weapon, ok := equipped[content.SlotMainHand]; ok && weapon.Element != "" {
		return weapon.Element
	}
	return content.ElementPhysical
}

func abyssElementReaction(previous, current content.Element) string {
	if previous == "" || current == "" || previous == current || previous == content.ElementPhysical || current == content.ElementPhysical {
		return ""
	}
	pair := map[content.Element]bool{previous: true, current: true}
	switch {
	case pair[content.ElementFire] && pair[content.ElementWater]:
		return "Steam Burst"
	case pair[content.ElementEarth] && pair[content.ElementAir]:
		return "Dust Cyclone"
	case pair[content.ElementFire] && pair[content.ElementAir]:
		return "Firestorm"
	case pair[content.ElementWater] && pair[content.ElementEarth]:
		return "Crushing Mire"
	default:
		return "Elemental Clash"
	}
}

func abyssSkillMasteryKey(uid, skillID string) string {
	return "abyss_skill_mastery_" + uid + "_" + skillID
}

func (b *Bot) loadAbyssSkillMastery(uid string) map[string]int {
	out := map[string]int{}
	prefix := "abyss_skill_mastery_" + uid + "_"
	rows, err := b.DB.Query("SELECT key, value FROM app_meta WHERE key LIKE $1", prefix+"%")
	if err != nil {
		return out
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var key, value string
		if rows.Scan(&key, &value) == nil {
			count, _ := strconv.Atoi(value)
			out[strings.TrimPrefix(key, prefix)] = count
		}
	}
	return out
}

func (b *Bot) recordAbyssSkillUse(uid, skillID string) int {
	var count int
	err := b.DB.QueryRow(`INSERT INTO app_meta (key, value) VALUES ($1, '1')
		ON CONFLICT (key) DO UPDATE SET value=(COALESCE(NULLIF(app_meta.value, ''), '0')::bigint + 1)::text
		RETURNING value::int`, abyssSkillMasteryKey(uid, skillID)).Scan(&count)
	if err != nil {
		return 0
	}
	return count
}

func (b *Bot) useAbyssActiveRelic(au *activeUser, round int, logs *[]string) bool {
	if au.relicCharges <= 0 {
		return false
	}
	relic, ok := au.u.Equipped[content.SlotRelic]
	if !ok {
		return false
	}
	flags := b.loadRunFlags(au.u.UID)
	if flags[abyssRunFlagRelicCharges] <= 0 {
		return false
	}
	flags[abyssRunFlagRelicCharges]--
	if b.saveRunFlags(au.u.UID, flags) != nil {
		return false
	}
	au.relicCharges--
	relicMultiplier := abyssTreeActionMultiplier(au.treeBonus, "relic_skill_power")
	heal := int(float64(au.u.Stats.HP/5) * relicMultiplier)
	au.u.CurrentHP += heal
	if au.u.CurrentHP > au.u.Stats.HP {
		au.u.CurrentHP = au.u.Stats.HP
	}
	au.u.DEFMod *= 1 + 0.5*relicMultiplier
	au.defendingRound = round
	*logs = append(*logs, fmt.Sprintf("🏺 %s invokes %s: restores %d HP and gains Guard. (No charges remain)", au.u.Nickname, relic.Name, heal))
	return true
}

func (s *WebServer) handleAbyssBuildRespec(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()
	if s.rejectDuringLiveCombat(w, uid) {
		return
	}
	var req struct {
		Mutation string `json:"mutation"`
	}
	if readJSON(r, &req) != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}
	run := s.bot.loadAbyssRun(uid)
	if !run.Active || run.Downed {
		writeJSON(w, map[string]any{"ok": false, "error": "no active run to respec"})
		return
	}
	mutation := normalizeAbyssSkillMutation(req.Mutation)
	if mutation != strings.ToLower(strings.TrimSpace(req.Mutation)) {
		writeJSON(w, map[string]any{"ok": false, "error": "unknown skill mutation"})
		return
	}
	flags := s.bot.loadRunFlags(uid)
	if flags[abyssRunFlagBuildRespec] != 0 {
		writeJSON(w, map[string]any{"ok": false, "error": "the one in-run respec is already spent"})
		return
	}
	if flags[abyssRunFlagSkillMutation] == abyssSkillMutations[mutation] {
		writeJSON(w, map[string]any{"ok": false, "error": "that mutation is already active"})
		return
	}
	flags[abyssRunFlagSkillMutation] = abyssSkillMutations[mutation]
	flags[abyssRunFlagBuildRespec] = 1
	if s.bot.saveRunFlags(uid, flags) != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	writeJSON(w, map[string]any{"ok": true, "msg": "Build shifted to " + mutation + ". The run's respec is now spent."})
}
