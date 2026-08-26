package bot

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestAbyssMobAffixSnapshotContract(t *testing.T) {
	t.Parallel()
	effect := liveVampiricMobAffix()
	encoded, err := json.Marshal(effect)
	if err != nil {
		t.Fatal(err)
	}
	for _, token := range []string{`"key":"vampiric"`, `"icon":"🩸"`, `"description":"Heals for 15% of direct damage dealt."`, `"tone":"sustain"`, `"affix":true`} {
		if !strings.Contains(string(encoded), token) {
			t.Errorf("affix JSON %s is missing %s", encoded, token)
		}
	}
	if abyssLiveSnapshotSchemaVersion < 2 {
		t.Fatalf("live schema version = %d, want additive affix schema", abyssLiveSnapshotSchemaVersion)
	}
}

func TestAbyssMobAffixPresentationAssets(t *testing.T) {
	t.Parallel()
	live, err := webAssets.ReadFile("webassets/abyss_live.html")
	if err != nil {
		t.Fatal(err)
	}
	pixel, err := webAssets.ReadFile("webassets/abyss_pixel.html")
	if err != nil {
		t.Fatal(err)
	}
	styles, err := webAssets.ReadFile("webassets/abyss_mob_affixes.css")
	if err != nil {
		t.Fatal(err)
	}
	page, err := webAssets.ReadFile("webassets/abyss.html")
	if err != nil {
		t.Fatal(err)
	}
	for _, token := range []string{"liveEffectChip", "liveAffixAria", `title="'+consEsc(tip)`, "consEsc(tip)"} {
		if !strings.Contains(string(live), token) {
			t.Errorf("live affix UI is missing %q", token)
		}
	}
	for _, token := range []string{"liveEffectChip(effect,true)", "liveAffixAria(unit)"} {
		if !strings.Contains(string(pixel), token) {
			t.Errorf("pixel affix UI is missing %q", token)
		}
	}
	for _, token := range []string{".ab-effect.ab-mob-affix", ".ab-pixel-effects i.ab-mob-affix", "forced-colors: active", "max-width: 520px"} {
		if !strings.Contains(string(styles), token) {
			t.Errorf("affix stylesheet is missing %q", token)
		}
	}
	if !strings.Contains(string(page), "abyss_mob_affixes.css") {
		t.Fatal("Abyss page does not load the dedicated mob-affix stylesheet")
	}
}
