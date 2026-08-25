package bot

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"ts3news/internal/content"
)

const (
	abyssParagonRanksPerPrestige = 10
	abyssParagonMaxRank          = 20
	abyssBestiaryTalentMaxRank   = 5
	abyssBestiaryKillsPerRank    = 10
)

type abyssParagonNode struct {
	Key    string  `json:"key"`
	Label  string  `json:"label"`
	Effect string  `json:"effect"`
	Pct    float64 `json:"pct"`
	Rank   int     `json:"rank"`
	Max    int     `json:"max"`
}

type abyssParagonView struct {
	Unlocked bool               `json:"unlocked"`
	Prestige int                `json:"prestige"`
	Points   int                `json:"points"`
	Used     int                `json:"used"`
	Nodes    []abyssParagonNode `json:"nodes"`
}

type abyssBestiaryTalent struct {
	Family    string `json:"family"`
	Label     string `json:"label"`
	Kills     int    `json:"kills"`
	Spent     int    `json:"spent"`
	Available int    `json:"available"`
	Rank      int    `json:"rank"`
	Max       int    `json:"max"`
	NextCost  int    `json:"next_cost"`
}

func abyssBestiaryKillsSpent(rank int) int {
	if rank <= 0 {
		return 0
	}
	return rank * (rank + 1) / 2 * abyssBestiaryKillsPerRank
}

func abyssBestiaryRanksSpent(ranks map[string]int) int {
	spent := 0
	for _, rank := range ranks {
		spent += abyssBestiaryKillsSpent(rank)
	}
	return spent
}

func abyssFullSetCount(counts map[string]int) int {
	full := 0
	for _, count := range counts {
		if count >= 6 {
			full++
		}
	}
	return full
}

var abyssParagonCatalog = []abyssParagonNode{
	{Key: "might", Label: "Might", Effect: "str_pct", Pct: 0.001},
	{Key: "vitality", Label: "Vitality", Effect: "hp_pct", Pct: 0.001},
	{Key: "guard", Label: "Guard", Effect: "def_pct", Pct: 0.001},
	{Key: "haste", Label: "Haste", Effect: "spd_pct", Pct: 0.001},
	{Key: "insight", Label: "Insight", Effect: "int_pct", Pct: 0.001},
	{Key: "fortune", Label: "Fortune", Effect: "loot_find", Pct: 0.001},
	{Key: "convergence", Label: "Convergence", Effect: "skill_damage", Pct: 0.001},
}

var abyssBestiaryTalentCatalog = []abyssBestiaryTalent{
	{Family: string(content.MobCommon), Label: "Common Hunter"},
	{Family: string(content.MobEliteMinion), Label: "Minion Breaker"},
	{Family: string(content.MobElite), Label: "Elite Hunter"},
	{Family: string(content.MobMiniboss), Label: "Miniboss Slayer"},
	{Family: string(content.MobBoss), Label: "Boss Slayer"},
	{Family: string(content.MobLegendary), Label: "Legend Breaker"},
}

func abyssParagonKey(uid string) string         { return "abyss_paragon_" + uid }
func abyssBestiaryTalentsKey(uid string) string { return "abyss_bestiary_talents_" + uid }

func decodeAbyssRankMap(stored string, maxRank int) (map[string]int, error) {
	ranks := map[string]int{}
	if stored == "" {
		return ranks, nil
	}
	if err := json.Unmarshal([]byte(stored), &ranks); err != nil {
		return nil, err
	}
	for key, rank := range ranks {
		if rank < 0 || rank > maxRank {
			return nil, fmt.Errorf("invalid rank for %s", key)
		}
	}
	return ranks, nil
}

func (b *Bot) loadAbyssRankMap(key string, maxRank int) map[string]int {
	var stored string
	_ = b.DB.QueryRow("SELECT value FROM app_meta WHERE key=$1", key).Scan(&stored)
	ranks, err := decodeAbyssRankMap(stored, maxRank)
	if err != nil {
		return map[string]int{}
	}
	return ranks
}

func (b *Bot) abyssParagonView(uid string) abyssParagonView {
	var prestige int
	_ = b.DB.QueryRow("SELECT abyss_prestige FROM users WHERE client_uid=$1", uid).Scan(&prestige)
	ranks := b.loadAbyssRankMap(abyssParagonKey(uid), abyssParagonMaxRank)
	view := abyssParagonView{Unlocked: prestige > 0, Prestige: prestige, Points: prestige * abyssParagonRanksPerPrestige}
	for _, definition := range abyssParagonCatalog {
		node := definition
		node.Rank, node.Max = ranks[node.Key], abyssParagonMaxRank
		view.Used += node.Rank
		view.Nodes = append(view.Nodes, node)
	}
	return view
}

func (b *Bot) abyssBestiaryTalentViews(uid string) []abyssBestiaryTalent {
	ranks := b.loadAbyssRankMap(abyssBestiaryTalentsKey(uid), abyssBestiaryTalentMaxRank)
	var bossKills int
	_ = b.DB.QueryRow(`SELECT COALESCE(SUM(kills),0) FROM abyss_bestiary
		WHERE client_uid=$1 AND mob_family=$2`, uid, string(content.MobBoss)).Scan(&bossKills)
	spent := abyssBestiaryRanksSpent(ranks)
	available := max(bossKills-spent, 0)
	views := make([]abyssBestiaryTalent, 0, len(abyssBestiaryTalentCatalog))
	for _, definition := range abyssBestiaryTalentCatalog {
		view := definition
		view.Rank, view.Max = ranks[view.Family], abyssBestiaryTalentMaxRank
		view.Kills, view.Spent, view.Available = bossKills, spent, available
		if view.Rank < view.Max {
			view.NextCost = (view.Rank + 1) * abyssBestiaryKillsPerRank
		}
		views = append(views, view)
	}
	return views
}

func (b *Bot) applyAbyssMasteryBonuses(uid string, bonus *content.TreeBonus) {
	if bonus.Pct == nil {
		bonus.Pct = map[string]float64{}
	}
	paragon := b.loadAbyssRankMap(abyssParagonKey(uid), abyssParagonMaxRank)
	for _, node := range abyssParagonCatalog {
		bonus.Pct[node.Effect] += float64(paragon[node.Key]) * node.Pct
	}
	bestiary := b.loadAbyssRankMap(abyssBestiaryTalentsKey(uid), abyssBestiaryTalentMaxRank)
	for family, rank := range bestiary {
		bonus.Pct["bestiary_damage_"+strings.ToLower(family)] += float64(rank) * 0.01
	}
}

func (s *WebServer) handleAbyssTreeParagon(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()
	var request struct {
		Key string `json:"key"`
	}
	if readJSON(r, &request) != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}
	valid := false
	for _, node := range abyssParagonCatalog {
		valid = valid || node.Key == request.Key
	}
	if !valid {
		writeJSON(w, map[string]any{"ok": false, "error": "unknown paragon node"})
		return
	}
	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()
	var prestige int
	if err := tx.QueryRow("SELECT abyss_prestige FROM users WHERE client_uid=$1 FOR UPDATE", uid).Scan(&prestige); err != nil || prestige < 1 {
		writeJSON(w, map[string]any{"ok": false, "error": "paragon unlocks after the first Abyss prestige"})
		return
	}
	key := abyssParagonKey(uid)
	var stored string
	err = tx.QueryRow("SELECT value FROM app_meta WHERE key=$1 FOR UPDATE", key).Scan(&stored)
	if err != nil && err != sql.ErrNoRows {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	ranks, err := decodeAbyssRankMap(stored, abyssParagonMaxRank)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "stored paragon board is invalid"})
		return
	}
	used := 0
	for _, rank := range ranks {
		used += rank
	}
	if used >= prestige*abyssParagonRanksPerPrestige {
		writeJSON(w, map[string]any{"ok": false, "error": "no paragon points available"})
		return
	}
	if ranks[request.Key] >= abyssParagonMaxRank {
		writeJSON(w, map[string]any{"ok": false, "error": "paragon node is maxed"})
		return
	}
	ranks[request.Key]++
	payload, _ := json.Marshal(ranks)
	if _, err := tx.Exec(`INSERT INTO app_meta (key,value) VALUES ($1,$2)
		ON CONFLICT (key) DO UPDATE SET value=EXCLUDED.value`, key, string(payload)); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	writeJSON(w, map[string]any{"ok": true, "view": s.bot.abyssParagonView(uid), "msg": "Paragon rank allocated: +0.1%."})
}

func (s *WebServer) handleAbyssTreeBestiaryTalent(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()
	var request struct {
		Family string `json:"family"`
	}
	if readJSON(r, &request) != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}
	valid := false
	for _, node := range abyssBestiaryTalentCatalog {
		valid = valid || node.Family == request.Family
	}
	if !valid {
		writeJSON(w, map[string]any{"ok": false, "error": "unknown bestiary family"})
		return
	}
	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()
	var kills int
	if err := tx.QueryRow(`SELECT COALESCE(SUM(kills),0) FROM abyss_bestiary
		WHERE client_uid=$1 AND mob_family=$2`, uid, string(content.MobBoss)).Scan(&kills); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	key := abyssBestiaryTalentsKey(uid)
	var stored string
	err = tx.QueryRow("SELECT value FROM app_meta WHERE key=$1 FOR UPDATE", key).Scan(&stored)
	if err != nil && err != sql.ErrNoRows {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	ranks, err := decodeAbyssRankMap(stored, abyssBestiaryTalentMaxRank)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "stored bestiary talents are invalid"})
		return
	}
	if ranks[request.Family] >= abyssBestiaryTalentMaxRank {
		writeJSON(w, map[string]any{"ok": false, "error": "bestiary talent is maxed"})
		return
	}
	required := (ranks[request.Family] + 1) * abyssBestiaryKillsPerRank
	available := max(kills-abyssBestiaryRanksSpent(ranks), 0)
	if available < required {
		writeJSON(w, map[string]any{"ok": false, "error": fmt.Sprintf("need %d available boss kills; %d remain", required, available)})
		return
	}
	ranks[request.Family]++
	payload, _ := json.Marshal(ranks)
	if _, err := tx.Exec(`INSERT INTO app_meta (key,value) VALUES ($1,$2)
		ON CONFLICT (key) DO UPDATE SET value=EXCLUDED.value`, key, string(payload)); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	writeJSON(w, map[string]any{"ok": true, "talents": s.bot.abyssBestiaryTalentViews(uid), "msg": "+1% damage against " + request.Family + " enemies."})
}
