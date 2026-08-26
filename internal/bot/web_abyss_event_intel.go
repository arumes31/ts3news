package bot

// Server-owned forecasting for the Sanctuary Map Table and the event-chain
// ribbon. Event previews are rolled once, kept out of the public run payload,
// and consumed when their scheduled floor arrives.

import (
	"database/sql"
	"encoding/json"
	"strconv"
)

type abyssEventPreview struct {
	Depth int    `json:"depth"`
	State string `json:"state"`
}

type abyssEventIntelView struct {
	MapOwned   bool
	NextDepth  int
	NextIn     int
	EventLabel string
	Chain      abyssEventChainView
}

func abyssEventPreviewKey(uid string) string { return "abyss_event_preview_" + uid }

func abyssEventTypeLabel(raw string) string {
	var state struct {
		Type string `json:"type"`
	}
	if json.Unmarshal([]byte(raw), &state) != nil {
		return "Unknown anomaly"
	}
	labels := map[string]string{
		"merchant": "Abyssal Market", "imp": "Gambling Imp", "shrine": "Cursed Shrine",
		"wishing_well": "Wishing Well", "gambler": "Card Dealer", "statue": "Heroic Statue",
		"fountain": "Fountain of Youth", "mimic": "Suspicious Chest", "buried_cache": "Buried Cache",
		"puzzle": "Three Chests", "cursed_library": "Cursed Library", "den": "Gambling Den",
		"rift": "Scrying Rift", "blood_altar": "Blood Altar", "alchemy_lab": "Alchemy Lab",
		"mirrors": "Hall of Mirrors", "challenge_room": "Trial Chamber", "cursed_door": "Cursed Door",
		"story_crossroads": "Memorial Crossroads", "lost_explorer": "Lost Explorer", "locked_vault": "Locked Vault",
		"collapsed_passage": "Collapsed Passage", "abyssal_garden": "Abyssal Garden",
		"cursed_elevator": "Cursed Elevator", "trap_chamber": "Trap Chamber",
		"unstable_portal": "Unstable Portal", "graveyard": "Delver Graveyard",
		"echo_floor": "Echo Floor", "bounty_board": "Bounty Board",
		abyssForgeFloorType:        "Silent Anvil",
		abyssEventChainType:        "Triune Sigil Hunt",
		abyssCartographerEventType: "Lost Cartographer",
	}
	if label := labels[state.Type]; label != "" {
		return label
	}
	return "Unknown anomaly"
}

func (b *Bot) ensureAbyssEventPreview(uid string, depth int) {
	if depth <= 0 || b.loadSanctuary(uid)["map"] <= 0 {
		return
	}
	var current sql.NullString
	_ = b.DB.QueryRow("SELECT value FROM app_meta WHERE key=$1", abyssEventPreviewKey(uid)).Scan(&current)
	if current.Valid {
		var preview abyssEventPreview
		if json.Unmarshal([]byte(current.String), &preview) == nil && preview.Depth == depth && preview.State != "" {
			return
		}
	}
	_, state := rollFloorDetail("event")
	payload, err := json.Marshal(abyssEventPreview{Depth: depth, State: state})
	if err != nil {
		return
	}
	_, _ = b.DB.Exec(`INSERT INTO app_meta (key, value) VALUES ($1, $2)
		ON CONFLICT (key) DO UPDATE SET value=EXCLUDED.value`, abyssEventPreviewKey(uid), string(payload))
}

func (b *Bot) takeAbyssEventPreview(uid string, depth int) (string, bool) {
	var raw sql.NullString
	_ = b.DB.QueryRow("SELECT value FROM app_meta WHERE key=$1", abyssEventPreviewKey(uid)).Scan(&raw)
	if !raw.Valid {
		return "", false
	}
	var preview abyssEventPreview
	if json.Unmarshal([]byte(raw.String), &preview) != nil || preview.Depth != depth || preview.State == "" {
		return "", false
	}
	_, _ = b.DB.Exec("DELETE FROM app_meta WHERE key=$1", abyssEventPreviewKey(uid))
	return preview.State, true
}

func (b *Bot) abyssEventIntel(uid string, run abyssRun) abyssEventIntelView {
	view := abyssEventIntelView{MapOwned: b.loadSanctuary(uid)["map"] > 0}
	flags := b.loadRunFlags(uid)
	view.Chain = abyssEventChainFromFlags(flags, run.Depth)
	if !view.MapOwned || !run.Active {
		return view
	}
	var rawDepth string
	_ = b.DB.QueryRow("SELECT value FROM app_meta WHERE key=$1", abyssNextEventDepthKey(uid)).Scan(&rawDepth)
	view.NextDepth, _ = strconv.Atoi(rawDepth)
	view.NextIn = view.NextDepth - run.Depth
	if view.NextIn < 0 {
		view.NextIn = 0
	}
	if view.NextIn > 2 {
		return view
	}
	var rawPreview string
	_ = b.DB.QueryRow("SELECT value FROM app_meta WHERE key=$1", abyssEventPreviewKey(uid)).Scan(&rawPreview)
	var preview abyssEventPreview
	if json.Unmarshal([]byte(rawPreview), &preview) == nil && preview.Depth == view.NextDepth {
		view.EventLabel = abyssEventTypeLabel(preview.State)
	}
	return view
}
