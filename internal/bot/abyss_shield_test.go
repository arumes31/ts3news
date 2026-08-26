package bot

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"ts3news/internal/content"
)

func TestAbyssShieldCapacity(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		maxHP, defense int
		want           int
	}{
		{name: "HP capped", maxHP: 1_000, defense: 20, want: 150},
		{name: "DEF limited", maxHP: 10_000, defense: 20, want: 200},
		{name: "tiny health", maxHP: 1, defense: 1, want: 1},
		{name: "no defense", maxHP: 1_000},
		{name: "invalid health", maxHP: -1, defense: 100},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := abyssShieldCapacity(test.maxHP, test.defense); got != test.want {
				t.Fatalf("abyssShieldCapacity(%d, %d) = %d, want %d", test.maxHP, test.defense, got, test.want)
			}
		})
	}

	maxInt := int(^uint(0) >> 1)
	if got := abyssShieldCapacity(maxInt, maxInt); got <= 0 || got > maxInt {
		t.Fatalf("maximum-input capacity = %d, want a positive bounded value", got)
	}
}

func TestInitializeAndAbsorbAbyssShield(t *testing.T) {
	t.Parallel()

	user := activeUser{u: &UserInCombat{
		Nickname: "Bulwark", EscrowLoot: true,
		Stats: content.Stats{HP: 1_000, DEF: 20},
	}}
	logLine := initializeAbyssShield(&user)
	if user.shield != 150 || user.maxShield != 150 {
		t.Fatalf("initialized shield = %d/%d, want 150/150", user.shield, user.maxShield)
	}
	if !strings.Contains(logLine, "150-point barrier") || !strings.Contains(logLine, "20 DEF") {
		t.Fatalf("initialization log lacks exact values: %q", logLine)
	}

	remaining, absorbed := absorbAbyssShield(&user, 80)
	if remaining != 0 || absorbed != 80 || user.shield != 70 {
		t.Fatalf("first hit = remaining %d, absorbed %d, shield %d", remaining, absorbed, user.shield)
	}
	remaining, absorbed = absorbAbyssShield(&user, 100)
	if remaining != 30 || absorbed != 70 || user.shield != 0 {
		t.Fatalf("breaking hit = remaining %d, absorbed %d, shield %d", remaining, absorbed, user.shield)
	}
	if got := abyssShieldAbsorbLog("Bulwark", absorbed, user.shield); !strings.Contains(got, "barrier broken") {
		t.Fatalf("break log = %q", got)
	}

	nonAbyss := activeUser{u: &UserInCombat{}, shield: 100, maxShield: 100}
	remaining, absorbed = absorbAbyssShield(&nonAbyss, 40)
	if remaining != 40 || absorbed != 0 || nonAbyss.shield != 100 {
		t.Fatalf("non-Abyss shield changed: remaining %d, absorbed %d, shield %d", remaining, absorbed, nonAbyss.shield)
	}
}

func TestAbyssShieldAbsorbsMobDamageBeforeHP(t *testing.T) {
	t.Parallel()

	user := activeUser{
		u: &UserInCombat{
			UID: "tank", Nickname: "Tank", EscrowLoot: true,
			CurrentHP: 1_000, Stats: content.Stats{HP: 1_000, DEF: 20},
			Equipped: map[content.GearSlot]content.Gear{}, DEFMod: 1,
		},
		effects: []content.ItemEffect{content.EffectThorns},
	}
	initializeAbyssShield(&user)
	users := []activeUser{user}
	mob := &content.Mob{
		Name: "Raider", Stats: content.Stats{HP: 1_000, STR: 100, SPD: 10},
		MaxHP: 1_000, STRMod: 1,
	}
	logs := []string{}
	totalMobDamage, totalUserDamage := 0, 0
	track := &abyssFightTrack{}
	random := fixedCombatRandom{float: 1, intn: 99}

	(&Bot{}).mobTurn(users, []*content.Mob{mob}, content.Zone{}, 1, &logs, &totalMobDamage, &totalUserDamage, 1, false, track, random)
	if users[0].u.CurrentHP != 1_000 || users[0].shield != 70 || totalMobDamage != 0 {
		t.Fatalf("first hit = HP %d, shield %d, enemy damage %d; want 1000, 70, 0", users[0].u.CurrentHP, users[0].shield, totalMobDamage)
	}
	if track.shields != 80 || track.thorns != 8 || totalUserDamage != 8 {
		t.Fatalf("first hit tracking = shield %d, thorns %d, party damage %d", track.shields, track.thorns, totalUserDamage)
	}

	(&Bot{}).mobTurn(users, []*content.Mob{mob}, content.Zone{}, 1, &logs, &totalMobDamage, &totalUserDamage, 2, false, track, random)
	if users[0].u.CurrentHP != 990 || users[0].shield != 0 || totalMobDamage != 10 || users[0].u.DamageTaken != 10 {
		t.Fatalf("breaking hit = HP %d, shield %d, enemy damage %d, taken %d", users[0].u.CurrentHP, users[0].shield, totalMobDamage, users[0].u.DamageTaken)
	}
	joined := strings.Join(logs, "\n")
	for _, want := range []string{"absorbs 80 damage", "absorbs 70 damage", "barrier broken"} {
		if !strings.Contains(joined, want) {
			t.Errorf("combat log lacks %q: %q", want, joined)
		}
	}
}

func TestAbyssShieldDoesNotAbsorbEnvironmentalDamage(t *testing.T) {
	t.Parallel()

	users := []activeUser{{
		u: &UserInCombat{
			Nickname: "Tank", EscrowLoot: true,
			CurrentHP: 1_000, Stats: content.Stats{HP: 1_000, DEF: 20},
		},
		shield: 150, maxShield: 150,
	}}
	zone := content.Zone{Effects: []content.ZoneEffect{{
		Name: "Falling stone", Type: content.ZoneHazard, Power: 0.4,
	}}}
	logs := []string{}

	(&Bot{}).applyEffects(users, nil, zone, 1, 1, 1, &logs)
	if users[0].u.CurrentHP != 990 || users[0].u.DamageTaken != 10 || users[0].shield != 150 {
		t.Fatalf("hazard result = HP %d, taken %d, shield %d; want 990, 10, 150",
			users[0].u.CurrentHP, users[0].u.DamageTaken, users[0].shield)
	}
}

func TestAbyssShieldLiveContract(t *testing.T) {
	t.Parallel()

	readAsset := func(name string) string {
		t.Helper()
		body, err := webAssets.ReadFile("webassets/" + name)
		if err != nil {
			t.Fatal(err)
		}
		return string(body)
	}
	source := readAsset("abyss_live.html") + readAsset("abyss_pixel.html") +
		readAsset("abyss_shield.html") + readAsset("abyss_shield.css") + readAsset("abyss.html")
	for _, token := range []string{
		"max_shield", "liveShieldBar(unit,'combatant')", "liveShieldBar(unit,'overhead')",
		"liveShieldAria", "shield-break", "ab-combatant-shield", "ab-overhead-shield",
		"prefers-reduced-motion", "/static/abyss_shield.css",
	} {
		if !strings.Contains(source, token) {
			t.Errorf("shield UI contract is missing %q", token)
		}
	}
	server, err := os.ReadFile("web.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(server), "/static/abyss_shield.css") {
		t.Error("shield stylesheet has no explicit asset route")
	}
	encoded, err := json.Marshal(abyssLiveCombatantView{Shield: 25, MaxShield: 100})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), `"shield":25`) || !strings.Contains(string(encoded), `"max_shield":100`) {
		t.Fatalf("live shield JSON = %s", encoded)
	}
}
