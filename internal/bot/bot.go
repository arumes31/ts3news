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
	levelGroups map[int]int // manual level milestone -> existing TS3 server group id
	xpGroups    map[int]int // auto-created level group -> TS3 server group id (XP_SERVER_GROUPS)
	parties     [][]string  // each party is a list of lowercased member nicknames
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
		parties:     parseParties(cfg.Parties),
	}
	if cfg.XPServerGroups {
		b.loadLevelGroups()
	}
	return b
}

// parseParties parses "Nick1,Nick2;Nick3,Nick4" into lowercased member lists.
func parseParties(s string) [][]string {
	var out [][]string
	for _, party := range strings.Split(s, ";") {
		var members []string
		for _, m := range strings.Split(party, ",") {
			if m = strings.ToLower(strings.TrimSpace(m)); m != "" {
				members = append(members, m)
			}
		}
		if len(members) > 0 {
			out = append(out, members)
		}
	}
	return out
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

	targetNick := strings.TrimSpace(b.Cfg.TargetNick)
	log.Printf("Online clients: %d. Target filter: '%s'. Found %d free games.", len(clients), targetNick, len(freeGames))

	// Diagnostic: list the online normal (voice) clients so it is easy to see who
	// is connected and pick the right TS3_TARGET_NICK.
	var onlineNames []string
	for _, cl := range clients {
		if cl.Type == 0 {
			onlineNames = append(onlineNames, cl.Nickname)
		}
	}
	log.Printf("Online normal clients: %v", onlineNames)

	theme := b.activeTheme()
	if theme != nil {
		log.Printf("Holiday theme active: %s %s", theme.Emoji, theme.Name)
	}

	// Remove any auto-created level groups that have been left empty.
	if b.Cfg.XPServerGroups {
		b.cleanupEmptyLevelGroups(c)
	}

	// Shared per-cycle context, then apply the inactivity (Sloth) decay to users
	// who have been offline past the grace period.
	ctx := b.buildCycleContext(clients)
	b.slothDecay(c, ctx.today)

	pokedCount := 0
	usedDynamicNick := false
	for _, client := range clients {
		if client.Type != 0 {
			continue
		}
		if targetNick != "" && !strings.EqualFold(strings.TrimSpace(client.Nickname), targetNick) {
			continue
		}
		if client.UID == "" {
			continue // need a stable unique id to track per user
		}

		// Mark every eligible user as seen (drives dead-user cleanup).
		if err := b.touchUser(client.UID, client.Nickname); err != nil {
			log.Printf("touchUser failed for %s: %v", client.Nickname, err)
		}

		alreadySent, err := b.getSentGames(client.UID)
		if err != nil {
			log.Printf("Failed to get sent games for %s: %v", client.Nickname, err)
			continue
		}

		candidates := filterNewGames(freeGames, alreadySent)
		// Prefer GamerPower links when any are available.
		var gpCandidates []games.Game
		for _, g := range candidates {
			if strings.EqualFold(g.Source, "GamerPower") {
				gpCandidates = append(gpCandidates, g)
			}
		}
		if len(gpCandidates) > 0 {
			candidates = gpCandidates
		}

		hasGame := len(candidates) > 0
		var game games.Game
		var shortURL string
		if hasGame {
			game = candidates[rand.Intn(len(candidates))]
			if shortURL, err = games.ShortenURL(game.URL); err != nil {
				log.Printf("Shortening failed: %v", err)
				shortURL = game.URL
			}
		}

		// Process XP (login bonus, multipliers, loot boxes, artifact roll). With no
		// new game a 50% penalty applies but XP/levels/groups still progress.
		var lvl *levelResult
		var notes []string
		var artifactPoke string
		if b.Cfg.EnableLeveling {
			lvl, notes, artifactPoke = b.processUserXP(client.UID, client.Nickname, b.xpForGame(game), hasGame, ctx)
			if lvl != nil {
				b.applyMilestones(c, client.CLID, client.Nickname, lvl)
				if b.Cfg.XPServerGroups {
					b.applyLevelGroup(c, client.CLID, client.UID, client.Nickname, lvl.NewLevel)
				}
			}
		}

		// Extra poke announcing a freshly granted corrupted artifact (game or not).
		if artifactPoke != "" {
			if err := c.Poke(client.CLID, artifactPoke); err != nil {
				log.Printf("Artifact poke failed for %s: %v", client.Nickname, err)
			}
		}

		if !hasGame {
			continue // XP processed; nothing new to announce
		}

		// Bot adopts a game-themed nickname while announcing.
		if b.Cfg.DynamicNickname {
			usedDynamicNick = true
			if err := c.SetNickname(content.NicknameForGame(game.Title)); err != nil {
				log.Printf("SetNickname failed: %v", err)
			}
		}

		pokeMsg := composePoke(game, shortURL, theme, lvl)
		pmMsg := b.composePM(game, shortURL, theme, lvl, notes)

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
	if usedDynamicNick {
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
// trivia, holiday flavour, and any progression notes.
func (b *Bot) composePM(g games.Game, shortURL string, theme *content.Theme, lvl *levelResult, notes []string) string {
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
	fmt.Fprintf(&sb, "🎮 %s\n", name)
	if g.WorthShown() {
		fmt.Fprintf(&sb, "💰 Worth %s → FREE now\n", g.Worth)
	}
	fmt.Fprintf(&sb, "🔗 Claim: %s\n", shortURL)

	if b.Cfg.EnableYouTubeTrailer {
		fmt.Fprintf(&sb, "▶️ Trailer: %s\n", games.TrailerSearchURL(name))
	}

	if lvl != nil {
		fmt.Fprintf(&sb, "🏆 %s (Lvl %d) — +%d XP (%d total)\n",
			leveling.LevelName(lvl.NewLevel), lvl.NewLevel, lvl.Awarded, lvl.TotalXP)
		if lvl.NewLevel > lvl.OldLevel {
			fmt.Fprintf(&sb, "🎉 Level up! You are now a %s!\n", leveling.LevelName(lvl.NewLevel))
		}
	}
	
	for _, note := range notes {
		fmt.Fprintf(&sb, "✨ %s\n", note)
	}

	if b.Cfg.EnableTrivia {
		fmt.Fprintf(&sb, "💡 Did you know? %s\n", content.RandomTrivia())
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
	defer func() { _ = rows.Close() }()

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
