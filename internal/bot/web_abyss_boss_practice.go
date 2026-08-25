package bot

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"ts3news/internal/content"
)

type abyssBossPracticeResult struct {
	Victory      bool
	Tactic       string
	Rounds       int
	PlayerHP     int
	PlayerMaxHP  int
	BossHP       int
	BossMaxHP    int
	EstimatedDPS int
	Logs         []string
}

func normalizeAbyssPracticeTactic(tactic string) string {
	switch strings.ToLower(strings.TrimSpace(tactic)) {
	case "aggressive", "defensive":
		return strings.ToLower(strings.TrimSpace(tactic))
	default:
		return "balanced"
	}
}

func simulateAbyssBossPractice(name string, depth int, combatRating float64, tactic string) abyssBossPracticeResult {
	tactic = normalizeAbyssPracticeTactic(tactic)
	depth = max(1, depth)
	cr := max(1, int(combatRating))
	playerMaxHP := max(600, cr*18)
	playerHP := playerMaxHP
	bossMaxHP := max(800, depth*120+cr*8)
	bossHP := bossMaxHP
	playerDamage := max(60, cr*2)
	bossDamage := max(12, depth*3+cr/5)
	switch tactic {
	case "aggressive":
		playerDamage = playerDamage * 5 / 4
		bossDamage = bossDamage * 115 / 100
	case "defensive":
		playerDamage = playerDamage * 4 / 5
		bossDamage = bossDamage * 13 / 20
	}

	result := abyssBossPracticeResult{
		Tactic: tactic, PlayerHP: playerHP, PlayerMaxHP: playerMaxHP,
		BossHP: bossHP, BossMaxHP: bossMaxHP,
		Logs: []string{fmt.Sprintf("R0 · %s rehearsal loaded at recorded depth %d · %s tactic", name, depth, tactic)},
	}
	totalDamage := 0
	phase50, phase25 := false, false
	for round := 1; round <= combatBossEnrageRound(true) && playerHP > 0 && bossHP > 0; round++ {
		damage := playerDamage
		if round%5 == 0 {
			damage = damage * 5 / 4
		}
		damage = min(damage, bossHP)
		bossHP -= damage
		totalDamage += damage
		result.Logs = append(result.Logs, fmt.Sprintf("R%d · you deal %d · boss %d/%d HP", round, damage, bossHP, bossMaxHP))
		if !phase50 && bossHP > 0 && bossHP*2 <= bossMaxHP {
			phase50 = true
			result.Logs = append(result.Logs, fmt.Sprintf("R%d · 50%% phase: %s", round, abyssBossTip(name)))
		}
		if !phase25 && bossHP > 0 && bossHP*4 <= bossMaxHP {
			phase25 = true
			result.Logs = append(result.Logs, fmt.Sprintf("R%d · 25%% phase: stagger before the enrage sequence", round))
		}
		result.Rounds = round
		if bossHP <= 0 {
			break
		}
		incoming := bossDamage
		if round == 1 {
			incoming /= 2
		}
		if combatBossShouldEnrage(round, true) {
			incoming *= 2
		}
		incoming = min(max(1, incoming), playerHP)
		playerHP -= incoming
		result.Logs = append(result.Logs, fmt.Sprintf("R%d · %s retaliates for %d · you %d/%d HP", round, name, incoming, playerHP, playerMaxHP))
	}
	result.Victory = bossHP <= 0
	result.PlayerHP = playerHP
	result.BossHP = bossHP
	if result.Rounds > 0 {
		result.EstimatedDPS = totalDamage / result.Rounds
	}
	outcome := "defeat"
	if result.Victory {
		outcome = "victory"
	}
	result.Logs = append(result.Logs, fmt.Sprintf("Result · %s in %d rounds · sandbox reset with no rewards or state changes",
		outcome, result.Rounds))
	return result
}

func (s *WebServer) handleAbyssBossPractice(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	var req struct {
		Name   string `json:"name"`
		Tactic string `json:"tactic"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "invalid request"})
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" || len(req.Name) > 160 {
		writeJSON(w, map[string]any{"ok": false, "error": "invalid boss name"})
		return
	}
	var depth int
	if err := s.bot.DB.QueryRow(
		"SELECT COALESCE(MAX(depth),0) FROM abyss_boss_kills WHERE client_uid=$1 AND boss_name=$2",
		uid, req.Name,
	).Scan(&depth); err != nil || depth <= 0 {
		writeJSON(w, map[string]any{"ok": false, "error": "defeat this boss once to unlock its practice drill"})
		return
	}
	mob := &content.Mob{Name: req.Name, Type: content.MobBoss}
	practice := simulateAbyssBossPractice(req.Name, depth, s.bot.abyssPlayerCR(uid), req.Tactic)
	writeJSON(w, map[string]any{
		"ok": true, "boss": req.Name, "role": abyssEnemyRole(mob),
		"faction": abyssEnemyFaction(mob), "pattern": abyssEnemyPattern(mob),
		"rewards": false, "victory": practice.Victory, "tactic": practice.Tactic,
		"rounds": practice.Rounds, "estimated_dps": practice.EstimatedDPS,
		"player_hp": practice.PlayerHP, "player_max_hp": practice.PlayerMaxHP,
		"boss_hp": practice.BossHP, "boss_max_hp": practice.BossMaxHP,
		"practice_log": practice.Logs,
		"drill": []string{
			"Opening: read the intent and establish Guard before committing resources.",
			"At 50%: hold an Ultimate to interrupt the telegraphed summon.",
			"At 25%: break the stagger bar before the enrage sequence resolves.",
		},
	})
}
