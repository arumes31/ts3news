package bot

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"math/rand/v2"
	"mime"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"ts3news/internal/content"
	"ts3news/internal/db"
	"ts3news/internal/rift"
)

var errRiftConflict = errors.New("the expedition changed; reload its saved state")

type riftRequest struct {
	LevelID   int        `json:"level_id,omitempty"`
	Kind      string     `json:"kind"`
	RunID     string     `json:"run_id"`
	RequestID string     `json:"request_id"`
	Revision  int        `json:"revision"`
	Skills    []string   `json:"skills,omitempty"`
	Input     rift.Input `json:"input"`
}

func (s *WebServer) handleRiftPage(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	s.render(w, "rift", map[string]any{"Title": "Rift Brawl", "Nav": "rift", "EnableAbyss": true, "AccountNav": true})
}

func (b *Bot) riftBuild(ctx context.Context, uid string) (rift.Build, error) {
	stats, _, _, _ := b.calculateTotalStats(uid, time.Now())
	var name sql.NullString
	var level int
	if err := b.DB.QueryRowContext(ctx, "SELECT nickname, level FROM users WHERE client_uid=$1", uid).Scan(&name, &level); err != nil {
		return rift.Build{}, err
	}
	u := UserInCombat{Stats: stats, Skills: b.getSkills(uid), Equipped: abyssPlayerEquipment(b.getEquippedItems(uid)), Pets: b.getPets(uid), Ultimates: b.getActiveUltimates(uid)}
	state, err := b.loadAbyssClassState(ctx, uid)
	if err != nil {
		return rift.Build{}, err
	}
	b.applyAbyssClassBuild(&u, state)
	return riftBuildFromUser(u, name.String, level), nil
}

func riftBuildFromUser(u UserInCombat, name string, level int) rift.Build {
	// Compress the RPG's very large stat range for readable action combat while
	// retaining permanent progression and equipment differences in this mode.
	scale := func(n int) float64 { return math.Log2(1 + float64(max(0, n))) }
	build := rift.Build{Name: name, BaseClass: u.AbyssClass, Class: u.AbyssSubclass, Level: level, HP: 160 + scale(u.Stats.HP)*15, Damage: 12 + scale(max(u.Stats.STR, u.Stats.INT))*3, Armor: scale(u.Stats.DEF), Weapon: "Unarmed", Skills: []rift.Skill{}, Signatures: []rift.Skill{}, Gear: []string{}}
	if build.Class == "" {
		build.Class = u.AbyssClass
	}
	if build.Class == "" {
		build.Class = "adventurer"
	}
	if build.Name == "" {
		build.Name = "Delver"
	}
	if style, ok := content.AbyssCombatStyle(build.Class); ok {
		build.BaseClass = style.ClassID
		build.ClassName = style.Name
		build.Resource = style.Resource
		build.Sequence = style.Sequence
	}
	for _, pet := range u.Pets {
		if pet != nil && pet.Stats.HP > 0 {
			build.Pets++
		}
	}
	_, build.Relic = u.Equipped[content.SlotRelic]
	for _, g := range u.Equipped {
		build.Gear = append(build.Gear, g.Name)
		if g.Slot == content.SlotMainHand {
			build.Weapon = g.Name
		}
	}
	sort.Strings(build.Gear)
	seen := map[string]bool{}
	for _, s := range u.Skills {
		if seen[s.ID] {
			continue
		}
		seen[s.ID] = true
		kind := "fire"
		switch {
		case s.HealPercent > 0 && s.Power == 0:
			kind = "heal"
		case s.Type == content.SkillBuff:
			kind = "shield"
		case s.Type == content.SkillPhysical:
			kind = "slash"
		case strings.Contains(strings.ToLower(string(s.Element)), "frost"), strings.Contains(strings.ToLower(s.Name), "ice"):
			kind = "ice"
		case strings.Contains(strings.ToLower(string(s.Element)), "void"), s.Type == content.SkillDebuff:
			kind = "void"
		}
		role := abyssClassSkillRole(s)
		if role != "" {
			kind = riftSignatureKind(build.Class, role, kind)
		}
		skill := rift.Skill{ID: s.ID, Name: s.Name, Kind: kind, Role: role, Power: math.Max(0, math.Min(4, s.Power)), Damage: 12 + scale(abyssSkillBase(&u, s))*3, Heal: s.HealPercent, Pierce: s.IgnoreDef, Cost: math.Max(12, math.Min(45, float64(s.ManaCost))), Cooldown: math.Max(2, math.Min(12, float64(s.CooldownRounds)*1.5))}
		if role != "" {
			build.Signatures = append(build.Signatures, skill)
		} else {
			build.Skills = append(build.Skills, skill)
		}
	}
	sort.SliceStable(build.Signatures, func(i, j int) bool {
		return build.Signatures[i].Role == "builder" && build.Signatures[j].Role != "builder"
	})
	for _, ult := range u.Ultimates {
		if ult != nil {
			build.Ultimate = &rift.Skill{ID: ult.ID, Name: ult.Name, Kind: "ultimate", Power: math.Max(3, math.Min(7, ult.Power)), Damage: build.Damage, Cost: 70, Cooldown: math.Max(15, float64(ult.CooldownRounds)*1.5)}
			break
		}
	}
	return build
}

func riftSignatureKind(style, role, fallback string) string {
	forms := map[string][2]string{"vanguard": {"shield", "slash"}, "berserker": {"slash", "slash"}, "marksman": {"arrow", "arrow"}, "beastmaster": {"arrow", "pack"}, "elementalist": {"fire", "ice"}, "chronomancer": {"void", "void"}, "oracle": {"heal", "radiant"}, "geomancer": {"shield", "quake"}, "bloodblade": {"slash", "slash"}, "voidwalker": {"void", "void"}, "runesmith": {"rune", "rune"}, "alchemist": {"poison", "fire"}}
	if kinds, ok := forms[style]; ok {
		if role == "builder" {
			return kinds[0]
		}
		return kinds[1]
	}
	return fallback
}

func riftRarities() []map[string]any {
	var values []map[string]any
	for rarity := content.RarityCommon; rarity <= content.RarityEternal; rarity++ {
		values = append(values, map[string]any{"value": int(rarity), "name": rarity.String(), "color": rarity.Color()})
	}
	return values
}

func (s *WebServer) handleRiftAPI(w http.ResponseWriter, r *http.Request, uid string) {
	w.Header().Set("Cache-Control", "no-store")
	if r.Method == http.MethodGet {
		build, err := s.bot.riftBuild(r.Context(), uid)
		if err != nil {
			riftFailure(w, r, err)
			return
		}
		run, err := loadRift(r.Context(), s.bot.DB, uid)
		if err != nil {
			riftFailure(w, r, err)
			return
		}
		writeJSON(w, map[string]any{"ok": true, "run": run, "build": build, "rooms": rift.Rooms, "levels": rift.Campaign(), "bestiary": riftBestiary(time.Now()), "rarities": riftRarities()})
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "GET or POST only", http.StatusMethodNotAllowed)
		return
	}
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		http.Error(w, "JSON required", http.StatusUnsupportedMediaType)
		return
	}
	if origin := r.Header.Get("Origin"); origin != "" {
		parsed, parseErr := url.Parse(origin)
		if parseErr != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") || !strings.EqualFold(parsed.Host, r.Host) {
			http.Error(w, "same origin required", http.StatusForbidden)
			return
		}
	}
	if r.Header.Get("Sec-Fetch-Site") == "cross-site" {
		http.Error(w, "same origin required", http.StatusForbidden)
		return
	}
	var req riftRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil || decoder.Decode(&struct{}{}) != io.EOF || !validRiftRequest(req) {
		http.Error(w, "invalid expedition controls", http.StatusBadRequest)
		return
	}
	var build rift.Build
	if req.Kind == "start" {
		build, err = s.bot.riftBuild(r.Context(), uid)
		if err != nil {
			riftFailure(w, r, err)
			return
		}
		if req.Skills != nil {
			selected := []rift.Skill{}
			for _, id := range req.Skills {
				for _, sk := range build.Skills {
					if sk.ID == id {
						selected = append(selected, sk)
						break
					}
				}
			}
			if len(selected) != len(req.Skills) {
				http.Error(w, "choose owned skills", http.StatusBadRequest)
				return
			}
			build.Skills = selected
		} else if len(build.Skills) > 3 {
			build.Skills = build.Skills[:3]
		}
	}
	run, err := s.bot.updateRift(r.Context(), uid, req, build, time.Now())
	if errors.Is(err, errRiftConflict) {
		writeJSONStatus(w, http.StatusConflict, map[string]any{"ok": false, "error": errRiftConflict.Error()})
		return
	}
	if err != nil {
		riftFailure(w, r, err)
		return
	}
	writeJSON(w, map[string]any{"ok": true, "run": run})
}

func validRiftRequest(r riftRequest) bool {
	if r.LevelID < 0 || r.LevelID > rift.LevelCount {
		return false
	}
	if len(r.RunID) > 80 || len(r.RequestID) < 16 || len(r.RequestID) > 80 || r.Revision < 0 || len(r.Input.Skill) > 100 || !r.Input.ValidMovement() || len(r.Skills) > 3 {
		return false
	}
	seen := map[string]bool{}
	for _, id := range r.Skills {
		if len(id) > 100 || seen[id] {
			return false
		}
		seen[id] = true
	}
	switch r.Kind {
	case "start", "step", "pause", "resume", "bank", "next", "advance", "exit":
		return true
	}
	return false
}

func riftFailure(w http.ResponseWriter, r *http.Request, err error) {
	slog.ErrorContext(r.Context(), "rift expedition request failed", "error", err)
	writeJSONStatus(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "The expedition could not be confirmed. Reload to recover its saved state."})
}

func decodeRift(saved string) (*rift.Run, error) {
	var run rift.Run
	if len(saved) > 256_000 {
		return nil, errors.New("rift snapshot exceeds limit")
	}
	if err := json.Unmarshal([]byte(saved), &run); err != nil {
		return nil, fmt.Errorf("decode rift: %w", err)
	}
	if run.Schema != 1 || run.ID == "" || run.Room < 0 || run.Room >= len(rift.Rooms) || run.SkillTimers == nil {
		return nil, errors.New("unsupported rift snapshot")
	}
	return &run, nil
}

func loadRift(ctx context.Context, database *sql.DB, uid string) (*rift.Run, error) {
	var saved string
	err := database.QueryRowContext(ctx, "SELECT value FROM app_meta WHERE key=$1", "rift_brawl:"+uid).Scan(&saved)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	run, err := decodeRift(saved)
	if err != nil {
		return nil, err
	}
	var epoch string
	if err := database.QueryRowContext(ctx, "SELECT COALESCE((SELECT value FROM app_meta WHERE key='gold_economy_version'),'0')").Scan(&epoch); err != nil {
		return nil, err
	}
	if run.Epoch != epoch {
		run.Status = "expired"
		run.Gold = 0
		run.Drops = []rift.Drop{}
		run.BankedGold = 0
		run.BankedItems = []string{}
	}
	return run, nil
}

func (b *Bot) updateRift(ctx context.Context, uid string, req riftRequest, build rift.Build, now time.Time) (*rift.Run, error) {
	tx, err := b.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	var owner string
	if err := tx.QueryRowContext(ctx, "SELECT client_uid FROM users WHERE client_uid=$1 FOR UPDATE", uid).Scan(&owner); err != nil {
		return nil, err
	}
	var epoch string
	if err := tx.QueryRowContext(ctx, "SELECT COALESCE((SELECT value FROM app_meta WHERE key='gold_economy_version'),'0')").Scan(&epoch); err != nil {
		return nil, err
	}
	key := "rift_brawl:" + uid
	var saved string
	var run *rift.Run
	err = tx.QueryRowContext(ctx, "SELECT value FROM app_meta WHERE key=$1", key).Scan(&saved)
	if err == nil {
		run, err = decodeRift(saved)
		if err != nil {
			return nil, err
		}
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	if req.Kind == "start" {
		if run != nil && run.StartKey == req.RequestID && run.Epoch == epoch {
			return run, nil
		}
		if run != nil && run.Epoch == epoch && (run.Status == "fighting" || run.Status == "cleared") {
			return nil, errRiftConflict
		}
		if run != nil && req.RunID != run.ID {
			return nil, errRiftConflict
		}
		id, err := newAccountToken()
		if err != nil {
			return nil, err
		}
		var completed []int
		if run != nil {
			completed = run.CompletedLevels
		}
		run = rift.NewRunAtLevel(id, build, now, riftMobCatalog(now), req.LevelID)
		run.CompletedLevels = completed
		run.StartKey = req.RequestID
		run.Epoch = epoch
	} else {
		if run == nil || req.RunID != run.ID || epoch != run.Epoch {
			return nil, errRiftConflict
		}
		if req.Revision <= run.Revision {
			return run, nil
		}
		if req.Revision != run.Revision+1 {
			return nil, errRiftConflict
		}
		switch req.Kind {
		case "step":
			run.Step(req.Input, now)
		case "pause":
			run.Paused = true
			run.LastMS = max(run.LastMS, now.UnixMilli())
		case "resume":
			run.Paused = false
			run.LastMS = max(run.LastMS, now.UnixMilli())
		case "bank", "exit", "next", "advance":
			if run.Status != "cleared" {
				return nil, errRiftConflict
			}
			if err := bankRift(ctx, tx, uid, req.RequestID, run); err != nil {
				return nil, err
			}
			run.FinishCheckpoint(req.Kind, riftMobCatalog(now))
			run.LastMS = max(run.LastMS, now.UnixMilli())
		}
		run.Revision = req.Revision
	}
	for i := range run.Drops {
		drop := &run.Drops[i]
		if drop.NeedsGear && drop.Gear == nil {
			gear, err := rollRiftGear(run.Room, now)
			if err != nil {
				return nil, err
			}
			drop.Gear = &gear
			drop.Gear.FoundBoss = riftGearOrigin(run)
		}
	}
	data, err := json.Marshal(run)
	if err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, "INSERT INTO app_meta (key,value) VALUES ($1,$2) ON CONFLICT (key) DO UPDATE SET value=EXCLUDED.value", key, string(data)); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return run, nil
}

func riftGearOrigin(run *rift.Run) string {
	name := rift.Rooms[run.Room]
	if run.Level != nil {
		name = run.Level.Name
	}
	return fmt.Sprintf("Rift Brawl: %s · Tier %d", name, run.Room+1)
}

func rollRiftGear(room int, now time.Time) (content.Gear, error) {
	catalog := content.AbyssGearCatalog()
	pool := []content.Gear{}
	capRarity := rift.LootRarityCap(room)
	for _, gear := range catalog {
		if gear.Rarity <= capRarity {
			pool = append(pool, gear)
		}
	}
	if len(pool) == 0 {
		return content.Gear{}, errors.New("rift loot catalog is unavailable")
	}
	// #nosec G404 -- randomized server-owned game loot, not a security token.
	gear := pool[rand.IntN(len(pool))]
	gear.FoundAt = now.UTC().Format(time.RFC3339)
	gear.FoundBoss = "Rift Brawl: " + rift.Rooms[room]
	return gear, nil
}

func bankRift(ctx context.Context, tx *sql.Tx, uid, requestID string, run *rift.Run) error {
	if err := db.SetEconomyContext(ctx, tx, "rift_brawl", requestID, run.ID, ""); err != nil {
		return err
	}
	var gold int64
	for i := range run.Drops {
		drop := &run.Drops[i]
		if drop.Banked || !drop.Collected {
			continue
		}
		gold += drop.Gold
		if drop.Gear != nil {
			gear := drop.Gear
			data, err := json.Marshal(gear)
			if err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, "INSERT INTO user_inventory (client_uid,gear_id,durability,item_data) VALUES ($1,$2,$3,$4)", uid, gear.ID, gear.MaxDurability, string(data)); err != nil {
				return err
			}
			run.BankedItems = append(run.BankedItems, gear.Name)
		}
		drop.Banked = true
		drop.Collected = true
	}
	if gold > 0 {
		if _, err := tx.ExecContext(ctx, "/* economy:bot.bankRift */ UPDATE users SET gold=gold+$1 WHERE client_uid=$2", gold, uid); err != nil {
			return err
		}
	}
	run.BankedGold += gold
	run.Gold = 0
	return nil
}
