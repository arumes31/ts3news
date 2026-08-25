package bot

import (
	"strings"
	"testing"

	"ts3news/internal/content"
)

func TestCombatTimelineCarriesAuthoritativePetHealth(t *testing.T) {
	t.Parallel()

	pet := &content.Mob{Name: "Mossling", Stats: content.Stats{HP: 42}, MaxHP: 100}
	user := &UserInCombat{CurrentHP: 80, Stats: content.Stats{HP: 100}, Pets: []*content.Mob{pet}}
	frames := []combatTimelineFrame{}
	appendCombatTimelineFrame(&frames, 3, 2, []activeUser{{u: user}}, nil)
	markCombatTimelineExchange(&frames, "player", 4)

	if len(frames) != 1 {
		t.Fatalf("timeline frames = %d, want 1", len(frames))
	}
	frame := frames[0]
	if frame.PetName != "Mossling" || frame.PetHP != 42 || frame.PetMax != 100 {
		t.Fatalf("pet frame = %#v, want Mossling at 42/100 HP", frame)
	}
	if frame.Round != 2 {
		t.Fatalf("round = %d, want 2", frame.Round)
	}
	if frame.Side != "player" || frame.Actions != 4 {
		t.Fatalf("exchange metadata = %#v, want four player actions", frame)
	}
}

func TestAbyssStagePresentationContracts(t *testing.T) {
	t.Parallel()

	page, err := webAssets.ReadFile("webassets/abyss.html")
	if err != nil {
		t.Fatalf("read Abyss page: %v", err)
	}
	pageSource := string(page)
	for _, required := range []string{
		`id="biomeChip"`,
		`id="wavePips"`,
		`id="eventIcon"`,
		`id="restEmbers"`,
		`id="petHPBar" role="progressbar"`,
		"function elemGlyph",
		"function setBiomeChip",
		"function firstStrike",
		"function updateWavePips",
		"function bossFrameClass",
		"classList.toggle('ab-downed'",
		"function updateStageScene",
		"updatePetCombatFrame(f)",
		`id="combatExchange"`,
		"animateCombatExchange(f)",
		`{{template "abyss-stage-presentation" .}}`,
	} {
		if !strings.Contains(pageSource, required) {
			t.Errorf("Abyss stage is missing %q", required)
		}
	}

	server, err := NewWebServer(nil)
	if err != nil {
		t.Fatalf("NewWebServer: %v", err)
	}
	partial := server.tmpl.Lookup("abyss-stage-presentation")
	if partial == nil {
		t.Fatal("stage presentation template is missing")
	}
	partialSource := partial.Tree.Root.String()
	for _, required := range []string{
		"function updatePetCombatFrame",
		"frame.pet_hp",
		"frame.pet_max",
		"fill.style.width=percentage+'%'",
		"aria-valuetext",
		"classList.toggle('is-low'",
	} {
		if !strings.Contains(partialSource, required) {
			t.Errorf("pet presentation is missing %q", required)
		}
	}

	uiCSS, err := webAssets.ReadFile("webassets/abyss_ui200.css")
	if err != nil {
		t.Fatalf("read UI-200 CSS: %v", err)
	}
	pixelCSS, err := webAssets.ReadFile("webassets/abyss_pixel.css")
	if err != nil {
		t.Fatalf("read pixel CSS: %v", err)
	}
	commandCSS, err := webAssets.ReadFile("webassets/abyss_command.css")
	if err != nil {
		t.Fatalf("read command CSS: %v", err)
	}
	styles := string(uiCSS) + string(pixelCSS) + string(commandCSS)
	for _, required := range []string{
		"@keyframes ab-pixel-idle",
		".ab-scene.ab-fs-you::before",
		".ab-petbar.is-low",
		".ab-pip.cur",
		".abyss-stage.ab-boss-void",
		".abyss-stage.ab-downed",
		".ab-rest-embers",
		".ab-event-icon",
		"@keyframes ab-exchange-player",
		"@keyframes ab-exchange-enemy",
	} {
		if !strings.Contains(styles, required) {
			t.Errorf("stage presentation CSS is missing %q", required)
		}
	}
}
