package bot

import (
	"fmt"

	"ts3news/internal/content"
)

const (
	abyssSecretBossChainLength = 3
	abyssSecretBossAchievement = "lore_secret_chain"
)

type abyssSecretBossDefinition struct {
	Name        string
	Title       string
	Tip         string
	Element     content.Element
	HPScale     float64
	DamageScale float64
	RewardXP    int
}

var abyssSecretBosses = []abyssSecretBossDefinition{
	{Name: "The Scribe Without Eyes", Title: "Author of the First Wound", Tip: "Its unwritten sigils punish slow openings.", Element: content.ElementAir, HPScale: 1.15, DamageScale: 1.05, RewardXP: 650},
	{Name: "Mnemos, Keeper of Names", Title: "Jailer of Every Forgotten Delver", Tip: "Survive the opening verdict before spending your reserves.", Element: content.ElementWater, HPScale: 1.35, DamageScale: 1.15, RewardXP: 800},
	{Name: "The Abyss That Remembers", Title: "The Mind Beneath the Last Descent", Tip: "No affinity protects it; bring your strongest complete build.", Element: content.ElementPhysical, HPScale: 1.60, DamageScale: 1.25, RewardXP: 1_000},
}

type abyssSecretBossChainView struct {
	LoreFound int
	LoreTotal int
	Unlocked  bool
	Completed bool
	Stage     int
	NextStage int
	NextDepth int
	Boss      string
	Title     string
}

func (b *Bot) ensureAbyssSecretBossChain(uid string) (int, bool) {
	var loreFound int
	if err := b.DB.QueryRow("SELECT COUNT(*) FROM abyss_lore_unlocked WHERE client_uid=$1 AND lore_id BETWEEN 1 AND $2", uid, len(abyssLoreFragments)).Scan(&loreFound); err != nil {
		return 0, false
	}
	return b.ensureAbyssSecretBossChainWithLore(uid, loreFound)
}

func (b *Bot) ensureAbyssSecretBossChainWithLore(uid string, loreFound int) (int, bool) {
	if loreFound < len(abyssLoreFragments) {
		return 0, false
	}
	if _, err := b.DB.Exec(`INSERT INTO abyss_secret_boss_chains (client_uid,stage)
		VALUES ($1,0) ON CONFLICT (client_uid) DO NOTHING`, uid); err != nil {
		return 0, false
	}
	var stage int
	if err := b.DB.QueryRow("SELECT stage FROM abyss_secret_boss_chains WHERE client_uid=$1", uid).Scan(&stage); err != nil {
		return 0, false
	}
	return min(max(stage, 0), abyssSecretBossChainLength), true
}

func (b *Bot) abyssSecretBossChain(uid string, run abyssRun) abyssSecretBossChainView {
	var loreFound int
	_ = b.DB.QueryRow("SELECT COUNT(*) FROM abyss_lore_unlocked WHERE client_uid=$1 AND lore_id BETWEEN 1 AND $2", uid, len(abyssLoreFragments)).Scan(&loreFound)
	return b.abyssSecretBossChainWithLore(uid, run, loreFound)
}

func (b *Bot) abyssSecretBossChainWithLore(uid string, run abyssRun, loreFound int) abyssSecretBossChainView {
	view := abyssSecretBossChainView{LoreFound: loreFound, LoreTotal: len(abyssLoreFragments)}
	if view.LoreFound < view.LoreTotal {
		return view
	}
	stage, unlocked := b.ensureAbyssSecretBossChainWithLore(uid, view.LoreFound)
	view.Unlocked, view.Stage = unlocked, stage
	view.Completed = unlocked && stage >= abyssSecretBossChainLength
	if unlocked && !view.Completed {
		view.NextStage = stage + 1
		view.Boss = abyssSecretBosses[stage].Name
		view.Title = abyssSecretBosses[stage].Title
		view.NextDepth = ((max(0, run.Depth) / abyssBossEvery) + 1) * abyssBossEvery
	}
	return view
}

func (b *Bot) abyssSecretBossForFloor(uid string, depth int, naturalBoss bool) (abyssSecretBossDefinition, int, bool) {
	if !naturalBoss || depth <= 0 || depth%abyssBossEvery != 0 {
		return abyssSecretBossDefinition{}, 0, false
	}
	stage, unlocked := b.ensureAbyssSecretBossChain(uid)
	if !unlocked || stage >= abyssSecretBossChainLength {
		return abyssSecretBossDefinition{}, stage, false
	}
	return abyssSecretBosses[stage], stage, true
}

func abyssSecretBossEncounter(def abyssSecretBossDefinition, mobLevel int, difficulty float64) []content.Mob {
	lvlScale, effectiveDiff := abyssMobScalars(mobLevel, difficulty)
	return []content.Mob{{
		Name:  def.Name,
		Type:  content.MobBoss,
		Level: mobLevel + 2,
		Stats: content.Stats{
			HP:  int(1_000 * lvlScale * effectiveDiff * def.HPScale),
			STR: int(50 * lvlScale * abyssMobDamageMult * effectiveDiff * def.DamageScale),
			DEF: min(14+mobLevel/2, 92),
			SPD: 108 + def.RewardXP/200,
		},
		RewardXP: def.RewardXP,
		Element:  def.Element,
	}}
}

func (b *Bot) advanceAbyssSecretBossChain(uid string, expectedStage int) (int, bool, string) {
	if expectedStage < 0 || expectedStage >= abyssSecretBossChainLength {
		return expectedStage, false, ""
	}
	tx, err := b.DB.Begin()
	if err != nil {
		return expectedStage, false, ""
	}
	defer func() { _ = tx.Rollback() }()
	next := expectedStage + 1
	res, err := tx.Exec(`UPDATE abyss_secret_boss_chains SET stage=$1,
		completed_at=CASE WHEN $1=$2 THEN NOW() ELSE completed_at END
		WHERE client_uid=$3 AND stage=$4`, next, abyssSecretBossChainLength, uid, expectedStage)
	if err != nil {
		return expectedStage, false, ""
	}
	changed, _ := res.RowsAffected()
	if changed != 1 {
		return expectedStage, false, ""
	}
	achievement := ""
	if next == abyssSecretBossChainLength {
		res, err = tx.Exec("INSERT INTO abyss_achievements (client_uid,code) VALUES ($1,$2) ON CONFLICT DO NOTHING", uid, abyssSecretBossAchievement)
		if err != nil {
			return expectedStage, false, ""
		}
		if awarded, _ := res.RowsAffected(); awarded > 0 {
			achievement = abyssAchievementName(abyssSecretBossAchievement)
		}
	}
	if err := tx.Commit(); err != nil {
		return expectedStage, false, ""
	}
	return next, next == abyssSecretBossChainLength, achievement
}

func abyssSecretBossIntro(def abyssSecretBossDefinition, stage, depth int) []string {
	return []string{
		"[hr]",
		fmt.Sprintf("[center][size=12][color=#d7a93b]🔐 HIDDEN SOVEREIGN %d/%d — %s[/color][/size][/center]", stage+1, abyssSecretBossChainLength, def.Name),
		fmt.Sprintf("[center][color=#f0b35a][b]%s[/b][/color][/center]", def.Title),
		fmt.Sprintf("[center][color=#8a93a8][i]Depth %d · the completed codex has opened a forbidden chapter.[/i][/color][/center]", depth),
		fmt.Sprintf("[center][color=#ffd991]Scout tip: %s · %s affinity[/color][/center]", def.Tip, def.Element),
		"[hr]",
	}
}
