package bot

import (
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"

	"ts3news/internal/content"
)

const (
	abyssPetTrainingCost = int64(1_000)
	abyssPetTrainingCap  = 3
	abyssDuoUnlocksAt    = 5
	abyssDuoBonusPct     = 2
	abyssRivalReward     = 15
	abyssWeeklyBossHP    = int64(100_000_000)
)

type abyssSocialPetView struct {
	ID          int64
	Name        string
	Type        string
	Level       int
	HP          int
	MaxHP       int
	STR         int
	DEF         int
	SPD         int
	Loyalty     int
	ActiveSlot  int
	Mood        string
	MoodIcon    string
	MoodPct     int
	Training    int
	HealEnabled bool
	Equipment   string
}

type abyssDeathView struct {
	Depth        int
	KillerName   string
	KillerFamily string
	When         string
}

type abyssMemorialView struct {
	Name    string
	Type    string
	Level   int
	Loyalty int
	When    string
}

type abyssTrophyView struct {
	Boss string
	Date string
}

type abyssBankFeedView struct {
	Nick  string
	Depth int
	Gold  int64
	When  string
}

type abyssRivalView struct {
	Available   bool
	Nick        string
	TargetDepth int
	NextDepth   int
	Current     int
	Passed      bool
	Claimed     bool
}

type abyssWeeklyBossView struct {
	Week         string
	Name         string
	HP           int64
	MaxHP        int64
	Percent      int
	Contributed  bool
	Damage       int64
	Contributors int
	Defeated     bool
}

type abyssNotificationView struct {
	Message string
	When    string
}

type abyssSocialHubView struct {
	Pets              []abyssSocialPetView
	SecondPetUnlocked bool
	Deaths            []abyssDeathView
	Memorials         []abyssMemorialView
	Trophies          []abyssTrophyView
	RevengeFamily     string
	Rival             abyssRivalView
	BankFeedEnabled   bool
	BankFeed          []abyssBankFeedView
	WeeklyBoss        abyssWeeklyBossView
	Notifications     []abyssNotificationView
}

func abyssPetMood(currentHP, maxHP, loyalty int) (string, string, int) {
	switch {
	case maxHP > 0 && currentHP*4 <= maxHP:
		return "scared", "😨", -2
	case loyalty < 50:
		return "hungry", "🍖", -2
	default:
		return "content", "😊", 2
	}
}

func abyssPetMoodScale(value, pct int) int {
	if value <= 0 {
		return value
	}
	return max(1, value*(100+pct)/100)
}

func abyssPetLowestStat(strength, defense, speed int) (string, int) {
	stats := []struct {
		name  string
		value int
	}{{"str", strength}, {"def", defense}, {"spd", speed}}
	sort.SliceStable(stats, func(i, j int) bool { return stats[i].value < stats[j].value })
	return stats[0].name, stats[0].value
}

func abyssPair(uid, other string) (string, string, bool) {
	if uid == "" || other == "" || uid == other {
		return "", "", false
	}
	if uid < other {
		return uid, other, true
	}
	return other, uid, true
}

func (b *Bot) abyssDuoAssists(uid, other string) int {
	low, high, ok := abyssPair(uid, other)
	if !ok {
		return 0
	}
	var assists int
	_ = b.DB.QueryRow("SELECT assists FROM abyss_helper_bonds WHERE uid_low=$1 AND uid_high=$2", low, high).Scan(&assists)
	return assists
}

func (b *Bot) recordAbyssDuoAssist(uid, other string) int {
	low, high, ok := abyssPair(uid, other)
	if !ok {
		return 0
	}
	var assists int
	_ = b.DB.QueryRow(`INSERT INTO abyss_helper_bonds (uid_low,uid_high,assists) VALUES ($1,$2,1)
		ON CONFLICT (uid_low,uid_high) DO UPDATE SET assists=abyss_helper_bonds.assists+1,updated_at=NOW()
		RETURNING assists`, low, high).Scan(&assists)
	return assists
}

func applyAbyssDuoBonus(users []UserInCombat, assists int) bool {
	if len(users) < 2 || assists < abyssDuoUnlocksAt {
		return false
	}
	for i := range users {
		users[i].Stats.HP = users[i].Stats.HP * (100 + abyssDuoBonusPct) / 100
		users[i].Stats.STR = users[i].Stats.STR * (100 + abyssDuoBonusPct) / 100
		users[i].Stats.DEF = users[i].Stats.DEF * (100 + abyssDuoBonusPct) / 100
		users[i].CurrentHP = min(users[i].CurrentHP*(100+abyssDuoBonusPct)/100, users[i].Stats.HP)
	}
	return true
}

func abyssPetEquipmentLabel(gear content.Gear) string {
	if gear.Name == "" {
		return "No collar/charm equipped"
	}
	parts := make([]string, 0, 3)
	if gear.Stats.STR != 0 {
		parts = append(parts, fmt.Sprintf("%+d STR", gear.Stats.STR))
	}
	if gear.Stats.DEF != 0 {
		parts = append(parts, fmt.Sprintf("%+d DEF", gear.Stats.DEF))
	}
	if gear.Stats.SPD != 0 {
		parts = append(parts, fmt.Sprintf("%+d SPD", gear.Stats.SPD))
	}
	if len(parts) == 0 {
		parts = append(parts, "utility effect")
	}
	return gear.Name + " · " + strings.Join(parts, " · ")
}

func (b *Bot) abyssSocialPets(uid string) []abyssSocialPetView {
	rows, err := b.DB.Query(`SELECT pet_id,name,mob_type,level,hp,max_hp,str,def,spd,loyalty,active_slot,
		CASE WHEN trained_on=CURRENT_DATE THEN training_count ELSE 0 END,
		COALESCE((autoskills->>'heal')::boolean,TRUE)
		FROM user_pets WHERE client_uid=$1 ORDER BY active_slot DESC,captured_at,pet_id`, uid)
	if err != nil {
		return nil
	}
	defer rows.Close()
	equipped := b.getEquippedItems(uid)
	views := make([]abyssSocialPetView, 0)
	for rows.Next() {
		var view abyssSocialPetView
		if rows.Scan(&view.ID, &view.Name, &view.Type, &view.Level, &view.HP, &view.MaxHP,
			&view.STR, &view.DEF, &view.SPD, &view.Loyalty, &view.ActiveSlot, &view.Training, &view.HealEnabled) != nil {
			return nil
		}
		view.Mood, view.MoodIcon, view.MoodPct = abyssPetMood(view.HP, view.MaxHP, view.Loyalty)
		slot := content.SlotPet1
		if view.ActiveSlot == 2 {
			slot = content.SlotPet2
		}
		view.Equipment = abyssPetEquipmentLabel(equipped[slot])
		views = append(views, view)
	}
	if rows.Err() != nil {
		return nil
	}
	return views
}

func (b *Bot) abyssDeathWall(uid string) []abyssDeathView {
	rows, err := b.DB.Query(`SELECT depth,killer_name,killer_family,died_at
		FROM abyss_deaths WHERE client_uid=$1 ORDER BY died_at DESC LIMIT 10`, uid)
	if err != nil {
		return nil
	}
	defer rows.Close()
	deaths := make([]abyssDeathView, 0, 10)
	for rows.Next() {
		var death abyssDeathView
		var at time.Time
		if rows.Scan(&death.Depth, &death.KillerName, &death.KillerFamily, &at) != nil {
			return nil
		}
		death.When = at.Format("Jan 2 15:04")
		deaths = append(deaths, death)
	}
	return deaths
}

func (b *Bot) abyssRevengeFamily(uid string) string {
	var family string
	_ = b.DB.QueryRow(`SELECT killer_family FROM abyss_deaths WHERE client_uid=$1
		GROUP BY killer_family ORDER BY COUNT(*) DESC,killer_family LIMIT 1`, uid).Scan(&family)
	return family
}

func (b *Bot) abyssPetMemorials(uid string) []abyssMemorialView {
	rows, err := b.DB.Query(`SELECT name,mob_type,level,loyalty,fallen_at FROM abyss_pet_memorials
		WHERE client_uid=$1 ORDER BY fallen_at DESC LIMIT 20`, uid)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var views []abyssMemorialView
	for rows.Next() {
		var view abyssMemorialView
		var at time.Time
		if rows.Scan(&view.Name, &view.Type, &view.Level, &view.Loyalty, &at) != nil {
			return nil
		}
		view.When = at.Format("2006-01-02")
		views = append(views, view)
	}
	return views
}

func (b *Bot) abyssBossTrophies(uid string) []abyssTrophyView {
	rows, err := b.DB.Query(`SELECT boss_name,MIN(killed_at) FROM abyss_boss_kills
		WHERE client_uid=$1 GROUP BY boss_name ORDER BY MIN(killed_at) DESC`, uid)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var views []abyssTrophyView
	for rows.Next() {
		var view abyssTrophyView
		var at time.Time
		if rows.Scan(&view.Boss, &at) != nil {
			return nil
		}
		view.Date = at.Format("2006-01-02")
		views = append(views, view)
	}
	return views
}

func (b *Bot) ensureAbyssWeeklyRival(uid string) abyssRivalView {
	week := abyssCurrentWeek(time.Now())
	var current int
	if b.DB.QueryRow("SELECT abyss_best_depth FROM users WHERE client_uid=$1", uid).Scan(&current) != nil {
		return abyssRivalView{}
	}
	var rivalUID string
	var target int
	if b.DB.QueryRow(`SELECT client_uid,abyss_best_depth FROM users
		WHERE client_uid<>$1 AND abyss_best_depth>$2 ORDER BY abyss_best_depth,client_uid LIMIT 1`, uid, current).Scan(&rivalUID, &target) == nil {
		_, _ = b.DB.Exec(`INSERT INTO abyss_weekly_rivals (week_key,client_uid,rival_uid,target_depth)
			VALUES ($1,$2,$3,$4) ON CONFLICT (week_key,client_uid) DO NOTHING`, week, uid, rivalUID, target)
	}
	var nick string
	var claimed sql.NullTime
	err := b.DB.QueryRow(`SELECT COALESCE(NULLIF(u.nickname,''),'Adventurer'),r.target_depth,r.claimed_at
		FROM abyss_weekly_rivals r JOIN users u ON u.client_uid=r.rival_uid
		WHERE r.week_key=$1 AND r.client_uid=$2`, week, uid).Scan(&nick, &target, &claimed)
	if err != nil {
		return abyssRivalView{}
	}
	return abyssRivalView{Available: true, Nick: nick, TargetDepth: target, NextDepth: target + 1, Current: current,
		Passed: current > target, Claimed: claimed.Valid}
}

func abyssWeeklyBossDefinition(now time.Time) (string, string) {
	week := abyssCurrentWeek(now)
	names := []string{"Nhal, the Starved Horizon", "Veyra of the Thousand Eyes", "The Iron Leviathan", "Mournroot Prime"}
	sum := 0
	for _, char := range week {
		sum += int(char)
	}
	return week, names[sum%len(names)]
}

func (b *Bot) abyssWeeklyBossStatus(uid string) abyssWeeklyBossView {
	week, name := abyssWeeklyBossDefinition(time.Now())
	view := abyssWeeklyBossView{Week: week, Name: name, HP: abyssWeeklyBossHP, MaxHP: abyssWeeklyBossHP, Percent: 100}
	var defeated sql.NullTime
	_ = b.DB.QueryRow(`SELECT boss_name,current_hp,max_hp,defeated_at FROM abyss_weekly_bosses WHERE week_key=$1`, week).
		Scan(&view.Name, &view.HP, &view.MaxHP, &defeated)
	_ = b.DB.QueryRow(`SELECT COUNT(DISTINCT client_uid),COALESCE(SUM(damage) FILTER (WHERE client_uid=$2),0),
		EXISTS(SELECT 1 FROM abyss_weekly_boss_contributions WHERE week_key=$1 AND client_uid=$2 AND contribution_date=CURRENT_DATE)
		FROM abyss_weekly_boss_contributions WHERE week_key=$1`, week, uid).
		Scan(&view.Contributors, &view.Damage, &view.Contributed)
	view.Defeated = defeated.Valid || view.HP == 0
	if view.MaxHP > 0 {
		view.Percent = int(view.HP * 100 / view.MaxHP)
	}
	return view
}

func (b *Bot) abyssBankFeed(uid string) (bool, []abyssBankFeedView) {
	var enabled bool
	_ = b.DB.QueryRow("SELECT bank_feed_opt_in FROM abyss_social_profiles WHERE client_uid=$1", uid).Scan(&enabled)
	if !enabled {
		return false, nil
	}
	rows, err := b.DB.Query(`SELECT COALESCE(NULLIF(u.nickname,''),'Adventurer'),r.depth,r.gold_banked,r.created_at
		FROM abyss_runs r JOIN users u ON u.client_uid=r.client_uid
		JOIN abyss_social_profiles publisher ON publisher.client_uid=r.client_uid AND publisher.bank_feed_opt_in=TRUE
		WHERE r.victory=TRUE AND r.client_uid<>$1
		  AND ABS(r.depth-(SELECT abyss_best_depth FROM users WHERE client_uid=$1))<=5
		ORDER BY r.created_at DESC LIMIT 8`, uid)
	if err != nil {
		return true, nil
	}
	defer rows.Close()
	var feed []abyssBankFeedView
	for rows.Next() {
		var item abyssBankFeedView
		var at time.Time
		if rows.Scan(&item.Nick, &item.Depth, &item.Gold, &at) != nil {
			return true, nil
		}
		item.When = at.Format("15:04")
		feed = append(feed, item)
	}
	return true, feed
}

func (b *Bot) abyssSocialNotifications(uid string) []abyssNotificationView {
	rows, err := b.DB.Query(`SELECT message,created_at FROM abyss_social_notifications
		WHERE client_uid=$1 ORDER BY created_at DESC LIMIT 5`, uid)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var views []abyssNotificationView
	for rows.Next() {
		var view abyssNotificationView
		var at time.Time
		if rows.Scan(&view.Message, &at) != nil {
			return nil
		}
		view.When = at.Format("Jan 2 15:04")
		views = append(views, view)
	}
	return views
}

func (b *Bot) abyssSocialHub(uid string, prestige int) abyssSocialHubView {
	deaths := b.abyssDeathWall(uid)
	bankEnabled, bankFeed := b.abyssBankFeed(uid)
	return abyssSocialHubView{
		Pets: b.abyssSocialPets(uid), SecondPetUnlocked: prestige >= 2,
		Deaths: deaths, Memorials: b.abyssPetMemorials(uid), Trophies: b.abyssBossTrophies(uid),
		RevengeFamily: b.abyssRevengeFamily(uid), Rival: b.ensureAbyssWeeklyRival(uid), BankFeedEnabled: bankEnabled,
		BankFeed: bankFeed, WeeklyBoss: b.abyssWeeklyBossStatus(uid), Notifications: b.abyssSocialNotifications(uid),
	}
}
