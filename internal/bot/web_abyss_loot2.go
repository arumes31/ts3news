package bot

import (
	"fmt"
	"log"
	"math/rand/v2"
	"strconv"
	"time"

	"ts3news/internal/clientquery"
	"ts3news/internal/content"
)

// Abyss loot & itemization extras (AB-76..100)
// -----------------------------------------------------------------------------
// Companion to web_abyss_loot.go: drop-quality forecasts, the rest-floor loot
// vacuum, per-user loot toggles and collectible counters in app_meta, beam
// intensity classes, the Eternal-drop TS3 fanfare, boss relic lore, stacked
// consumable grants and the corrupted/lucid drop-variant helpers.

// abyssBeamClass maps a drop's rarity to its loot-beam intensity class
// (AB-88): subtle rare → dramatic eternal, with the red-black "beam-doomed"
// reserved for cursed+eldritch co-occurrences (AB-79). The web manifest
// attaches the class to the loot row; chat labels stay BBCode-only.
func abyssBeamClass(r content.Rarity, doomed bool) string {
	if doomed {
		return "beam-doomed"
	}
	switch {
	case r >= content.RarityEternal:
		return "beam-eternal"
	case r >= content.RarityCelestial:
		return "beam-celestial"
	case r >= content.RarityMythic:
		return "beam-high"
	case r >= content.RarityLegendary:
		return "beam-legendary"
	case r >= content.RarityEpic:
		return "beam-epic"
	case r >= content.RarityRare:
		return "beam-rare"
	default:
		return "beam-common"
	}
}

// ---- app_meta-backed per-user loot state -------------------------------------

// abyssMetaGet/Set read and write per-user Abyss flags in the generic
// app_meta key/value table (no schema changes).
func (b *Bot) abyssMetaGet(key string) string {
	var v string
	if err := b.DB.QueryRow("SELECT value FROM app_meta WHERE key=$1", key).Scan(&v); err != nil {
		return ""
	}
	return v
}

func (b *Bot) abyssMetaSet(key, value string) {
	if _, err := b.DB.Exec(`INSERT INTO app_meta (key, value) VALUES ($1, $2)
		ON CONFLICT (key) DO UPDATE SET value=EXCLUDED.value`, key, value); err != nil {
		log.Printf("abyss meta write failed (%s): %v", key, err)
	}
}

// ---- Split pity display (AB-95) ----------------------------------------------

func (b *Bot) abyssCelestialPity(uid string) int {
	n, _ := strconv.Atoi(b.abyssMetaGet("abyss_celestial_pity_" + uid))
	return n
}

func (b *Bot) abyssSetCelestialPity(uid string, n int) {
	b.abyssMetaSet("abyss_celestial_pity_"+uid, strconv.Itoa(n))
}

// abyssPityDisplay returns the split pity counters for the HUD (AB-95): the
// legendary pity (users.legendary_pity, capped at abyssLegendaryPityCap) and
// the celestial counter (gear drops since the last Celestial+ — display only,
// there is no celestial pity cap mechanic).
func (b *Bot) abyssPityDisplay(uid string) (legendary, legendaryCap, celestial int) {
	_ = b.DB.QueryRow("SELECT legendary_pity FROM users WHERE client_uid=$1", uid).Scan(&legendary)
	return legendary, abyssLegendaryPityCap, b.abyssCelestialPity(uid)
}

// ---- Duplicate-legendary auto-convert toggle (AB-87) -------------------------

// abyssDupLegendConvert reports whether the player opted into auto-converting
// duplicate legendaries (owned 2+) into 5 Umbral Cores.
func (b *Bot) abyssDupLegendConvert(uid string) bool {
	return b.loadAbyssLootSettings(uid).DuplicateLegendConvert
}

// setAbyssDupLegendConvert is the toggle the settings UI writes.
func (b *Bot) setAbyssDupLegendConvert(uid string, on bool) error {
	settings := b.loadAbyssLootSettings(uid)
	settings.DuplicateLegendConvert = on
	return b.saveAbyssLootSettings(uid, settings)
}

// ---- Goblin token collectibles (AB-96) ---------------------------------------

// abyssGoblinTitleAt is the number of banked goblin tokens that unlocks the
// cosmetic "Goblin King" title.
const abyssGoblinTitleAt = 5

func (b *Bot) abyssGoblinTokens(uid string) int {
	n, _ := strconv.Atoi(b.abyssMetaGet("abyss_goblin_tokens_" + uid))
	return n
}

// abyssAddGoblinTokens banks goblin tokens into the player's collectible
// counter, returning the new total and whether this grant unlocked the Goblin
// King title (granted immediately through the normal title slot; if an
// unexpired title occupies the slot the tokens stay banked and the unlock
// retried on the next token).
func (b *Bot) abyssAddGoblinTokens(uid string, n int) (int, bool) {
	if n <= 0 {
		return b.abyssGoblinTokens(uid), false
	}
	total := b.abyssGoblinTokens(uid) + n
	unlocked := false
	if total >= abyssGoblinTitleAt {
		t, ok := content.GetTitleByName("Goblin King")
		if !ok {
			t = content.Title{Name: "Goblin King", XPMultiplier: 1.10}
		}
		res, err := b.DB.Exec("UPDATE users SET title=$2, title_mult=$3, title_expires=NOW() + INTERVAL '7 days', title_source='goblin' WHERE client_uid=$1 AND (title IS NULL OR title_expires < NOW())",
			uid, t.Name, t.XPMultiplier)
		if err == nil {
			if rows, _ := res.RowsAffected(); rows > 0 {
				unlocked = true
				total -= abyssGoblinTitleAt
			}
		}
	}
	b.abyssMetaSet("abyss_goblin_tokens_"+uid, strconv.Itoa(total))
	return total, unlocked
}

// ---- Drop-quality forecast (AB-77) -------------------------------------------

// abyssDropForecast is the expected per-type drop distribution of one loot
// roll: each field an individual probability, Common the leftover "common
// gear or small potion" band. Surfaced by the threat-meter tooltip.
type abyssDropForecast struct {
	Ultimate   float64 `json:"ultimate"`
	Title      float64 `json:"title"`
	Unique     float64 `json:"unique"`
	Artifact   float64 `json:"artifact"`
	Enchant    float64 `json:"enchant"`
	Skill      float64 `json:"skill"`
	Consumable float64 `json:"consumable"`
	Gear       float64 `json:"gear"`
	Common     float64 `json:"common"`
}

// abyssDropForecastData computes the individual band probabilities for the
// given quality multiplier and rarity scale — the same bands
// rollAbyssLootToEscrow accumulates into its drop cascade.
func abyssDropForecastData(qualityMult, rareScale float64) abyssDropForecast {
	remaining := 1.0
	claim := func(probability float64) float64 {
		probability = max(0, min(probability, remaining))
		remaining -= probability
		return probability
	}
	f := abyssDropForecast{
		Ultimate: claim(ultimateSkillChance * qualityMult * rareScale),
		Title:    claim(titleChance * qualityMult * rareScale),
		Unique:   claim(uniqueItemChance * qualityMult * rareScale),
		Artifact: claim(artifactChance * qualityMult * rareScale),
		Enchant:  claim(enchChance * qualityMult * rareScale),
		Skill:    claim(skillChance * qualityMult),
	}
	f.Consumable = claim(consChance * qualityMult)
	f.Gear = claim(gearChance * qualityMult)
	f.Common = remaining
	return f
}

// abyssNextFloorForecast estimates the drop-quality distribution for the
// run's next combat floor (the threat-meter tooltip, AB-77). ok is false when
// no run is active. The rarity scale uses the player level as an
// approximation of next floor's mob level.
func (b *Bot) abyssNextFloorForecast(uid string) (f abyssDropForecast, ok bool) {
	run := b.loadAbyssRun(uid)
	if !run.Active {
		return f, false
	}
	diff, _ := abyssDifficulty(run.Depth + 1)
	qualityMult := diff
	if qualityMult < 1.0 {
		qualityMult = 1.0
	}
	var level int
	_ = b.DB.QueryRow("SELECT level FROM users WHERE client_uid=$1", uid).Scan(&level)
	return abyssDropForecastData(qualityMult, lootRarityScale(level)), true
}

// ---- Rest-floor loot vacuum (AB-76) ------------------------------------------

// abyssRestFloorVacuum grants the rest-floor "loot vacuum" consolation: the
// party sweeps up scraps of the drops missed on the way down — a small gold
// find (and from depth 20 occasionally a material) scaled to depth, sealed
// into the run escrow like any other drop so it stays forfeitable on death.
// Adaptation: tracking every individual missed common drop would need
// per-floor drop records, so the vacuum pays a depth-scaled consolation
// instead. Wired up from handleAbyssNonCombatProceed (rest floors only).
func (b *Bot) abyssRestFloorVacuum(uid string, depth int) []string {
	if depth < 1 {
		return nil
	}
	var labels []string
	gold := int64(depth * 10) // ~50% of the value of the common drops walked past
	label := fmt.Sprintf("🧹 Loot vacuum: %d gold swept from the floors above", gold)
	if b.escrowAbyssLoot(uid, depth, label, abyssLootGrant{Type: "gold", Gold: gold}) {
		labels = append(labels, label)
	}
	// #nosec G404 -- non-cryptographic loot roll
	if depth >= 20 && rand.Float64() < 0.25 {
		mat, n := "dust", 1+rand.IntN(2) // #nosec G404
		if depth >= 40 {
			mat = "shard"
		}
		label = fmt.Sprintf("🧹 Loot vacuum: %s ×%d", abyssMaterialName(mat), n)
		if b.escrowAbyssLoot(uid, depth, label, abyssLootGrant{Type: "mat", MatID: mat, MatN: n}) {
			labels = append(labels, label)
		}
	}
	return labels
}

// ---- Eternal drop TS3 fanfare (AB-93) ----------------------------------------

// broadcastAbyssEternalDrop pushes a TS3 channel-wide announcement for an
// Eternal drop, mirroring BroadcastAbyssRecord's nickname+poke fanfare.
// Eternal gear never drops from the roller today (forge ascension/fusion
// only) — the ascension path should call this as well.
func (b *Bot) broadcastAbyssEternalDrop(uid, itemName string) {
	var nick string
	if err := b.DB.QueryRow("SELECT nickname FROM users WHERE client_uid=$1", uid).Scan(&nick); err != nil || nick == "" {
		return
	}
	addr := b.Cfg.ClientQueryAddr
	if addr == "" {
		addr = "127.0.0.1:25639"
	}
	c, err := clientquery.Dial(addr, 2*time.Second)
	if err != nil {
		return
	}
	defer func() { _ = c.Close() }()
	if apiKey := b.getAPIKey(); apiKey != "" {
		_ = c.Auth(apiKey)
	}
	_ = c.Use(1)

	// Nickname is user-controlled; neutralize BBCode so the broadcast can't
	// inject formatting or links (same guard as BroadcastAbyssRecord).
	nick = sanitizeBBCode(nick)

	oldNick := b.Cfg.TS3Nickname
	_ = c.SetNickname("Eternal Drop!")

	clients, err := c.ClientList()
	if err == nil {
		msg := fmt.Sprintf("🌟 ETERNAL! %s has obtained %s — the rarest treasure of the Abyss!", nick, itemName)
		for _, cl := range clients {
			if cl.Type == 0 { // normal user
				_ = c.Poke(cl.CLID, msg)
				// Respect the configured anti-flood poke delay, like the main cycle.
				time.Sleep(time.Duration(b.Cfg.PokeDelayMS) * time.Millisecond)
			}
		}
	}

	time.Sleep(3 * time.Second)
	_ = c.SetNickname(oldNick)
}

// ---- Boss relic flavor text (AB-97) ------------------------------------------

// abyssBossRelicLore maps each named Abyss boss to the flavour inspect text
// of its unique relic and is appended to the escrowed loot label.
func abyssBossRelicLore(bossName string) string {
	switch bossName {
	case "Gorgoroth the Firelord":
		return "Still warm. The Firelord's heart beats once for every delver it outlasted."
	case "Malakor the Voidweaver":
		return "Threads of nothing, spun into something. It hums when you look away."
	case "Azazoth the Slumbering Eye":
		return "It dreams of you now. Try to be interesting."
	case "Abyssus, Heart of the Void":
		return "A shard of the Abyss itself, cracked and sulking."
	default:
		return "A trophy pried from a nameless horror of the deep."
	}
}

// ---- Consumable stacking (AB-98) ---------------------------------------------

// abyssConsumableStackCap is the maximum stack size for identical consumables
// granted from the Abyss escrow (merged stacks render as "x5" count badges).
const abyssConsumableStackCap = 5

// grantConsumableStacked merges an escrowed consumable into the player's
// stack (remaining_fights add), capped at abyssConsumableStackCap charges.
func (b *Bot) grantConsumableStacked(uid, consID string, fights int) error {
	if fights <= 0 {
		fights = 1
	}
	if _, err := b.DB.Exec(
		`INSERT INTO user_consumables (client_uid, cons_id, remaining_fights)
		 VALUES ($1, $2, LEAST($3, $4))
		 ON CONFLICT (client_uid, cons_id)
		 DO UPDATE SET remaining_fights = LEAST(user_consumables.remaining_fights + EXCLUDED.remaining_fights, $4)`,
		uid, consID, fights, abyssConsumableStackCap); err != nil {
		return err
	}
	b.autoCombineConsumable(uid, consID)
	return nil
}

// ---- Drop variants: corrupted consumables (AB-85), lucid insanity (AB-81) ----

// rollAbyssConsumable returns a random consumable drop; 5% of the time it is
// upgraded to its corrupted variant (stronger effect plus self-damage) when
// one exists.
func rollAbyssConsumable() content.Consumable {
	c := content.RandomConsumable()
	// #nosec G404 -- non-cryptographic loot roll
	if rand.Float64() < 0.05 {
		if vid := content.CorruptedConsumableVariant(c.ID); vid != "" {
			if v, ok := content.GetConsumableByID(vid); ok {
				return v
			}
		}
	}
	return c
}

// lucidInsanityVariant strips the negative trade-off stat line from an
// Insanity-tier drop and scales the remaining stats down 20%.
func lucidInsanityVariant(g content.Gear) content.Gear {
	g.Lucid = true
	g.Stats = g.Stats.WithoutNegatives().Scaled(0.8)
	g.Name = "Lucid " + g.Name
	return g
}
