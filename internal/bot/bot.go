package bot

import (
	"bufio"
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"

	_ "github.com/lib/pq"
	"ts3news/internal/clientquery"
	"ts3news/internal/config"
	"ts3news/internal/content"
	"ts3news/internal/db"
	"ts3news/internal/games"
	"ts3news/internal/leveling"
)

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

	if b.Cfg.XPServerGroups {
		b.cleanupEmptyLevelGroups(c)
	}

	// Group normal users by channel
	chanUsers := map[int][]UserInCombat{}
	for _, cl := range clients {
		if cl.Type != 0 || (targetNick != "" && !strings.EqualFold(cl.Nickname, targetNick)) || cl.UID == "" {
			continue
		}
		stats, _, _ := b.calculateTotalStats(cl.UID, ctx.today)
		skills := b.getSkills(cl.UID)
		
		var lvl, curHP, regen int
		_ = b.DB.QueryRow("SELECT level, current_hp, regen_stacks FROM users WHERE client_uid=$1", cl.UID).Scan(&lvl, &curHP, &regen)
		if curHP <= 0 { curHP = stats.HP } // Auto-fill if new/dead
		pets := b.getPets(cl.UID)

		chanUsers[cl.CID] = append(chanUsers[cl.CID], UserInCombat{
			UID: cl.UID, Nickname: cl.Nickname, CLID: cl.CLID, Stats: stats, Level: lvl, Skills: skills,
			CurrentHP: curHP, RegenStacks: regen, Pets: pets,
		})
	}

	theme := b.activeTheme()
	pokedCount := 0

	for cid, users := range chanUsers {
		if len(users) == 0 { continue }

		// 1. Party Stats & Difficulty
		totalLvl := 0
		totalStatScore := 0
		for _, u := range users {
			totalLvl += u.Level
			totalStatScore += u.Stats.Score()
		}
		avgLvl := totalLvl / len(users)
		if avgLvl < 1 { avgLvl = 1 }

		expectedScore := 45 + (avgLvl / 5)
		diffFactor := float64(totalStatScore) / float64(len(users)) / float64(expectedScore)
		if diffFactor < 0.5 { diffFactor = 0.5 }
		if diffFactor > 1.5 { diffFactor = 1.5 }

		// 2. Select Zone
		zone := content.GetRandomZone(avgLvl, totalStatScore/len(users))
		battleLogs := []string{zone.Display()}

		mobs := content.SpawnMobGroup(avgLvl, zone, diffFactor*zone.Difficulty)
		var mobPtrs []*content.Mob
		for i := range mobs {
			mobPtrs = append(mobPtrs, &mobs[i])
		}

		// 3. Resolve Group Combat
		resLogs, rewardXP, victory := b.resolveChannelCombat(users, mobPtrs, avgLvl, diffFactor, zone)
		battleLogs = append(battleLogs, resLogs...)

		// 4. Pool Loot for Channel (Shared cross-channel)
		type lootResult struct {
			uid  string
			note string
		}
		var channelLoot []lootResult
		if victory {
			for _, mob := range mobPtrs {
				// Each mob can drop items to ONE random member of the party
				winner := users[rand.Intn(len(users))]
				if note := b.rollLootForUser(winner.UID, *mob, zone.Difficulty); note != "" {
					channelLoot = append(channelLoot, lootResult{uid: winner.UID, note: note})
				}
			}
		}

		// 5. Post-battle processing for each user
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
				game = candidates[rand.Intn(len(candidates))]
				shortURL, _ = games.ShortenURL(game.URL)
			}

			// XP, Leveling, Loot
			baseXP := b.xpForGame(game)
			lr, notes, artifactPoke := b.processUserXP(user.UID, user.Nickname, cid, baseXP+rewardXP, hasGame, ctx)

			// Durability & Loot Drops
			b.applyDurabilityLoss(user.UID, !victory)
			
			userLootFound := false
			for _, cl := range channelLoot {
				if cl.uid == user.UID {
					notes = append(notes, cl.note)
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
			pokeMsg := composePoke(game, shortURL, theme, lr)
			pmMsg := b.composePM(game, shortURL, theme, lr, notes, user.Stats.Score())

			// Persona check
			botNick := b.Cfg.TS3Nickname
			if userLootFound || artifactPoke != "" {
				botNick = "godsfinger"
			}
			_ = c.SetNickname(botNick)

			if artifactPoke != "" {
				_ = c.Poke(user.CLID, artifactPoke)
			}
			_ = c.Poke(user.CLID, pokeMsg)

			for _, chunk := range splitMessage(pmMsg, 1000) {
				_ = c.SendPrivateMessage(user.CLID, chunk)
			}

			_ = c.SetNickname(b.Cfg.TS3Nickname)
			if hasGame {
				_ = b.markAsSent(user.UID, user.Nickname, game.Key(), game.DisplayTitle())
			}
			pokedCount++
			time.Sleep(time.Duration(b.Cfg.PokeDelayMS) * time.Millisecond)
		}
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
	for len(msg) > limit {
		idx := strings.LastIndex(msg[:limit], "\n")
		if idx == -1 {
			idx = limit
		}
		chunks = append(chunks, msg[:idx])
		// Skip the newline itself to prevent leading newlines in chunks
		msg = strings.TrimLeft(msg[idx:], "\n")
	}
	if len(msg) > 0 {
		chunks = append(chunks, msg)
	}
	return chunks
}

func composePoke(g games.Game, shortURL string, theme *content.Theme, lvl *levelResult) string {
	prefix := "Free: "
	if theme != nil && theme.Emoji != "" {
		prefix = theme.Emoji + " Free: "
	}
	suffix := ""
	if lvl != nil {
		sign := "+"
		amt := lvl.Awarded
		if amt < 0 {
			sign = "" // amt already has "-"
		}
		suffix = fmt.Sprintf(" %s%dXP L%d", sign, amt, lvl.NewLevel)
	}
	title := g.DisplayTitle()
	avail := 100 - len(prefix) - 1 - len(shortURL) - len(suffix)
	if avail > 4 && len(title) > avail {
		title = title[:avail-3] + "..."
	}
	return fmt.Sprintf("%s%s %s%s", prefix, title, shortURL, suffix)
}

func (b *Bot) composePM(g games.Game, shortURL string, theme *content.Theme, lvl *levelResult, notes []string, totalGS int) string {
	var sb strings.Builder
	if theme != nil {
		sb.WriteString(theme.Emoji + " " + theme.Banner)
	} else if b.Cfg.EnableGreetings {
		sb.WriteString(content.RandomGreeting())
	} else {
		sb.WriteString("Daily Free Game!")
	}
	sb.WriteString("\n")

	name := g.DisplayTitle()
	fmt.Fprintf(&sb, "🎮 %s\n", name)
	if g.WorthShown() {
		fmt.Fprintf(&sb, "💰 Worth %s → FREE now\n", g.Worth)
	}
	fmt.Fprintf(&sb, "🔗 Claim: %s\n", shortURL)

	if b.Cfg.EnableYouTubeTrailer {
		fmt.Fprintf(&sb, "▶️ Trailer: %s\n", games.TrailerSearchURL(name))
	}

	if lvl != nil {
		fmt.Fprintf(&sb, "🏆 %s (Lvl %d) [GS: %d] — +%d XP (%d total)\n",
			leveling.LevelName(lvl.NewLevel), lvl.NewLevel, totalGS, lvl.Awarded, lvl.TotalXP)
		if lvl.NewLevel > lvl.OldLevel {
			fmt.Fprintf(&sb, "🎉 Level up! You are now a %s!\n", leveling.LevelName(lvl.NewLevel))
		}
	}
	for _, note := range notes {
		fmt.Fprintf(&sb, "✨ %s\n", note)
	}
	if theme != nil && theme.Signoff != "" {
		sb.WriteString(theme.Signoff)
	}
	return strings.TrimRight(sb.String(), "\n")
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
		if sessionMS > lastMS { deltaSec = (sessionMS - lastMS) / 1000 } else { deltaSec = sessionMS / 1000 }
	} else { deltaSec = sessionMS / 1000 }
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
