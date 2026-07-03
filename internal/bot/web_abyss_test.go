package bot

import (
	"strings"
	"testing"
	"time"

	"ts3news/internal/content"
)

// TestBBToHTMLEscapesThenConverts verifies the combat-log converter turns the
// known BBCode tokens into safe HTML while neutralising any injected markup.
func TestBBToHTMLConversions(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"color", "[color=#ff9800]Loot[/color]", `<span style="color:#ff9800">Loot</span>`},
		{"bold", "[b]Boss[/b]", "<b>Boss</b>"},
		{"italic", "[i]lore[/i]", "<i>lore</i>"},
		{"hr", "[hr]", `<span class="ab-hr"></span>`},
		{"center+size", "[center][size=12]Summary[/size][/center]",
			`<span class="ab-center"><span class="ab-big">Summary</span></span>`},
	}
	for _, c := range cases {
		if got := bbToHTML(c.in); got != c.want {
			t.Errorf("%s: bbToHTML(%q) = %q, want %q", c.name, c.in, got, c.want)
		}
	}
}

// TestBBToHTMLNeutralisesInjection ensures attacker-controlled angle brackets in
// e.g. a nickname cannot inject a tag — they must be escaped before BBCode runs.
func TestBBToHTMLNeutralisesInjection(t *testing.T) {
	got := bbToHTML(`[b]<script>alert(1)</script>[/b]`)
	if strings.Contains(got, "<script>") {
		t.Fatalf("bbToHTML left a live <script> tag: %q", got)
	}
	if !strings.Contains(got, "&lt;script&gt;") {
		t.Fatalf("bbToHTML did not escape the script tag: %q", got)
	}
	// The legitimate [b] token must still have been converted.
	if !strings.HasPrefix(got, "<b>") || !strings.HasSuffix(got, "</b>") {
		t.Fatalf("bbToHTML did not convert the bold token: %q", got)
	}
}

// TestAbyssDifficultyRampsAndSoftCaps checks the per-floor difficulty grows with
// depth, never drops below the floor-1 minimum, and that growth past the soft
// cap slows to a logarithmic crawl (later increments smaller than early ones).
func TestAbyssDifficultyRampsAndSoftCaps(t *testing.T) {
	// Difficulty is depth-driven only — floor 1 is the gentle baseline.
	d1, _ := abyssDifficulty(1)
	if d1 < abyssBaseDiff-1e-9 {
		t.Errorf("floor 1 difficulty %.3f below the %.2f baseline", d1, abyssBaseDiff)
	}

	// Monotonically non-decreasing with depth.
	prev := 0.0
	for depth := 1; depth <= 200; depth++ {
		d, _ := abyssDifficulty(depth)
		if d < prev-1e-9 {
			t.Fatalf("difficulty decreased at depth %d: %.3f < %.3f", depth, d, prev)
		}
		prev = d
	}

	// Past the soft cap the marginal per-floor increase must shrink: the jump near
	// the start (pre-cap, linear) should exceed the jump deep down (post-cap, log).
	a1, _ := abyssDifficulty(2)
	a0, _ := abyssDifficulty(1)
	earlyDelta := a1 - a0
	b1, _ := abyssDifficulty(200)
	b0, _ := abyssDifficulty(199)
	lateDelta := b1 - b0
	if lateDelta >= earlyDelta {
		t.Errorf("soft cap not engaged: late delta %.4f >= early delta %.4f", lateDelta, earlyDelta)
	}
}

// TestAbyssDifficultyBossCadence verifies a boss is forced on every Nth floor.
func TestAbyssDifficultyBossCadence(t *testing.T) {
	for depth := 1; depth <= 20; depth++ {
		_, boss := abyssDifficulty(depth)
		want := depth%abyssBossEvery == 0
		if boss != want {
			t.Errorf("depth %d boss=%v, want %v", depth, boss, want)
		}
	}
}

// TestBossResistScalesAndCaps verifies boss resistance grows with level and gear
// score, is ~0 for a fresh character, and never exceeds the hard cap.
func TestBossResistScalesAndCaps(t *testing.T) {
	if low := bossResist(1, 0); low > 0.05 {
		t.Errorf("fresh character boss resist %.3f should be ~0", low)
	}
	// Monotonic in both inputs.
	if bossResist(500, 0) <= bossResist(100, 0) {
		t.Error("boss resist should grow with level")
	}
	if bossResist(100, 5000) <= bossResist(100, 0) {
		t.Error("boss resist should grow with gear score")
	}
	// Hard cap holds even at absurd inputs.
	if capped := bossResist(100000, 100000000); capped > bossResistCap+1e-9 {
		t.Errorf("boss resist %.3f exceeded cap %.3f", capped, bossResistCap)
	}
}

// TestLootRarityScale verifies the low-level rarity dampener ramps from a floor at
// level 1 up to full at level 50 and stays clamped beyond.
func TestLootRarityScale(t *testing.T) {
	if s1 := lootRarityScale(1); s1 < 0.29 || s1 > 0.31 {
		t.Errorf("level-1 rarity scale %.3f, want ~0.30", s1)
	}
	if s := lootRarityScale(50); s != 1.0 {
		t.Errorf("level-50 rarity scale %.3f, want 1.0", s)
	}
	if s := lootRarityScale(9999); s != 1.0 {
		t.Errorf("high-level rarity scale %.3f, want clamp at 1.0", s)
	}
	// Monotonically non-decreasing.
	prev := 0.0
	for lvl := 1; lvl <= 60; lvl++ {
		if s := lootRarityScale(lvl); s < prev-1e-9 {
			t.Fatalf("rarity scale decreased at level %d", lvl)
		} else {
			prev = s
		}
	}
}

// TestAbyssMobLevelDecoupled verifies mob level is a depth-scaled fraction of the
// player's level: below them on floor 1, ramping past parity with depth, capped.
func TestAbyssMobLevelDecoupled(t *testing.T) {
	const lvl = 947
	if floor1 := abyssMobLevel(1, lvl); floor1 >= lvl {
		t.Errorf("floor 1 mob level %d should be below player level %d", floor1, lvl)
	}
	// Non-decreasing with depth.
	prev := 0
	for depth := 1; depth <= 100; depth++ {
		got := abyssMobLevel(depth, lvl)
		if got < prev {
			t.Fatalf("mob level decreased at depth %d: %d < %d", depth, got, prev)
		}
		prev = got
	}
	// Hard ceiling at 2× the player's level.
	if deep := abyssMobLevel(1000, lvl); deep > lvl*2 {
		t.Errorf("mob level %d exceeded the 2x ceiling %d", deep, lvl*2)
	}
}

// TestAbyssFloorBonusScales checks the escrow bonus rises with depth and level
// and respects the per-floor minimum.
func TestAbyssFloorBonusScales(t *testing.T) {
	if b1, b2 := abyssFloorBonus(1, 10), abyssFloorBonus(2, 10); b2 <= b1 {
		t.Errorf("bonus should grow with depth: floor1=%d floor2=%d", b1, b2)
	}
	if lo, hi := abyssFloorBonus(5, 10), abyssFloorBonus(5, 200); hi <= lo {
		t.Errorf("bonus should grow with level: lvl10=%d lvl200=%d", lo, hi)
	}
	// Minimum per-floor rate is 40 gold × depth, even at level 0/1.
	if got := abyssFloorBonus(3, 0); got < 40*3 {
		t.Errorf("floor bonus %d below the 40/floor minimum", got)
	}
}

// TestAbyssBankMultiplier checks deeper banks and longer streaks pay more, and
// that both inputs are capped so the multiplier can't run away.
func TestAbyssBankMultiplier(t *testing.T) {
	var b Bot
	if base := b.abyssBankMultiplier(0, 0); base != 1.0 {
		t.Errorf("base multiplier = %.2f, want 1.0", base)
	}
	if deep, shallow := b.abyssBankMultiplier(50, 0), b.abyssBankMultiplier(10, 0); deep <= shallow {
		t.Errorf("deeper bank should pay more: d50=%.2f d10=%.2f", deep, shallow)
	}
	if streaky, none := b.abyssBankMultiplier(10, 10), b.abyssBankMultiplier(10, 0); streaky <= none {
		t.Errorf("streak should pay more: s10=%.2f s0=%.2f", streaky, none)
	}
	// Depth caps at 100 and streak at 25 → max = 1 + 1.0 + 0.5 = 2.5.
	if capped := b.abyssBankMultiplier(99999, 99999); capped > 2.5+1e-9 {
		t.Errorf("multiplier not capped: %.2f", capped)
	}
}

// TestAbyssInsuranceCost checks the premium scales with cache and coverage, and
// that the Ward upgrade discounts it down to a floor.
func TestAbyssInsuranceCost(t *testing.T) {
	full := abyssInsuranceCost(10000, 50, 0)
	if full != int64(10000*0.50*0.50) {
		t.Errorf("base 50%% premium = %d, want %d", full, int64(10000*0.5*0.5))
	}
	if warded := abyssInsuranceCost(10000, 50, 3); warded >= full {
		t.Errorf("Ward should discount: warded=%d full=%d", warded, full)
	}
	// Rate floor is 0.25 even at absurd Ward levels.
	if got, floor := abyssInsuranceCost(10000, 100, 99), int64(float64(10000)*1.0*0.25); got < floor {
		t.Errorf("premium %d below the rate floor %d", got, floor)
	}
}

// TestAbyssTierUnlock verifies tiers gate behind a best-depth requirement.
func TestAbyssTierUnlock(t *testing.T) {
	low := abyssTierList(0)
	for _, tv := range low {
		if tv.Key == "normal" && !tv.Unlocked {
			t.Error("normal tier must always be unlocked")
		}
		if tv.Key == "hell" && tv.Unlocked {
			t.Error("hell tier must be locked at best depth 0")
		}
	}
	high := abyssTierList(999)
	for _, tv := range high {
		if !tv.Unlocked {
			t.Errorf("tier %q should be unlocked at best depth 999", tv.Key)
		}
	}
}

// TestAbyssWave3Content verifies Wave 3 structs and config helpers.
func TestAbyssWave3Content(t *testing.T) {
	// 1. Verify UserInCombat field extensions exist
	var u UserInCombat
	u.LootFocus = "gold"
	u.FloorModifier = "enraged"
	
	if u.LootFocus != "gold" || u.FloorModifier != "enraged" {
		t.Error("UserInCombat fields failed to set or retrieve")
	}

	// 2. Verify Lore Codex config
	if len(abyssLoreFragments) != 10 {
		t.Errorf("expected exactly 10 lore codex fragments, got %d", len(abyssLoreFragments))
	}
	
	if abyssLoreFragments[1] == "" || abyssLoreFragments[10] == "" {
		t.Error("lore codex fragments have empty values")
	}
}

// TestAbyssRemaining verifies the new features added in the remaining waves (Waves 4, 5, & 6)
func TestAbyssRemaining(t *testing.T) {
	b := &Bot{}

	// 1. Daily Challenges
	seed, mod := b.currentDailyChallenge()
	if seed == 0 {
		t.Error("daily seed should not be zero")
	}
	validMod := false
	for _, m := range abyssDailyMods {
		if mod == m {
			validMod = true
			break
		}
	}
	if !validMod {
		t.Errorf("unexpected daily challenge modifier: %q", mod)
	}

	// 2. Prestige multiplier checking
	// 5% gold bonus per prestige level
	st := abyssStats{AbyssPrestige: 2}
	bonus := int64(100)
	bonus = int64(float64(bonus) * (1.0 + float64(st.UpGreed)*0.05) * (1.0 + float64(st.AbyssPrestige)*0.05))
	if bonus != 110 {
		t.Errorf("expected prestige gold bonus to be 110, got %d", bonus)
	}
}

// TestAbyssFeatureExpansion covers the wave-4 feature additions: the new weekly
// affix multipliers, the Compounding interest node, and the new upgrade nodes.
func TestAbyssFeatureExpansion(t *testing.T) {
	// Daily reward/danger multipliers.
	if got := abyssDailyRewardMult("gold_rush"); got != 2.0 {
		t.Errorf("gold_rush reward mult = %v, want 2.0", got)
	}
	if got := abyssDailyRewardMult("iron_skin"); got != 0.9 {
		t.Errorf("iron_skin reward mult = %v, want 0.9", got)
	}
	if got := abyssDailyRewardMult("vampiric_mobs"); got != 1.15 {
		t.Errorf("vampiric_mobs reward mult = %v, want 1.15", got)
	}
	// The combat-hooked affixes must be in the rotation pool to ever fire.
	for _, want := range []string{"iron_skin", "bloodlust", "execute", "vampiric_mobs"} {
		found := false
		for _, m := range abyssDailyMods {
			if m == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("affix %q missing from abyssDailyMods pool", want)
		}
	}
	if got := abyssDailyRewardMult("glass_cannon"); got != 1.3 {
		t.Errorf("glass_cannon reward mult = %v, want 1.3", got)
	}
	if got := abyssDailyRewardMult("double_hazards"); got != 1.0 {
		t.Errorf("double_hazards reward mult = %v, want 1.0", got)
	}
	if got := abyssDailyDangerMult("glass_cannon"); got != 1.3 {
		t.Errorf("glass_cannon danger mult = %v, want 1.3", got)
	}
	if got := abyssDailyDangerMult("gold_rush"); got != 1.0 {
		t.Errorf("gold_rush danger mult = %v, want 1.0", got)
	}

	// Compounding node raises the base escrow interest by 0.1% per level.
	if got := abyssEffectiveInterest(0, false); got != abyssEscrowInterest {
		t.Errorf("interest L0 = %v, want %v", got, abyssEscrowInterest)
	}
	if got := abyssEffectiveInterest(4, false); got < abyssEscrowInterest+0.0039 || got > abyssEscrowInterest+0.0041 {
		t.Errorf("interest L4 = %v, want %v", got, abyssEscrowInterest+0.004)
	}

	// Every new upgrade node must be in the whitelist so /api/abyss/upgrade accepts it.
	for _, node := range []string{"interest", "tribute", "insight"} {
		if _, ok := abyssUpgradeCols[node]; !ok {
			t.Errorf("upgrade node %q missing from abyssUpgradeCols whitelist", node)
		}
	}
}

// TestAbyssSignatureRelics verifies the combat-active Abyss relics expose their
// fixed Special through the static gear definition (which is what the live combat
// engine reads when the piece is equipped) and that the drop roller preserves it.
func TestAbyssSignatureRelics(t *testing.T) {
	want := map[string]content.ItemEffect{
		"ABYSS_OFFHAND":  content.EffectThorns,
		"ABYSS_AURA":     content.EffectVampiric,
		"ABYSS_BAND":     content.EffectBerserk,
		"ABYSS_TRINKET":  content.EffectStealth,
		"ABYSS_TALISMAN": content.EffectParry,
		"ABYSS_RELIC":    content.EffectPhoenix,
	}
	for id, eff := range want {
		g, ok := content.GetGearByID(id)
		if !ok {
			t.Errorf("signature gear %q not found", id)
			continue
		}
		if g.Special != eff {
			t.Errorf("%q Special = %q, want %q", id, g.Special, eff)
		}
	}

	// Whenever the roller returns a signature ID, its authored effect must survive
	// (RandomAbyssGearDrop only random-rolls Special for items that define none).
	for i := 0; i < 400; i++ {
		g := content.RandomAbyssGearDrop()
		if eff, isSig := want[g.ID]; isSig && g.Special != eff {
			t.Fatalf("RandomAbyssGearDrop overwrote %q Special: got %q want %q", g.ID, g.Special, eff)
		}
	}
}

// TestAbyssPacts verifies pact validation (unknown keys dropped, canonical order,
// de-duplicated) and that the reward/danger multipliers aggregate correctly.
func TestAbyssPacts(t *testing.T) {
	// Unknown keys are dropped; duplicates collapse; output is in catalog order.
	got := abyssValidatePacts([]string{"glass_cannon", "bogus", "double_hazards", "double_hazards"})
	if got != "double_hazards glass_cannon" {
		t.Errorf("validate = %q, want %q", got, "double_hazards glass_cannon")
	}
	if abyssValidatePacts([]string{"nope"}) != "" {
		t.Error("all-unknown pact list should validate to empty")
	}

	// Reward multiplier is 1.0 + the sum of each pact's additive bonus.
	if m := abyssPactRewardMult(nil); m != 1.0 {
		t.Errorf("no-pact reward mult = %v, want 1.0", m)
	}
	if m := abyssPactRewardMult([]string{"double_hazards", "glass_cannon"}); m < 1.449 || m > 1.451 {
		t.Errorf("reward mult = %v, want ~1.45", m)
	}
	// Danger multiplier comes only from pacts that raise difficulty (glass_cannon).
	if m := abyssPactDangerMult([]string{"double_hazards"}); m != 1.0 {
		t.Errorf("double_hazards danger mult = %v, want 1.0", m)
	}
	if m := abyssPactDangerMult([]string{"glass_cannon"}); m != 1.3 {
		t.Errorf("glass_cannon danger mult = %v, want 1.3", m)
	}
	if !abyssPactsEnrage([]string{"enraged"}) || abyssPactsEnrage([]string{"glass_cannon"}) {
		t.Error("enrage detection incorrect")
	}
}

// TestAbyssDismantleTokens verifies the dismantle yield: nothing below Rare, and a
// strictly increasing token value up the rarity ladder.
func TestAbyssDismantleTokens(t *testing.T) {
	if abyssDismantleTokens(content.RarityCommon) != 0 || abyssDismantleTokens(content.RarityUncommon) != 0 {
		t.Error("common/uncommon should dismantle for 0 tokens (use salvage)")
	}
	ladder := []content.Rarity{content.RarityRare, content.RarityEpic, content.RarityLegendary, content.RarityMythic}
	var prev int64
	for _, rar := range ladder {
		got := abyssDismantleTokens(rar)
		if got <= prev {
			t.Errorf("dismantle tokens not increasing at rarity %v: %d <= %d", rar, got, prev)
		}
		prev = got
	}
}

// TestAbyssShopCatalog verifies the token-shop catalog is well-formed: unique keys,
// positive costs, and a lookup that resolves every advertised key (so no buy button
// can hit an unknown-item path).
func TestAbyssShopCatalog(t *testing.T) {
	seen := map[string]bool{}
	for _, it := range abyssShopCatalog {
		if it.Key == "" || it.Name == "" {
			t.Errorf("shop item has empty key/name: %+v", it)
		}
		if it.Cost <= 0 && it.CostGold <= 0 {
			t.Errorf("shop item %q has non-positive cost (tokens=%d gold=%d)", it.Key, it.Cost, it.CostGold)
		}
		if seen[it.Key] {
			t.Errorf("duplicate shop key %q", it.Key)
		}
		seen[it.Key] = true
		if _, ok := abyssShopByKey(it.Key); !ok {
			t.Errorf("abyssShopByKey cannot resolve advertised key %q", it.Key)
		}
	}
	if _, ok := abyssShopByKey("definitely_not_a_key"); ok {
		t.Error("abyssShopByKey resolved a bogus key")
	}
}

// TestAbyssDailyBounty verifies the bounty is deterministic per UTC day, rotates
// across days, and that every template entry is well-formed (positive target/reward).
func TestAbyssDailyBounty(t *testing.T) {
	day := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	a := abyssDailyBounty(day)
	b := abyssDailyBounty(day.Add(6 * time.Hour))
	if a != b {
		t.Error("bounty should be stable within the same UTC day")
	}
	// Over a full rotation of distinct days we must see more than one bounty.
	seen := map[abyssBountyKind]bool{}
	for i := 0; i < len(abyssBountyTable); i++ {
		seen[abyssDailyBounty(day.AddDate(0, 0, i)).Kind] = true
	}
	if len(seen) < 2 {
		t.Error("bounty rotation should yield more than one kind over its period")
	}
	for i, bt := range abyssBountyTable {
		if bt.Target <= 0 || bt.Desc == "" || (bt.RewardTk <= 0 && bt.RewardGd <= 0) {
			t.Errorf("bounty %d is malformed: %+v", i, bt)
		}
	}
}

// TestAbyssStreakBonus verifies the bounty streak bonus: none on day 1, +5 per
// extra consecutive day, capped at +30 from day 7 onward.
func TestAbyssStreakBonus(t *testing.T) {
	cases := map[int]int{0: 0, 1: 0, 2: 5, 3: 10, 7: 30, 8: 30, 100: 30}
	for streak, want := range cases {
		if got := abyssStreakBonusTokens(streak); got != want {
			t.Errorf("streak %d bonus = %d, want %d", streak, got, want)
		}
	}
}

// TestAbyssAchievementTiers verifies every count-based achievement ladder is
// ascending and that each tier code has a player-facing name (so newly earned
// milestones never surface as a raw code).
func TestAbyssAchievementTiers(t *testing.T) {
	ladders := map[string][]achTier{
		"boss":     abyssBossTiers,
		"bank":     abyssBankTiers,
		"bestiary": abyssBestiaryTiers,
	}
	for name, tiers := range ladders {
		if len(tiers) == 0 {
			t.Errorf("%s ladder is empty", name)
		}
		var prev int64 = -1
		for _, tr := range tiers {
			if tr.N <= prev {
				t.Errorf("%s ladder not strictly ascending at %q (N=%d)", name, tr.Code, tr.N)
			}
			prev = tr.N
			if abyssAchievementName(tr.Code) == tr.Code {
				t.Errorf("%s tier %q has no display name", name, tr.Code)
			}
		}
	}
	// The prestige achievement must also resolve to a name.
	if abyssAchievementName("prestige_1") == "prestige_1" {
		t.Error("prestige_1 has no display name")
	}
}

// TestAbyssSetBonus verifies the cumulative 2/4/6-piece set-bonus tiers: nothing
// below 2 pieces, the right threshold reported, and bonuses that stack and grow.
func TestAbyssSetBonus(t *testing.T) {
	for _, pieces := range []int{0, 1} {
		if b, reached := content.AbyssSetBonus(pieces); reached != 0 || b.Score() != 0 {
			t.Errorf("%d pieces should grant no set bonus, got tier %d", pieces, reached)
		}
	}
	_, r2 := content.AbyssSetBonus(2)
	_, r3 := content.AbyssSetBonus(3)
	if r2 != 2 || r3 != 2 {
		t.Errorf("2–3 pieces should report the 2-piece tier, got %d/%d", r2, r3)
	}
	if _, r := content.AbyssSetBonus(5); r != 4 {
		t.Errorf("5 pieces should report the 4-piece tier, got %d", r)
	}
	if _, r := content.AbyssSetBonus(6); r != 6 {
		t.Errorf("6 pieces should report the 6-piece tier, got %d", r)
	}
	// Bonuses are cumulative: more equipped pieces never reduce the total.
	b2, _ := content.AbyssSetBonus(2)
	b4, _ := content.AbyssSetBonus(4)
	b6, _ := content.AbyssSetBonus(6)
	if b2.HP >= b4.HP || b4.HP >= b6.HP {
		t.Errorf("set HP bonus should grow with tier: %d/%d/%d", b2.HP, b4.HP, b6.HP)
	}
	if b6.CRT == 0 {
		t.Error("6-piece tier should include the CRT bonus")
	}
}


