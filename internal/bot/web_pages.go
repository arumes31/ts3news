package bot

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"strings"
	"time"

	"ts3news/internal/content"
)

// statKV is a single non-zero gear stat for display.
type statKV struct {
	Label string
	Value int
}

// gearView is a template-friendly view of a gear piece.
type gearView struct {
	InvID       int64
	Slot        string
	Icon        string
	IconName    string // game-icons.net SVG basename for the slot
	ID          string
	Name        string
	Rarity      string
	RarityColor string
	RarityIcon  string // game-icons.net SVG basename for the rarity
	CR          float64
	Score       int
	Durability  int
	Empty       bool
	AHPrice     int64 // auto-calculated auction house listing price

	// Detail surfaced in the armoury/inventory.
	Element    string
	Effect     string
	EffectIcon string // game-icons.net SVG basename for the effect
	EffectDesc string
	XPBonusPct int
	Stats      []statKV

	Unidentified bool
	Sockets      int
	Gemstones    []string
	RarityVal    int
	Insured      bool // whether the piece is death-insured (drives the forge picker)
	Corrupted    bool // carries an HP malus, cleansable at the forge (#83)
	Temper       int  // forge temper level (#106)
}

// gearEffectDescriptions maps each special effect to a short player-facing blurb.
var gearEffectDescriptions = map[content.ItemEffect]string{
	content.EffectThorns:         "Reflects 10% of damage taken",
	content.EffectVampiric:       "Heals for 5% of damage dealt",
	content.EffectBerserk:        "+20% STR while below 50% HP",
	content.EffectLucky:          "+10% Luck",
	content.EffectTreasureHunter: "+5% item find chance",
	content.EffectQuick:          "+10% Speed",
	content.EffectBulwark:        "+10% Defense",
	content.EffectRadiant:        "+10% XP gained",
	content.EffectFragile:        "+30% STR but double durability loss",
	content.EffectSteady:         "-50% stun chance",
	content.EffectMindControl:    "Chance to capture low-health mobs",
	content.EffectRegenStack:     "Permanent regen stack on victory",
	content.EffectPhoenix:        "Revive once per fight at 50% HP",
	content.EffectStealth:        "Skip first-round mob damage",
	content.EffectParry:          "10% chance to negate a hit and counter",
	content.EffectCleanse:        "Removes a negative effect each turn",
}

// gearStatList returns the gear's non-zero combat stats, largest first.
func gearStatList(s content.Stats) []statKV {
	pairs := []statKV{
		{"HP", s.HP}, {"MNA", s.MNA}, {"STR", s.STR}, {"DEF", s.DEF}, {"SPD", s.SPD},
		{"CRT%", s.CRT}, {"DGE%", s.DGE}, {"LCK", s.LCK}, {"INT", s.INT}, {"STA", s.STA},
	}
	var out []statKV
	for _, p := range pairs {
		if p.Value != 0 {
			out = append(out, p)
		}
	}
	return out
}

func toGearView(slot content.GearSlot, g content.Gear) gearView {
	name := g.Name
	stats := gearStatList(g.Stats)
	effDesc := gearEffectDescriptions[g.Special]

	if g.Unidentified {
		name = "Unidentified " + string(slot)
		stats = []statKV{{"???", 0}}
		effDesc = "Identify this item to reveal its stats and effects."
	} else {
		if g.GearLevel > 0 {
			name = fmt.Sprintf("%s +%d", name, g.GearLevel)
		}
		if g.Cursed {
			name = "💀 Cursed " + name
			if effDesc != "" {
				effDesc += " | "
			}
			effDesc += "Cursed: -2% HP per round"
		}
		if g.Eldritch {
			name = "🌌 Eldritch " + name
			if effDesc != "" {
				effDesc += " | "
			}
			effDesc += "Eldritch: Cosmic horror affix applied"
		}
		if g.Rune != "" {
			name = fmt.Sprintf("%s (%s Rune)", name, g.Rune)
		}
		if g.Sockets > 0 {
			gemsText := ""
			for i := 0; i < g.Sockets; i++ {
				if i < len(g.Gemstones) {
					gemsText += " 💎" + g.Gemstones[i]
				} else {
					gemsText += " 🔘Empty"
				}
			}
			if effDesc != "" {
				effDesc += " | "
			}
			effDesc += "Sockets:" + gemsText
		}
		if g.Insured {
			name = "🛡️ " + name
		}
		if g.Temper > 0 {
			name = fmt.Sprintf("%s ⚒+%d", name, g.Temper)
		}
		if len(g.BonusEffects) > 0 {
			names := make([]string, 0, len(g.BonusEffects))
			for _, e := range g.BonusEffects {
				names = append(names, string(e))
			}
			if effDesc != "" {
				effDesc += " | "
			}
			effDesc += "Bonus Effects: " + strings.Join(names, ", ")
		}
	}

	v := gearView{
		Slot:        string(slot),
		Icon:        content.SlotIcon(slot),
		IconName:    content.SlotIconName(slot),
		ID:          g.ID,
		Name:        name,
		Rarity:      g.Rarity.String(),
		RarityColor: g.Rarity.Color(),
		RarityIcon:  content.RarityIconName(g.Rarity),
		CR:          g.CombatRating(),
		Score:       g.Stats.Score(),
		Stats:       stats,
		XPBonusPct:  int(math.Round((g.XPMultiplier - 1.0) * 100)),
		Unidentified: g.Unidentified,
		Sockets:      g.Sockets,
		Gemstones:    g.Gemstones,
		RarityVal:    int(g.Rarity),
		Insured:      g.Insured,
		Corrupted:    g.Corrupted,
		Temper:       g.Temper,
	}
	if g.Element != "" && g.Element != content.ElementPhysical {
		v.Element = string(g.Element)
	}
	if g.Special != content.EffectNone && !g.Unidentified {
		v.Effect = string(g.Special)
		v.EffectIcon = content.EffectIconName(g.Special)
		v.EffectDesc = effDesc
	} else if !g.Unidentified && len(g.BonusEffects) > 0 {
		// No base Special, but Mythic/Divine bonus affixes: show the first as the tag.
		v.Effect = string(g.BonusEffects[0])
		v.EffectIcon = content.EffectIconName(g.BonusEffects[0])
		v.EffectDesc = effDesc
	} else if g.Unidentified {
		v.EffectDesc = effDesc
	} else if effDesc != "" {
		// Identified gear with no base Special but with cursed/eldritch/socket affixes
		// still has an assembled description; surface it so those affixes render.
		v.EffectDesc = effDesc
	}
	return v
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("web: json encode failed: %v", err)
	}
}

// artifactView is the equipped corrupted/blessed artifact, for the armoury.
type artifactView struct {
	Name       string
	XPPct      int // signed XP bonus percentage (+boon, -curse)
	Boon       bool
	Durability int
}

// petView is a combat companion captured via the Mind Control effect, for the
// armoury. These are living mobs from user_pets, not the Pet1/Pet2 gear slots.
type petView struct {
	Name  string
	Type  string
	Level int
	HP    int
	MaxHP int
	STR   int
	DEF   int
	SPD   int
	Score int
}

// loadPetViews returns the user's captured combat pets.
func (b *Bot) loadPetViews(uid string) []petView {
	var out []petView
	for _, m := range b.getPets(uid) {
		out = append(out, petView{
			Name:  m.Name,
			Type:  string(m.Type),
			Level: m.Level,
			HP:    m.Stats.HP,
			MaxHP: m.MaxHP,
			STR:   m.Stats.STR,
			DEF:   m.Stats.DEF,
			SPD:   m.Stats.SPD,
			Score: m.Score(),
		})
	}
	return out
}

// titleView is the active, time-limited title and its XP bonus.
type titleView struct {
	Name      string
	XPPct     int
	ExpiresIn string // human-readable remaining time, "" if permanent
}

// humanDur renders a duration as a compact "Xd Yh" / "Xh Ym" / "Xm" string.
func humanDur(d time.Duration) string {
	if d <= 0 {
		return "expired"
	}
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	mins := int(d.Minutes()) % 60
	switch {
	case days > 0:
		return fmt.Sprintf("%dd %dh", days, hours)
	case hours > 0:
		return fmt.Sprintf("%dh %dm", hours, mins)
	default:
		return fmt.Sprintf("%dm", mins)
	}
}

// artifactView returns the user's active artifact, or nil if none is equipped.
func (b *Bot) loadArtifactView(uid string) *artifactView {
	var mult sql.NullFloat64
	var name sql.NullString
	var dura sql.NullInt64
	if err := b.DB.QueryRow(
		"SELECT artifact_mult, artifact_name, artifact_durability FROM users WHERE client_uid=$1", uid,
	).Scan(&mult, &name, &dura); err != nil {
		return nil
	}
	if !name.Valid || name.String == "" {
		return nil
	}
	m := 1.0
	if mult.Valid {
		m = mult.Float64
	}
	return &artifactView{
		Name:       name.String,
		XPPct:      int(math.Round((m - 1.0) * 100)),
		Boon:       m >= 1.0,
		Durability: int(dura.Int64),
	}
}

// loadTitleView returns the user's active title, or nil if none is held.
func (b *Bot) loadTitleView(uid string) *titleView {
	var name sql.NullString
	var mult sql.NullFloat64
	var expires sql.NullTime
	if err := b.DB.QueryRow(
		"SELECT title, title_mult, title_expires FROM users WHERE client_uid=$1", uid,
	).Scan(&name, &mult, &expires); err != nil {
		return nil
	}
	if !name.Valid || name.String == "" {
		return nil
	}
	m := 1.0
	if mult.Valid {
		m = mult.Float64
	}
	tv := &titleView{Name: name.String, XPPct: int(math.Round((m - 1.0) * 100))}
	if expires.Valid {
		tv.ExpiresIn = humanDur(time.Until(expires.Time))
	}
	return tv
}

func (s *WebServer) handleArmory(w http.ResponseWriter, r *http.Request, uid string) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	u, err := s.loadWebUser(uid)
	if err != nil {
		http.Redirect(w, r, "/denied", http.StatusSeeOther)
		return
	}

	equipped := s.bot.getEquippedItems(uid)
	var slots []gearView
	for _, slot := range content.AllSlots {
		if g, ok := equipped[slot]; ok {
			slots = append(slots, toGearView(slot, g))
		} else {
			slots = append(slots, gearView{Slot: string(slot), Icon: content.SlotIcon(slot), IconName: content.SlotIconName(slot), Empty: true})
		}
	}

	skills := s.bot.getSkills(uid)
	ultimates := s.bot.getActiveUltimates(uid)
	artifact := s.bot.loadArtifactView(uid)
	title := s.bot.loadTitleView(uid)
	pets := s.bot.loadPetViews(uid)

	s.render(w, "armory", map[string]any{
		"Title":       "Armoury",
		"Nav":         "armory",
		"U":           u,
		"Slots":       slots,
		"Skills":      skills,
		"Ultimates":   ultimates,
		"Artifact":    artifact,
		"PlayerTitle": title,
		"Pets":        pets,
	})
}

func (s *WebServer) handleInventory(w http.ResponseWriter, r *http.Request, uid string) {
	u, err := s.loadWebUser(uid)
	if err != nil {
		http.Redirect(w, r, "/denied", http.StatusSeeOther)
		return
	}

	items := s.bot.inventoryItems(uid)
	cons := s.bot.consumableCounts(uid)

	s.render(w, "inventory", map[string]any{
		"Title":       "Inventory",
		"Nav":         "inventory",
		"U":           u,
		"Items":       items,
		"Consumables": cons,
	})
}

// inventoryItems returns the user's owned, unequipped gear.
func (b *Bot) inventoryItems(uid string) []gearView {
	rows, err := b.DB.Query("SELECT id, gear_id, durability, item_data FROM user_inventory WHERE client_uid=$1 ORDER BY id DESC", uid)
	if err != nil {
		return nil
	}
	defer func() { _ = rows.Close() }()
	var out []gearView
	for rows.Next() {
		var id int64
		var gid string
		var dur int
		var itemData sql.NullString
		if err := rows.Scan(&id, &gid, &dur, &itemData); err != nil {
			continue
		}
		g, ok := b.makeGear(gid, itemData)
		if !ok {
			continue
		}
		v := toGearView(g.Slot, g)
		v.InvID = id
		v.Durability = dur
		// Auto-calculate AH listing price: (CR×10 + GS×5) × (Rarity+1)
		price := int64(g.CombatRating()*10+float64(g.Stats.Score())*5) * (int64(g.Rarity) + 1)
		if price < 10 {
			price = 10
		}
		v.AHPrice = price
		out = append(out, v)
	}
	return out
}

type consumableView struct {
	Name  string
	Count int
}

func (b *Bot) consumableCounts(uid string) []consumableView {
	rows, err := b.DB.Query("SELECT cons_id, remaining_fights FROM user_consumables WHERE client_uid=$1", uid)
	if err != nil {
		return nil
	}
	defer func() { _ = rows.Close() }()
	var out []consumableView
	for rows.Next() {
		var cid string
		var n int
		if err := rows.Scan(&cid, &n); err != nil {
			continue
		}
		name := cid
		if c, ok := content.GetConsumableByID(cid); ok {
			name = c.Name
		}
		out = append(out, consumableView{Name: name, Count: n})
	}
	return out
}

// handleEquipAPI equips an inventory item, swapping any currently-equipped piece
// in that slot back into the inventory.
func (s *WebServer) handleEquipAPI(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		InvID int64 `json:"inv_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}

	// Look up the inventory item.
	var gid string
	var dur int
	var itemData sql.NullString
	if err := s.bot.DB.QueryRow("SELECT gear_id, durability, item_data FROM user_inventory WHERE id=$1 AND client_uid=$2", req.InvID, uid).Scan(&gid, &dur, &itemData); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "item not found"})
		return
	}
	g, ok := s.bot.makeGear(gid, itemData)
	if !ok {
		writeJSON(w, map[string]any{"ok": false, "error": "unknown gear"})
		return
	}

	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "tx"})
		return
	}
	defer func() { _ = tx.Rollback() }()

	// Remove from inventory.
	if _, err := tx.Exec("DELETE FROM user_inventory WHERE id=$1 AND client_uid=$2", req.InvID, uid); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "remove"})
		return
	}

	// Equip the new piece (automatically displacing the old one if any)
	if err := s.bot.equipGear(tx, uid, g, dur, itemData); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "equip"})
		return
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "commit"})
		return
	}
	writeJSON(w, map[string]any{"ok": true, "equipped": g.Name, "slot": string(g.Slot)})
}

// handleSellAPI vendors an inventory item for half its fair price.
func (s *WebServer) handleSellAPI(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		InvID int64 `json:"inv_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}

	var gid string
	if err := s.bot.DB.QueryRow("SELECT gear_id FROM user_inventory WHERE id=$1 AND client_uid=$2", req.InvID, uid).Scan(&gid); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "item not found"})
		return
	}
	g, ok := content.GetGearByID(gid)
	if !ok {
		writeJSON(w, map[string]any{"ok": false, "error": "unknown gear"})
		return
	}
	value := gearPrice(g) / 2
	if value < 1 {
		value = 1
	}

	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "tx"})
		return
	}
	defer func() { _ = tx.Rollback() }()
	res, err := tx.Exec("DELETE FROM user_inventory WHERE id=$1 AND client_uid=$2", req.InvID, uid)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "remove"})
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		writeJSON(w, map[string]any{"ok": false, "error": "already sold"})
		return
	}
	var gold int64
	if err := tx.QueryRow("UPDATE users SET gold = gold + $1 WHERE client_uid=$2 RETURNING gold", value, uid).Scan(&gold); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "gold"})
		return
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "commit"})
		return
	}

	writeJSON(w, map[string]any{"ok": true, "value": value, "gold": gold})
}
