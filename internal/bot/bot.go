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
	levelGroups map[int]int // level milestone -> TS3 server group id
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

	return &Bot{
		Cfg:         cfg,
		DB:          database,
		levelGroups: leveling.ParseLevelGroups(cfg.LevelGroups),
	}
}

func (b *Bot) Close() {
	if b.DB != nil {
		if err := b.DB.Close(); err != nil {
			log.Printf("Error closing database: %v", err)
		}
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

// RunCycle fetches free games and notifies every eligible online client once per
// new game, awarding XP and updating per-user state. The ClientQuery connection
// must already be authenticated and connected to the server.
func (b *Bot) RunCycle(c *clientquery.Client) error {
	freeGames, err := games.FetchFreeGames(b.fetchOptions())
	if err != nil {
		return fmt.Errorf("failed to fetch games: %w", err)
	}

	clients, err := c.ClientList()
	if err != nil {
		return fmt.Errorf("failed to list clients: %w", err)
	}

	targetNick := b.Cfg.TargetNick
	log.Printf("Online clients: %d. Target filter: '%s'. Found %d free games.", len(clients), targetNick, len(freeGames))

	theme := b.activeTheme()
	if theme != nil {
		log.Printf("Holiday theme active: %s %s", theme.Emoji, theme.Name)
	}

	pokedCount := 0
	for _, client := range clients {
		if client.Type != 0 {
			continue
		}
		if targetNick != "" && client.Nickname != targetNick {
			continue
		}
		if client.UID == "" {
			continue // need a stable unique id to track per user
		}

		// Mark every eligible user as seen (drives dead-user cleanup) even if there
		// is nothing new to send this cycle.
		if err := b.touchUser(client.UID, client.Nickname); err != nil {
			log.Printf("touchUser failed for %s: %v", client.Nickname, err)
		}

		alreadySent, err := b.getSentGames(client.UID)
		if err != nil {
			log.Printf("Failed to get sent games for %s: %v", client.Nickname, err)
			continue
		}

		candidates := filterNewGames(freeGames, alreadySent)
		if len(candidates) == 0 {
			continue
		}
		game := candidates[rand.Intn(len(candidates))]

		shortURL, err := games.ShortenURL(game.URL)
		if err != nil {
			log.Printf("Shortening failed: %v", err)
			shortURL = game.URL
		}

		// Award XP / level up before composing the message so the PM is accurate.
		var lvl *levelResult
		if b.Cfg.EnableLeveling {
			lr, err := b.awardXP(client.UID, client.Nickname, b.xpForGame(game))
			if err != nil {
				log.Printf("awardXP failed for %s: %v", client.Nickname, err)
			} else {
				lvl = lr
				b.applyMilestones(c, client.CLID, client.Nickname, lr)
			}
		}

		// Feature: bot adopts a game-themed nickname while announcing.
		if b.Cfg.DynamicNickname {
			if err := c.SetNickname(content.NicknameForGame(game.Title)); err != nil {
				log.Printf("SetNickname failed: %v", err)
			}
		}

		pokeMsg := composePoke(game, shortURL, theme, lvl)
		pmMsg := b.composePM(game, shortURL, theme, lvl)

		log.Printf("Notifying %s about %s [%s] (%s)", client.Nickname, game.DisplayTitle(), game.Source, shortURL)

		if err := c.Poke(client.CLID, pokeMsg); err != nil {
			log.Printf("Poke failed for %s: %v", client.Nickname, err)
		}
		if err := c.SendPrivateMessage(client.CLID, pmMsg); err != nil {
			log.Printf("PM failed for %s: %v", client.Nickname, err)
		}

		if err := b.markAsSent(client.UID, client.Nickname, game.Key(), game.DisplayTitle()); err != nil {
			log.Printf("Failed to mark game as sent for %s: %v", client.Nickname, err)
		}
		pokedCount++
		time.Sleep(time.Duration(b.Cfg.PokeDelayMS) * time.Millisecond)
	}

	// Restore the bot's configured nickname after a dynamic-nickname cycle.
	if b.Cfg.DynamicNickname && pokedCount > 0 {
		if err := c.SetNickname(b.Cfg.TS3Nickname); err != nil {
			log.Printf("Restoring nickname failed: %v", err)
		}
	}

	log.Printf("Cycle finished. Poked %d users.", pokedCount)
	return nil
}

// activeTheme returns the current holiday theme, or nil when disabled / ordinary day.
func (b *Bot) activeTheme() *content.Theme {
	if !b.Cfg.EnableHolidayThemes {
		return nil
	}
	return content.CurrentTheme(time.Now())
}

// composePoke builds the <=100 char poke (must contain the link). The clean game
// name is used (no "Giveaway"/platform tags) and a short XP/level note is appended
// when leveling is active.
func composePoke(g games.Game, shortURL string, theme *content.Theme, lvl *levelResult) string {
	prefix := "Free: "
	if theme != nil && theme.Emoji != "" {
		prefix = theme.Emoji + " Free: "
	}
	suffix := ""
	if lvl != nil {
		suffix = fmt.Sprintf(" +%dXP L%d", lvl.Awarded, lvl.NewLevel)
	}
	title := g.DisplayTitle()
	avail := 100 - len(prefix) - 1 - len(shortURL) - len(suffix)
	if avail > 4 && len(title) > avail {
		title = title[:avail-3] + "..."
	}
	return fmt.Sprintf("%s%s %s%s", prefix, title, shortURL, suffix)
}

// composePM builds the rich private message with greeting, trailer, XP/level,
// trivia and holiday flavour.
func (b *Bot) composePM(g games.Game, shortURL string, theme *content.Theme, lvl *levelResult) string {
	var sb strings.Builder

	// Greeting / themed banner.
	if theme != nil {
		sb.WriteString(theme.Emoji + " " + theme.Banner)
	} else if b.Cfg.EnableGreetings {
		sb.WriteString(content.RandomGreeting())
	} else {
		sb.WriteString("Daily Free Game!")
	}
	sb.WriteString("\n")

	name := g.DisplayTitle()
	sb.WriteString(fmt.Sprintf("🎮 %s\n", name))
	if g.WorthShown() {
		sb.WriteString(fmt.Sprintf("💰 Worth %s → FREE now\n", g.Worth))
	}
	sb.WriteString(fmt.Sprintf("🔗 Claim: %s\n", shortURL))

	if b.Cfg.EnableYouTubeTrailer {
		sb.WriteString(fmt.Sprintf("▶️ Trailer: %s\n", games.TrailerSearchURL(name)))
	}

	if lvl != nil {
		sb.WriteString(fmt.Sprintf("🏆 %s (Lvl %d) — +%d XP (%d total)\n",
			leveling.LevelName(lvl.NewLevel), lvl.NewLevel, lvl.Awarded, lvl.TotalXP))
		if lvl.NewLevel > lvl.OldLevel {
			sb.WriteString(fmt.Sprintf("🎉 Level up! You are now a %s!\n", leveling.LevelName(lvl.NewLevel)))
		}
	}

	if b.Cfg.EnableTrivia {
		sb.WriteString(fmt.Sprintf("💡 Did you know? %s\n", content.RandomTrivia()))
	}

	if theme != nil && theme.Signoff != "" {
		sb.WriteString(theme.Signoff)
	}

	return strings.TrimRight(sb.String(), "\n")
}

// ---- per-user game dedup ----

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

// getSentGames returns the game keys already sent to a user within the resend
// window (ResendAfterDays <= 0 => never expire).
func (b *Bot) getSentGames(uid string) ([]string, error) {
	var rows *sql.Rows
	var err error
	if b.Cfg.ResendAfterDays > 0 {
		rows, err = b.DB.Query(
			"SELECT game_key FROM sent_notifications WHERE client_uid = $1 AND sent_at > NOW() - ($2 * INTERVAL '1 day')",
			uid, b.Cfg.ResendAfterDays)
	} else {
		rows, err = b.DB.Query("SELECT game_key FROM sent_notifications WHERE client_uid = $1", uid)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []string
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}

func (b *Bot) markAsSent(uid, nickname, gameKey, gameTitle string) error {
	_, err := b.DB.Exec(
		`INSERT INTO sent_notifications (client_uid, game_key, game_title, client_nickname, sent_at)
		 VALUES ($1, $2, $3, $4, NOW())
		 ON CONFLICT (client_uid, game_key)
		 DO UPDATE SET sent_at = NOW(), client_nickname = $4, game_title = $3`,
		uid, gameKey, gameTitle, nickname)
	return err
}

// ---- users / leveling ----

type levelResult struct {
	OldLevel int
	NewLevel int
	TotalXP  int
	Awarded  int
}

// touchUser records that a user was seen (upsert, refreshing last_seen).
func (b *Bot) touchUser(uid, nickname string) error {
	_, err := b.DB.Exec(
		`INSERT INTO users (client_uid, nickname, last_seen)
		 VALUES ($1, $2, NOW())
		 ON CONFLICT (client_uid)
		 DO UPDATE SET last_seen = NOW(), nickname = $2`,
		uid, nickname)
	return err
}

// xpForGame returns the XP award for a game, scaled by its price (pricier games
// grant more XP by default; configurable via CHEAPER_MORE_XP). Falls back to a
// randomised award when the price is unknown.
func (b *Bot) xpForGame(g games.Game) int {
	if p, ok := g.PriceEUR(); ok {
		return leveling.XPForPrice(p, b.Cfg.CheaperMoreXP)
	}
	return leveling.XPPerPoke()
}

// awardXP grants the given XP for a poke and recomputes the user's level.
func (b *Bot) awardXP(uid, nickname string, awarded int) (*levelResult, error) {
	var curXP, curLevel int
	err := b.DB.QueryRow("SELECT xp, level FROM users WHERE client_uid = $1", uid).Scan(&curXP, &curLevel)
	if err == sql.ErrNoRows {
		curXP, curLevel = 0, 1
	} else if err != nil {
		return nil, err
	}

	total := curXP + awarded
	newLevel := leveling.LevelForXP(total)

	_, err = b.DB.Exec(
		`INSERT INTO users (client_uid, nickname, xp, level, last_seen)
		 VALUES ($1, $2, $3, $4, NOW())
		 ON CONFLICT (client_uid)
		 DO UPDATE SET xp = $3, level = $4, nickname = $2, last_seen = NOW()`,
		uid, nickname, total, newLevel)
	if err != nil {
		return nil, err
	}

	return &levelResult{OldLevel: curLevel, NewLevel: newLevel, TotalXP: total, Awarded: awarded}, nil
}

// applyMilestones grants TS3 server groups for any level milestones newly crossed.
func (b *Bot) applyMilestones(c *clientquery.Client, clid int, nickname string, lr *levelResult) {
	if len(b.levelGroups) == 0 || lr.NewLevel <= lr.OldLevel {
		return
	}
	groups := leveling.MilestonesCrossed(lr.OldLevel, lr.NewLevel, b.levelGroups)
	if len(groups) == 0 {
		return
	}
	cldbid, err := c.ClientDBID(clid)
	if err != nil {
		log.Printf("Cannot resolve cldbid for %s (skipping group grant): %v", nickname, err)
		return
	}
	for _, sgid := range groups {
		if err := c.AddServerGroup(sgid, cldbid); err != nil {
			log.Printf("Granting server group %d to %s failed (needs permission): %v", sgid, nickname, err)
		} else {
			log.Printf("Granted server group %d to %s (level %d)", sgid, nickname, lr.NewLevel)
		}
	}
}

// CleanupDeadUsers purges users not seen for DeadUserDays days, plus their
// notification history. Returns the number of users removed.
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

// ---- ClientQuery API key ----

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
