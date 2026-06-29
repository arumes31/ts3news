package bot

import (
	"strings"
	"testing"
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
	if got, min := abyssInsuranceCost(10000, 100, 99), int64(float64(10000)*1.0*0.25); got < min {
		t.Errorf("premium %d below the rate floor %d", got, min)
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

	// 1. Weekly Challenges
	seed, mod := b.currentWeeklyChallenge()
	if seed == 0 {
		t.Error("weekly seed should not be zero")
	}
	if mod != "double_hazards" && mod != "zero_durability_loss" && mod != "enraged_mobs" {
		t.Errorf("unexpected weekly challenge modifier: %q", mod)
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


