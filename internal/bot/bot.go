// Package bot implements the headless TS3 free-game bot and its RPG
// progression system: XP/combat, gear/loot, the web portal, and the Abyss
// dungeon, all backed by PostgreSQL.
package bot

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math/rand/v2"
	"os"
	"strings"
	"time"
	"unicode"

	"ts3news/internal/clientquery"
	"ts3news/internal/config"
	"ts3news/internal/content"
	"ts3news/internal/db"
	"ts3news/internal/games"
	"ts3news/internal/i18n"
	"ts3news/internal/leveling"

	_ "github.com/lib/pq" // registers the "postgres" database/sql driver
)

// Bot is the top-level RPG bot instance: its config and database handle, plus
// caches of TS3 server-group IDs already created for level/XP progression.
type Bot struct {
	Cfg         *config.Config
	DB          *sql.DB
	levelGroups map[int]int
	xpGroups    map[int]int
}

type levelResult struct {
	OldLevel int
	NewLevel int
	TotalXP  int
	Awarded  int
}

// NewBot opens the database, runs pending migrations, and returns a ready-to-use Bot.
func NewBot(cfg *config.Config) *Bot {
	database, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	if err := database.Ping(); err != nil {
		log.Printf("Warning: Database ping failed: %v", err)
	}

	// Schema is managed by versioned, embedded migrations (golang-migrate).
	if err := db.Migrate(database); err != nil {
		log.Fatalf("Failed to run database migrations: %v", err)
	}

	b := &Bot{
		Cfg:         cfg,
		DB:          database,
		levelGroups: leveling.ParseLevelGroups(cfg.LevelGroups),
		xpGroups:    map[int]int{},
	}
	if cfg.XPServerGroups {
		b.loadLevelGroups()
	}
	return b
}

// Close releases the bot's database connection.
func (b *Bot) Close() {
	if b.DB != nil {
		_ = b.DB.Close()
	}
}

// fetchOptions builds the games fetch options from config.
func (b *Bot) fetchOptions() games.Options {
	return games.Options{
		DRMFilter:        b.Cfg.DRMFilter,
		EnableGamerPower: b.Cfg.EnableGamerPower,
		EnableEpic:       b.Cfg.EnableEpic,
		EnableReddit:     b.Cfg.EnableReddit,
		EnableITAD:       b.Cfg.ITADKey != "",
		ITADKey:          b.Cfg.ITADKey,
	}
}

// RunCycle now resolves group combat by channel
func (b *Bot) RunCycle(c *clientquery.Client) error {
	freeGames, err := games.FetchFreeGames(b.fetchOptions())
	if err != nil {
		return fmt.Errorf("failed to fetch games: %w", err)
	}

	clients, err := c.ClientList()
	if err != nil {
		return fmt.Errorf("failed to list clients: %w", err)
	}

	targetNick := strings.TrimSpace(b.Cfg.TargetNick)
	ctx := b.buildCycleContext(clients)
	b.slothDecay(c, ctx.today)
	b.CleanupAuctionHouse()

	if b.Cfg.XPServerGroups {
		b.cleanupEmptyLevelGroups(c)
	}
	b.cleanupUnusedIcons(c)

	// Group normal users by channel
	chanUsers := map[int][]UserInCombat{}
	for _, cl := range clients {
		if cl.Type != 0 || (targetNick != "" && !strings.EqualFold(cl.Nickname, targetNick)) || cl.UID == "" {
			continue
		}
		stats, _, _, _ := b.calculateTotalStats(cl.UID, ctx.today)
		skills := b.getSkills(cl.UID)
		ultimate := b.getUltimateSkill(cl.UID)

		var lvl, prestige, curHP, regen, curseFights, bestDepth int
		var gold int64
		err := b.DB.QueryRow("SELECT level, prestige, current_hp, regen_stacks, gold, abyss_curse_fights, abyss_best_depth FROM users WHERE client_uid=$1", cl.UID).Scan(&lvl, &prestige, &curHP, &regen, &gold, &curseFights, &bestDepth)
		if err != nil && err != sql.ErrNoRows {
			log.Printf("Failed to scan user combat state for %s: %v", cl.UID, err)
		}
		if b.Cfg.EnableAbyss {
			b.applyAbyssMilestones(c, cl.CLID, cl.UID, cl.Nickname, bestDepth)
		}
		// Abyss "cursed bank": the player took a +20% payout in exchange for a hex
		// on their next few cycle fights. Sap their combat stats and tick it down.
		if b.Cfg.EnableAbyss && curseFights > 0 {
			// Guard against underflow and concurrent changes: only scale stats if the
			// decrement actually consumed a curse fight (row still had > 0).
			res, err := b.DB.Exec("UPDATE users SET abyss_curse_fights = abyss_curse_fights - 1 WHERE client_uid=$1 AND abyss_curse_fights > 0", cl.UID)
			if err != nil {
				log.Printf("Failed to decrement abyss curse for %s: %v", cl.UID, err)
			} else if n, _ := res.RowsAffected(); n > 0 {
				stats = stats.Scaled(0.85)
			}
		}
		if curHP <= 0 {
			curHP = stats.HP // Auto-fill if new/dead
		} else if curHP > stats.HP {
			curHP = stats.HP // Clamp to post-curse max
		}
		pets := b.getPets(cl.UID)
		equipped := b.getEquippedItems(cl.UID)

		chanUsers[cl.CID] = append(chanUsers[cl.CID], UserInCombat{
			UID: cl.UID, Nickname: cl.Nickname, CLID: cl.CLID, Stats: stats, Level: lvl, Skills: skills,
			UltimateSkill: ultimate, CurrentHP: curHP, RegenStacks: regen, Gold: gold, Pets: pets,
			Equipped: equipped,
		})
	}

	theme := b.activeTheme()
	pokedCount := 0

	for cid, users := range chanUsers {
		if len(users) == 0 {
			continue
		}

		// 1. Party Stats & Difficulty
		totalLvl := 0
		totalStatScore := 0
		for _, u := range users {
			totalLvl += u.Level
			totalStatScore += u.Stats.Score()
		}
		avgLvl := totalLvl / len(users)
		if avgLvl < 1 {
			avgLvl = 1
		}

		expectedScore := 45 + (avgLvl / 5)
		diffFactor := float64(totalStatScore) / float64(len(users)) / float64(expectedScore)
		if diffFactor < 0.5 {
			diffFactor = 0.5
		}
		if diffFactor > 1.5 {
			diffFactor = 1.5
		}

		// 2. Select Zone
		zone := content.GetRandomZone(avgLvl, float64(totalStatScore)/float64(len(users)))
		battleLogs := []string{zone.Display()}

		mobs := content.SpawnMobGroup(avgLvl, zone, diffFactor*zone.Difficulty, len(users), false)
		var mobPtrs []*content.Mob
		for i := range mobs {
			mobPtrs = append(mobPtrs, &mobs[i])
		}

		// 3. Resolve Group Combat
		resLogs, rewardXP, victory, combatLoots := b.resolveChannelCombat(users, mobPtrs, avgLvl, diffFactor, zone)
		battleLogs = append(battleLogs, resLogs...)

		// 4. Post-battle processing for each user
		for _, user := range users {
			_ = b.touchUser(user.UID, user.Nickname, 0)

			alreadySent, _ := b.getSentGames(user.UID)
			candidates := filterNewGames(freeGames, alreadySent)
			// Prioritize GamerPower
			var gp []games.Game
			for _, g := range candidates {
				if strings.EqualFold(g.Source, "GamerPower") {
					gp = append(gp, g)
				}
			}
			if len(gp) > 0 {
				candidates = gp
			}

			hasGame := len(candidates) > 0
			var game games.Game
			var shortURL string
			if hasGame {
				// #nosec G404
				game = candidates[rand.IntN(len(candidates))] // #nosec G404
				shortURL, _ = games.ShortenURL(game.URL)
			}

			// XP, Leveling, Loot
			baseXP := b.xpForGame(game)
			lr, notes, artifactPoke := b.processUserXP(user.UID, user.Nickname, cid, baseXP+rewardXP, hasGame, ctx)

			// Auction House auto-purchase
			if ahNote := b.autoPurchaseUpgrades(user.UID, user.Gold); ahNote != "" {
				notes = append(notes, ahNote)
			}

			extraPoke := artifactPoke

			// Auto-prestige at the level cap: reset to level 1, +1 prestige (with a
			// permanent stat bonus) and grant the prestige rank group. Future leveling
			// then resumes from level 1 at the new prestige.
			if lr != nil && lr.NewLevel >= PrestigeThreshold {
					newP := b.doPrestige(user.UID)
					notes = append(notes, i18n.T("bot.prestige.announce", newP, int(prestigeStatBonus*100)))
					if extraPoke != "" {
							extraPoke += " "
					}
					extraPoke += i18n.T("bot.prestige.poke", newP)
					lr.OldLevel, lr.NewLevel, lr.TotalXP = 1, 1, 0
					if b.Cfg.XPServerGroups {
							b.applyPrestigeGroup(c, user.CLID, user.UID, user.Nickname, newP)
					}
			}

			// Durability & Loot Drops
			duraNotes := b.applyDurabilityLoss(user.UID, !victory)
			notes = append(notes, duraNotes...)

			userLootFound := false
			for _, cl := range combatLoots {
				if cl.UID == user.UID {
					notes = append(notes, cl.Note)
					if cl.Poke != "" {
						if extraPoke != "" {
							extraPoke += " "
						}
						extraPoke += cl.Poke
					}
					userLootFound = true
				}
			}

			// Apply Groups & Titles
			if b.Cfg.EnableLeveling {
				b.applyMilestones(c, user.CLID, user.Nickname, lr)
				if b.Cfg.XPServerGroups {
					b.applyLevelGroup(c, user.CLID, user.UID, user.Nickname, lr.NewLevel)
				}
				b.applyTitleGroup(c, user.CLID, user.UID, user.Nickname)
				b.syncLootGroups(c, user.CLID, user.UID)
			}

			// Messaging
			notes = append(notes, battleLogs...)
			if lr != nil {
					outcome := i18n.T("xp.battle")
					if lr.Awarded < 0 {
							outcome = i18n.T("xp.lost")
					}
					notes = append(notes, i18n.T("xp.outcome", outcome, lr.Awarded, leveling.LevelName(lr.NewLevel), lr.NewLevel))
			}
			pokeMsg := composePoke(game, shortURL, theme, lr)
			pmMsg := b.composePM(game, shortURL, theme, lr, notes, user.Stats.Score())

			// Persona check
			botNick := b.Cfg.TS3Nickname
			if userLootFound || extraPoke != "" {
				botNick = "godsfinger"
			}
			_ = c.SetNickname(botNick)

			// Send Pokes
			if extraPoke != "" {
				_ = c.Poke(user.CLID, strings.TrimSpace(extraPoke))
			}

			if hasGame && shortURL != "" {
				_ = c.Poke(user.CLID, pokeMsg)
			}

			for _, chunk := range splitMessage(pmMsg, 1000) {
				_ = c.SendPrivateMessage(user.CLID, chunk)
			}

			// Personal web-portal login link (token-based, shortened).
			if b.Cfg.WebEnable {
				if msg := b.composeLoginPM(user.UID); msg != "" {
					_ = c.SendPrivateMessage(user.CLID, msg)
				}
			}

			_ = c.SetNickname(b.Cfg.TS3Nickname)
			if hasGame {
				_ = b.markAsSent(user.UID, user.Nickname, game.Key(), game.DisplayTitle())
			}
			pokedCount++
			time.Sleep(time.Duration(b.Cfg.PokeDelayMS) * time.Millisecond)
		}
	}

	// Update channel descriptions with all players' stats
	if err := b.UpdateChannelDescriptions(c); err != nil {
		log.Printf("Warning: Failed to update channel descriptions: %v", err)
	} else {
		log.Printf("Updated channel descriptions")
	}

	log.Printf("Cycle finished. Poked %d users.", pokedCount)
	return nil
}

func (b *Bot) activeTheme() *content.Theme {
	if !b.Cfg.EnableHolidayThemes {
		return nil
	}
	return content.CurrentTheme(time.Now())
}

func splitMessage(msg string, limit int) []string {
	if len(msg) <= limit {
		return []string{msg}
	}
	var chunks []string
	for len(msg) > 0 {
		if len(msg) <= limit {
			chunks = append(chunks, msg)
			break
		}
		idx := strings.LastIndex(msg[:limit], "\n")
		if idx == -1 {
			idx = limit
		}
		chunks = append(chunks, msg[:idx])
		msg = strings.TrimPrefix(msg[idx:], "\n")
	}
	return chunks
}

func composePoke(g games.Game, shortURL string, theme *content.Theme, lvl *levelResult) string {
	// Poke is just the clean game name + link (no XP/level — those go in the PM).
	_ = lvl
	prefix := "Free: "
	if theme != nil && theme.Emoji != "" {
		prefix = theme.Emoji + " Free: "
	}
	title := g.DisplayTitle()
	avail := 100 - len(prefix) - 1 - len(shortURL)
	if avail > 4 && len(title) > avail {
		title = title[:avail-3] + "..."
	}
	return fmt.Sprintf("%s%s %s", prefix, title, shortURL)
}

func (b *Bot) composePM(g games.Game, shortURL string, theme *content.Theme, lvl *levelResult, notes []string, totalGS int) string {
	var sb strings.Builder
	if theme != nil {
		sb.WriteString(theme.Emoji + " [b]" + theme.Banner + "[/b]")
	} else if b.Cfg.EnableGreetings {
		sb.WriteString(i18n.T("bot.pm.greeting", content.RandomGreeting()))
	} else {
		sb.WriteString(i18n.T("bot.poke.daily_game"))
	}
	sb.WriteString("\n")

	name := g.DisplayTitle()
	if name != "" {
		sb.WriteString(i18n.T("bot.pm.game_line", name) + "\n")
		if g.WorthShown() {
			sb.WriteString(i18n.T("bot.pm.worth_line", g.Worth) + "\n")
		}
	} else {
		sb.WriteString(i18n.T("bot.pm.no_game") + "\n")
	}

	if lvl != nil {
		sb.WriteString("\n" + i18n.T("bot.stats.header", leveling.LevelName(lvl.NewLevel), 0, lvl.NewLevel, float64(totalGS), 0.0) + "\n")
		sb.WriteString(i18n.T("xp.outcome", i18n.T("xp.battle"), lvl.Awarded, leveling.LevelName(lvl.NewLevel), lvl.NewLevel) + "\n")
		if lvl.NewLevel > lvl.OldLevel {
			sb.WriteString(i18n.T("bot.stats.level_up", leveling.LevelName(lvl.NewLevel)) + "\n")
		}
	}

	// Categorize notes
	var combatNotes []string
	var rewardNotes []string
	var equipNotes []string
	var miscNotes []string

	for _, note := range notes {
		note = strings.TrimSpace(note)
		if note == "" {
			continue
		}
		switch {
		case strings.Contains(note, "📍") || strings.Contains(note, "⚔️") || strings.Contains(note, "WAVE") ||
			strings.Contains(note, "☠️") || strings.Contains(note, "🏁") || strings.Contains(note, "💥") ||
			strings.Contains(note, "📊") || strings.Contains(note, "AMBUSH") || strings.Contains(note, "slain") ||
			strings.Contains(note, "cast") || strings.Contains(note, "used") || strings.Contains(note, "defeated") ||
			(strings.Contains(note, "✨") && strings.Contains(note, ":")): // Skill/Pet logs
			combatNotes = append(combatNotes, note)
		case strings.Contains(note, "🎁") || strings.Contains(note, "💰") || strings.Contains(note, "🌟") ||
			strings.Contains(note, "listed on AH") || strings.Contains(note, "Learned") || strings.Contains(note, "Equipped") ||
			strings.Contains(note, "XP") || strings.Contains(note, "🏆"):
			rewardNotes = append(rewardNotes, note)
		case strings.Contains(note, "dur)"):
			// Durability lines, e.g. "Novice Neck (47 dur)" — the bulk of the gear
			// loadout. (The old check matched "dura", which never appears, so these
			// fell through to misc and printed one slot per line.)
			equipNotes = append(equipNotes, note)
		default:
			miscNotes = append(miscNotes, note)
		}
	}

	// Bullets are packed several-per-line to use the available width and keep the
	// PM short. Misc keeps one item per line because it holds full-width elements
	// (the [hr] divider and the Party/Enemy health bars) that must not share a line.
	if len(miscNotes) > 0 {
		sb.WriteString("\n" + i18n.T("bot.section.bonuses") + "\n")
		for _, n := range miscNotes {
			sb.WriteString(" • " + n + "\n")
		}
	}

	if len(combatNotes) > 0 {
		sb.WriteString("\n" + i18n.T("bot.section.combat") + "\n")
		writeCombatLog(&sb, combatNotes, pmLineWidth)
	}

	if len(rewardNotes) > 0 {
		sb.WriteString("\n" + i18n.T("bot.section.loot") + "\n")
		writePackedBullets(&sb, rewardNotes, pmLineWidth)
	}

	if len(equipNotes) > 0 {
		sb.WriteString("\n" + i18n.T("bot.section.equipment") + "\n")
		writePackedBullets(&sb, equipNotes, pmLineWidth)
	}

	// Add game claim and YouTube trailer at the end for better readability
	if shortURL != "" {
		sb.WriteString("\n")
		sb.WriteString(i18n.T("bot.pm.claim_line", shortURL) + "\n")
		if b.Cfg.EnableYouTubeTrailer {
			sb.WriteString(i18n.T("bot.pm.trailer_line", games.TrailerSearchURL(name)) + "\n")
		}
	}

	if theme != nil && theme.Signoff != "" {
		sb.WriteString("\n" + theme.Signoff)
	}
	return strings.TrimRight(sb.String(), "\n")
}

// pmLineWidth is the soft target width (in bytes) for packed PM bullet lines.
// Wider than a typical chat line so we fit several items per line, but short
// enough to avoid an unreadable wall of text.
const pmLineWidth = 300

// writePackedBullets writes items onto as few " • "-prefixed lines as possible,
// separating items with " | " and wrapping once a line would exceed maxLen. A
// single item longer than maxLen still occupies its own line.
func writePackedBullets(sb *strings.Builder, items []string, maxLen int) {
	var line string
	for _, it := range items {
		seg := it
		if line != "" {
			seg = " | " + it
		}
		if line != "" && len(line)+len(seg) > maxLen {
			sb.WriteString(" • " + line + "\n")
			line = it
		} else {
			line += seg
		}
	}
	if line != "" {
		sb.WriteString(" • " + line + "\n")
	}
}

// writeCombatLog renders the combat notes in order, packing consecutive short
// blow-by-blow lines (kills, casts, ambush) together while keeping structural
// lines — the zone/wave headers and the battle summary — on their own line so
// the narrative stays readable.
func writeCombatLog(sb *strings.Builder, notes []string, maxLen int) {
	var run []string
	flush := func() {
		if len(run) > 0 {
			writePackedBullets(sb, run, maxLen)
			run = run[:0]
		}
	}
	for _, n := range notes {
		if isStandaloneCombatLine(n) {
			flush()
			sb.WriteString(" • " + n + "\n")
		} else {
			run = append(run, n)
		}
	}
	flush()
}

// isStandaloneCombatLine reports whether a combat note should occupy its own
// line: zone/wave headers and the BBCode-formatted battle summary, which read
// poorly when packed next to short kill lines.
func isStandaloneCombatLine(n string) bool {
	return strings.Contains(n, "WAVE") || strings.Contains(n, "📍") ||
		strings.Contains(n, "📊") || strings.Contains(n, "🏁") ||
		strings.Contains(n, "[center]") || strings.Contains(n, "[size") ||
		strings.Contains(n, "[hr]")
}

func filterNewGames(all []games.Game, alreadySent []string) []games.Game {
	sent := make(map[string]bool, len(alreadySent))
	for _, k := range alreadySent {
		sent[k] = true
	}
	var candidates []games.Game
	for _, g := range all {
		if !sent[g.Key()] {
			candidates = append(candidates, g)
		}
	}
	return candidates
}

func (b *Bot) getSentGames(uid string) ([]string, error) {
	var rows *sql.Rows
	var err error
	if b.Cfg.ResendAfterDays > 0 {
		rows, err = b.DB.Query("SELECT game_key FROM sent_notifications WHERE client_uid = $1 AND sent_at > NOW() - ($2 * INTERVAL '1 day')", uid, b.Cfg.ResendAfterDays)
	} else {
		rows, err = b.DB.Query("SELECT game_key FROM sent_notifications WHERE client_uid = $1", uid)
	}
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var keys []string
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err == nil {
			keys = append(keys, k)
		}
	}
	return keys, rows.Err()
}

func (b *Bot) markAsSent(uid, nickname, gameKey, gameTitle string) error {
	_, err := b.DB.Exec(`INSERT INTO sent_notifications (client_uid, game_key, game_title, client_nickname, sent_at) VALUES ($1, $2, $3, $4, NOW()) ON CONFLICT (client_uid, game_key) DO UPDATE SET sent_at = NOW(), client_nickname = $4, game_title = $3`, uid, gameKey, gameTitle, nickname)
	return err
}

func (b *Bot) touchUser(uid, nickname string, sessionMS int64) error {
	var lastMS int64
	err := b.DB.QueryRow("SELECT last_session_connected_ms FROM users WHERE client_uid = $1", uid).Scan(&lastMS)
	deltaSec := int64(0)
	if err == nil {
		if sessionMS > lastMS {
			deltaSec = (sessionMS - lastMS) / 1000
		} else {
			deltaSec = sessionMS / 1000
		}
	} else {
		deltaSec = sessionMS / 1000
	}
	_, err = b.DB.Exec(`INSERT INTO users (client_uid, nickname, last_seen, total_connection_seconds, last_session_connected_ms) VALUES ($1, $2, NOW(), $3, $4) ON CONFLICT (client_uid) DO UPDATE SET last_seen = NOW(), nickname = $2, total_connection_seconds = users.total_connection_seconds + $3, last_session_connected_ms = $4`, uid, nickname, deltaSec, sessionMS)
	return err
}

func (b *Bot) xpForGame(g games.Game) int {
	if p, ok := g.PriceEUR(); ok {
		return leveling.XPForPrice(p, b.Cfg.CheaperMoreXP)
	}
	return leveling.XPPerPoke()
}

func (b *Bot) getAPIKey() string {
	if b.Cfg.APIKey != "" {
		return b.Cfg.APIKey
	}
	f, err := os.Open(b.Cfg.ClientQueryINI)
	if err != nil {
		return ""
	}
	defer func() { _ = f.Close() }()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "api_key=") {
			return strings.TrimPrefix(line, "api_key=")
		}
	}
	return ""
}

// FormatGold formats a gold amount with a k/M/B suffix and TS3 BBCode styling.
func FormatGold(v int64) string {
	f := float64(v)
	switch {
	case v >= 1_000_000_000:
		return fmt.Sprintf("[b]%.1f[/b][color=#9e9e9e]B[/color]", f/1_000_000_000.0)
	case v >= 1_000_000:
		return fmt.Sprintf("[b]%.1f[/b][color=#9e9e9e]M[/color]", f/1_000_000.0)
	case v >= 1_000:
		return fmt.Sprintf("[b]%.1f[/b][color=#9e9e9e]k[/color]", f/1_000.0)
	default:
		return fmt.Sprintf("[b]%d[/b][color=#9e9e9e]g[/color]", v)
	}
}

// FormatGoldPlain is FormatGold without TS3 BBCode, for HTML/web rendering where
// the markup would otherwise be shown literally.
func FormatGoldPlain(v int64) string {
	f := float64(v)
	switch {
	case v >= 1_000_000_000:
		return fmt.Sprintf("%.1fB", f/1_000_000_000.0)
	case v >= 1_000_000:
		return fmt.Sprintf("%.1fM", f/1_000_000.0)
	case v >= 1_000:
		return fmt.Sprintf("%.1fk", f/1_000.0)
	default:
		return fmt.Sprintf("%dg", v)
	}
}

func (b *Bot) makeGear(gearID string, itemData sql.NullString) (content.Gear, bool) {
	if itemData.Valid && itemData.String != "" && itemData.String != "{}" {
		// Start from the catalog entry so any field the persisted JSON omits (null or
		// partial payloads) keeps its catalog default; the unmarshal then overlays only
		// the fields actually stored on the item.
		g, ok := content.GetGearByID(gearID)
		if !ok {
			g = content.Gear{}
		}
		if err := json.Unmarshal([]byte(itemData.String), &g); err == nil {
			return g, true
		}
	}
	return content.GetGearByID(gearID)
}

func (b *Bot) getEquippedItems(uid string) map[content.GearSlot]content.Gear {
	out := make(map[content.GearSlot]content.Gear)
	rows, err := b.DB.Query("SELECT slot, gear_id, item_data FROM user_gear WHERE client_uid = $1", uid)
	if err != nil {
		return out
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var slot string
		var id string
		var itemData sql.NullString
		if err := rows.Scan(&slot, &id, &itemData); err == nil {
			if gear, ok := b.makeGear(id, itemData); ok {
				out[content.GearSlot(slot)] = gear
			}
		}
	}
	return out
}

// CleanupDeadUsers deletes accounts inactive longer than Cfg.DeadUserDays and
// purges their notification history. The returned count is the number of deleted
// user rows only; the separate notification-history purge is not counted in it.
func (b *Bot) CleanupDeadUsers() (int, error) {
	if b.Cfg.DeadUserDays <= 0 {
		return 0, nil
	}
	_, err := b.DB.Exec(
		`DELETE FROM sent_notifications WHERE client_uid IN (
			SELECT client_uid FROM users WHERE last_seen < NOW() - ($1 * INTERVAL '1 day'))`,
		b.Cfg.DeadUserDays)
	if err != nil {
		return 0, err
	}
	res, err := b.DB.Exec(
		"DELETE FROM users WHERE last_seen < NOW() - ($1 * INTERVAL '1 day')",
		b.Cfg.DeadUserDays)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

// UpdateChannelDescriptions updates all channel descriptions with the stats of players in each channel
func (b *Bot) UpdateChannelDescriptions(c *clientquery.Client) error {
	log.Printf("Starting UpdateChannelDescriptions...")

	clients, err := c.ClientList()
	if err != nil {
		return fmt.Errorf("failed to list clients: %w", err)
	}
	log.Printf("Found %d clients", len(clients))

	// Group clients by channel
	chanUsers := make(map[int][]struct {
		UID  string
		Nick string
	})
	for _, cl := range clients {
		if cl.Type != 0 || cl.UID == "" {
			continue
		}
		chanUsers[cl.CID] = append(chanUsers[cl.CID], struct {
			UID  string
			Nick string
		}{UID: cl.UID, Nick: cl.Nickname})
	}
	log.Printf("Found %d channels with players", len(chanUsers))

	// Fetch channel metadata once: the default ("home") channel is excluded from
	// renaming so the server's landing channel keeps its name. We also seed a
	// "taken" set with every current channel name so randomly picked names stay
	// server-wide unique (TeamSpeak rejects a duplicate with id=771).
	defaultCID := -1
	channelName := make(map[int]string)
	taken := make(map[string]bool)
	if chans, derr := c.ChannelList(); derr != nil {
		log.Printf("Channel rename: could not list channels (will not exclude default or guarantee unique names): %v", derr)
	} else {
		for _, ch := range chans {
			channelName[ch.CID] = ch.Name
			taken[ch.Name] = true
			if ch.IsDefault {
				defaultCID = ch.CID
			}
		}
		log.Printf("Channel rename: default (home) channel is %d; it will be excluded", defaultCID)
	}

	// Update each channel's description
	for cid, users := range chanUsers {
		if len(users) == 0 {
			continue
		}

		// The default ("home") channel is left completely untouched — neither its
		// description nor its name is changed.
		if cid == defaultCID {
			log.Printf("Channel update: skipping default/home channel %d (no description or name changes)", cid)
			continue
		}

		log.Printf("Updating channel %d with %d users", cid, len(users))

		var sb strings.Builder
		sb.WriteString(i18n.T("channel.header", len(users)) + "\n[hr]\n")

		for _, u := range users {
			var level, prestige int
			var gold int64
			var currentHP sql.NullInt64
			err := b.DB.QueryRow("SELECT level, prestige, gold, current_hp FROM users WHERE client_uid=$1", u.UID).Scan(&level, &prestige, &gold, &currentHP)
			if err != nil {
				log.Printf("Failed to get user info for %s: %v", u.UID, err)
				continue
			}

			stats, _, gearScore, _ := b.calculateTotalStats(u.UID, time.Now())

			actualCurrentHP := stats.HP
			if currentHP.Valid {
				actualCurrentHP = int(currentHP.Int64)
			}

			hpColor := "#4caf50" // Green
			if float64(actualCurrentHP) < float64(stats.HP)*0.3 {
				hpColor = "#f44336" // Red
			} else if float64(actualCurrentHP) < float64(stats.HP)*0.6 {
				hpColor = "#ff9800" // Orange
			}

			sb.WriteString(i18n.T("channel.player_line", u.Nick, prestige, level, float64(gearScore), 0.0, hpColor, actualCurrentHP, stats.HP, FormatGold(gold)) + "\n")
			sb.WriteString(i18n.T("channel.stats_line", stats.STR, stats.DEF, stats.SPD, stats.LCK, stats.INT, stats.STA, stats.CRT, stats.DGE) + "\n")
		}

		// Truncate if too long (TeamSpeak channel description limit is ~8000 chars)
		desc := sb.String()
		if len(desc) > 5000 {
			desc = desc[:5000] + "..."
		}

		if err := c.SetChannelDescription(cid, desc); err != nil {
			log.Printf("Failed to set channel %d description: %v", cid, err)
		} else {
			log.Printf("Updated channel %d description", cid)
		}

		// Update channel name from the 1000-name pool.
		if !b.Cfg.EnableChannelRename {
			log.Printf("Channel rename: skipping channel %d (ENABLE_CHANNEL_RENAME is off)", cid)
			continue
		}
		// A channel's own current name shouldn't block its rename; everything else
		// stays reserved so picks remain server-wide unique.
		delete(taken, channelName[cid])
		newName, foreign := pickChannelName(taken)
		if newName == "" {
			log.Printf("Channel rename: skipping channel %d (no client-safe name available for lang %q)", cid, b.Cfg.Lang)
			taken[channelName[cid]] = true
			continue
		}
		origin := "local"
		if foreign {
			origin = "foreign"
		}
		log.Printf("Channel rename: setting channel %d -> %q (%s)", cid, newName, origin)
		if err := c.SetChannelName(cid, newName); err != nil {
			log.Printf("Channel rename: FAILED for channel %d name=%q: %v", cid, newName, err)
			taken[channelName[cid]] = true // keep the old name reserved on failure
		} else {
			log.Printf("Channel rename: OK channel %d is now %q", cid, newName)
			taken[newName] = true
		}
	}

	log.Printf("Completed UpdateChannelDescriptions")
	return nil
}

// foreignChannelNameChance is the probability that a channel rename draws its
// name from another language's pool instead of the active locale's.
const foreignChannelNameChance = 0.08

// pickChannelName returns a random channel name from the active locale's pool,
// occasionally (foreignChannelNameChance) borrowing from a random other
// language — but only names whose characters the TS3 client renders reliably
// (see clientSafeChannelName). It avoids any name already in taken so renames
// stay server-wide unique. The second return value reports whether the chosen
// name came from a foreign pool.
func pickChannelName(taken map[string]bool) (string, bool) {
	// Occasionally try another language first, but fall back to the active locale
	// if that foreign pool has no free client-safe name left.
	// #nosec G404 -- non-cryptographic cosmetic channel-name pick
	if rand.Float64() < foreignChannelNameChance {
		var others []i18n.LocaleID
		for _, id := range i18n.AllLocales {
			if id != i18n.CurrentLocale() {
				others = append(others, id)
			}
		}
		if len(others) > 0 {
			// #nosec G404 -- non-cryptographic cosmetic channel-name pick
			id := others[rand.IntN(len(others))]
			var safe []string
			for _, n := range i18n.PoolForLocale(id, "channel.name") {
				if clientSafeChannelName(n) {
					safe = append(safe, n)
				}
			}
			if n := pickUnused(safe, taken); n != "" {
				return n, true
			}
		}
	}
	// Pool keys are stored without the "pool." prefix (see poolKeyFromYAMLKey).
	return pickUnused(i18n.Pool("channel.name"), taken), false
}

// pickUnused returns a random entry of pool not present in taken, or "" if every
// entry is taken (or the pool is empty). It tries a few random draws before
// falling back to a linear scan.
func pickUnused(pool []string, taken map[string]bool) string {
	if len(pool) == 0 {
		return ""
	}
	for i := 0; i < 8; i++ {
		// #nosec G404 -- non-cryptographic cosmetic channel-name pick
		if n := pool[rand.IntN(len(pool))]; !taken[n] {
			return n
		}
	}
	for _, n := range pool {
		if !taken[n] {
			return n
		}
	}
	return ""
}

// clientSafeChannelName reports whether every rune in s is something the TS3
// client renders reliably: ASCII or Latin-script letters, digits, spaces and a
// few common name punctuation marks. This filters out borrowed names written in
// non-Latin scripts (Cyrillic, CJK, Arabic, …).
func clientSafeChannelName(s string) bool {
	if strings.TrimSpace(s) == "" {
		return false
	}
	for _, r := range s {
		switch {
		case r == ' ' || r == '\'' || r == '-' || r == '.' || r == '&':
		case unicode.IsDigit(r):
		case unicode.IsLetter(r) && (r < 128 || unicode.Is(unicode.Latin, r)):
		default:
			return false
		}
	}
	return true
}

// BroadcastAbyssRecord temporarily renames the TS3 bot to celebrate a new depth record
// and pokes all online normal clients with the news.
func (b *Bot) BroadcastAbyssRecord(nick string, depth int) {
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

	// The record holder's nickname is user-controlled; neutralize BBCode so the
	// broadcast (nickname + poke) can't inject formatting or links. [supervisor.go]
	nick = sanitizeBBCode(nick)

	oldNick := b.Cfg.TS3Nickname
	_ = c.SetNickname(fmt.Sprintf("Abyss Record - %s", nick))

	clients, err := c.ClientList()
	if err == nil {
		msg := fmt.Sprintf("🏆 NEW RECORD! %s has banked a record-breaking run at floor %d in the Abyss!", nick, depth)
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

type dbOrTx interface {
	Exec(query string, args ...any) (sql.Result, error)
	QueryRow(query string, args ...any) *sql.Row
}

// equipGear equips a gear piece, displacing any previously equipped item in that slot to inventory.
func (b *Bot) equipGear(db dbOrTx, uid string, g content.Gear, dur int, itemData any) error {
	// 1. Displace old gear to inventory
	var oldGID string
	var oldDur int
	var oldItemData sql.NullString
	err := db.QueryRow("SELECT gear_id, durability, item_data FROM user_gear WHERE client_uid=$1 AND slot=$2", uid, string(g.Slot)).Scan(&oldGID, &oldDur, &oldItemData)
	if err == nil {
		// Move it to inventory
		_, _ = db.Exec("INSERT INTO user_inventory (client_uid, gear_id, durability, item_data) VALUES ($1, $2, $3, $4)", uid, oldGID, oldDur, oldItemData)
	}

	// 2. Equip the new item directly
	_, err = db.Exec(`INSERT INTO user_gear (client_uid, slot, gear_id, durability, item_data)
	                     VALUES ($1, $2, $3, $4, $5)
	                     ON CONFLICT (client_uid, slot) DO UPDATE SET gear_id = EXCLUDED.gear_id, durability = EXCLUDED.durability, item_data = EXCLUDED.item_data`,
		uid, string(g.Slot), g.ID, dur, itemData)
	return err
}

