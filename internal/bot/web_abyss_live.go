package bot

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"ts3news/internal/content"
)

const (
	abyssLiveRoundTime  = 4 * time.Second
	abyssLiveHybridTime = 9 * time.Second
)

var (
	errAbyssLiveNotFound = errors.New("live combat not found")
	errAbyssLiveStale    = errors.New("live combat round is stale")
)

type abyssLiveAction struct {
	SessionID      string `json:"session_id,omitempty"`
	Kind           string `json:"kind"`
	AbilityID      string `json:"ability_id,omitempty"`
	TargetID       string `json:"target_id,omitempty"`
	Round          int    `json:"round"`
	IdempotencyKey string `json:"idempotency_key,omitempty"`
	Automatic      bool   `json:"automatic,omitempty"`
}

type abyssLiveOption struct {
	Kind        string  `json:"kind"`
	ID          string  `json:"id,omitempty"`
	Name        string  `json:"name"`
	Description string  `json:"description,omitempty"`
	Target      string  `json:"target"`
	Mana        int     `json:"mana,omitempty"`
	Cooldown    int     `json:"cooldown,omitempty"`
	Count       int     `json:"count,omitempty"`
	Power       float64 `json:"power,omitempty"`
}

type abyssLiveCombatantView struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	HP       int    `json:"hp"`
	MaxHP    int    `json:"max_hp"`
	Mana     int    `json:"mana,omitempty"`
	MaxMana  int    `json:"max_mana,omitempty"`
	Ready    bool   `json:"ready,omitempty"`
	IsPlayer bool   `json:"is_player,omitempty"`
	IsSelf   bool   `json:"is_self,omitempty"`
}

type abyssLiveSnapshot struct {
	OK            bool                     `json:"ok"`
	SessionID     string                   `json:"session_id"`
	OwnerUID      string                   `json:"-"`
	Phase         string                   `json:"phase"`
	Round         int                      `json:"round"`
	Version       int64                    `json:"version"`
	Deadline      time.Time                `json:"deadline,omitempty"`
	PauseReason   string                   `json:"pause_reason,omitempty"`
	Tactic        string                   `json:"tactic"`
	Allies        []abyssLiveCombatantView `json:"allies"`
	Enemies       []abyssLiveCombatantView `json:"enemies"`
	Options       []abyssLiveOption        `json:"options"`
	Queued        *abyssLiveAction         `json:"queued,omitempty"`
	RecentLogs    []string                 `json:"recent_logs"`
	Result        map[string]any           `json:"result,omitempty"`
	PreviousDepth int                      `json:"previous_depth,omitempty"`
}

type abyssLiveCombat struct {
	mu sync.Mutex

	server        *WebServer
	id            string
	ownerUID      string
	participants  map[string]bool
	tactics       map[string]string
	phase         string
	round         int
	version       int64
	deadline      time.Time
	pauseReason   string
	allies        []abyssLiveCombatantView
	enemies       []abyssLiveCombatantView
	options       map[string][]abyssLiveOption
	queued        map[string]abyssLiveAction
	recentLogs    []string
	result        map[string]any
	lastLogCount  int
	idempotency   map[string]bool
	previousDepth int
}

func newAbyssLiveSessionID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generating live combat id: %w", err)
	}
	return hex.EncodeToString(buf), nil
}

func normalizeAbyssTactic(tactic string) string {
	switch strings.ToLower(strings.TrimSpace(tactic)) {
	case "aggressive", "defensive", "conserve_items":
		return strings.ToLower(strings.TrimSpace(tactic))
	default:
		return "balanced"
	}
}

func (s *WebServer) loadAbyssTactic(uid string) string {
	var tactic string
	err := s.bot.DB.QueryRow(
		"SELECT value FROM app_meta WHERE key=$1",
		"abyss_live_tactic_"+uid,
	).Scan(&tactic)
	if err != nil {
		return "balanced"
	}
	return normalizeAbyssTactic(tactic)
}

func (s *WebServer) startAbyssLiveCombat(
	uid string,
	run abyssRun,
	depth int,
	tier abyssTier,
	modifier string,
	focus string,
) (*abyssLiveCombat, error) {
	id, err := newAbyssLiveSessionID()
	if err != nil {
		return nil, err
	}

	participants := map[string]bool{uid: true}
	var coopUID string
	_ = s.bot.DB.QueryRow(
		"SELECT COALESCE(coop_uid, '') FROM abyss_active WHERE client_uid=$1",
		uid,
	).Scan(&coopUID)
	if coopUID != "" {
		participants[coopUID] = true
	}

	c := &abyssLiveCombat{
		server:        s,
		id:            id,
		ownerUID:      uid,
		participants:  participants,
		tactics:       make(map[string]string, len(participants)),
		phase:         "starting",
		options:       make(map[string][]abyssLiveOption, len(participants)),
		queued:        make(map[string]abyssLiveAction, len(participants)),
		idempotency:   make(map[string]bool),
		previousDepth: run.Depth,
	}
	for memberUID := range participants {
		c.tactics[memberUID] = s.loadAbyssTactic(memberUID)
	}

	tx, err := s.bot.DB.Begin()
	if err != nil {
		return nil, fmt.Errorf("starting live combat transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	for memberUID := range participants {
		if _, err := tx.Exec(
			"DELETE FROM abyss_combat_sessions WHERE session_id IN (SELECT session_id FROM abyss_combat_members WHERE client_uid=$1)",
			memberUID,
		); err != nil {
			return nil, fmt.Errorf("clearing previous live combat: %w", err)
		}
	}
	if _, err := tx.Exec(
		`INSERT INTO abyss_combat_sessions (session_id, owner_uid, depth, phase, state)
		 VALUES ($1, $2, $3, 'starting', '{}'::jsonb)`,
		id,
		uid,
		run.Depth,
	); err != nil {
		return nil, fmt.Errorf("inserting live combat: %w", err)
	}
	for memberUID := range participants {
		if _, err := tx.Exec(
			`INSERT INTO abyss_combat_members (session_id, client_uid, tactic)
			 VALUES ($1, $2, $3)`,
			id,
			memberUID,
			c.tactics[memberUID],
		); err != nil {
			return nil, fmt.Errorf("inserting live combat member: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("committing live combat: %w", err)
	}

	for memberUID := range participants {
		if old, ok := s.liveCombatByUID.Load(memberUID); ok {
			oldID, isString := old.(string)
			if isString {
				s.liveCombats.Delete(oldID)
			}
			s.liveCombatByUID.Delete(memberUID)
		}
	}
	s.liveCombats.Store(id, c)
	for memberUID := range participants {
		s.liveCombatByUID.Store(memberUID, id)
	}
	c.persist()

	go func() {
		res, fightErr := s.bot.fightAbyssFloorLive(uid, depth, tier, modifier, focus, c)
		if fightErr != nil {
			_, _ = s.bot.DB.Exec(
				"UPDATE abyss_active SET depth=$1, modifier='', event_state=NULL, last_action_at=NOW() WHERE client_uid=$2",
				run.Depth,
				uid,
			)
			c.complete(map[string]any{"ok": false, "error": "combat"})
			return
		}
		c.complete(s.finishDescendData(uid, run, depth, run.Escrow, tier, res, modifier, focus))
	}()

	return c, nil
}

func (s *WebServer) liveCombatForUID(uid string) (*abyssLiveCombat, bool) {
	value, ok := s.liveCombatByUID.Load(uid)
	if !ok {
		return nil, false
	}
	sessionID, ok := value.(string)
	if !ok {
		return nil, false
	}
	combatValue, ok := s.liveCombats.Load(sessionID)
	if !ok {
		return nil, false
	}
	return combatValue.(*abyssLiveCombat), true
}

func (s *WebServer) rejectDuringLiveCombat(w http.ResponseWriter, uid string) bool {
	combat, ok := s.liveCombatForUID(uid)
	if !ok || !combat.isActive() {
		return false
	}
	writeJSON(w, map[string]any{
		"ok": false, "error": "finish the active combat first",
		"state": combat.snapshotFor(uid),
	})
	return true
}

func (c *abyssLiveCombat) isActive() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.phase != "complete" && c.phase != "failed"
}

func (c *abyssLiveCombat) snapshotFor(uid string) abyssLiveSnapshot {
	c.mu.Lock()
	defer c.mu.Unlock()

	allies := append([]abyssLiveCombatantView{}, c.allies...)
	for i := range allies {
		uidFromTarget := strings.TrimPrefix(allies[i].ID, "ally:")
		_, allies[i].Ready = c.queued[uidFromTarget]
		allies[i].IsSelf = uidFromTarget == uid
	}
	enemies := append([]abyssLiveCombatantView{}, c.enemies...)
	options := append([]abyssLiveOption{}, c.options[uid]...)
	recentLogs := append([]string{}, c.recentLogs...)
	var queued *abyssLiveAction
	if action, ok := c.queued[uid]; ok {
		copyAction := action
		queued = &copyAction
	}
	return abyssLiveSnapshot{
		OK:            true,
		SessionID:     c.id,
		OwnerUID:      c.ownerUID,
		Phase:         c.phase,
		Round:         c.round,
		Version:       c.version,
		Deadline:      c.deadline,
		PauseReason:   c.pauseReason,
		Tactic:        c.tactics[uid],
		Allies:        allies,
		Enemies:       enemies,
		Options:       options,
		Queued:        queued,
		RecentLogs:    recentLogs,
		Result:        c.result,
		PreviousDepth: c.previousDepth,
	}
}

func (s *WebServer) persistedAbyssLiveSnapshot(uid string) (abyssLiveSnapshot, bool) {
	var stateJSON, ownerUID, phase, sessionID string
	var depth int
	err := s.bot.DB.QueryRow(
		`SELECT m.state::text, s.owner_uid, s.phase, s.session_id, s.depth
		   FROM abyss_combat_sessions s
		   JOIN abyss_combat_members m ON m.session_id=s.session_id
		  WHERE m.client_uid=$1
		  ORDER BY s.updated_at DESC
		  LIMIT 1`,
		uid,
	).Scan(&stateJSON, &ownerUID, &phase, &sessionID, &depth)
	if err != nil {
		return abyssLiveSnapshot{}, false
	}
	var snapshot abyssLiveSnapshot
	if err := json.Unmarshal([]byte(stateJSON), &snapshot); err != nil {
		return abyssLiveSnapshot{}, false
	}
	if phase == "starting" || phase == "planning" || phase == "resolving" {
		if depth > 0 {
			_, _ = s.bot.DB.Exec(
				"UPDATE abyss_active SET depth=$1, modifier='', event_state=NULL, last_action_at=NOW() WHERE client_uid=$2",
				depth,
				ownerUID,
			)
		}
		snapshot.Phase = "failed"
		snapshot.Result = map[string]any{
			"ok": false, "error": "combat was safely cancelled after a server restart; descend again",
		}
		snapshot.Version++
		updatedState, marshalErr := json.Marshal(snapshot)
		if marshalErr == nil {
			_, _ = s.bot.DB.Exec(
				"UPDATE abyss_combat_sessions SET phase='failed', state=$1, version=$2, deadline=NULL, updated_at=NOW() WHERE session_id=$3",
				string(updatedState),
				snapshot.Version,
				sessionID,
			)
			_, _ = s.bot.DB.Exec(
				"UPDATE abyss_combat_members SET state=$1, queued_action=NULL WHERE session_id=$2 AND client_uid=$3",
				string(updatedState),
				sessionID,
				uid,
			)
		}
	}
	snapshot.OK = true
	return snapshot, true
}

func (c *abyssLiveCombat) persist() {
	snapshot := c.snapshotFor(c.ownerUID)
	state, err := json.Marshal(snapshot)
	if err != nil {
		return
	}
	var deadline any
	if !snapshot.Deadline.IsZero() {
		deadline = snapshot.Deadline
	}
	_, _ = c.server.bot.DB.Exec(
		`UPDATE abyss_combat_sessions
		    SET phase=$1, round=$2, version=$3, deadline=$4,
		        pause_reason=$5, state=$6, updated_at=NOW()
		  WHERE session_id=$7`,
		snapshot.Phase,
		snapshot.Round,
		snapshot.Version,
		deadline,
		snapshot.PauseReason,
		string(state),
		c.id,
	)
	c.mu.Lock()
	members := make(map[string]struct {
		tactic string
		action *abyssLiveAction
	}, len(c.participants))
	for uid := range c.participants {
		member := struct {
			tactic string
			action *abyssLiveAction
		}{tactic: c.tactics[uid]}
		if action, ok := c.queued[uid]; ok {
			copyAction := action
			member.action = &copyAction
		}
		members[uid] = member
	}
	c.mu.Unlock()
	for uid, member := range members {
		var actionJSON any
		var target string
		var submittedRound int
		if member.action != nil {
			if encoded, err := json.Marshal(member.action); err == nil {
				actionJSON = string(encoded)
			}
			target = member.action.TargetID
			submittedRound = member.action.Round
		}
		memberSnapshot, snapshotErr := json.Marshal(c.snapshotFor(uid))
		if snapshotErr != nil {
			continue
		}
		_, _ = c.server.bot.DB.Exec(
			`UPDATE abyss_combat_members
			    SET tactic=$1, selected_target=$2, queued_action=$3,
			        submitted_round=$4, state=$5
			  WHERE session_id=$6 AND client_uid=$7`,
			member.tactic,
			target,
			actionJSON,
			submittedRound,
			string(memberSnapshot),
			c.id,
			uid,
		)
	}
}

func (c *abyssLiveCombat) publishRound(
	round int,
	users []activeUser,
	mobs []*content.Mob,
	logs []string,
) {
	options := make(map[string][]abyssLiveOption, len(users))
	for i := range users {
		if users[i].u == nil {
			continue
		}
		options[users[i].u.UID] = c.optionsFor(&users[i])
	}

	allies := make([]abyssLiveCombatantView, 0, len(users))
	critical := false
	for i := range users {
		au := &users[i]
		if au.u == nil {
			continue
		}
		if au.u.Stats.HP > 0 && au.u.CurrentHP*10 <= au.u.Stats.HP*3 {
			critical = true
		}
		allies = append(allies, abyssLiveCombatantView{
			ID:       "ally:" + au.u.UID,
			Name:     au.u.Nickname,
			HP:       max(0, au.u.CurrentHP),
			MaxHP:    max(1, au.u.Stats.HP),
			Mana:     max(0, au.CurrentMana),
			MaxMana:  max(0, au.MaxMana),
			Ready:    false,
			IsPlayer: true,
		})
	}

	enemies := make([]abyssLiveCombatantView, 0, len(mobs))
	boss := false
	for i, mob := range mobs {
		if mob == nil || mob.Stats.HP <= 0 {
			continue
		}
		boss = boss || mob.Type == content.MobBoss
		enemies = append(enemies, abyssLiveCombatantView{
			ID:    fmt.Sprintf("enemy:%d", i),
			Name:  mob.Name,
			HP:    max(0, mob.Stats.HP),
			MaxHP: max(1, mob.MaxHP),
		})
	}

	duration := abyssLiveRoundTime
	pauseReason := ""
	phaseEvent := false
	if c.lastLogCount < len(logs) {
		for _, line := range logs[c.lastLogCount:] {
			if strings.Contains(line, "summoning ritual") ||
				strings.Contains(line, "ENRAGE") ||
				strings.Contains(line, "Void Barrier") ||
				strings.Contains(line, "opens his slumbering eye") ||
				strings.Contains(line, "cataclysmic sleep") {
				phaseEvent = true
				break
			}
		}
	}
	switch {
	case boss && round == 1:
		duration = abyssLiveHybridTime
		pauseReason = "Boss engaged — plan your opening"
	case phaseEvent:
		duration = abyssLiveHybridTime
		pauseReason = "Boss phase changed — reassess the field"
	case critical:
		duration = abyssLiveHybridTime
		pauseReason = "Critical health — coordinate your response"
	case boss && round%5 == 0:
		duration = abyssLiveHybridTime
		pauseReason = "Boss pressure rising — tactical pause"
	}

	recentLogs := []string{}
	if c.lastLogCount < len(logs) {
		for _, line := range logs[c.lastLogCount:] {
			recentLogs = append(recentLogs, bbToHTML(line))
		}
	}

	c.mu.Lock()
	c.round = round
	c.phase = "planning"
	c.version++
	c.deadline = time.Now().Add(duration)
	c.pauseReason = pauseReason
	c.allies = allies
	c.enemies = enemies
	c.options = options
	c.queued = make(map[string]abyssLiveAction, len(users))
	c.recentLogs = recentLogs
	c.lastLogCount = len(logs)
	c.mu.Unlock()
	c.persist()
}

func (c *abyssLiveCombat) awaitActions(
	round int,
	users []activeUser,
	mobs []*content.Mob,
	logs []string,
) map[string]abyssLiveAction {
	c.publishRound(round, users, mobs, logs)

	c.mu.Lock()
	wait := time.Until(c.deadline)
	c.mu.Unlock()
	if wait > 0 {
		timer := time.NewTimer(wait)
		<-timer.C
	}

	c.mu.Lock()
	for i := range users {
		au := &users[i]
		if au.u == nil || au.u.CurrentHP <= 0 {
			continue
		}
		if _, ok := c.queued[au.u.UID]; !ok {
			c.queued[au.u.UID] = c.bestActionLocked(au.u.UID)
		}
	}
	c.phase = "resolving"
	c.version++
	actions := make(map[string]abyssLiveAction, len(c.queued))
	for uid, action := range c.queued {
		actions[uid] = action
	}
	c.mu.Unlock()
	c.persist()
	return actions
}

func (c *abyssLiveCombat) optionsFor(au *activeUser) []abyssLiveOption {
	options := []abyssLiveOption{
		{Kind: "attack", Name: "Basic Attack", Target: "enemy", Power: 1},
		{Kind: "defend", Name: "Defend", Target: "self"},
	}
	spellCost := 20
	if chest, ok := au.u.Equipped[content.SlotChest]; ok && chest.ID == "ABYSS_ARCHMAGE_ROBES" {
		spellCost -= 5
	}
	if spellCost < 5 {
		spellCost = 5
	}
	for i := range au.u.Skills {
		skill := au.u.Skills[i]
		target := "enemy"
		if skill.HealPercent > 0 && skill.Power == 0 {
			target = "ally"
		}
		options = append(options, abyssLiveOption{
			Kind:        "skill",
			ID:          skill.ID,
			Name:        skill.Name,
			Description: skill.Description,
			Target:      target,
			Mana:        spellCost,
			Power:       skill.Power + skill.HealPercent*2,
		})
	}
	for _, ultimate := range au.u.Ultimates {
		if ultimate == nil {
			continue
		}
		options = append(options, abyssLiveOption{
			Kind:        "ultimate",
			ID:          ultimate.ID,
			Name:        ultimate.Name,
			Description: ultimate.Description,
			Target:      "enemy",
			Cooldown:    ultimate.CurrentCooldown,
			Power:       ultimate.Power,
		})
	}
	for _, item := range c.server.bot.getConsumables(au.u.UID) {
		if item.Type != content.ConsumableHealing && item.Type != content.ConsumableBuff {
			continue
		}
		if item.Duration <= 0 {
			continue
		}
		target := "self"
		if item.Type == content.ConsumableHealing {
			target = "ally"
		}
		options = append(options, abyssLiveOption{
			Kind:        "item",
			ID:          item.ID,
			Name:        item.Name,
			Description: item.Description,
			Target:      target,
			Count:       item.Duration,
			Power:       item.EffectValue,
		})
	}
	return options
}

func (c *abyssLiveCombat) bestActionLocked(uid string) abyssLiveAction {
	action := abyssLiveAction{Kind: "defend", Round: c.round, Automatic: true}
	if len(c.enemies) > 0 {
		target := c.enemies[0]
		for _, enemy := range c.enemies[1:] {
			if enemy.HP < target.HP {
				target = enemy
			}
		}
		action = abyssLiveAction{
			Kind:      "attack",
			TargetID:  target.ID,
			Round:     c.round,
			Automatic: true,
		}
	}

	ally := abyssLiveCombatantView{}
	for _, candidate := range c.allies {
		if candidate.HP <= 0 {
			continue
		}
		if ally.ID == "" || candidate.HP*ally.MaxHP < ally.HP*candidate.MaxHP {
			ally = candidate
		}
	}
	ratio := 1.0
	if ally.ID != "" && ally.MaxHP > 0 {
		ratio = float64(ally.HP) / float64(ally.MaxHP)
	}
	tactic := c.tactics[uid]
	healThreshold := 0.40
	itemThreshold := 0.28
	switch tactic {
	case "aggressive":
		healThreshold, itemThreshold = 0.22, 0.14
	case "defensive":
		healThreshold, itemThreshold = 0.68, 0.45
	case "conserve_items":
		healThreshold, itemThreshold = 0.42, 0.12
	}

	if ratio <= healThreshold {
		for _, option := range c.options[uid] {
			if option.Kind == "skill" && option.Target == "ally" {
				return abyssLiveAction{
					Kind: option.Kind, AbilityID: option.ID, TargetID: ally.ID,
					Round: c.round, Automatic: true,
				}
			}
		}
	}
	if ratio <= itemThreshold {
		for _, option := range c.options[uid] {
			if option.Kind == "item" && option.Target == "ally" && option.Count > 0 {
				return abyssLiveAction{
					Kind: option.Kind, AbilityID: option.ID, TargetID: ally.ID,
					Round: c.round, Automatic: true,
				}
			}
		}
	}

	best := abyssLiveOption{}
	currentMana := 0
	for _, candidate := range c.allies {
		if candidate.ID == "ally:"+uid {
			currentMana = candidate.Mana
			break
		}
	}
	for _, option := range c.options[uid] {
		if option.Target != "enemy" || option.Cooldown > 0 {
			continue
		}
		if option.Mana > currentMana {
			continue
		}
		if option.Kind != "skill" && option.Kind != "ultimate" {
			continue
		}
		if option.Power > best.Power {
			best = option
		}
	}
	if best.ID != "" && action.TargetID != "" {
		action.Kind = best.Kind
		action.AbilityID = best.ID
	}
	return action
}

func (c *abyssLiveCombat) submit(uid string, action abyssLiveAction) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.participants[uid] {
		return errAbyssLiveNotFound
	}
	if action.SessionID != c.id {
		return errAbyssLiveStale
	}
	if action.IdempotencyKey != "" && c.idempotency[uid+":"+action.IdempotencyKey] {
		return nil
	}
	if c.phase != "planning" || action.Round != c.round || time.Now().After(c.deadline) {
		return errAbyssLiveStale
	}
	validOption := false
	for _, option := range c.options[uid] {
		if option.Kind == action.Kind && option.ID == action.AbilityID {
			validOption = option.Cooldown == 0
			if option.Kind == "item" && option.Count <= 0 {
				validOption = false
			}
			if option.Mana > 0 {
				for _, ally := range c.allies {
					if ally.ID == "ally:"+uid && ally.Mana < option.Mana {
						validOption = false
					}
				}
			}
			break
		}
	}
	if !validOption {
		return fmt.Errorf("invalid or unavailable action")
	}
	if !c.validTargetLocked(uid, action) {
		return fmt.Errorf("invalid target")
	}
	action.Automatic = false
	c.queued[uid] = action
	if action.IdempotencyKey != "" {
		c.idempotency[uid+":"+action.IdempotencyKey] = true
	}
	c.version++
	return nil
}

func (c *abyssLiveCombat) validTargetLocked(uid string, action abyssLiveAction) bool {
	if action.Kind == "defend" {
		return action.TargetID == "" || action.TargetID == "ally:"+uid
	}
	var targetType string
	for _, option := range c.options[uid] {
		if option.Kind == action.Kind && option.ID == action.AbilityID {
			targetType = option.Target
			break
		}
	}
	switch targetType {
	case "self":
		return action.TargetID == "" || action.TargetID == "ally:"+uid
	case "ally":
		for _, ally := range c.allies {
			if ally.ID == action.TargetID && ally.HP > 0 {
				return true
			}
		}
	case "enemy":
		for _, enemy := range c.enemies {
			if enemy.ID == action.TargetID && enemy.HP > 0 {
				return true
			}
		}
	}
	return false
}

func (c *abyssLiveCombat) setTactic(uid, tactic string) error {
	tactic = normalizeAbyssTactic(tactic)
	c.mu.Lock()
	if !c.participants[uid] {
		c.mu.Unlock()
		return errAbyssLiveNotFound
	}
	c.tactics[uid] = tactic
	c.version++
	c.mu.Unlock()
	_, err := c.server.bot.DB.Exec(
		`INSERT INTO app_meta (key, value) VALUES ($1, $2)
		 ON CONFLICT (key) DO UPDATE SET value=EXCLUDED.value`,
		"abyss_live_tactic_"+uid,
		tactic,
	)
	if err == nil {
		_, err = c.server.bot.DB.Exec(
			"UPDATE abyss_combat_members SET tactic=$1 WHERE session_id=$2 AND client_uid=$3",
			tactic,
			c.id,
			uid,
		)
	}
	return err
}

func (c *abyssLiveCombat) complete(result map[string]any) {
	c.mu.Lock()
	c.phase = "complete"
	if ok, _ := result["ok"].(bool); !ok {
		c.phase = "failed"
	}
	c.result = result
	c.deadline = time.Time{}
	c.pauseReason = ""
	c.version++
	c.mu.Unlock()
	c.persist()
}

func (s *WebServer) handleAbyssCombatState(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	c, ok := s.liveCombatForUID(uid)
	if !ok {
		if snapshot, found := s.persistedAbyssLiveSnapshot(uid); found {
			writeJSON(w, snapshot)
			return
		}
		writeJSON(w, map[string]any{"ok": false, "error": "no active combat"})
		return
	}
	writeJSON(w, c.snapshotFor(uid))
}

func (s *WebServer) handleAbyssCombatAction(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	c, ok := s.liveCombatForUID(uid)
	if !ok {
		writeJSON(w, map[string]any{"ok": false, "error": "no active combat"})
		return
	}
	var action abyssLiveAction
	if err := readJSON(r, &action); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}
	if err := c.submit(uid, action); err != nil {
		if errors.Is(err, errAbyssLiveStale) {
			writeJSON(w, map[string]any{"ok": false, "error": "round closed", "state": c.snapshotFor(uid)})
			return
		}
		writeJSON(w, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	c.persist()
	writeJSON(w, c.snapshotFor(uid))
}

func (s *WebServer) handleAbyssCombatTactics(w http.ResponseWriter, r *http.Request, uid string) {
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
		Tactic string `json:"tactic"`
	}
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}
	if err := c.setTactic(uid, req.Tactic); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	writeJSON(w, c.snapshotFor(uid))
}

func (s *WebServer) handleAbyssCombatEvents(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	c, ok := s.liveCombatForUID(uid)
	if !ok {
		http.Error(w, "no active combat", http.StatusNotFound)
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	var lastVersion int64 = -1
	for {
		snapshot := c.snapshotFor(uid)
		if snapshot.Version != lastVersion {
			data, err := json.Marshal(snapshot)
			if err != nil {
				return
			}
			if _, err := fmt.Fprintf(w, "event: combat\ndata: %s\n\n", data); err != nil {
				return
			}
			flusher.Flush()
			lastVersion = snapshot.Version
			if snapshot.Phase == "complete" || snapshot.Phase == "failed" {
				return
			}
		}
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
		}
	}
}

func liveMobFromTarget(targetID string, mobs []*content.Mob) *content.Mob {
	var index int
	if _, err := fmt.Sscanf(targetID, "enemy:%d", &index); err != nil {
		return nil
	}
	if index < 0 || index >= len(mobs) || mobs[index] == nil || mobs[index].Stats.HP <= 0 {
		return nil
	}
	return mobs[index]
}

func liveAllyFromTarget(targetID string, users []activeUser) *UserInCombat {
	uid := strings.TrimPrefix(targetID, "ally:")
	for i := range users {
		if users[i].u != nil && users[i].u.UID == uid && users[i].u.CurrentHP > 0 {
			return users[i].u
		}
	}
	return nil
}

func findLiveSkill(u *UserInCombat, id string) *content.Skill {
	for i := range u.Skills {
		if u.Skills[i].ID == id {
			return &u.Skills[i]
		}
	}
	return nil
}

func findLiveUltimate(u *UserInCombat, id string) *content.UltimateSkill {
	for _, ultimate := range u.Ultimates {
		if ultimate != nil && ultimate.ID == id {
			return ultimate
		}
	}
	return nil
}

func (b *Bot) useLiveConsumable(
	actor *UserInCombat,
	target *UserInCombat,
	consumableID string,
	logs *[]string,
) bool {
	consumable, ok := content.GetConsumableByID(consumableID)
	if !ok || (consumable.Type != content.ConsumableHealing && consumable.Type != content.ConsumableBuff) {
		return false
	}
	result, err := b.DB.Exec(
		`UPDATE user_consumables
		    SET remaining_fights=remaining_fights-1
		  WHERE client_uid=$1 AND cons_id=$2 AND remaining_fights>0`,
		actor.UID,
		consumableID,
	)
	if err != nil {
		return false
	}
	rows, err := result.RowsAffected()
	if err != nil || rows == 0 {
		return false
	}
	_, _ = b.DB.Exec(
		"DELETE FROM user_consumables WHERE client_uid=$1 AND cons_id=$2 AND remaining_fights<=0",
		actor.UID,
		consumableID,
	)
	b.abyssSpendLoadout(actor.UID, consumableID)
	_, _ = b.DB.Exec("UPDATE abyss_active SET momentum=0 WHERE client_uid=$1", actor.UID)

	switch consumable.Type {
	case content.ConsumableHealing:
		if target == nil {
			target = actor
		}
		heal := int(consumable.EffectValue)
		if consumable.EffectValue <= 1 {
			heal = int(float64(target.Stats.HP) * consumable.EffectValue)
		}
		if heal < 1 {
			heal = 1
		}
		target.CurrentHP = min(target.Stats.HP, target.CurrentHP+heal)
		*logs = append(*logs, fmt.Sprintf(
			"🧪 %s uses %s on %s, restoring %d HP.",
			actor.Nickname,
			consumable.Name,
			target.Nickname,
			heal,
		))
	case content.ConsumableBuff:
		amount := int(consumable.EffectValue)
		switch consumableID {
		case "iron_skin_brew":
			actor.Stats.DEF += amount
		case "speed_elixir":
			actor.Stats.SPD += amount
		case "intellect_elixir":
			actor.Stats.INT += amount
		default:
			actor.Stats.STR += amount
		}
		*logs = append(*logs, fmt.Sprintf("🧪 %s uses %s and surges with power.", actor.Nickname, consumable.Name))
	}
	return true
}
