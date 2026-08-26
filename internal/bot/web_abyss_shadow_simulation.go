package bot

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"math/rand/v2"
	"net/http"
	"sort"
	"strings"
	"time"

	"ts3news/internal/content"
)

const (
	abyssShadowSimulationCost   = 2
	abyssShadowSimulationTrials = 100
)

var errAbyssShadowInsufficientTokens = errors.New("not enough Abyss Tokens")

type abyssFightMode struct {
	ctx           context.Context
	live          *abyssLiveCombat
	shadowTrials  int
	encounterSeed [2]uint64
	trialSeed     [2]uint64
}

type abyssFightExecution struct {
	floor      abyssFloorResult
	simulation *abyssShadowSimulationResult
}

type abyssShadowSimulationResult struct {
	Depth             int    `json:"depth"`
	Encounter         string `json:"encounter"`
	Trials            int    `json:"trials"`
	Wins              int    `json:"wins"`
	Losses            int    `json:"losses"`
	WinPct            int    `json:"win_pct"`
	ConfidenceLowPct  int    `json:"confidence_low_pct"`
	ConfidenceHighPct int    `json:"confidence_high_pct"`
	MedianWinHPPct    int    `json:"median_win_hp_pct"`
	RiskPct           int    `json:"risk_pct"`
	Cost              int    `json:"cost"`
	Tokens            int    `json:"tokens"`
}

func (b *Bot) prepareAbyssShadowUser(u *UserInCombat) {
	if u == nil {
		return
	}
	u.shadow = true
	u.IsClone = true
	u.shadowConsumables = append([]content.Consumable{}, b.getConsumables(u.UID)...)
	_, _, _, _, u.shadowEffects = b.activeLootMult(u.UID, time.Now())
	u.shadowHoldMana = b.abyssHoldMana(u.UID)
	u.shadowPetCommand = b.loadAbyssPetCommand(u.UID)
	u.shadowPetFocus = b.abyssPetFocus(u.UID)
	u.shadowRunFlags = b.loadRunFlags(u.UID)
	u.shadowUpInsight = b.loadAbyssStats(u.UID).UpInsight
	u.shadowBackups = b.loadBackupWeapons(u.UID)
	var titleName sql.NullString
	_ = b.DB.QueryRow("SELECT title FROM users WHERE client_uid=$1", u.UID).Scan(&titleName)
	if titleName.Valid {
		if title, ok := content.GetTitleByName(titleName.String); ok {
			u.shadowLifesteal = title.Lifesteal
			u.shadowMultiStrike = title.MultiStrike
		}
	}
	for _, gear := range u.Equipped {
		if gear.Special == content.EffectMindControl {
			u.shadowMindControl += int(gear.Rarity) + 1
		}
	}
}

func cloneAbyssShadowUsers(users []UserInCombat) []UserInCombat {
	clones := make([]UserInCombat, len(users))
	for i := range users {
		clones[i] = users[i]
		clones[i].Skills = append([]content.Skill{}, users[i].Skills...)
		clones[i].Equipped = make(map[content.GearSlot]content.Gear, len(users[i].Equipped))
		for slot, gear := range users[i].Equipped {
			clones[i].Equipped[slot] = gear
		}
		clones[i].shadowBackups = append([]content.Gear{}, users[i].shadowBackups...)
		clones[i].abyssSkillsUsed = make(map[string]struct{}, len(users[i].abyssSkillsUsed))
		for skillID := range users[i].abyssSkillsUsed {
			clones[i].abyssSkillsUsed[skillID] = struct{}{}
		}
		clones[i].shadowConsumables = append(
			[]content.Consumable{},
			users[i].shadowConsumables...,
		)
		clones[i].shadowEffects = append(
			[]content.ItemEffect{},
			users[i].shadowEffects...,
		)
		clones[i].Pets = make([]*content.Mob, len(users[i].Pets))
		for petIndex, pet := range users[i].Pets {
			if pet != nil {
				clones[i].Pets[petIndex] = pet.Clone()
			}
		}
		clones[i].Ultimates = make([]*content.UltimateSkill, len(users[i].Ultimates))
		for ultimateIndex, ultimate := range users[i].Ultimates {
			if ultimate == nil {
				continue
			}
			ultimateCopy := *ultimate
			clones[i].Ultimates[ultimateIndex] = &ultimateCopy
		}
		if users[i].abyssSupport != nil {
			supportCopy := *users[i].abyssSupport
			clones[i].abyssSupport = &supportCopy
		}
	}
	return clones
}

func abyssShadowEncounterLabel(mobs []*content.Mob) string {
	names := make([]string, 0, len(mobs))
	for _, mob := range mobs {
		if mob != nil {
			names = append(names, mob.DisplayNameShort())
		}
	}
	sort.Strings(names)
	if len(names) == 0 {
		return "Unknown encounter"
	}
	return strings.Join(names, " · ")
}

func abyssWilsonInterval(wins, trials int) (lowPct, highPct int) {
	if trials <= 0 {
		return 0, 0
	}
	const z = 1.959963984540054
	n := float64(trials)
	p := float64(wins) / n
	zSquared := z * z
	denominator := 1 + zSquared/n
	center := (p + zSquared/(2*n)) / denominator
	margin := z * math.Sqrt((p*(1-p)+zSquared/(4*n))/n) / denominator
	lowPct = int(math.Round(100 * max(0.0, center-margin)))
	highPct = int(math.Round(100 * min(1.0, center+margin)))
	return lowPct, highPct
}

func medianInt(values []int) int {
	if len(values) == 0 {
		return 0
	}
	sort.Ints(values)
	middle := len(values) / 2
	if len(values)%2 == 1 {
		return values[middle]
	}
	return (values[middle-1] + values[middle]) / 2
}

func (b *Bot) simulatePreparedAbyssCombat(
	ctx context.Context,
	users []UserInCombat,
	mobs []*content.Mob,
	avgLevel int,
	difficulty float64,
	zone content.Zone,
	trials int,
	seed [2]uint64,
) (abyssShadowSimulationResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	trials = min(max(trials, 1), abyssShadowSimulationTrials)
	wins := 0
	winningHPPct := make([]int, 0, trials)
	for trial := range trials {
		if err := ctx.Err(); err != nil {
			return abyssShadowSimulationResult{}, err
		}
		trialUsers := cloneAbyssShadowUsers(users)
		random := rand.New(rand.NewPCG(
			seed[0]+uint64(trial)*0x9e3779b97f4a7c15,
			seed[1]^uint64(trial+1)*0xbf58476d1ce4e5b9,
		))
		_, _, victory, _, _, _ := b.resolveChannelCombatDetailedWithRandom(
			trialUsers,
			mobs,
			avgLevel,
			difficulty,
			zone,
			random,
		)
		if !victory {
			continue
		}
		wins++
		maxHP := max(1, trialUsers[0].Stats.HP)
		winningHPPct = append(
			winningHPPct,
			min(100, max(0, trialUsers[0].CurrentHP*100/maxHP)),
		)
	}
	lowPct, highPct := abyssWilsonInterval(wins, trials)
	return abyssShadowSimulationResult{
		Encounter:         abyssShadowEncounterLabel(mobs),
		Trials:            trials,
		Wins:              wins,
		Losses:            trials - wins,
		WinPct:            int(math.Round(float64(wins) * 100 / float64(trials))),
		ConfidenceLowPct:  lowPct,
		ConfidenceHighPct: highPct,
		MedianWinHPPct:    medianInt(winningHPPct),
		Cost:              abyssShadowSimulationCost,
	}, nil
}

func (b *Bot) simulateAbyssNextFloor(
	ctx context.Context,
	uid string,
	depth int,
	tier abyssTier,
	modifier string,
	focus string,
) (abyssShadowSimulationResult, error) {
	mode := abyssFightMode{
		ctx:           ctx,
		shadowTrials:  abyssShadowSimulationTrials,
		encounterSeed: [2]uint64{rand.Uint64(), rand.Uint64()}, // #nosec G404 -- gameplay simulation
		trialSeed:     [2]uint64{rand.Uint64(), rand.Uint64()}, // #nosec G404 -- gameplay simulation
	}
	execution, err := b.fightAbyssFloorMode(uid, depth, tier, modifier, focus, mode)
	if err != nil {
		return abyssShadowSimulationResult{}, err
	}
	if execution.simulation == nil {
		return abyssShadowSimulationResult{}, errors.New("shadow simulation produced no result")
	}
	return *execution.simulation, nil
}

func chargeAbyssShadowSimulation(ctx context.Context, db *sql.DB, uid string) (int, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()

	var tokens int
	err = tx.QueryRowContext(
		ctx,
		`UPDATE users SET abyss_tokens=abyss_tokens-$1
		 WHERE client_uid=$2 AND abyss_tokens >= $1 RETURNING abyss_tokens`,
		abyssShadowSimulationCost,
		uid,
	).Scan(&tokens)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, errAbyssShadowInsufficientTokens
	}
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return tokens, nil
}

func (s *WebServer) handleAbyssShadowSimulation(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()
	if s.rejectDuringLiveCombat(w, uid) {
		return
	}
	run := s.bot.loadAbyssRun(uid)
	if !run.Active {
		writeJSON(w, map[string]any{"ok": false, "error": "not in a run"})
		return
	}
	if run.Downed {
		writeJSON(w, map[string]any{"ok": false, "error": "revive or concede before scouting"})
		return
	}
	if run.FloorType != "combat" {
		writeJSON(w, map[string]any{"ok": false, "error": "resolve the current room before scouting"})
		return
	}
	var pendingChoice sql.NullString
	if err := s.bot.DB.QueryRowContext(
		r.Context(),
		"SELECT pending_floor_choice FROM abyss_active WHERE client_uid=$1",
		uid,
	).Scan(&pendingChoice); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "shadow fights are temporarily unavailable"})
		return
	}
	if pendingChoice.Valid && pendingChoice.String != "" {
		writeJSON(w, map[string]any{"ok": false, "error": "choose the prepared floor before scouting"})
		return
	}
	if s.bot.abyssTokens(uid) < abyssShadowSimulationCost {
		writeJSON(w, map[string]any{"ok": false, "error": "not enough Abyss Tokens"})
		return
	}
	tier, ok := abyssTierByKey(run.Tier)
	if !ok {
		writeJSON(w, map[string]any{"ok": false, "error": "run tier is unavailable"})
		return
	}
	depth := run.Depth + 1
	modifier := ""
	if abyssWatcherAmbushDue(run, time.Now()) {
		modifier = "watcher"
	}
	result, err := s.bot.simulateAbyssNextFloor(
		r.Context(),
		uid,
		depth,
		tier,
		modifier,
		s.selectedAbyssFocus(uid, run),
	)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "shadow fights could not be completed"})
		return
	}

	result.Depth = depth
	result.RiskPct = abyssRiskPct(depth, tier, s.bot.abyssPlayerCR(uid))

	tokens, err := chargeAbyssShadowSimulation(r.Context(), s.bot.DB, uid)
	if errors.Is(err, errAbyssShadowInsufficientTokens) {
		writeJSON(w, map[string]any{"ok": false, "error": "not enough Abyss Tokens"})
		return
	}
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "shadow fights could not be purchased"})
		return
	}
	result.Tokens = tokens
	writeJSON(w, map[string]any{
		"ok":         true,
		"simulation": result,
		"message": fmt.Sprintf(
			"%d shadow fights complete: %d%% observed wins.",
			result.Trials,
			result.WinPct,
		),
	})
}
