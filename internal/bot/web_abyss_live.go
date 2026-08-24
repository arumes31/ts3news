package bot

import (
	"errors"
	"fmt"
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

func normalizeAbyssTactic(tactic string) string {
	switch strings.ToLower(strings.TrimSpace(tactic)) {
	case "aggressive", "defensive", "conserve_items":
		return strings.ToLower(strings.TrimSpace(tactic))
	default:
		return "balanced"
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
