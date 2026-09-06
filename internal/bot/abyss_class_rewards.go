package bot

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"sort"
	"ts3news/internal/content"
)

type abyssClassKill struct {
	ID string `json:"id"`
	XP int64  `json:"xp"`
}
type abyssClassReceipt struct {
	MaximumXP  int64           `json:"maximum_xp"`
	Version    int             `json:"version"`
	Class      string          `json:"class"`
	Kills      map[string]bool `json:"kills"`
	Cleared    bool            `json:"cleared"`
	Efficiency int64           `json:"efficiency"`
}

// Only original hostile pointers are eligible. Removed captures and summons do
// not enter this list; fled goblins are explicitly excluded by the caller.
func abyssWaveClassKills(original, current []*content.Mob, wave, waves int, fled map[*content.Mob]bool) []abyssClassKill {
	eligible := []*content.Mob{}
	for _, m := range original {
		if m != nil && !abyssEnemyHazard(m) {
			eligible = append(eligible, m)
		}
	}
	original = eligible
	hostile := map[*content.Mob]bool{}
	for _, m := range current {
		hostile[m] = true
	}
	result := []abyssClassKill{}
	if len(original) == 0 {
		return result
	}
	budget := int64(1000 / max(1, waves))
	if wave == waves {
		budget = 1000 - int64(waves-1)*(1000/int64(max(1, waves)))
	}
	for i, m := range original {
		if m == nil || !hostile[m] || m.Stats.HP > 0 || fled[m] {
			continue
		}
		xp := budget / int64(len(original))
		if i == len(original)-1 {
			xp = budget - xp*int64(len(original)-1)
		}
		if m.Type == content.MobBoss || m.Type == content.MobLegendary {
			xp += xp / 10
		}
		result = append(result, abyssClassKill{ID: fmt.Sprintf("%d:%d", wave, i), XP: xp})
	}
	return result
}
func applyAbyssClassCredit(p abyssClassProgress, receipt abyssClassReceipt, kills []abyssClassKill, cleared bool, depth int) (abyssClassProgress, abyssClassReceipt, int64) {
	if receipt.Kills == nil {
		receipt.Kills = map[string]bool{}
	}
	if receipt.Efficiency == 0 {
		receipt.Efficiency = 100
		if p.XP >= 75000 && p.BestDepth >= 100 && depth < p.BestDepth/4 {
			receipt.Efficiency = 25
		}
	}
	// Replayed waves can contain different monster counts because action RNG
	// also affects later spawns. Credit only the best completed fraction of
	// this encounter, capped at 1.1 floor equivalents including boss bonuses.
	earned := int64(0)
	seen := map[string]bool{}
	for _, k := range kills {
		if k.ID != "" && k.XP > 0 && !seen[k.ID] {
			seen[k.ID] = true
			earned += min(k.XP, 1100)
		}
	}
	earned = min(1100, earned)
	delta := max(int64(0), earned*receipt.Efficiency/100-receipt.MaximumXP*receipt.Efficiency/100)
	receipt.MaximumXP = max(receipt.MaximumXP, earned)
	delta = min(delta, max(int64(0), content.AbyssClassPointFloors[14]*1000-p.XP))
	p.XP += delta
	if cleared && !receipt.Cleared {
		receipt.Cleared = true
		p.Clears = min(math.MaxInt64/2, p.Clears+1)
		p.BestDepth = max(p.BestDepth, depth)
	}
	return p, receipt, delta
}
func (b *Bot) grantAbyssClassCredit(ctx context.Context, uid, classID, owner string, seed [2]uint64, depth int, kills []abyssClassKill, cleared bool) (int64, error) {
	if _, ok := content.AbyssClassByID(classID); !ok || uid == "" || seed == [2]uint64{} || depth < 1 {
		return 0, errors.New("invalid class reward identity")
	}
	receiptKey := fmt.Sprintf("abyss_class_reward:%s:%s:%016x%016x", uid, owner, seed[0], seed[1])
	tx, err := b.DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	var locked string
	if err = tx.QueryRowContext(ctx, "SELECT client_uid FROM users WHERE client_uid=$1 FOR UPDATE", uid).Scan(&locked); err != nil {
		return 0, err
	}
	var raw string
	state := newAbyssClassState()
	err = tx.QueryRowContext(ctx, "SELECT value FROM app_meta WHERE key=$1", abyssClassKey(uid)).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, errors.New("class selection missing")
	}
	if err != nil {
		return 0, err
	}
	state, err = decodeAbyssClassState(raw)
	if err != nil {
		return 0, err
	}
	receipt := abyssClassReceipt{Version: 1, Class: classID, Kills: map[string]bool{}}
	err = tx.QueryRowContext(ctx, "SELECT value FROM app_meta WHERE key=$1", receiptKey).Scan(&raw)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return 0, err
	}
	if err == nil {
		receipt = abyssClassReceipt{}
		if err = json.Unmarshal([]byte(raw), &receipt); err != nil {
			return 0, err
		}
		if receipt.Version != 1 || receipt.MaximumXP < 0 || receipt.MaximumXP > 1100 || (receipt.Efficiency != 25 && receipt.Efficiency != 100) {
			return 0, errors.New("unsupported class reward receipt")
		}
	}
	// A partially credited floor belongs to its original class even if a stale
	// delivery arrives after the player has changed their build.
	if receipt.Class != classID {
		return 0, errors.New("class reward changed its class")
	}
	progress, updated, delta := applyAbyssClassCredit(state.Progress[classID], receipt, kills, cleared, depth)
	state.Progress[classID] = progress
	// XP updates do not change build revision: a reward arriving after banking
	// must not invalidate an unrelated allocation snapshot.
	stateRaw, err := json.Marshal(state)
	if err != nil {
		return 0, err
	}
	receiptRaw, err := json.Marshal(updated)
	if err != nil {
		return 0, err
	}
	for _, entry := range [][2]string{{abyssClassKey(uid), string(stateRaw)}, {receiptKey, string(receiptRaw)}} {
		if _, err = tx.ExecContext(ctx, "INSERT INTO app_meta (key,value) VALUES ($1,$2) ON CONFLICT (key) DO UPDATE SET value=EXCLUDED.value", entry[0], entry[1]); err != nil {
			return 0, err
		}
	}
	if err = tx.Commit(); err != nil {
		return 0, err
	}
	return delta, nil
}
func (b *Bot) awardAbyssClassCredits(ctx context.Context, users []UserInCombat, owner string, seed [2]uint64, depth int, victory bool) []string {
	indexes := map[string]int{}
	for i, u := range users {
		if !u.shadow && u.UID != "" && u.AbyssClass != "" {
			indexes[u.UID] = i
		}
	}
	ids := make([]string, 0, len(indexes))
	for id := range indexes {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	messages := []string{}
	for _, id := range ids {
		u := users[indexes[id]]
		if len(u.classKillCredit) == 0 && !victory {
			continue
		}
		var delta int64
		var err error
		for attempt := 0; attempt < 3; attempt++ {
			delta, err = b.deliverAbyssClassPending(ctx, abyssClassPending{UID: id, Class: u.AbyssClass, Owner: owner, Seed: seed, Depth: depth, Kills: append([]abyssClassKill(nil), u.classKillCredit...), Cleared: victory})
			if err == nil {
				break
			}
		}
		if err != nil {
			log.Printf("abyss class XP save failed for %s: %v", id, err)
			b.retryAbyssClassPending(abyssClassPending{UID: id, Class: u.AbyssClass, Owner: owner, Seed: seed, Depth: depth, Kills: append([]abyssClassKill(nil), u.classKillCredit...), Cleared: victory})
			messages = append(messages, "Class XP is waiting for a save retry. Your existing class progress is unchanged.")
			continue
		}
		if delta > 0 {
			messages = append(messages, fmt.Sprintf("%s gains %.3f class XP for %s. Permanent progress is kept on defeat.", u.Nickname, float64(delta)/1000, u.AbyssClass))
		}
	}
	return messages
}
