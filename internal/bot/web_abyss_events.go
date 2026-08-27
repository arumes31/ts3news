package bot

// Focused expansion mechanics for Abyss event floors. Keeping new encounters
// here prevents the legacy non-combat switch in web_abyss.go from growing into
// an even larger mixed UI/economy monolith.

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/rand/v2"
	"net/http"

	"ts3news/internal/content"
)

const abyssMarketMysteryPrice int64 = 750

const abyssRunFlagRiftPeeks = "rift_peeks"

type abyssEventEnvelope struct {
	Type             string   `json:"type"`
	Depth            int      `json:"depth"`
	MemoryMultiplier float64  `json:"mem_mult"`
	MysteryAvailable bool     `json:"mystery_available"`
	MysteryPrice     int64    `json:"mystery_price"`
	Options          []string `json:"options"`
}

func parseAbyssEventEnvelope(raw string) abyssEventEnvelope {
	var state abyssEventEnvelope
	_ = json.Unmarshal([]byte(raw), &state)
	if state.MemoryMultiplier < 1 {
		state.MemoryMultiplier = 1
	}
	if state.MemoryMultiplier > 1.5 {
		state.MemoryMultiplier = 1.5
	}
	return state
}

func abyssEventOffer(base int64, raw string) int64 {
	return int64(math.Round(float64(base) * parseAbyssEventEnvelope(raw).MemoryMultiplier))
}

func abyssAltarBuffDuration(raw string, corrupted bool) int {
	duration := int(math.Ceil(3 * parseAbyssEventEnvelope(raw).MemoryMultiplier))
	if corrupted {
		duration *= 2
	}
	return duration
}

func abyssRiskyBrewDuration(base int) int {
	return max(1, int(math.Ceil(float64(base)*1.5)))
}

func abyssMirrorBuffDuration(streak int) int {
	if streak >= 3 {
		return 4
	}
	return 3
}

func advanceAbyssMirrorMemory(memory abyssMirrorMemory, pick, runID string) abyssMirrorMemory {
	if memory.LastRun == runID {
		return memory
	}
	if memory.Pick == pick {
		memory.Streak++
	} else {
		memory.Pick = pick
		memory.Streak = 1
	}
	memory.LastRun = runID
	return memory
}

type abyssDenGame struct {
	Stake int64
	Prize int64
	Odds  float64
	Label string
}

// abyssDenGameFor keeps posted odds and authoritative settlement values on one
// path. High-roller actions are the same games at 10x stakes and payouts and
// cannot be invoked before depth 41.
func abyssDenGameFor(action string, depth int) (abyssDenGame, bool) {
	high := false
	const suffix = "_high"
	if len(action) > len(suffix) && action[len(action)-len(suffix):] == suffix {
		high = true
		action = action[:len(action)-len(suffix)]
	}
	var game abyssDenGame
	switch action {
	case "den_dice":
		game = abyssDenGame{Stake: 300, Prize: 600, Odds: 0.50, Label: "🎲 Dice"}
	case "den_card":
		game = abyssDenGame{Stake: 300, Prize: 900, Odds: 0.33, Label: "🃏 High Card"}
	case "den_wheel":
		game = abyssDenGame{Stake: 500, Prize: 4000, Odds: 0.10, Label: "🎡 Wheel"}
	case "den_longshot":
		game = abyssDenGame{Stake: 200, Prize: 4000, Odds: 0.05, Label: "🎯 Long Shot"}
	case "den_cascade":
		game = abyssDenGame{Stake: 400, Prize: 600, Odds: 0.75, Label: "🪙 Coin Cascade"}
	default:
		return abyssDenGame{}, false
	}
	if high {
		if depth <= 40 {
			return abyssDenGame{}, false
		}
		game.Stake *= 10
		game.Prize *= 10
		game.Label += " · High Roller"
	}
	return game, true
}

// rollAbyssMysteryGear follows the exact posted market odds. The result is
// intentionally rolled server-side after purchase, never embedded in state.
func rollAbyssMysteryGear() (content.Gear, int) {
	roll := rand.IntN(100) // #nosec G404 -- non-cryptographic loot roll
	rarity := content.RarityCommon
	switch {
	case roll >= 99:
		rarity = content.RarityLegendary
	case roll >= 95:
		rarity = content.RarityEpic
	case roll >= 83:
		rarity = content.RarityRare
	case roll >= 55:
		rarity = content.RarityUncommon
	}
	candidates := content.GearByMinRarity(rarity)
	exact := candidates[:0]
	for _, gear := range candidates {
		if gear.Rarity == rarity {
			exact = append(exact, gear)
		}
	}
	if len(exact) == 0 {
		return content.RandomGearDrop(), roll
	}
	gear := exact[rand.IntN(len(exact))] // #nosec G404 -- non-cryptographic loot roll
	gear.Special = content.RandomItemEffect()
	return gear, roll
}

func addAbyssWellLifetime(exec dbExecQuerier, uid string, amount int64) (int64, error) {
	var raw string
	err := exec.QueryRow(`INSERT INTO app_meta (key, value) VALUES ($1, $2)
		ON CONFLICT (key) DO UPDATE SET value = (COALESCE(NULLIF(app_meta.value, '')::bigint, 0) + $3)::text
		RETURNING value`, abyssWellLifetimeKey(uid), fmt.Sprintf("%d", amount), amount).Scan(&raw)
	if err != nil {
		return 0, err
	}
	var total int64
	_, err = fmt.Sscan(raw, &total)
	return total, err
}

// handleAbyssExpandedEventAction owns only actions added by the event expansion.
// It is called inside the existing per-UID lock and returns false for legacy
// actions so their established handler remains unchanged.
func (s *WebServer) handleAbyssExpandedEventAction(
	w http.ResponseWriter,
	uid string,
	run abyssRun,
	action string,
) bool {
	state := parseAbyssEventEnvelope(run.EventState)
	switch action {
	case "library_trade_spd":
		if state.Type != "cursed_library" {
			writeJSON(w, map[string]any{"ok": false, "error": "wrong floor type for library_trade_spd"})
			return true
		}
		const curseFights = 3
		tx, err := s.bot.DB.Begin()
		if err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return true
		}
		defer func() { _ = tx.Rollback() }()
		flags, err := loadAbyssRunFlagsInTx(tx, uid)
		if err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return true
		}
		flags["spd_curse"] += curseFights
		if err := saveAbyssRunFlagsInTx(tx, uid, flags); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return true
		}
		fragID := 1 + rand.IntN(10) // #nosec G404 -- non-cryptographic lore roll
		loreUnlocked, loreTokens, err := grantAbyssLoreFragment(tx, uid, fragID)
		if err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return true
		}
		if _, err := tx.Exec("UPDATE abyss_active SET event_state=NULL, last_action_at=NOW() WHERE client_uid=$1", uid); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return true
		}
		if err := tx.Commit(); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return true
		}
		elixirFights := int(math.Ceil(state.MemoryMultiplier))
		if consumable, ok := content.GetConsumableByID("intellect_elixir"); ok {
			elixirFights = int(math.Ceil(float64(max(consumable.Duration, 1)) * state.MemoryMultiplier))
		}
		s.bot.grantConsumable(uid, "intellect_elixir", elixirFights)
		msg := fmt.Sprintf("📚 The library takes your speed for %d fights and yields lore plus an Intellect Elixir for %d fight(s).", curseFights, elixirFights)
		if !loreUnlocked && loreTokens > 0 {
			msg = fmt.Sprintf("📚 Familiar pages become %d Abyss Tokens; the speed curse lasts %d fights and the elixir %d.", loreTokens, curseFights, elixirFights)
		}
		if recipe := s.bot.discoverRandomRecipe(uid); recipe != "" {
			msg += " 📖 Recipe discovered: " + recipe + "!"
		}
		writeJSON(w, map[string]any{"ok": true, "resolved": true, "msg": msg, "consumables": s.bot.getConsumables(uid)})
		return true

	case "market_mystery":
		if state.Type != "merchant" || !state.MysteryAvailable {
			writeJSON(w, map[string]any{"ok": false, "error": "no mystery box is available"})
			return true
		}
		price := state.MysteryPrice
		if price <= 0 {
			price = abyssMarketMysteryPrice
		}
		gear, _ := rollAbyssMysteryGear()
		grant, err := json.Marshal(abyssLootGrant{Type: "gear", Gear: &gear})
		if err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return true
		}
		var raw map[string]any
		if err := json.Unmarshal([]byte(run.EventState), &raw); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "invalid event state"})
			return true
		}
		raw["mystery_available"] = false
		updated, err := json.Marshal(raw)
		if err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return true
		}
		label := fmt.Sprintf("%s [s:%s] (gs:%d R:%s)", gear.Name, string(gear.Slot), gear.Stats.Score(), gear.Rarity.String())
		tx, err := s.bot.DB.Begin()
		if err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return true
		}
		defer func() { _ = tx.Rollback() }()
		var newGold int64
		err = tx.QueryRow("UPDATE users SET gold=gold-$1 WHERE client_uid=$2 AND gold >= $1 RETURNING gold", price, uid).Scan(&newGold)
		if errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, map[string]any{"ok": false, "error": "not enough gold"})
			return true
		}
		if err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return true
		}
		if _, err := tx.Exec("INSERT INTO abyss_escrow_loot (client_uid, item_type, label, item_data, depth) VALUES ($1,$2,$3,$4,$5)", uid, "gear", label, grant, run.Depth); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return true
		}
		if _, err := tx.Exec("UPDATE abyss_active SET event_state=$1, last_action_at=NOW() WHERE client_uid=$2", string(updated), uid); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return true
		}
		if err := tx.Commit(); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return true
		}
		writeJSON(w, map[string]any{
			"ok": true, "gold": newGold, "event_state": string(updated),
			"msg": "🎁 The mystery seal breaks: " + label + " is secured in your run cache.",
		})
		return true
	}
	return false
}
