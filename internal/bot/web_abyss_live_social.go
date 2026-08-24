package bot

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"ts3news/internal/content"
)

const abyssLiveDisconnectReserve = 90 * time.Second

type abyssSocialPreferences struct {
	Role       string `json:"role"`
	Pace       string `json:"pace"`
	Difficulty string `json:"difficulty"`
	AllowRisky bool   `json:"allow_risky"`
}

type abyssLiveSocialSignal struct {
	Name     string    `json:"name"`
	Kind     string    `json:"kind"`
	TargetID string    `json:"target_id,omitempty"`
	Message  string    `json:"message"`
	At       time.Time `json:"at"`
}

type abyssLiveContribution struct {
	Name    string `json:"name"`
	Actions int    `json:"actions"`
	Ready   int    `json:"ready_rounds"`
	Manual  int    `json:"manual_actions"`
}

type abyssLiveMemberPresence struct {
	Name          string    `json:"name"`
	Role          string    `json:"role"`
	Connected     bool      `json:"connected"`
	ReservedUntil time.Time `json:"reserved_until,omitempty"`
}

type abyssLiveSocialSnapshot struct {
	PreferredRole      string                    `json:"preferred_role"`
	PartyTactic        string                    `json:"party_tactic"`
	TacticVotes        map[string]int            `json:"tactic_votes"`
	LootRule           string                    `json:"loot_rule"`
	Signals            []abyssLiveSocialSignal   `json:"signals,omitempty"`
	Contributions      []abyssLiveContribution   `json:"contributions,omitempty"`
	Members            []abyssLiveMemberPresence `json:"members,omitempty"`
	ComboOpportunities []string                  `json:"combo_opportunities,omitempty"`
	ReviveYes          int                       `json:"revive_yes,omitempty"`
	ReviveNeeded       int                       `json:"revive_needed,omitempty"`
	AutoResolve        bool                      `json:"auto_resolve,omitempty"`
	Spectating         bool                      `json:"spectating,omitempty"`
}

type abyssLiveSocialState struct {
	preferences  map[string]abyssSocialPreferences
	tacticVotes  map[string]string
	signals      []abyssLiveSocialSignal
	lastSeen     map[string]time.Time
	actionCounts map[string]int
	manualCounts map[string]int
	readyCounts  map[string]int
	reviveVotes  map[string]bool
	abandonVotes map[string]bool
	readyAwarded map[int]bool
	comboAwarded map[int]bool
	partyTactic  string
	lootRule     string
	autoResolve  bool
}

func (c *abyssLiveCombat) ensureSocialLocked() {
	if c.social.preferences == nil {
		c.social.preferences = map[string]abyssSocialPreferences{}
	}
	if c.social.tacticVotes == nil {
		c.social.tacticVotes = map[string]string{}
	}
	if c.social.lastSeen == nil {
		c.social.lastSeen = map[string]time.Time{}
	}
	if c.social.actionCounts == nil {
		c.social.actionCounts = map[string]int{}
	}
	if c.social.manualCounts == nil {
		c.social.manualCounts = map[string]int{}
	}
	if c.social.readyCounts == nil {
		c.social.readyCounts = map[string]int{}
	}
	if c.social.reviveVotes == nil {
		c.social.reviveVotes = map[string]bool{}
	}
	if c.social.abandonVotes == nil {
		c.social.abandonVotes = map[string]bool{}
	}
	if c.social.readyAwarded == nil {
		c.social.readyAwarded = map[int]bool{}
	}
	if c.social.comboAwarded == nil {
		c.social.comboAwarded = map[int]bool{}
	}
	if c.social.lootRule == "" {
		c.social.lootRule = "owner"
	}
}

func normalizeAbyssSocialPreferences(pref abyssSocialPreferences) abyssSocialPreferences {
	switch pref.Role {
	case "tank", "damage", "support", "flex":
	default:
		pref.Role = "flex"
	}
	switch pref.Pace {
	case "fast", "standard", "deliberate":
	default:
		pref.Pace = "standard"
	}
	switch pref.Difficulty {
	case "normal", "nightmare", "hell", "insanity", "any":
	default:
		pref.Difficulty = "any"
	}
	return pref
}

func abyssSocialPreferenceKey(uid string) string { return "abyss_social_pref_" + uid }

func (b *Bot) loadAbyssSocialPreferences(uid string) abyssSocialPreferences {
	pref := abyssSocialPreferences{}
	var raw string
	if b.DB.QueryRow("SELECT value FROM app_meta WHERE key=$1", abyssSocialPreferenceKey(uid)).Scan(&raw) == nil {
		_ = json.Unmarshal([]byte(raw), &pref)
	}
	return normalizeAbyssSocialPreferences(pref)
}

func (b *Bot) saveAbyssSocialPreferences(uid string, pref abyssSocialPreferences) error {
	pref = normalizeAbyssSocialPreferences(pref)
	encoded, err := json.Marshal(pref)
	if err != nil {
		return err
	}
	_, err = b.DB.Exec(`INSERT INTO app_meta (key, value) VALUES ($1,$2)
		ON CONFLICT (key) DO UPDATE SET value=EXCLUDED.value`, abyssSocialPreferenceKey(uid), string(encoded))
	return err
}

func newAbyssLiveSocialState(server *WebServer, participants map[string]bool, lootRule string) abyssLiveSocialState {
	state := abyssLiveSocialState{
		preferences: make(map[string]abyssSocialPreferences, len(participants)),
		tacticVotes: make(map[string]string), lastSeen: make(map[string]time.Time, len(participants)),
		actionCounts: make(map[string]int), manualCounts: make(map[string]int), readyCounts: make(map[string]int),
		reviveVotes: make(map[string]bool), abandonVotes: make(map[string]bool),
		readyAwarded: make(map[int]bool), comboAwarded: make(map[int]bool),
		lootRule: normalizeAbyssPartyLootRule(lootRule),
	}
	for uid := range participants {
		state.preferences[uid] = server.bot.loadAbyssSocialPreferences(uid)
		state.lastSeen[uid] = time.Now()
	}
	return state
}

func normalizeAbyssPartyLootRule(rule string) string {
	switch strings.ToLower(strings.TrimSpace(rule)) {
	case "round_robin", "need_before_greed":
		return strings.ToLower(strings.TrimSpace(rule))
	default:
		return "owner"
	}
}

func abyssPartyLootRuleID(rule string) int64 {
	switch normalizeAbyssPartyLootRule(rule) {
	case "round_robin":
		return 2
	case "need_before_greed":
		return 3
	default:
		return 1
	}
}

func abyssPartyLootRuleFromID(id int64) string {
	switch id {
	case 2:
		return "round_robin"
	case 3:
		return "need_before_greed"
	default:
		return "owner"
	}
}

func seedAbyssSocialRunFlagsInTx(tx *sql.Tx, uid, lootRule string) error {
	flags, err := loadAbyssRunFlagsInTx(tx, uid)
	if err != nil {
		return err
	}
	flags["party_loot_rule"] = abyssPartyLootRuleID(lootRule)
	return saveAbyssRunFlagsInTx(tx, uid, flags)
}

func (c *abyssLiveCombat) touchMember(uid string) {
	c.mu.Lock()
	c.ensureSocialLocked()
	if c.participants[uid] {
		c.social.lastSeen[uid] = time.Now()
	}
	c.mu.Unlock()
}

func (c *abyssLiveCombat) socialSnapshotLocked(uid string) abyssLiveSocialSnapshot {
	c.ensureSocialLocked()
	voteCounts := map[string]int{}
	for _, tactic := range c.social.tacticVotes {
		voteCounts[tactic]++
	}
	uidNames := make(map[string]string, len(c.allies))
	spectating := false
	for _, ally := range c.allies {
		memberUID := strings.TrimPrefix(ally.ID, "ally:")
		uidNames[memberUID] = ally.Name
		if memberUID == uid && ally.HP <= 0 {
			spectating = true
		}
	}
	keys := make([]string, 0, len(c.participants))
	for memberUID := range c.participants {
		keys = append(keys, memberUID)
	}
	sort.Strings(keys)
	contributions := make([]abyssLiveContribution, 0, len(keys))
	members := make([]abyssLiveMemberPresence, 0, len(keys))
	now := time.Now()
	for _, memberUID := range keys {
		name := uidNames[memberUID]
		if name == "" {
			name = "Adventurer"
		}
		contributions = append(contributions, abyssLiveContribution{
			Name: name, Actions: c.social.actionCounts[memberUID], Ready: c.social.readyCounts[memberUID], Manual: c.social.manualCounts[memberUID],
		})
		lastSeen := c.social.lastSeen[memberUID]
		connected := now.Sub(lastSeen) <= 10*time.Second
		presence := abyssLiveMemberPresence{Name: name, Role: c.social.preferences[memberUID].Role, Connected: connected}
		if !connected && now.Sub(lastSeen) < abyssLiveDisconnectReserve {
			presence.ReservedUntil = lastSeen.Add(abyssLiveDisconnectReserve)
		}
		members = append(members, presence)
	}
	reviveYes := 0
	for _, yes := range c.social.reviveVotes {
		if yes {
			reviveYes++
		}
	}
	return abyssLiveSocialSnapshot{
		PreferredRole: c.social.preferences[uid].Role, PartyTactic: c.social.partyTactic,
		TacticVotes: voteCounts, LootRule: c.social.lootRule, Signals: append([]abyssLiveSocialSignal{}, c.social.signals...),
		Contributions: contributions, Members: members, ComboOpportunities: c.comboOpportunitiesLocked(),
		ReviveYes: reviveYes, ReviveNeeded: len(c.participants)/2 + 1, AutoResolve: c.social.autoResolve, Spectating: spectating,
	}
}

func (c *abyssLiveCombat) comboOpportunitiesLocked() []string {
	ready := 0
	for memberUID := range c.participants {
		for _, option := range c.options[memberUID] {
			if option.Kind == "skill" && len(option.Tags) > 0 && option.Cooldown == 0 {
				ready++
				break
			}
		}
	}
	if ready >= 2 {
		return []string{"Two combo-ready skills are available — focus the same enemy for a chain bonus."}
	}
	return nil
}

func (c *abyssLiveCombat) setSocialPreference(uid string, pref abyssSocialPreferences) error {
	pref = normalizeAbyssSocialPreferences(pref)
	if err := c.server.bot.saveAbyssSocialPreferences(uid, pref); err != nil {
		return err
	}
	c.mu.Lock()
	c.ensureSocialLocked()
	c.social.preferences[uid] = pref
	c.social.lastSeen[uid] = time.Now()
	c.version++
	c.mu.Unlock()
	return nil
}

func (c *abyssLiveCombat) votePartyTactic(uid, tactic string) error {
	tactic = normalizeAbyssTactic(tactic)
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ensureSocialLocked()
	if !c.participants[uid] {
		return errAbyssLiveNotFound
	}
	if tactic == "aggressive" && !c.social.preferences[uid].AllowRisky {
		return fmt.Errorf("enable risky party decisions before voting aggressive")
	}
	c.social.tacticVotes[uid] = tactic
	counts := map[string]int{}
	for _, vote := range c.social.tacticVotes {
		counts[vote]++
	}
	best, bestN := "", 0
	for _, candidate := range []string{"balanced", "defensive", "conserve_items", "aggressive"} {
		if counts[candidate] > bestN {
			best, bestN = candidate, counts[candidate]
		}
	}
	if bestN > len(c.participants)/2 {
		c.social.partyTactic = best
	}
	c.version++
	return nil
}

func (c *abyssLiveCombat) addSocialSignal(uid, kind, targetID, message string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ensureSocialLocked()
	if !c.participants[uid] {
		return errAbyssLiveNotFound
	}
	name := "Adventurer"
	for _, ally := range c.allies {
		if ally.ID == "ally:"+uid {
			name = ally.Name
			break
		}
	}
	switch kind {
	case "target":
		valid := false
		for _, enemy := range c.enemies {
			valid = valid || enemy.ID == targetID
		}
		if !valid {
			return fmt.Errorf("choose a living enemy to ping")
		}
		message = "Focus this target"
	case "danger":
		message = "Danger — prepare defenses"
	case "emote":
		message = map[string]string{"cheer": "✨ Ready!", "thanks": "🙏 Thanks!", "help": "🆘 Need help!", "nice": "👏 Nice combo!"}[message]
		if message == "" {
			return fmt.Errorf("unknown emote")
		}
	default:
		return fmt.Errorf("unknown signal")
	}
	c.social.signals = append(c.social.signals, abyssLiveSocialSignal{Name: name, Kind: kind, TargetID: targetID, Message: message, At: time.Now().UTC()})
	if len(c.social.signals) > 12 {
		c.social.signals = c.social.signals[len(c.social.signals)-12:]
	}
	c.version++
	return nil
}

func (c *abyssLiveCombat) voteRevive(uid string, yes bool) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ensureSocialLocked()
	if !c.participants[uid] {
		return errAbyssLiveNotFound
	}
	c.social.reviveVotes[uid] = yes
	c.version++
	return nil
}

func (c *abyssLiveCombat) reviveApproved() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ensureSocialLocked()
	if len(c.participants) <= 1 {
		return true
	}
	yes := 0
	for _, vote := range c.social.reviveVotes {
		if vote {
			yes++
		}
	}
	return yes > len(c.participants)/2
}

func (c *abyssLiveCombat) voteAbandon(uid string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ensureSocialLocked()
	if !c.participants[uid] {
		return errAbyssLiveNotFound
	}
	c.social.abandonVotes[uid] = true
	if len(c.social.abandonVotes) > len(c.participants)/2 {
		c.social.autoResolve = true
		c.deadline = time.Now()
		select {
		case c.deadlineSignal <- struct{}{}:
		default:
		}
	}
	c.version++
	return nil
}

func (c *abyssLiveCombat) timeoutActionLocked(uid string) abyssLiveAction {
	switch normalizeLivePolicy(c.policies[uid]).TimeoutAction {
	case "defend":
		return abyssLiveAction{Kind: "defend", Round: c.round, Automatic: true}
	case "attack":
		if len(c.enemies) > 0 {
			return abyssLiveAction{Kind: "attack", TargetID: c.selectEnemyLocked(uid, "attack").ID, Round: c.round, Automatic: true}
		}
	}
	return c.bestActionLocked(uid)
}

func mentorScaledStats(mentor, host content.Stats) content.Stats {
	capStat := func(value, hostValue int) int {
		capValue := hostValue * 6 / 5
		if hostValue > 0 && value > capValue {
			return capValue
		}
		return value
	}
	mentor.HP = capStat(mentor.HP, host.HP)
	mentor.MNA = capStat(mentor.MNA, host.MNA)
	mentor.STR = capStat(mentor.STR, host.STR)
	mentor.DEF = capStat(mentor.DEF, host.DEF)
	mentor.SPD = capStat(mentor.SPD, host.SPD)
	mentor.INT = capStat(mentor.INT, host.INT)
	mentor.CRT = capStat(mentor.CRT, host.CRT)
	mentor.DGE = capStat(mentor.DGE, host.DGE)
	mentor.LCK = capStat(mentor.LCK, host.LCK)
	return mentor
}

func (s *WebServer) handleAbyssCombatSocial(w http.ResponseWriter, r *http.Request, uid string) {
	if !s.abyssFeatures.enabled("social", uid) {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	c, ok := s.liveCombatForUID(uid)
	if !ok {
		writeJSON(w, map[string]any{"ok": false, "error": "no active combat"})
		return
	}
	var req struct {
		Action      string                 `json:"action"`
		Value       string                 `json:"value"`
		TargetID    string                 `json:"target_id"`
		Vote        bool                   `json:"vote"`
		Preferences abyssSocialPreferences `json:"preferences"`
	}
	if readJSON(r, &req) != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}
	var err error
	switch req.Action {
	case "preferences":
		err = c.setSocialPreference(uid, req.Preferences)
	case "tactic_vote":
		err = c.votePartyTactic(uid, req.Value)
	case "target_ping", "danger_ping":
		err = c.addSocialSignal(uid, strings.TrimSuffix(req.Action, "_ping"), req.TargetID, "")
	case "emote":
		err = c.addSocialSignal(uid, "emote", "", req.Value)
	case "revive_vote":
		err = c.voteRevive(uid, req.Vote)
	case "abandon_vote":
		err = c.voteAbandon(uid)
	default:
		err = fmt.Errorf("unknown social action")
	}
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	c.persistOrLog("persisting social combat state")
	writeJSON(w, c.snapshotFor(uid))
}

// abyssReplayCode is deliberately anonymous: it contains only round count and
// enemy labels. The player decides where to paste the code; no public endpoint
// stores or exposes raw combat snapshots, player names, IDs, actions, or logs.
type abyssReplayCode struct {
	Version int      `json:"v"`
	Rounds  int      `json:"rounds"`
	Enemies []string `json:"enemies"`
}

func encodeAbyssReplayCode(snapshot abyssLiveSnapshot) (string, error) {
	code := abyssReplayCode{Version: 1, Rounds: snapshot.Round}
	for _, enemy := range snapshot.Enemies {
		code.Enemies = append(code.Enemies, enemy.Name)
	}
	encoded, err := json.Marshal(code)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(encoded), nil
}

func decodeAbyssReplayCode(code string) (abyssReplayCode, error) {
	if len(code) == 0 || len(code) > 2048 {
		return abyssReplayCode{}, fmt.Errorf("invalid replay code")
	}
	raw, err := base64.RawURLEncoding.DecodeString(code)
	if err != nil {
		return abyssReplayCode{}, fmt.Errorf("invalid replay code")
	}
	var replay abyssReplayCode
	if json.Unmarshal(raw, &replay) != nil || replay.Version != 1 || replay.Rounds < 1 || replay.Rounds > 10000 || len(replay.Enemies) > 64 {
		return abyssReplayCode{}, fmt.Errorf("invalid replay code")
	}
	return replay, nil
}

func (s *WebServer) handleAbyssReplayCode(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		SessionID string `json:"session_id"`
		Code      string `json:"code"`
		Ghost     bool   `json:"ghost"`
	}
	if readJSON(r, &req) != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}
	if req.Ghost {
		replay, err := decodeAbyssReplayCode(req.Code)
		if err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		_, err = s.bot.DB.Exec(`INSERT INTO app_meta (key,value) VALUES ($1,$2)
			ON CONFLICT (key) DO UPDATE SET value=EXCLUDED.value`, "abyss_ghost_rounds_"+uid, fmt.Sprint(replay.Rounds))
		writeJSON(w, map[string]any{"ok": err == nil, "msg": "Ghost-party challenge armed for your next live fight."})
		return
	}
	var owned, raw string
	if s.bot.DB.QueryRow("SELECT value FROM app_meta WHERE key=$1", "abyss_live_replay_user_"+uid+"_"+req.SessionID).Scan(&owned) != nil ||
		s.bot.DB.QueryRow("SELECT value FROM app_meta WHERE key=$1", "abyss_live_replay_session_"+req.SessionID).Scan(&raw) != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "replay not found"})
		return
	}
	var archive abyssLiveReplayArchive
	if json.Unmarshal([]byte(raw), &archive) != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "replay unavailable"})
		return
	}
	code, err := encodeAbyssReplayCode(archive.State.Snapshot)
	writeJSON(w, map[string]any{"ok": err == nil, "code": code})
}

func (b *Bot) resolveAbyssGhostChallenge(uid string, rounds int) {
	key := "abyss_ghost_rounds_" + uid
	var raw string
	if b.DB.QueryRow("SELECT value FROM app_meta WHERE key=$1", key).Scan(&raw) != nil {
		return
	}
	var target int
	if _, err := fmt.Sscan(raw, &target); err == nil && target > 0 && rounds <= target {
		b.recordAbyssProgression(uid, "ghost_party")
	}
	_, _ = b.DB.Exec("DELETE FROM app_meta WHERE key=$1", key)
}

func (b *Bot) incrementCommunityExpedition() {
	week := abyssCurrentWeek(time.Now())
	_, _ = b.DB.Exec(`INSERT INTO app_meta (key,value) VALUES ($1,'1')
		ON CONFLICT (key) DO UPDATE SET value=(COALESCE(NULLIF(app_meta.value,''),'0')::bigint+1)::text`, "abyss_community_expedition_"+week)
}

func (b *Bot) communityExpeditionStatus() map[string]any {
	week := abyssCurrentWeek(time.Now())
	var raw string
	_ = b.DB.QueryRow("SELECT value FROM app_meta WHERE key=$1", "abyss_community_expedition_"+week).Scan(&raw)
	var floors int64
	_, _ = fmt.Sscan(raw, &floors)
	return map[string]any{"Week": week, "Floors": floors, "Target": int64(1000)}
}
