package bot

import (
	"bufio"
	"database/sql"
	"fmt"
	"log"
	"math/rand/v2"
	"os"
	"strings"
	"time"

	"ts3news/internal/clientquery"
	"ts3news/internal/config"
	"ts3news/internal/content"
	"ts3news/internal/db"
	"ts3news/internal/games"
	"ts3news/internal/i18n"
	"ts3news/internal/leveling"

	_ "github.com/lib/pq"
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

// UserInCombat represents a user participating in combat
type UserInCombat struct {
	UID           string
	Nickname      string
	CLID          int
	Stats         content.Stats
	Level         int
	Skills        []content.Skill
	UltimateSkill *content.UltimateSkill
	CurrentHP     int
	RegenStacks   int
	Gold          int64
	Equipped      map[content.GearSlot]content.Gear
	GearScore     float64
}

// LootResult represents the result of a loot drop
type LootResult struct {
	UID   string
	Poke  string
	Loot  interface{}
	Count int
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

// calculateTotalStats calculates a user's total stats from gear and base stats
func (b *Bot) calculateTotalStats(uid string, now time.Time) (content.Stats, int, float64, error) {
	// Base stats from DB
	var baseHP, baseSTR, baseDEF, baseSPD, baseLCK, baseINT, baseSTA, baseCRT, baseDGE int
	err := b.DB.QueryRow(`
		SELECT hp, str, def, spd, lck, int, sta, crt, dge
		FROM users
		WHERE client_uid = $1`, uid).
		Scan(&baseHP, &baseSTR, &baseDEF, &baseSPD, &baseLCK, &baseINT, &baseSTA, &baseCRT, &baseDGE)
	if err != nil {
		return content.Stats{}, 0, 0, err
	}

	// Calculate gear stats
	var totalStats content.Stats
	totalStats.HP = baseHP
	totalStats.STR = baseSTR
	totalStats.DEF = baseDEF
	totalStats.SPD = baseSPD
	totalStats.LCK = baseLCK
	totalStats.INT = baseINT
	totalStats.STA = baseSTA
	totalStats.CRT = baseCRT
	totalStats.DGE = baseDGE

	// Get equipped gear
	rows, err := b.DB.Query(`
		SELECT g.id, g.stats, g.rarity, g.combat_rating
		FROM user_gear ug
		JOIN gear g ON ug.gear_id = g.id
		WHERE ug.client_uid = $1`, uid)
	if err != nil {
		return content.Stats{}, 0, 0, err
	}
	defer func() { _ = rows.Close() }()

	var gearScore float64
	for rows.Next() {
		var gearID string
		var gearStats content.Stats
		var rarity int
		var combatRating float64
		if err := rows.Scan(&gearID, &gearStats, &rarity, &combatRating); err != nil {
			log.Printf("Warning: failed to scan gear row (gearID=%s, row index): %v", gearID, err)
			continue
		}
		totalStats.HP += gearStats.HP
		totalStats.STR += gearStats.STR
		totalStats.DEF += gearStats.DEF
		totalStats.SPD += gearStats.SPD
		totalStats.LCK += gearStats.LCK
		totalStats.INT += gearStats.INT
		totalStats.STA += gearStats.STA
		totalStats.CRT += gearStats.CRT
		totalStats.DGE += gearStats.DGE
		gearScore += combatRating * float64(rarity)
	}

	// Get enchantments
	enchantRows, err := b.DB.Query(`
		SELECT e.stats, e.rarity
		FROM user_enchantments ue
		JOIN enchantments e ON ue.enchantment_id = e.id
		WHERE ue.client_uid = $1`, uid)
	if err != nil {
		return content.Stats{}, 0, 0, err
	}
	defer func() { _ = enchantRows.Close() }()

	for enchantRows.Next() {
		var enchantStats content.Stats
		var rarity int
		if err := enchantRows.Scan(&enchantStats, &rarity); err != nil {
			log.Printf("Warning: failed to scan enchantment row: %v", err)
			continue
		}
		totalStats.HP += enchantStats.HP
		totalStats.STR += enchantStats.STR
		totalStats.DEF += enchantStats.DEF
		totalStats.SPD += enchantStats.SPD
		totalStats.LCK += enchantStats.LCK
		totalStats.INT += enchantStats.INT
		totalStats.STA += enchantStats.STA
		totalStats.CRT += enchantStats.CRT
		totalStats.DGE += enchantStats.DGE
	}

	return totalStats, 0, gearScore, nil
}

// getSkills retrieves a user's skills
func (b *Bot) getSkills(uid string) []content.Skill {
	var skills []content.Skill
	rows, err := b.DB.Query(`
		SELECT s.id, s.name, s.type, s.rarity, s.power, s.ignore_def, s.stun_chance, s.heal_percent, s.description, s.special
		FROM user_skills us
		JOIN skills s ON us.skill_id = s.id
		WHERE us.client_uid = $1`, uid)
	if err != nil {
		return skills
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var skill content.Skill
		if err := rows.Scan(&skill.ID, &skill.Name, &skill.Type, &skill.Rarity, &skill.Power, &skill.IgnoreDef, &skill.StunChance, &skill.HealPercent, &skill.Description, &skill.Special); err != nil {
			log.Printf("Warning: failed to scan skill row: %v", err)
			continue
		}
		skills = append(skills, skill)
	}
	return skills
}

// getUltimateSkill retrieves a user's ultimate skill
func (b *Bot) getUltimateSkill(uid string) *content.UltimateSkill {
	var ultimate *content.UltimateSkill
	rows, err := b.DB.Query(`
		SELECT u.id, u.name, u.rarity, u.power, u.cooldown_rounds, u.current_cooldown, u.description, u.special
		FROM user_ultimate_skills uus
		JOIN ultimate_skills u ON uus.ultimate_id = u.id
		WHERE uus.client_uid = $1`, uid)
	if err != nil {
		return ultimate
	}
	defer func() { _ = rows.Close() }()

	if rows.Next() {
		ultimate = &content.UltimateSkill{}
		if err := rows.Scan(&ultimate.ID, &ultimate.Name, &ultimate.Rarity, &ultimate.Power, &ultimate.CooldownRounds, &ultimate.CurrentCooldown, &ultimate.Description, &ultimate.Special); err != nil {
			log.Printf("Warning: failed to scan ultimate skill row: %v", err)
			return nil
		}
		return ultimate
	}
	return nil
}

// resolveChannelCombat resolves combat between users in a channel
func (b *Bot) resolveChannelCombat(users []UserInCombat, mobs []*content.Mob, avgLvl int, difficulty float64, zone content.Zone) ([]string, int, bool, []LootResult) {
	_ = avgLvl
	_ = difficulty
	if len(users) == 0 || len(mobs) == 0 {
		return nil, 0, false, nil
	}

	var logs []string

	// Party offensive power (mob stats already bake in level & difficulty scaling).
	partyPower := 0.0
	for _, u := range users {
		partyPower += float64(u.Stats.STR + u.Stats.SPD + u.Stats.HP/5 + u.Stats.CRT)
	}

	// Mob threat + wave header.
	mobThreat := 0
	names := make([]string, 0, len(mobs))
	for _, m := range mobs {
		mobThreat += m.Score()
		names = append(names, m.DisplayName())
	}
	logs = append(logs, i18n.T("bot.combat.wave_header", 1, mobThreat, strings.Join(names, ", ")))

	victory := partyPower >= float64(mobThreat)*0.6
	partyDmg := int(partyPower)
	mobDmg := mobThreat / 2

	var loots []LootResult
	rewardXP := 0

	if victory {
		for _, m := range mobs {
			// #nosec G404
			killer := users[rand.IntN(len(users))]
			logs = append(logs, i18n.T("bot.combat.defeated", m.DisplayNameShort(), killer.Nickname))
			rewardXP += m.RewardXP
			for _, note := range b.rollLootForUser(killer.UID, killer.Nickname, m) {
				logs = append(logs, note)
				loots = append(loots, LootResult{UID: killer.UID})
			}
		}
		logs = append(logs, i18n.T("bot.combat.battle_summary", partyDmg, mobDmg))
		logs = append(logs, i18n.T("bot.combat.victory", len(mobs), zone.Name))
	} else {
		logs = append(logs, i18n.T("bot.combat.battle_summary", partyDmg, mobDmg))
		logs = append(logs, i18n.T("bot.combat.defeat", zone.Name))
	}

	return logs, rewardXP, victory, loots
}

// processUserXP processes XP gain for a user
func (b *Bot) processUserXP(uid, nickname string, cid int, baseXP int, hasGame bool, now time.Time) (*levelResult, []string, string) {
	// Placeholder implementation
	return nil, []string{}, ""
}

// applyDurabilityLoss applies durability loss to gear
func (b *Bot) applyDurabilityLoss(uid string, isVictory bool) string {
	// Placeholder implementation
	return ""
}

// shouldEquip reports whether a gear item is an upgrade over what the player has
// equipped in that slot (and should therefore be equipped on pickup).
func (b *Bot) shouldEquip(uid string, gear content.Gear) bool {
	return decideGearFate(gear, b.equippedInSlot(uid, gear.Slot)) == fateEquip
}

// equipSkill learns a skill, filling an empty skill slot (1..5) or replacing the
// weakest currently-known skill if the new one is stronger. It reports whether
// the skill was actually equipped.
func (b *Bot) equipSkill(uid string, skill content.Skill) (bool, error) {
	rows, err := b.DB.Query("SELECT slot, skill_id FROM user_skills WHERE client_uid = $1", uid)
	if err != nil {
		return false, err
	}
	used := map[int]string{}
	for rows.Next() {
		var slot int
		var id string
		if err := rows.Scan(&slot, &id); err == nil {
			used[slot] = id
		}
	}
	_ = rows.Close()

	// Fill the first empty slot.
	for slot := 1; slot <= 5; slot++ {
		if _, taken := used[slot]; !taken {
			_, err := b.DB.Exec(
				`INSERT INTO user_skills (client_uid, slot, skill_id) VALUES ($1, $2, $3)
				 ON CONFLICT (client_uid, slot) DO UPDATE SET skill_id = $3`,
				uid, slot, skill.ID)
			return err == nil, err
		}
	}

	// All slots full: replace the weakest known skill if the new one beats it.
	worstSlot, worstScore := 0, int(^uint(0)>>1)
	for slot, id := range used {
		if cur, ok := content.GetSkillByID(id); ok && cur.Score() < worstScore {
			worstScore, worstSlot = cur.Score(), slot
		}
	}
	if worstSlot != 0 && skill.Score() > worstScore {
		_, err := b.DB.Exec("UPDATE user_skills SET skill_id = $1 WHERE client_uid = $2 AND slot = $3",
			skill.ID, uid, worstSlot)
		return err == nil, err
	}
	return false, nil
}

// applyEnchantment attaches an enchantment to one of the player's equipped gear
// pieces that does not already carry one.
func (b *Bot) applyEnchantment(uid string, enchantment content.Enchantment) error {
	var slot string
	if err := b.DB.QueryRow(
		`SELECT slot FROM user_gear
		 WHERE client_uid = $1 AND (enchantment_id IS NULL OR enchantment_id = '')
		 ORDER BY slot LIMIT 1`, uid).Scan(&slot); err != nil {
		return err
	}
	_, err := b.DB.Exec("UPDATE user_gear SET enchantment_id = $1 WHERE client_uid = $2 AND slot = $3",
		enchantment.ID, uid, slot)
	return err
}

// RunCycle now resolves group combat by channel
func (b *Bot) RunCycle(c *clientquery.Client) error {
	cycleTime := time.Now()
	var freeGames []games.Game
	var err error
	if b.Cfg.EnableGameNews {
		freeGames, err = games.FetchFreeGames(b.fetchOptions())
		if err != nil {
			return fmt.Errorf("failed to fetch games: %w", err)
		}
	}

	clients, err := c.ClientList()
	if err != nil {
		return fmt.Errorf("failed to list clients: %w", err)
	}

	targetNick := strings.TrimSpace(b.Cfg.TargetNick)
	// ctx := b.buildCycleContext(clients)

	if b.Cfg.EnableRPG {
		// b.slothDecay(c, cycleTime)
		if b.Cfg.XPServerGroups {
			b.cleanupEmptyLevelGroups(c)
		}
	}

	// Group normal users by channel
	chanUsers := map[int][]UserInCombat{}
	for _, cl := range clients {
		if cl.Type != 0 || (targetNick != "" && !strings.EqualFold(cl.Nickname, targetNick)) || cl.UID == "" {
			continue
		}

		if !b.Cfg.EnableRPG {
			chanUsers[cl.CID] = append(chanUsers[cl.CID], UserInCombat{
				UID: cl.UID, Nickname: cl.Nickname, CLID: cl.CLID,
			})
			continue
		}

		stats, _, gearScore, _ := b.calculateTotalStats(cl.UID, cycleTime)
		skills := b.getSkills(cl.UID)
		ultimate := b.getUltimateSkill(cl.UID)

		var lvl, prestige, curHP, regen int
		var gold int64
		err := b.DB.QueryRow("SELECT level, prestige, current_hp, regen_stacks, gold FROM users WHERE client_uid=$1", cl.UID).Scan(&lvl, &prestige, &curHP, &regen, &gold)
		if err != nil && err != sql.ErrNoRows {
			log.Printf("Failed to scan user combat state for %s: %v", cl.UID, err)
		}
		if curHP <= 0 {
			curHP = stats.HP
		} // Auto-fill if new/dead
		equipped := b.getEquippedItems(cl.UID)

		chanUsers[cl.CID] = append(chanUsers[cl.CID], UserInCombat{
			UID: cl.UID, Nickname: cl.Nickname, CLID: cl.CLID, Stats: stats, Level: lvl, Skills: skills,
			UltimateSkill: ultimate, CurrentHP: curHP, RegenStacks: regen, Gold: gold,
			Equipped: equipped, GearScore: gearScore,
		})
	}

	theme := b.activeTheme()
	pokedCount := 0

	for cid, users := range chanUsers {
		if len(users) == 0 {
			continue
		}

		var resLogs []string
		var rewardXP int
		var victory bool
		var combatLoots []LootResult
		var zone *content.Zone

		if b.Cfg.EnableRPG {
			// 1. Party Stats & Difficulty
			totalLvl := 0
			totalGS := 0.0
			for _, u := range users {
				totalLvl += u.Level
				totalGS += u.GearScore
			}
			avgLvl := totalLvl / len(users)
			if avgLvl < 1 {
				avgLvl = 1
			}
			avgGS := totalGS / float64(len(users))

			// 2. Select Zone
			z := content.GetRandomZone(avgLvl, avgGS)
			zone = &z
			battleLogs := []string{zone.Display()}

			mobs := content.SpawnMobGroup(avgLvl, *zone, zone.Difficulty, len(users))
			var mobPtrs []*content.Mob
			for i := range mobs {
				mobPtrs = append(mobPtrs, &mobs[i])
			}

			// 3. Resolve Group Combat
			resLogs, rewardXP, victory, combatLoots = b.resolveChannelCombat(users, mobPtrs, avgLvl, zone.Difficulty, *zone)
			resLogs = append(battleLogs, resLogs...)
		}

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

			var lr *levelResult
			var notes []string
			var extraPoke string
			var userLootFound bool

			if b.Cfg.EnableRPG {
				baseXP := b.xpForGame(game)
				var artifactPoke string
				lr, notes, artifactPoke = b.processUserXP(user.UID, user.Nickname, cid, baseXP+rewardXP, hasGame, cycleTime)
				extraPoke = artifactPoke

				// Auto-prestige at the level cap
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
				if duraNote := b.applyDurabilityLoss(user.UID, !victory); duraNote != "" {
					notes = append(notes, duraNote)
				}

				for _, cl := range combatLoots {
					if cl.UID == user.UID {
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
					if lr != nil {
						b.applyMilestones(c, user.CLID, user.Nickname, lr)
						if b.Cfg.XPServerGroups {
							b.applyLevelGroup(c, user.CLID, user.UID, user.Nickname, lr.NewLevel)
						}
					}
					b.applyTitleGroup(c, user.CLID, user.UID, user.Nickname)
					if b.Cfg.EnableRPG {
						b.syncLootGroups(c, user.CLID, user.UID)
					}
				}
			}

			// Messaging
			var allLogs []string
			if b.Cfg.EnableRPG {
				allLogs = append(notes, resLogs...)
				if lr != nil {
					outcome := i18n.T("xp.battle")
					if lr.Awarded < 0 {
						outcome = i18n.T("xp.lost")
					}
					allLogs = append(allLogs, i18n.T("xp.outcome", outcome, lr.Awarded, leveling.LevelName(lr.NewLevel), lr.NewLevel))
				}
			}

			pokeMsg := composePoke(game, shortURL, theme, lr)
			var pmMsg string
			if b.Cfg.EnableRPG {
				pmMsg = b.composePM(user.UID, game, shortURL, theme, lr, allLogs, user.GearScore, user.CurrentHP)
			} else {
				pmMsg = b.composeSimplePM(game, shortURL, theme)
			}

			// Persona check
			botNick := b.Cfg.TS3Nickname
			if b.Cfg.EnableRPG && (userLootFound || extraPoke != "") {
				botNick = "godsfinger"
			}
			_ = c.SetNickname(botNick)

			// Send Pokes
			if b.Cfg.EnableRPG && extraPoke != "" {
				_ = c.Poke(user.CLID, strings.TrimSpace(extraPoke))
			}

			if hasGame && shortURL != "" {
				_ = c.Poke(user.CLID, pokeMsg)
			}

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

	if b.Cfg.EnableRPG {
		// Resolve Auction House purchases for all online players
		b.ResolveGlobalAH(c, clients)

		// Update channel descriptions with all players' stats
		if err := b.UpdateChannelDescriptions(c); err != nil {
			log.Printf("Warning: Failed to update channel descriptions: %v", err)
		} else {
			log.Printf("Updated channel descriptions")
		}

		// Process expired AH items (7+ days)
		b.processExpiredAHItems(c)
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
	emojiPrefix := ""
	if theme != nil && theme.Emoji != "" {
		emojiPrefix = theme.Emoji + " "
	}
	prefix := i18n.T("bot.poke.free_prefix", emojiPrefix)
	title := g.DisplayTitle()
	avail := 100 - len(prefix) - 1 - len(shortURL)
	if avail > 4 && len(title) > avail {
		title = title[:avail-3] + "..."
	}
	return fmt.Sprintf("%s%s %s", prefix, title, shortURL)
}

func (b *Bot) composePM(uid string, g games.Game, shortURL string, theme *content.Theme, lvl *levelResult, notes []string, gearScore float64, currentHP int) string {
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
		var prestige int
		_ = b.DB.QueryRow("SELECT prestige FROM users WHERE client_uid=$1", uid).Scan(&prestige)
		stats, _, _, _ := b.calculateTotalStats(uid, time.Now())
		cr := float64(stats.Score()) / 10.0
		xpNext := leveling.XPForLevel(lvl.NewLevel + 1)
		xpReq := xpNext - lvl.TotalXP

		// Format: Nick [P:XX] [Lvl:XX] [gs:XX.X] [CR:XX]
		sb.WriteString("\n" + i18n.T("bot.stats.header",
			leveling.LevelName(lvl.NewLevel), prestige, lvl.NewLevel, gearScore, cr) + "\n")
		sb.WriteString(i18n.T("bot.stats.xp_line",
			lvl.Awarded, FormatLarge(float64(lvl.TotalXP)), FormatLarge(float64(xpReq))) + "\n")

		if lvl.NewLevel > lvl.OldLevel {
			sb.WriteString(i18n.T("bot.stats.level_up", leveling.LevelName(lvl.NewLevel)) + "\n")
		}

		// Compact Player Info
		sb.WriteString(i18n.T("bot.stats.hp_line",
			currentHP, stats.HP, stats.STR, stats.DEF, stats.SPD,
			stats.LCK, stats.INT, stats.STA, stats.CRT, stats.DGE) + "\n")
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
			strings.Contains(note, "💢") || strings.Contains(note, "🐾") ||
			strings.Contains(note, "📊") || strings.Contains(note, "AMBUSH") || strings.Contains(note, "slain") ||
			strings.Contains(note, "cast") || strings.Contains(note, "used") || strings.Contains(note, "defeated") ||
			(strings.Contains(note, "✨") && (strings.Contains(note, ":") || strings.Contains(note, "!"))): // Skill/Pet logs
			combatNotes = append(combatNotes, note)
		case strings.Contains(note, "🎁") || strings.Contains(note, "💰") || strings.Contains(note, "🌟") ||
			strings.Contains(note, "Listed on AH") || strings.Contains(note, "Learned") || strings.Contains(note, "Equipped") ||
			strings.Contains(note, "XP") || strings.Contains(note, "Unique:") || strings.Contains(note, "Artifact:") ||
			strings.Contains(note, "Duplicate"):
			rewardNotes = append(rewardNotes, note)
		case strings.Contains(note, "dur") || strings.Contains(note, "🛡️") || strings.Contains(note, "Salvaged") || strings.Contains(note, "Scrap"):
			equipNotes = append(equipNotes, note)
		default:
			miscNotes = append(miscNotes, note)
		}
	}

	if len(miscNotes) > 0 {
		sb.WriteString("\n" + i18n.T("bot.section.bonuses") + "\n")
		const maxMiscLineLen = 950
		var line string
		for i, n := range miscNotes {
			entry := n
			if i > 0 {
				entry = " | " + n
			}
			if len(line)+len(entry) > maxMiscLineLen && line != "" {
				sb.WriteString(line + "\n")
				line = n
			} else {
				line += entry
			}
		}
		if line != "" {
			sb.WriteString(line + "\n")
		}
	}

	if len(combatNotes) > 0 {
		sb.WriteString("\n" + i18n.T("bot.section.combat") + "\n")
		const maxCombatLineLen = 950
		var line string
		for i, cn := range combatNotes {
			entry := cn
			if i > 0 {
				entry = " | " + cn
			}
			if len(line)+len(entry) > maxCombatLineLen && line != "" {
				sb.WriteString(line + "\n")
				line = cn
			} else {
				line += entry
			}
		}
		if line != "" {
			sb.WriteString(line + "\n")
		}
	}

	if len(rewardNotes) > 0 {
		sb.WriteString("\n" + i18n.T("bot.section.loot") + "\n")
		for _, n := range rewardNotes {
			fmt.Fprintf(&sb, " • %s\n", n)
		}
	}

	if len(equipNotes) > 0 {
		sb.WriteString("\n" + i18n.T("bot.section.equipment") + "\n")
		const maxNoteLineLen = 950
		var line string
		for i, gn := range equipNotes {
			entry := gn
			if i > 0 {
				entry = " | " + gn
			}
			if len(line)+len(entry) > maxNoteLineLen && line != "" {
				sb.WriteString(line + "\n")
				line = gn
			} else {
				line += entry
			}
		}
		if line != "" {
			sb.WriteString(line + "\n")
		}
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

func (b *Bot) composeSimplePM(g games.Game, shortURL string, theme *content.Theme) string {
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

func FormatGold(v int64) string {
	return i18n.FormatGold(v)
}

func FormatLarge(v float64) string {
	return i18n.FormatLarge(v)
}

func (b *Bot) getEquippedItems(uid string) map[content.GearSlot]content.Gear {
	out := make(map[content.GearSlot]content.Gear)
	rows, err := b.DB.Query("SELECT slot, gear_id FROM user_gear WHERE client_uid = $1", uid)
	if err != nil {
		return out
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var slot string
		var id string
		if err := rows.Scan(&slot, &id); err == nil {
			if gear, ok := content.GetGearByID(id); ok {
				out[content.GearSlot(slot)] = gear
			}
		}
	}
	return out
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

	// Update each channel's description
	for cid, users := range chanUsers {
		if len(users) == 0 {
			continue
		}

		log.Printf("Updating channel %d with %d users", cid, len(users))

		var sb strings.Builder
		sb.WriteString(i18n.T("channel.header", len(users)) + "\n[hr]\n")

		totalCR := 0.0
		totalGS := 0.0

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
			cr := float64(stats.Score()) / 10.0 // Combat Rating estimate
			totalCR += cr
			totalGS += float64(gearScore)

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

			sb.WriteString(i18n.T("channel.player_line",
				u.Nick, prestige, level, gearScore, cr, hpColor, actualCurrentHP, stats.HP, FormatGold(gold)) + "\n")

			sb.WriteString("  " + i18n.T("channel.stats_line",
				stats.STR, stats.DEF, stats.SPD, stats.LCK, stats.INT, stats.STA, stats.CRT, stats.DGE) + "\n")
		}

		sb.WriteString("\n[hr]\n" + i18n.T("channel.group_power", totalCR, totalGS/float64(len(users))))

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
	}

	log.Printf("Completed UpdateChannelDescriptions")
	return nil
}
