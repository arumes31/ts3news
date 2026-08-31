package bot

import (
	"testing"

	"ts3news/internal/content"
)

func TestAbyssElementalPreviewMatchesCombatMultipliers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		player       content.Element
		enemy        content.Element
		weakTo       content.Element
		resists      content.Element
		wantOutcome  string
		wantMultiple string
	}{
		{
			name:   "strong counter",
			player: content.ElementWater, enemy: content.ElementFire,
			weakTo: content.ElementWater, resists: content.ElementAir,
			wantOutcome: "ADVANTAGE", wantMultiple: "2×",
		},
		{
			name:   "resisted attack",
			player: content.ElementAir, enemy: content.ElementFire,
			weakTo: content.ElementWater, resists: content.ElementAir,
			wantOutcome: "RESISTED", wantMultiple: "½×",
		},
		{
			name:   "neutral elemental attack",
			player: content.ElementEarth, enemy: content.ElementFire,
			weakTo: content.ElementWater, resists: content.ElementAir,
			wantOutcome: "NEUTRAL", wantMultiple: "1×",
		},
		{
			name:   "physical encounter",
			player: content.ElementFire, enemy: content.ElementPhysical,
			wantOutcome: "NEUTRAL", wantMultiple: "1×",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			view := abyssElementalPreview(
				abyssBossAffinityForecastView{
					Name: "Scouted foe", Element: string(test.enemy),
					WeakTo: string(test.weakTo), StrongAgainst: string(test.resists),
					TargetDepth: 15,
				},
				map[content.GearSlot]content.Gear{
					content.SlotMainHand: {Element: test.player},
				},
			)
			if !view.Active || view.Outcome != test.wantOutcome || view.Multiplier != test.wantMultiple {
				t.Fatalf("preview = %+v", view)
			}
			if view.NeutralEncounter != (test.enemy == content.ElementPhysical) {
				t.Fatalf("neutral encounter = %t", view.NeutralEncounter)
			}
		})
	}
}

func TestAbyssElementalPreviewUsesPhysicalForMissingWeapon(t *testing.T) {
	t.Parallel()

	view := abyssElementalPreview(
		abyssBossAffinityForecastView{
			Name: "Cinder Crown", Element: string(content.ElementFire),
			WeakTo: string(content.ElementWater), StrongAgainst: string(content.ElementAir),
			TargetDepth: 15,
		},
		nil,
	)
	if view.PlayerElement != string(content.ElementPhysical) || view.Outcome != "NEUTRAL" {
		t.Fatalf("preview = %+v", view)
	}
	if view.StrongElement != string(content.ElementWater) || view.ResistedElement != string(content.ElementAir) {
		t.Fatalf("counter guidance = %+v", view)
	}
}

func TestAbyssElementalPreviewUsesPhysicalForUnidentifiedWeapon(t *testing.T) {
	t.Parallel()

	view := abyssElementalPreview(
		abyssBossAffinityForecastView{
			Name: "Cinder Crown", Element: string(content.ElementFire),
			WeakTo: string(content.ElementWater), StrongAgainst: string(content.ElementAir),
			TargetDepth: 15,
		},
		map[content.GearSlot]content.Gear{
			content.SlotMainHand: {Element: content.ElementWater, Unidentified: true},
		},
	)
	if view.PlayerElement != string(content.ElementPhysical) || view.Outcome != "NEUTRAL" {
		t.Fatalf("unidentified weapon preview = %+v", view)
	}
}

func TestAbyssElementalPreviewNormalizesUnknownElements(t *testing.T) {
	t.Parallel()

	view := abyssElementalPreview(
		abyssBossAffinityForecastView{Name: "Unknown", Element: "Void", TargetDepth: 5},
		map[content.GearSlot]content.Gear{content.SlotMainHand: {Element: "Spirit"}},
	)
	if view.EnemyElement != string(content.ElementPhysical) ||
		view.PlayerElement != string(content.ElementPhysical) || !view.NeutralEncounter {
		t.Fatalf("preview = %+v", view)
	}
}
