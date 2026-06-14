package bot

import (
	"math/rand/v2"
	"strconv"
)

// This file adds 20 one-shot arcade games on top of the original five. Every
// game is tuned to a ~5% house edge (return-to-player ≈ 0.95) and is covered by
// TestArcadeBalance, which enforces 0.90 ≤ RTP < 1.00 for each game and each
// selectable choice so no option is ever player-favoured.
//
// Each game is a func(rng, bet, choice) (payout, detail): payout is the gross
// amount returned to the player (0 = lost the bet), detail is the result text.

// arcadeGame is the signature shared by all extra games.
type arcadeGame func(rng *rand.Rand, bet int64, choice string) (int64, string)

// extraGames maps a game key to its implementation. handleArcadeAPI/playArcade
// dispatch through this table for anything not handled by the five originals.
var extraGames = map[string]arcadeGame{
	"roulette":   playRoulette,
	"rps":        playRPS,
	"war":        playWar,
	"plinko":     playPlinko,
	"scratch":    playScratch,
	"sicbo":      playSicbo,
	"lucky":      playLucky,
	"gems":       playGems,
	"crash":      playCrash,
	"megawheel":  playMegaWheel,
	"diceduel":   playDiceDuel,
	"chests":     playChests,
	"horserace":  playHorseRace,
	"colorpick":  playColorPick,
	"fortune":    playFortune,
	"lightning":  playLightning,
	"minefield":  playMinefield,
	"darts":      playDarts,
	"slotdeluxe": playSlotDeluxe,
	"keno":       playKeno,
}

func mulBet(bet int64, m float64) int64 { return int64(float64(bet) * m) }

// ---- 1. Roulette (European, single zero) ---------------------------------

var rouletteReds = map[int]bool{
	1: true, 3: true, 5: true, 7: true, 9: true, 12: true, 14: true, 16: true,
	18: true, 19: true, 21: true, 23: true, 25: true, 27: true, 30: true, 32: true, 34: true, 36: true,
}

func playRoulette(rng *rand.Rand, bet int64, choice string) (int64, string) {
	n := rng.IntN(37) // 0..36
	color := "Green"
	if n != 0 {
		if rouletteReds[n] {
			color = "Red"
		} else {
			color = "Black"
		}
	}
	desc := "Spun " + itoa(n) + " (" + color + ")"
	win := false
	mult := 0.0
	if num, err := strconv.Atoi(choice); err == nil && num >= 0 && num <= 36 {
		// Straight-up number bet pays 36x.
		win, mult = n == num, 36
	} else {
		mult = 2 // all even-money bets
		switch choice {
		case "red":
			win = n != 0 && rouletteReds[n]
		case "black":
			win = n != 0 && !rouletteReds[n]
		case "even":
			win = n != 0 && n%2 == 0
		case "odd":
			win = n%2 == 1
		case "low":
			win = n >= 1 && n <= 18
		case "high":
			win = n >= 19 && n <= 36
		default:
			win = n != 0 && rouletteReds[n] // default to red
		}
	}
	if win {
		return mulBet(bet, mult), desc + " — win ×" + ftoa(mult)
	}
	return 0, desc + " — loss"
}

// ---- 2. Rock / Paper / Scissors ------------------------------------------

func playRPS(rng *rand.Rand, bet int64, choice string) (int64, string) {
	moves := []string{"rock", "paper", "scissors"}
	icons := map[string]string{"rock": "✊", "paper": "✋", "scissors": "✌️"}
	if choice != "rock" && choice != "paper" && choice != "scissors" {
		choice = "rock"
	}
	house := moves[rng.IntN(3)]
	desc := "You " + icons[choice] + " vs " + icons[house]
	if choice == house {
		return bet, desc + " — push"
	}
	beats := map[string]string{"rock": "scissors", "paper": "rock", "scissors": "paper"}
	if beats[choice] == house {
		return mulBet(bet, 1.9), desc + " — win ×1.9"
	}
	return 0, desc + " — loss"
}

// ---- 3. War (single high card vs dealer) ---------------------------------

func playWar(rng *rand.Rand, bet int64, _ string) (int64, string) {
	you := rng.IntN(13) + 1
	dealer := rng.IntN(13) + 1
	desc := "You drew " + itoa(you) + " vs dealer " + itoa(dealer)
	switch {
	case you > dealer:
		return mulBet(bet, 1.9), desc + " — win ×1.9"
	case you == dealer:
		return bet, desc + " — push"
	default:
		return 0, desc + " — loss"
	}
}

// ---- 4. Plinko (8 rows → 9 bins) -----------------------------------------

var plinkoMults = []float64{15, 4, 1.5, 0.6, 0, 0.6, 1.5, 4, 15}

func playPlinko(rng *rand.Rand, bet int64, _ string) (int64, string) {
	bin := 0
	for i := 0; i < 8; i++ {
		if rng.IntN(2) == 0 {
			bin++
		}
	}
	m := plinkoMults[bin]
	if m <= 0 {
		return 0, "Ball landed center — ×0"
	}
	return mulBet(bet, m), "Ball landed ×" + ftoa(m)
}

// ---- 5. Scratch card (3 symbols) -----------------------------------------

var scratchSyms = []string{"🍀", "💎", "🔔", "⭐", "🍒", "💰"}

func playScratch(rng *rand.Rand, bet int64, _ string) (int64, string) {
	a, b, c := scratchSyms[rng.IntN(6)], scratchSyms[rng.IntN(6)], scratchSyms[rng.IntN(6)]
	row := a + b + c
	switch {
	case a == b && b == c:
		return mulBet(bet, 15), row + " — JACKPOT ×15"
	case a == b || b == c || a == c:
		return mulBet(bet, 1.3), row + " — pair ×1.3"
	default:
		return 0, row + " — no match"
	}
}

// ---- 6. Sic Bo lite (2d6: under/over/seven) ------------------------------

func playSicbo(rng *rand.Rand, bet int64, choice string) (int64, string) {
	d1, d2 := rng.IntN(6)+1, rng.IntN(6)+1
	sum := d1 + d2
	if choice != "under" && choice != "over" && choice != "seven" {
		choice = "under"
	}
	desc := "Rolled " + itoa(d1) + "+" + itoa(d2) + "=" + itoa(sum)
	switch choice {
	case "under":
		if sum < 7 {
			return mulBet(bet, 2.28), desc + " — under wins ×2.28"
		}
	case "over":
		if sum > 7 {
			return mulBet(bet, 2.28), desc + " — over wins ×2.28"
		}
	case "seven":
		if sum == 7 {
			return mulBet(bet, 5.7), desc + " — seven! ×5.7"
		}
	}
	return 0, desc + " — loss"
}

// ---- 7. Lucky number (1..10) ---------------------------------------------

func playLucky(rng *rand.Rand, bet int64, choice string) (int64, string) {
	pick, err := strconv.Atoi(choice)
	if err != nil || pick < 1 || pick > 10 {
		pick = 7
	}
	draw := rng.IntN(10) + 1
	desc := "Drew " + itoa(draw) + " (you picked " + itoa(pick) + ")"
	if draw == pick {
		return mulBet(bet, 9.5), desc + " — win ×9.5"
	}
	return 0, desc + " — loss"
}

// ---- 8. Gem dig (weighted rarity) ----------------------------------------

func playGems(rng *rand.Rand, bet int64, _ string) (int64, string) {
	roll := rng.Float64()
	switch {
	case roll < 0.55:
		return 0, "💩 Worthless rock — ×0"
	case roll < 0.85:
		return mulBet(bet, 1.0), "🟢 Uncommon gem — ×1.0"
	case roll < 0.95:
		return mulBet(bet, 2.5), "🔵 Rare gem — ×2.5"
	case roll < 0.99:
		return mulBet(bet, 6.0), "🟣 Epic gem — ×6"
	default:
		return mulBet(bet, 15.0), "🟡 Legendary gem — ×15"
	}
}

// ---- 9. Crash (choose a cashout multiplier) ------------------------------

var crashTargets = map[string]float64{"1.5": 1.5, "2": 2, "3": 3, "5": 5, "10": 10}

func playCrash(rng *rand.Rand, bet int64, choice string) (int64, string) {
	target, ok := crashTargets[choice]
	if !ok {
		target = 2
	}
	// House edge 3%: P(crash >= x) = 0.97/x.
	u := rng.Float64()
	if u < 1e-9 {
		u = 1e-9
	}
	crash := 0.97 / u
	desc := "Crashed at ×" + ftoa(round2(crash)) + " (cashout ×" + ftoa(target) + ")"
	if crash >= target {
		return mulBet(bet, target), desc + " — win"
	}
	return 0, desc + " — busted"
}

func round2(f float64) float64 { return float64(int(f*100)) / 100 }

// ---- 10. Mega wheel (12 segments) ----------------------------------------

var megaWheelSegs = []float64{0, 0, 0, 0, 0, 1.5, 0, 0, 2, 0, 0, 8}

func playMegaWheel(rng *rand.Rand, bet int64, _ string) (int64, string) {
	seg := rng.IntN(len(megaWheelSegs))
	m := megaWheelSegs[seg]
	if m <= 0 {
		return 0, "Landed on a blank"
	}
	return mulBet(bet, m), "Won ×" + ftoa(m)
}

// ---- 11. Dice duel (your die vs house die) -------------------------------

func playDiceDuel(rng *rand.Rand, bet int64, _ string) (int64, string) {
	you, house := rng.IntN(6)+1, rng.IntN(6)+1
	desc := "You " + itoa(you) + " vs house " + itoa(house)
	switch {
	case you > house:
		return mulBet(bet, 1.88), desc + " — win ×1.88"
	case you == house:
		return bet, desc + " — push"
	default:
		return 0, desc + " — loss"
	}
}

// ---- 12. Three chests (pick one) -----------------------------------------

func playChests(rng *rand.Rand, bet int64, choice string) (int64, string) {
	pick, err := strconv.Atoi(choice)
	if err != nil || pick < 0 || pick > 2 {
		pick = 0
	}
	prize := rng.IntN(3)
	desc := "Gold was in chest " + itoa(prize+1) + " (you opened " + itoa(pick+1) + ")"
	if pick == prize {
		return mulBet(bet, 2.85), desc + " — win ×2.85"
	}
	return 0, desc + " — empty"
}

// ---- 13. Horse race (pick a horse 1..4) ----------------------------------

func playHorseRace(rng *rand.Rand, bet int64, choice string) (int64, string) {
	pick, err := strconv.Atoi(choice)
	if err != nil || pick < 0 || pick > 3 {
		pick = 0
	}
	winner := rng.IntN(4)
	desc := "Horse " + itoa(winner+1) + " won (you backed " + itoa(pick+1) + ")"
	if pick == winner {
		return mulBet(bet, 3.8), desc + " — win ×3.8"
	}
	return 0, desc + " — loss"
}

// ---- 14. Colour pick -----------------------------------------------------

func playColorPick(rng *rand.Rand, bet int64, choice string) (int64, string) {
	colors := []string{"red", "green", "blue"}
	valid := map[string]bool{"red": true, "green": true, "blue": true}
	if !valid[choice] {
		choice = "red"
	}
	drawn := colors[rng.IntN(3)]
	desc := "Drew " + drawn + " (you picked " + choice + ")"
	if drawn == choice {
		return mulBet(bet, 2.85), desc + " — win ×2.85"
	}
	return 0, desc + " — loss"
}

// ---- 15. Fortune dice (3d6: triple / pair) -------------------------------

func playFortune(rng *rand.Rand, bet int64, _ string) (int64, string) {
	a, b, c := rng.IntN(6)+1, rng.IntN(6)+1, rng.IntN(6)+1
	desc := "Rolled " + itoa(a) + "-" + itoa(b) + "-" + itoa(c)
	switch {
	case a == b && b == c:
		return mulBet(bet, 14), desc + " — TRIPLE ×14"
	case a == b || b == c || a == c:
		return mulBet(bet, 1.35), desc + " — pair ×1.35"
	default:
		return 0, desc + " — loss"
	}
}

// ---- 16. Lightning strike (weighted multiplier) --------------------------

func playLightning(rng *rand.Rand, bet int64, _ string) (int64, string) {
	roll := rng.Float64()
	switch {
	case roll < 0.70:
		return 0, "⚡ Fizzled out — ×0"
	case roll < 0.91:
		return mulBet(bet, 2), "⚡ Spark — ×2"
	case roll < 0.97:
		return mulBet(bet, 4), "⚡ Bolt — ×4"
	case roll < 0.995:
		return mulBet(bet, 8), "⚡ Storm — ×8"
	default:
		return mulBet(bet, 25), "⚡ THUNDERGOD — ×25"
	}
}

// ---- 17. Minefield (reveal 3 of 9, 2 mines) ------------------------------

func playMinefield(rng *rand.Rand, bet int64, _ string) (int64, string) {
	// Place 2 mines among 9 tiles, reveal 3 random tiles.
	tiles := make([]bool, 9) // true = mine
	placed := 0
	for placed < 2 {
		i := rng.IntN(9)
		if !tiles[i] {
			tiles[i] = true
			placed++
		}
	}
	order := rng.Perm(9)
	for _, idx := range order[:3] {
		if tiles[idx] {
			return 0, "💥 Hit a mine on reveal — loss"
		}
	}
	return mulBet(bet, 2.28), "💎 Cleared 3 tiles — win ×2.28"
}

// ---- 18. Darts -----------------------------------------------------------

func playDarts(rng *rand.Rand, bet int64, _ string) (int64, string) {
	roll := rng.Float64()
	switch {
	case roll < 0.05:
		return mulBet(bet, 10), "🎯 Bullseye! ×10"
	case roll < 0.20:
		return mulBet(bet, 1.5), "🎯 Inner ring ×1.5"
	case roll < 0.50:
		return mulBet(bet, 0.8), "🎯 Outer ring ×0.8"
	default:
		return 0, "🎯 Missed the board — ×0"
	}
}

// ---- 19. Slots Deluxe (3 reels, 7 symbols) -------------------------------

var deluxeSyms = []string{"🍇", "🍊", "🔔", "⭐", "💎", "👑", "7️⃣"}

func playSlotDeluxe(rng *rand.Rand, bet int64, _ string) (int64, string) {
	a, b, c := deluxeSyms[rng.IntN(7)], deluxeSyms[rng.IntN(7)], deluxeSyms[rng.IntN(7)]
	row := a + b + c
	switch {
	case a == b && b == c:
		return mulBet(bet, 20), row + " — 3 of a kind ×20"
	case a == b || b == c || a == c:
		return mulBet(bet, 1.45), row + " — pair ×1.45"
	default:
		return 0, row + " — no match"
	}
}

// ---- 20. Keno (5 spots vs 5 drawn from 20) -------------------------------

func playKeno(rng *rand.Rand, bet int64, _ string) (int64, string) {
	// Player spots are 1..5 for simplicity; draw 5 distinct from 1..20.
	spots := map[int]bool{1: true, 2: true, 3: true, 4: true, 5: true}
	drawn := rng.Perm(20)[:5]
	hits := 0
	for _, d := range drawn {
		if spots[d+1] { // Perm yields 0..19
			hits++
		}
	}
	var m float64
	switch hits {
	case 5:
		m = 400
	case 4:
		m = 35
	case 3:
		m = 5
	case 2:
		m = 1.4
	}
	desc := "Matched " + itoa(hits) + "/5"
	if m > 0 {
		return mulBet(bet, m), desc + " — win ×" + ftoa(m)
	}
	return 0, desc + " — loss"
}
