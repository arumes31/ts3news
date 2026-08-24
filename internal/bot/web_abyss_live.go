package bot

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"ts3news/internal/content"
)

const (
	abyssLiveRoundTime                  = 4 * time.Second
	abyssLiveHybridTime                 = 9 * time.Second
	abyssLiveSnapshotSchemaVersion      = 1
	abyssLiveMaxIdempotencyKeyLength    = 128
	abyssLiveMaxIdempotencyKeysPerRound = 64
	abyssLiveInitialTimeBank            = 6 * time.Second
	abyssLiveTimeBankSpend              = 2 * time.Second
)

var (
	errAbyssLiveNotFound            = errors.New("live combat not found")
	errAbyssLiveStale               = errors.New("live combat round is stale")
	errAbyssLiveIdempotencyConflict = errors.New("idempotency key conflicts with an accepted action")
	errAbyssLiveIdempotencyLimit    = errors.New("too many action attempts for this round")
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
	Kind         string   `json:"kind"`
	ID           string   `json:"id,omitempty"`
	Name         string   `json:"name"`
	Description  string   `json:"description,omitempty"`
	Target       string   `json:"target"`
	Mana         int      `json:"mana,omitempty"`
	Cooldown     int      `json:"cooldown,omitempty"`
	Count        int      `json:"count,omitempty"`
	Power        float64  `json:"power,omitempty"`
	EffectLabel  string   `json:"effect_label,omitempty"`
	MinEffect    int      `json:"min_effect,omitempty"`
	MaxEffect    int      `json:"max_effect,omitempty"`
	Tags         []string `json:"tags,omitempty"`
	EffectRounds int      `json:"effect_rounds,omitempty"`
	Modifiers    []string `json:"modifiers,omitempty"`
}

type abyssLiveEffect struct {
	Name            string `json:"name"`
	RemainingRounds int    `json:"remaining_rounds,omitempty"`
	Duration        string `json:"duration,omitempty"`
}

type abyssLiveCombatantView struct {
	ID       string            `json:"id"`
	Name     string            `json:"name"`
	HP       int               `json:"hp"`
	MaxHP    int               `json:"max_hp"`
	Mana     int               `json:"mana,omitempty"`
	MaxMana  int               `json:"max_mana,omitempty"`
	Ready    bool              `json:"ready,omitempty"`
	IsPlayer bool              `json:"is_player,omitempty"`
	IsSelf   bool              `json:"is_self,omitempty"`
	Element  string            `json:"element,omitempty"`
	WeakTo   string            `json:"weak_to,omitempty"`
	Position string            `json:"position,omitempty"`
	Speed    int               `json:"speed,omitempty"`
	Threat   int               `json:"threat,omitempty"`
	Role     string            `json:"role,omitempty"`
	Faction  string            `json:"faction,omitempty"`
	Pattern  string            `json:"pattern,omitempty"`
	Break    int               `json:"break,omitempty"`
	MaxBreak int               `json:"max_break,omitempty"`
	Hazard   bool              `json:"hazard,omitempty"`
	Effects  []abyssLiveEffect `json:"effects,omitempty"`
}

type abyssLiveRecommendation struct {
	Action abyssLiveAction `json:"action"`
	Reason string          `json:"reason"`
}

type abyssLivePolicy struct {
	CriticalTactic string `json:"critical_tactic"`
	AttackPriority string `json:"attack_priority"`
	SkillPriority  string `json:"skill_priority"`
	TimeoutAction  string `json:"timeout_action"`
}

type abyssLiveEnemyIntent struct {
	EnemyID   string `json:"enemy_id"`
	EnemyName string `json:"enemy_name"`
	Kind      string `json:"kind"`
	Ability   string `json:"ability,omitempty"`
	TargetID  string `json:"target_id,omitempty"`
	Target    string `json:"target,omitempty"`
}

type abyssLiveInitiativeEntry struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Side  string `json:"side"`
	Speed int    `json:"speed"`
}

type abyssLiveSnapshot struct {
	OK            bool                       `json:"ok"`
	SchemaVersion int                        `json:"schema_version"`
	SessionID     string                     `json:"session_id"`
	OwnerUID      string                     `json:"-"`
	Phase         string                     `json:"phase"`
	Round         int                        `json:"round"`
	Version       int64                      `json:"version"`
	Deadline      time.Time                  `json:"deadline,omitempty"`
	PauseReason   string                     `json:"pause_reason,omitempty"`
	PauseMode     string                     `json:"pause_mode"`
	CanConfigure  bool                       `json:"can_configure_pause,omitempty"`
	Policy        abyssLivePolicy            `json:"policy"`
	Tactic        string                     `json:"tactic"`
	Allies        []abyssLiveCombatantView   `json:"allies"`
	Enemies       []abyssLiveCombatantView   `json:"enemies"`
	Options       []abyssLiveOption          `json:"options"`
	Queued        *abyssLiveAction           `json:"queued,omitempty"`
	Recommended   *abyssLiveRecommendation   `json:"recommended,omitempty"`
	TimeBankMS    int64                      `json:"time_bank_ms,omitempty"`
	EnemyIntents  []abyssLiveEnemyIntent     `json:"enemy_intents,omitempty"`
	Initiative    []abyssLiveInitiativeEntry `json:"initiative,omitempty"`
	RecentLogs    []string                   `json:"recent_logs"`
	RoundRecap    string                     `json:"round_recap,omitempty"`
	RandomSeed    [2]uint64                  `json:"random_seed"`
	RandomDraws   uint64                     `json:"random_draws"`
	Result        map[string]any             `json:"result,omitempty"`
	PreviousDepth int                        `json:"previous_depth,omitempty"`
	Social        abyssLiveSocialSnapshot    `json:"social"`
}

type abyssLiveCombat struct {
	mu        sync.Mutex
	persistMu sync.Mutex
	rngMu     sync.Mutex
	rng       combatRandomSource

	server         *WebServer
	id             string
	ownerUID       string
	participants   map[string]bool
	tactics        map[string]string
	policies       map[string]abyssLivePolicy
	phase          string
	round          int
	version        int64
	deadline       time.Time
	pauseReason    string
	pauseMode      string
	allies         []abyssLiveCombatantView
	enemies        []abyssLiveCombatantView
	options        map[string][]abyssLiveOption
	queued         map[string]abyssLiveAction
	ready          map[string]bool
	readySignal    chan struct{}
	timeBank       map[string]time.Duration
	deadlineSignal chan struct{}
	recentLogs     []string
	roundRecap     string
	result         map[string]any
	lastLogCount   int
	idempotency    map[string]abyssLiveIdempotency
	history        []abyssLiveEvent
	enemyPlans     map[int]abyssLiveEnemyPlan
	actionCounts   map[string]int
	bossAdaptation string
	initiative     []abyssLiveInitiativeEntry
	social         abyssLiveSocialState
	previousDepth  int
	randomSeed     [2]uint64
	randomDraws    uint64
	createdAt      time.Time
	finishedAt     time.Time
}

func normalizeAbyssTactic(tactic string) string {
	switch strings.ToLower(strings.TrimSpace(tactic)) {
	case "aggressive", "defensive", "conserve_items":
		return strings.ToLower(strings.TrimSpace(tactic))
	default:
		return "balanced"
	}
}

func normalizeAbyssPauseMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "bosses", "danger", "fast":
		return strings.ToLower(strings.TrimSpace(mode))
	default:
		return "adaptive"
	}
}

func (c *abyssLiveCombat) isActive() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.phase != "complete" && c.phase != "failed"
}

func (c *abyssLiveCombat) snapshotFor(uid string) abyssLiveSnapshot {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.snapshotForLocked(uid)
}

func (c *abyssLiveCombat) snapshotForLocked(uid string) abyssLiveSnapshot {
	allies := append([]abyssLiveCombatantView{}, c.allies...)
	for i := range allies {
		uidFromTarget := strings.TrimPrefix(allies[i].ID, "ally:")
		allies[i].Ready = c.ready[uidFromTarget]
		allies[i].IsSelf = uidFromTarget == uid
		if action, ok := c.queued[uidFromTarget]; ok && action.Kind == "defend" {
			allies[i].Effects = append(allies[i].Effects, abyssLiveEffect{
				Name: "Guard queued", Duration: "next enemy phase",
			})
		}
	}
	enemies := append([]abyssLiveCombatantView{}, c.enemies...)
	options := append([]abyssLiveOption{}, c.options[uid]...)
	recentLogs := append([]string{}, c.recentLogs...)
	enemyIntents := make([]abyssLiveEnemyIntent, 0, len(c.enemyPlans))
	for i := range c.enemies {
		if plan, ok := c.enemyPlans[liveEnemyIndex(c.enemies[i].ID)]; ok {
			enemyIntents = append(enemyIntents, plan.Intent)
		}
	}
	initiative := append([]abyssLiveInitiativeEntry{}, c.initiative...)
	timeBankMS := c.timeBank[uid].Milliseconds()
	var queued *abyssLiveAction
	if action, ok := c.queued[uid]; ok {
		copyAction := action
		queued = &copyAction
	}
	var recommended *abyssLiveRecommendation
	if c.phase == "planning" && c.participants[uid] {
		action, reason := c.bestActionWithReasonLocked(uid)
		recommended = &abyssLiveRecommendation{Action: action, Reason: reason}
	}
	return abyssLiveSnapshot{
		OK:            true,
		SchemaVersion: abyssLiveSnapshotSchemaVersion,
		SessionID:     c.id,
		OwnerUID:      c.ownerUID,
		Phase:         c.phase,
		Round:         c.round,
		Version:       c.version,
		Deadline:      c.deadline,
		PauseReason:   c.pauseReason,
		PauseMode:     normalizeAbyssPauseMode(c.pauseMode),
		CanConfigure:  uid == c.ownerUID,
		Policy:        normalizeLivePolicy(c.policies[uid]),
		Tactic:        c.tactics[uid],
		Allies:        allies,
		Enemies:       enemies,
		Options:       options,
		Queued:        queued,
		Recommended:   recommended,
		TimeBankMS:    timeBankMS,
		EnemyIntents:  enemyIntents,
		Initiative:    initiative,
		RecentLogs:    recentLogs,
		RoundRecap:    c.roundRecap,
		RandomSeed:    c.randomSeed,
		RandomDraws:   c.randomDrawCount(),
		Result:        c.result,
		PreviousDepth: c.previousDepth,
		Social:        c.socialSnapshotLocked(uid),
	}
}

func (c *abyssLiveCombat) publishRound(
	round int,
	users []activeUser,
	mobs []*content.Mob,
	logs []string,
	playerStarts bool,
) {
	logs = c.applyLiveBossAdaptation(mobs, logs)
	options := make(map[string][]abyssLiveOption, len(users))
	for i := range users {
		if users[i].u == nil || users[i].u.CurrentHP <= 0 {
			continue
		}
		options[users[i].u.UID] = c.optionsFor(&users[i], users, mobs)
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
			Element:  string(liveUserElement(au.u)),
			Position: string(au.u.Position),
			Speed:    au.u.Stats.SPD,
			Threat:   liveThreat(au.u),
			Role:     c.social.preferences[au.u.UID].Role,
			Effects:  liveAllyEffects(au),
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
			ID:       fmt.Sprintf("enemy:%d", i),
			Name:     mob.Name,
			HP:       max(0, mob.Stats.HP),
			MaxHP:    max(1, mob.MaxHP),
			Element:  string(mob.Element),
			WeakTo:   string(liveElementWeakness(mob.Element)),
			Speed:    mob.Stats.SPD,
			Role:     abyssEnemyRole(mob),
			Faction:  abyssEnemyFaction(mob),
			Pattern:  abyssEnemyPattern(mob),
			Break:    max(0, mob.Break),
			MaxBreak: max(0, mob.MaxBreak),
			Hazard:   abyssEnemyHazard(mob),
			Effects:  liveMobEffects(mob),
		})
	}

	duration := abyssLiveRoundTime
	pauseReason := ""
	pauseMode := c.currentPauseMode()
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
	case boss && round == 1 && livePauseEnabled(pauseMode, "boss"):
		duration = abyssLiveHybridTime
		pauseReason = "Boss engaged — plan your opening"
	case phaseEvent && livePauseEnabled(pauseMode, "phase"):
		duration = abyssLiveHybridTime
		pauseReason = "Boss phase changed — reassess the field"
	case critical && livePauseEnabled(pauseMode, "critical"):
		duration = abyssLiveHybridTime
		pauseReason = "Critical health — coordinate your response"
	case boss && round%5 == 0 && livePauseEnabled(pauseMode, "boss"):
		duration = abyssLiveHybridTime
		pauseReason = "Boss pressure rising — tactical pause"
	}

	recentLogs := []string{}
	if c.lastLogCount < len(logs) {
		for _, line := range logs[c.lastLogCount:] {
			recentLogs = append(recentLogs, bbToHTML(line))
		}
	}
	enemyPlans := planLiveEnemyIntentsWithRandom(round, users, mobs, logs, c)
	initiative := liveInitiative(users, mobs, playerStarts)

	c.mu.Lock()
	roundRecap := liveRoundRecap(c.round, c.allies, allies, c.enemies, enemies)
	c.round = round
	c.pruneIdempotencyLocked()
	c.phase = "planning"
	c.version++
	c.deadline = time.Now().Add(duration)
	if c.social.autoResolve {
		c.deadline = time.Now()
	}
	c.pauseReason = pauseReason
	c.allies = allies
	c.enemies = enemies
	c.options = options
	c.queued = make(map[string]abyssLiveAction, len(users))
	c.ready = make(map[string]bool, len(users))
	c.readySignal = make(chan struct{}, 1)
	if c.timeBank == nil {
		c.timeBank = make(map[string]time.Duration, len(users))
	}
	for i := range users {
		if users[i].u != nil {
			if _, ok := c.timeBank[users[i].u.UID]; !ok {
				c.timeBank[users[i].u.UID] = abyssLiveInitialTimeBank
			}
		}
	}
	c.deadlineSignal = make(chan struct{}, 1)
	c.recentLogs = recentLogs
	c.roundRecap = roundRecap
	c.enemyPlans = enemyPlans
	c.initiative = initiative
	c.lastLogCount = len(logs)
	c.mu.Unlock()
	c.persistOrLog("publishing live combat round")
}

func (c *abyssLiveCombat) awaitActions(
	round int,
	users []activeUser,
	mobs []*content.Mob,
	logs []string,
	playerStarts bool,
) map[string]abyssLiveAction {
	planningStarted := time.Now()
	c.publishRound(round, users, mobs, logs, playerStarts)

	c.waitForPlanningDeadline()

	c.mu.Lock()
	c.ensureSocialLocked()
	for i := range users {
		au := &users[i]
		if au.u == nil || au.u.CurrentHP <= 0 {
			continue
		}
		if _, ok := c.queued[au.u.UID]; !ok {
			c.queued[au.u.UID] = c.timeoutActionLocked(au.u.UID)
		}
	}
	c.phase = "resolving"
	c.version++
	actions := make(map[string]abyssLiveAction, len(c.queued))
	coordinationUIDs := []string{}
	if c.allReadyLocked() && !c.social.readyAwarded[c.round] {
		c.social.readyAwarded[c.round] = true
		for uid := range c.participants {
			coordinationUIDs = append(coordinationUIDs, uid)
		}
	}
	comboTargets := map[string][]string{}
	for uid, action := range c.queued {
		actions[uid] = action
		c.server.abyssOps.observeAction(action.Automatic)
		if c.actionCounts == nil {
			c.actionCounts = make(map[string]int)
		}
		c.actionCounts[action.Kind]++
		c.social.actionCounts[uid]++
		if !action.Automatic {
			c.social.manualCounts[uid]++
		}
		if c.ready[uid] {
			c.social.readyCounts[uid]++
		}
		if action.Kind == "skill" && action.TargetID != "" {
			comboTargets[action.TargetID] = append(comboTargets[action.TargetID], uid)
		}
	}
	comboUIDs := []string{}
	for _, uids := range comboTargets {
		if len(uids) >= 2 && !c.social.comboAwarded[c.round] {
			c.social.comboAwarded[c.round] = true
			comboUIDs = append(comboUIDs, uids...)
		}
	}
	c.mu.Unlock()
	for _, uid := range coordinationUIDs {
		c.server.bot.recordAbyssProgression(uid, "coordinator")
	}
	for _, uid := range comboUIDs {
		c.server.bot.recordAbyssProgression(uid, "party_combo")
	}
	c.persistOrLog("resolving live combat round")
	c.server.abyssOps.observePlanning(time.Since(planningStarted))
	return actions
}

func (c *abyssLiveCombat) complete(result map[string]any) {
	c.mu.Lock()
	c.phase = "complete"
	if ok, _ := result["ok"].(bool); !ok {
		c.phase = "failed"
	}
	c.result = result
	c.finishedAt = time.Now()
	c.deadline = time.Time{}
	c.pauseReason = ""
	c.version++
	c.mu.Unlock()
	c.server.abyssOps.observeCompletion(c, result)
	if err := c.persist(); err != nil {
		log.Printf("abyss live: completing live combat: %v", err)
	}
	if victory, _ := result["victory"].(bool); victory {
		c.server.bot.resolveAbyssGhostChallenge(c.ownerUID, c.round)
	}
}
