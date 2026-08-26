package bot

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"ts3news/internal/content"
)

func TestAbyssWeaknessWindowArmsAfterStunningHit(t *testing.T) {
	t.Parallel()

	target := &content.Mob{Name: "Iron Warden", Stats: content.Stats{HP: 500}}
	logs := []string{}

	if damage, critical := consumeAbyssWeaknessCritical(target, 80, true); damage != 80 || critical {
		t.Fatalf("stunning hit consumed a window: damage %d, critical %t", damage, critical)
	}
	if !armAbyssWeaknessWindow(target, true, &logs) || !target.WeaknessWindow {
		t.Fatal("successful stun did not arm the next-hit window")
	}
	if armAbyssWeaknessWindow(target, true, &logs) {
		t.Fatal("repeated stun stacked a second weakness charge")
	}
	if len(logs) != 1 || !strings.Contains(logs[0], "next direct player hit") {
		t.Fatalf("arm logs = %q, want one exact promise", logs)
	}

	damage, critical := consumeAbyssWeaknessCritical(target, 80, true)
	if damage != 160 || !critical || target.WeaknessWindow {
		t.Fatalf("next hit = damage %d, critical %t, armed %t; want 160, true, false",
			damage, critical, target.WeaknessWindow)
	}
	if next, critical := consumeAbyssWeaknessCritical(target, 80, true); next != 80 || critical {
		t.Fatalf("spent window affected another hit: damage %d, critical %t", next, critical)
	}
}

func TestResolveAbyssWeaknessCriticalClearsFumbleAndTracksUse(t *testing.T) {
	t.Parallel()

	target := &content.Mob{Name: "Warden", Stats: content.Stats{HP: 100}, WeaknessWindow: true}
	user := &activeUser{u: &UserInCombat{Nickname: "Hero", EscrowLoot: true}, fumbled: true}
	track := &abyssFightTrack{}
	logs := []string{}

	damage, critical := resolveAbyssWeaknessCritical(
		abyssWeaknessCriticalContext{target: target, user: user, track: track, logs: &logs},
		40,
	)
	if damage != 80 || !critical || user.fumbled || track.weaknessCrits != 1 {
		t.Fatalf("resolved critical = damage %d, critical %t, fumbled %t, tracked %d",
			damage, critical, user.fumbled, track.weaknessCrits)
	}
	if len(logs) != 1 || !strings.Contains(logs[0], "WEAKNESS CRITICAL") {
		t.Fatalf("critical logs = %q", logs)
	}
}

func TestAbyssWeaknessWindowGuardsInvalidAndOverflowInputs(t *testing.T) {
	t.Parallel()

	nonAbyss := &content.Mob{Name: "Arena Mob", Stats: content.Stats{HP: 100}, WeaknessWindow: true}
	if damage, critical := consumeAbyssWeaknessCritical(nonAbyss, 25, false); damage != 25 || critical || !nonAbyss.WeaknessWindow {
		t.Fatalf("non-Abyss hit changed window: damage %d, critical %t, armed %t",
			damage, critical, nonAbyss.WeaknessWindow)
	}
	if armAbyssWeaknessWindow(&content.Mob{Stats: content.Stats{HP: 100}}, false, nil) {
		t.Fatal("non-Abyss stun armed a weakness window")
	}

	zeroDamage := &content.Mob{Stats: content.Stats{HP: 100}, WeaknessWindow: true}
	if damage, critical := consumeAbyssWeaknessCritical(zeroDamage, 0, true); damage != 0 || critical || !zeroDamage.WeaknessWindow {
		t.Fatalf("zero damage spent window: damage %d, critical %t, armed %t",
			damage, critical, zeroDamage.WeaknessWindow)
	}

	maxInt := int(^uint(0) >> 1)
	overflow := &content.Mob{Stats: content.Stats{HP: 100}, WeaknessWindow: true}
	if damage, critical := consumeAbyssWeaknessCritical(overflow, maxInt, true); damage != maxInt || !critical {
		t.Fatalf("overflow-safe critical = damage %d, critical %t; want %d, true", damage, critical, maxInt)
	}
	if !strings.Contains(abyssWeaknessCriticalLog("Hero", "Warden"), "2× damage") {
		t.Fatal("critical log does not state the exact multiplier")
	}
}

func TestAbyssWeaknessWindowCombatIntegrationContract(t *testing.T) {
	t.Parallel()

	source, err := os.ReadFile("xp.go")
	if err != nil {
		t.Fatal(err)
	}
	combat := string(source)
	consume := strings.Index(combat, "resolveAbyssWeaknessCritical(")
	damage := strings.Index(combat, "target.Stats.HP -= dmg")
	arm := strings.LastIndex(combat, "armAbyssWeaknessWindow(target, stunnedThisHit, logs)")
	if consume < 0 || damage < 0 || arm < 0 || consume >= damage || damage >= arm {
		t.Fatalf("weakness ordering = consume %d, damage %d, arm %d; want consume < damage < arm", consume, damage, arm)
	}
	if strings.Count(combat, "resolveAbyssWeaknessCritical(") != 1 {
		t.Fatal("weakness consumption escaped the one direct player-hit resolver")
	}
	if !strings.Contains(combat, "chainDmg := secondaryBaseDamage / 2") ||
		!strings.Contains(combat, "cleaveDamage := max(1, secondaryOverkill/2)") {
		t.Fatal("derived chain or cleave damage can inherit the weakness critical multiplier")
	}
	for _, name := range []string{"abyss_rescue_support.go", "web_abyss_combat_rooms.go"} {
		other, readErr := os.ReadFile(name)
		if readErr != nil {
			t.Fatal(readErr)
		}
		if strings.Contains(string(other), "resolveAbyssWeaknessCritical") {
			t.Fatalf("%s can consume a player-only weakness window", name)
		}
	}
}

func TestAbyssWeaknessWindowLiveContract(t *testing.T) {
	t.Parallel()

	effects := liveMobEffects(&content.Mob{
		Stats: content.Stats{HP: 100}, StunRounds: 1, WeaknessWindow: true,
	})
	joinedEffects := ""
	for _, effect := range effects {
		joinedEffects += effect.Name + " " + effect.Duration
	}
	if !strings.Contains(joinedEffects, "Weakness Window") || !strings.Contains(joinedEffects, "guaranteed critical") {
		t.Fatalf("live effects = %+v, want exact weakness promise", effects)
	}

	encoded, err := json.Marshal(abyssLiveCombatantView{WeaknessReady: true})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), `"weakness_ready":true`) {
		t.Fatalf("live weakness JSON = %s", encoded)
	}
	empty, err := json.Marshal(abyssLiveCombatantView{})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(empty), "weakness_ready") {
		t.Fatalf("inactive weakness leaked into JSON: %s", empty)
	}

	readAsset := func(name string) string {
		t.Helper()
		body, readErr := webAssets.ReadFile("webassets/" + name)
		if readErr != nil {
			t.Fatal(readErr)
		}
		return string(body)
	}
	source := readAsset("abyss_live.html") + readAsset("abyss_pixel.html") +
		readAsset("abyss_weakness_window.html") + readAsset("abyss_weakness_window.css") + readAsset("abyss.html")
	for _, token := range []string{
		"weakness_ready", "liveWeaknessWindowMark", "liveWeaknessAria", "NEXT PLAYER HIT CRITS",
		"weakness-open", "prefers-reduced-motion", "forced-colors", "/static/abyss_weakness_window.css",
	} {
		if !strings.Contains(source, token) {
			t.Errorf("weakness UI contract is missing %q", token)
		}
	}
	server, err := os.ReadFile("web.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(server), "/static/abyss_weakness_window.css") {
		t.Error("weakness stylesheet has no explicit asset route")
	}
}
