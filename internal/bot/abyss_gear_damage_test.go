package bot

import (
	"os"
	"strings"
	"testing"

	"ts3news/internal/content"
)

func TestAbyssGearDamageContributions(t *testing.T) {
	equipped := map[content.GearSlot]content.Gear{
		content.SlotMainHand: {
			Element: content.ElementFire,
			Stats:   content.Stats{STR: 60, INT: 20},
		},
		content.SlotOffHand: {Stats: content.Stats{STR: 40, INT: 60}},
		content.SlotChest:   {Stats: content.Stats{STR: -10, INT: 20}},
		content.SlotHead: {
			Unidentified: true,
			Stats:        content.Stats{STR: 999, INT: 999},
		},
	}

	got := abyssGearDamageContributions(equipped)
	if len(got) != 3 {
		t.Fatalf("visible contributions = %d, want 3", len(got))
	}
	mainHand := got[content.SlotMainHand]
	if mainHand.BasePower != 60 || mainHand.BaseSharePct != 60 {
		t.Errorf("main-hand base contribution = %+v, want +60 and 60%%", mainHand)
	}
	if mainHand.SpellPower != 20 || mainHand.SpellSharePct != 20 {
		t.Errorf("main-hand spell contribution = %+v, want +20 and 20%%", mainHand)
	}
	if mainHand.AttackType != string(content.ElementFire) {
		t.Errorf("main-hand attack type = %q, want Fire", mainHand.AttackType)
	}
	offHand := got[content.SlotOffHand]
	if offHand.BaseSharePct != 40 || offHand.SpellSharePct != 60 || offHand.AttackType != "" {
		t.Errorf("off-hand contribution = %+v, want 40%% STR, 60%% INT, no attack type", offHand)
	}
	chest := got[content.SlotChest]
	if chest.BasePower != -10 || chest.BaseSharePct != 0 || chest.SpellSharePct != 20 {
		t.Errorf("penalty contribution = %+v, want -10 STR penalty and 20%% INT", chest)
	}
	if _, ok := got[content.SlotHead]; ok {
		t.Error("unidentified gear exposed a damage contribution")
	}
}

func TestAbyssGearDamageMainHandDefaultsToPhysical(t *testing.T) {
	got := abyssGearDamageContributions(map[content.GearSlot]content.Gear{
		content.SlotMainHand: {},
	})
	if got[content.SlotMainHand].AttackType != string(content.ElementPhysical) {
		t.Fatalf("attack type = %q, want Physical", got[content.SlotMainHand].AttackType)
	}
}

func TestAbyssGearContributionPct(t *testing.T) {
	for _, tc := range []struct {
		name         string
		value, total int
		want         int
	}{
		{name: "rounded", value: 1, total: 6, want: 17},
		{name: "minimum visible", value: 1, total: 1000, want: 1},
		{name: "negative", value: -2, total: 10, want: 0},
		{name: "zero total", value: 2, total: 0, want: 0},
		{name: "bounded", value: 12, total: 10, want: 100},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := abyssGearContributionPct(tc.value, tc.total); got != tc.want {
				t.Fatalf("abyssGearContributionPct(%d, %d) = %d, want %d", tc.value, tc.total, got, tc.want)
			}
		})
	}
}

func TestAbyssGearDamageDisclosureContract(t *testing.T) {
	read := func(name string) string {
		t.Helper()
		body, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		return string(body)
	}
	page := read("webassets/abyss.html")
	partial := read("webassets/abyss_gear_damage.html")
	styles := read("webassets/abyss_gear_damage.css")
	server := read("web.go")

	for _, token := range []string{
		`/static/abyss_gear_damage.css`, `role="button"`, `aria-expanded="false"`,
		`aria-controls="abGearDamage-{{.Slot}}"`, `event.detail!==0`, `event.stopPropagation()`,
		`abyssGearDamageBreakdown`, `abyssGearDamageJS`,
	} {
		if !strings.Contains(page, token) {
			t.Errorf("Abyss page is missing %q", token)
		}
	}
	for _, token := range []string{
		`role="region"`, `hidden`, `equipped stats only`, `direct penalty`,
		`otherPanel.hidden=true`, `panel.hidden=!open`,
	} {
		if !strings.Contains(partial, token) {
			t.Errorf("damage disclosure is missing %q", token)
		}
	}
	for _, token := range []string{`.ab-gear-damage[hidden]`, `.ab-gear-damage-indicator::before`} {
		if !strings.Contains(styles, token) {
			t.Errorf("damage styles are missing %q", token)
		}
	}
	if strings.Contains(styles, `.abyss-side-gear[aria-controls]::after`) {
		t.Error("damage disclosure overwrites the gear row pseudo-element used by cracked items")
	}
	if !strings.Contains(server, `/static/abyss_gear_damage.css`) {
		t.Error("damage stylesheet has no explicit asset route")
	}
}
