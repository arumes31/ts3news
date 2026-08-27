package bot

import (
	"database/sql"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"ts3news/internal/content"
)

const (
	abyssBiomeMasteryClears = 50
	abyssLoreTitle          = "Abyss Chronicler"
	abyssLoreAchievement    = "lore_complete"
)

type abyssBiomeMasteryView struct {
	Name     string
	Affinity string
	Clears   int
	Goal     int
	Percent  int
	Mastered bool
}

type abyssSetBookRow struct {
	ID        string
	Name      string
	Collected int
	Total     int
	Percent   int
}

type abyssCollectionView struct {
	Biomes        []abyssBiomeMasteryView
	BiomeMastered int
	BiomeTotal    int
	Sets          []abyssSetBookRow
	SetCollected  int
	SetTotal      int
	SetPercent    int
	SetReward25   bool
	SetReward50   bool
	SetReward100  bool
	LoreComplete  bool
	LoreTitle     string
}

func abyssBiomeMasteryKey(uid, biome string) string {
	slug := strings.NewReplacer(" ", "_", "-", "_").Replace(strings.ToLower(biome))
	return "abyss_biome_mastery_" + uid + "_" + slug
}

func abyssSetBookPrefix(uid string) string {
	return "abyss_set_book_" + uid + "_"
}

func abyssSetBookCountKey(uid string) string {
	return "abyss_set_book_count_" + uid
}

func abyssSetBookBackfillKey(uid string) string {
	return "abyss_set_book_backfilled_" + uid
}

func abyssBadgeSuffixKey(uid string) string {
	return "abyss_badge_suffix_" + uid
}

func abyssLoreTitleKey(uid string) string {
	return "abyss_title_unlock_" + uid + "_lore_complete"
}

func (b *Bot) abyssBiomeClears(uid, biome string) int {
	var raw string
	if err := b.DB.QueryRow("SELECT value FROM app_meta WHERE key=$1", abyssBiomeMasteryKey(uid, biome)).Scan(&raw); err != nil {
		return 0
	}
	clears, _ := strconv.Atoi(raw)
	return max(clears, 0)
}

func (b *Bot) recordAbyssBiomeClear(uid, biome string) {
	_, _ = b.DB.Exec(
		`INSERT INTO app_meta (key,value) VALUES ($1,'1')
		 ON CONFLICT (key) DO UPDATE
		 SET value=(CASE WHEN app_meta.value ~ '^[0-9]+$' THEN app_meta.value::integer + 1 ELSE 1 END)::text`,
		abyssBiomeMasteryKey(uid, biome),
	)
}

func (b *Bot) applyAbyssBiomeMastery(uid string, biome content.AbyssBiome, stats content.Stats) (content.Stats, bool) {
	if b.abyssBiomeClears(uid, biome.Name) < abyssBiomeMasteryClears {
		return stats, false
	}
	bonus := content.TreeBonus{Pct: map[string]float64{
		"hp_pct": 0.02, "str_pct": 0.02, "def_pct": 0.02, "spd_pct": 0.02, "int_pct": 0.02,
	}}
	return bonus.ApplyCombatPct(stats), true
}

func abyssNamedSetName(id string) string {
	if id == "" {
		return ""
	}
	return strings.ToUpper(id[:1]) + id[1:] + " Set"
}

func abyssNamedSetGear(gear content.Gear) bool {
	for _, setID := range content.AbyssNamedSetIDs() {
		if gear.SetID != setID {
			continue
		}
		for _, catalogGear := range content.AbyssSetCatalog(setID) {
			if catalogGear.ID == gear.ID {
				return true
			}
		}
	}
	return false
}

func (b *Bot) recordAbyssSetBookGear(uid string, gear content.Gear) {
	if !abyssNamedSetGear(gear) {
		return
	}
	result, err := b.DB.Exec(
		"INSERT INTO app_meta (key,value) VALUES ($1,'1') ON CONFLICT (key) DO NOTHING",
		abyssSetBookPrefix(uid)+gear.ID,
	)
	if err == nil {
		if inserted, resultErr := result.RowsAffected(); resultErr == nil && inserted > 0 {
			b.refreshAbyssSetBookCount(uid)
		}
	}
}

func (b *Bot) abyssCollectedSetGear(uid string) map[string]bool {
	prefix := abyssSetBookPrefix(uid)
	rows, err := b.DB.Query(
		`SELECT gear_id FROM user_inventory WHERE client_uid=$1
		 UNION SELECT gear_id FROM user_gear WHERE client_uid=$1
		 UNION SELECT SUBSTRING(key FROM CHAR_LENGTH($2)+1) FROM app_meta
		       WHERE LEFT(key, CHAR_LENGTH($2))=$2`,
		uid, prefix,
	)
	if err != nil {
		return nil
	}
	defer func() { _ = rows.Close() }()
	collected := make(map[string]bool)
	for rows.Next() {
		var gearID string
		if err := rows.Scan(&gearID); err == nil && gearID != "" {
			collected[gearID] = true
		}
	}
	return collected
}

func (b *Bot) abyssSetBook(uid string) ([]abyssSetBookRow, int, int) {
	collected := b.abyssCollectedSetGear(uid)
	rows := make([]abyssSetBookRow, 0, len(content.AbyssNamedSetIDs()))
	totalCollected, totalItems := 0, 0
	for _, setID := range content.AbyssNamedSetIDs() {
		catalog := content.AbyssSetCatalog(setID)
		row := abyssSetBookRow{ID: setID, Name: abyssNamedSetName(setID), Total: len(catalog)}
		for _, gear := range catalog {
			if collected[gear.ID] {
				row.Collected++
			}
		}
		if row.Total > 0 {
			row.Percent = row.Collected * 100 / row.Total
		}
		rows = append(rows, row)
		totalCollected += row.Collected
		totalItems += row.Total
	}
	return rows, totalCollected, totalItems
}

func abyssNamedSetTotal() int {
	total := 0
	for _, setID := range content.AbyssNamedSetIDs() {
		total += len(content.AbyssSetCatalog(setID))
	}
	return total
}

func (b *Bot) refreshAbyssSetBookCount(uid string) {
	collected := b.abyssCollectedSetGear(uid)
	if collected == nil {
		return
	}
	count := 0
	for _, setID := range content.AbyssNamedSetIDs() {
		for _, gear := range content.AbyssSetCatalog(setID) {
			if collected[gear.ID] {
				count++
			}
		}
	}
	_, _ = b.DB.Exec(
		`INSERT INTO app_meta (key,value) VALUES ($1,$2)
		 ON CONFLICT (key) DO UPDATE SET value=EXCLUDED.value`,
		abyssSetBookCountKey(uid), strconv.Itoa(count),
	)
}

func (b *Bot) backfillAbyssSetBook(uid string) {
	var done string
	if err := b.DB.QueryRow("SELECT value FROM app_meta WHERE key=$1", abyssSetBookBackfillKey(uid)).Scan(&done); err == nil {
		return
	}
	owned := b.abyssCollectedSetGear(uid)
	if owned == nil {
		return
	}
	for _, setID := range content.AbyssNamedSetIDs() {
		for _, gear := range content.AbyssSetCatalog(setID) {
			if owned[gear.ID] {
				_, _ = b.DB.Exec(
					"INSERT INTO app_meta (key,value) VALUES ($1,'1') ON CONFLICT (key) DO NOTHING",
					abyssSetBookPrefix(uid)+gear.ID,
				)
			}
		}
	}
	b.refreshAbyssSetBookCount(uid)
	_, _ = b.DB.Exec(
		"INSERT INTO app_meta (key,value) VALUES ($1,'1') ON CONFLICT (key) DO NOTHING",
		abyssSetBookBackfillKey(uid),
	)
}

func (b *Bot) applyAbyssSetBookBonuses(uid string, bonus *content.TreeBonus) {
	var raw string
	if err := b.DB.QueryRow("SELECT value FROM app_meta WHERE key=$1", abyssSetBookCountKey(uid)).Scan(&raw); err != nil {
		return
	}
	collected, _ := strconv.Atoi(raw)
	total := abyssNamedSetTotal()
	if total == 0 {
		return
	}
	if bonus.Pct == nil {
		bonus.Pct = make(map[string]float64)
	}
	percent := collected * 100 / total
	if percent >= 25 {
		bonus.Pct["loot_find"] += 0.02
	}
	if percent >= 50 {
		bonus.Pct["gold_find"] += 0.03
	}
	if percent >= 100 {
		bonus.Pct["material_yield"] += 0.05
	}
}

func (b *Bot) abyssCollectionStatus(uid string, loreFound int) abyssCollectionView {
	b.backfillAbyssSetBook(uid)
	view := abyssCollectionView{BiomeTotal: len(content.AbyssBiomes())}
	for _, biome := range content.AbyssBiomes() {
		clears := b.abyssBiomeClears(uid, biome.Name)
		row := abyssBiomeMasteryView{
			Name: biome.Name, Affinity: biome.Affinity, Clears: clears, Goal: abyssBiomeMasteryClears,
			Percent: min(100, clears*100/abyssBiomeMasteryClears), Mastered: clears >= abyssBiomeMasteryClears,
		}
		if row.Mastered {
			view.BiomeMastered++
		}
		view.Biomes = append(view.Biomes, row)
	}
	view.Sets, view.SetCollected, view.SetTotal = b.abyssSetBook(uid)
	if view.SetTotal > 0 {
		view.SetPercent = view.SetCollected * 100 / view.SetTotal
	}
	view.SetReward25 = view.SetPercent >= 25
	view.SetReward50 = view.SetPercent >= 50
	view.SetReward100 = view.SetPercent >= 100
	view.LoreComplete = loreFound >= len(abyssLoreFragments)
	if view.LoreComplete {
		view.LoreTitle = abyssLoreTitle
	}
	return view
}

func grantAbyssLoreCompletion(db dbOrTx, uid string) error {
	var found int
	if err := db.QueryRow("SELECT COUNT(*) FROM abyss_lore_unlocked WHERE client_uid=$1", uid).Scan(&found); err != nil {
		return err
	}
	if found < len(abyssLoreFragments) {
		return nil
	}
	if _, err := db.Exec(
		"INSERT INTO abyss_achievements (client_uid,code) VALUES ($1,$2) ON CONFLICT DO NOTHING",
		uid, abyssLoreAchievement,
	); err != nil {
		return err
	}
	_, err := db.Exec(
		"INSERT INTO app_meta (key,value) VALUES ($1,$2) ON CONFLICT (key) DO NOTHING",
		abyssLoreTitleKey(uid), abyssLoreTitle,
	)
	return err
}

func (b *Bot) abyssBadgeSuffix(uid string) string {
	var code string
	_ = b.DB.QueryRow("SELECT value FROM app_meta WHERE key=$1", abyssBadgeSuffixKey(uid)).Scan(&code)
	return code
}

func (b *Bot) setAbyssBadgeSuffix(uid, code string) error {
	if code == "" {
		_, err := b.DB.Exec("DELETE FROM app_meta WHERE key=$1", abyssBadgeSuffixKey(uid))
		return err
	}
	_, err := b.DB.Exec(
		`INSERT INTO app_meta (key,value) VALUES ($1,$2)
		 ON CONFLICT (key) DO UPDATE SET value=EXCLUDED.value`,
		abyssBadgeSuffixKey(uid), code,
	)
	return err
}

func (b *Bot) abyssActiveBadgePrefix(uid string) string {
	var code sql.NullString
	_ = b.DB.QueryRow("SELECT abyss_active_badge FROM users WHERE client_uid=$1", uid).Scan(&code)
	if code.Valid {
		return code.String
	}
	return ""
}

func abyssSortedBadgeOptions(views []abyssAchievementView) []map[string]string {
	options := make([]map[string]string, 0, len(views))
	for _, achievement := range views {
		if achievement.Earned {
			options = append(options, map[string]string{"Code": achievement.Code, "Name": achievement.Name})
		}
	}
	sort.Slice(options, func(i, j int) bool { return options[i]["Name"] < options[j]["Name"] })
	return options
}

func abyssBadgeCombination(prefix, suffix string) string {
	switch {
	case prefix != "" && suffix != "":
		return fmt.Sprintf("%s · %s", abyssAchievementName(prefix), abyssAchievementName(suffix))
	case prefix != "":
		return abyssAchievementName(prefix)
	case suffix != "":
		return abyssAchievementName(suffix)
	default:
		return "none"
	}
}
