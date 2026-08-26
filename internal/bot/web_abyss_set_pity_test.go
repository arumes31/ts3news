package bot

import (
	"strings"
	"testing"

	"ts3news/internal/content"
)

func TestAbyssSetPityPanelUsesOnlyIdentifiedDistinctGear(t *testing.T) {
	t.Parallel()

	predator := content.AbyssSetCatalog("predator")
	warden := content.AbyssSetCatalog("warden")
	if len(predator) < 4 || len(warden) < 1 {
		t.Fatal("set catalogs do not satisfy the completion safeguard")
	}
	equipped := map[content.GearSlot]content.Gear{
		predator[0].Slot: predator[0],
		warden[0].Slot: {
			ID: warden[0].ID, Unidentified: true,
		},
	}
	inventory := []gearView{
		{ID: predator[1].ID},
		{ID: predator[0].ID}, // A duplicate surface must not advance the meter.
		{Unidentified: true},
	}
	escrow := []runLootRow{{GearID: predator[2].ID}, {Unidentified: true}}

	panel := abyssSetPityPanel(equipped, inventory, escrow)
	if panel.ChancePct != 25 || panel.HiddenItems != 3 || len(panel.Sets) != 2 {
		t.Fatalf("panel summary = %#v", panel)
	}
	got := panel.Sets[0]
	if got.ID != "predator" || got.Owned != 3 || !got.Active || got.Complete || got.Percent != 75 || got.Remaining != 1 {
		t.Fatalf("predator progress = %#v", got)
	}
	if got := panel.Sets[1]; got.Owned != 0 || got.Active || got.Complete {
		t.Fatalf("unidentified Warden identity leaked into progress: %#v", got)
	}
}

func TestAbyssSetPityPanelMarksFourPieceMilestone(t *testing.T) {
	t.Parallel()

	predator := content.AbyssSetCatalog("predator")
	inventory := make([]gearView, 0, abyssSetPityMilestone)
	for _, gear := range predator[:abyssSetPityMilestone] {
		inventory = append(inventory, gearView{ID: gear.ID})
	}
	progress := abyssSetPityPanel(nil, inventory, nil).Sets[0]
	if !progress.Complete || progress.Active || progress.Percent != 100 || progress.Remaining != 0 {
		t.Fatalf("four-piece progress = %#v", progress)
	}
}

func TestAbyssSetPityPanelPresentationContracts(t *testing.T) {
	t.Parallel()

	templateBody, err := webAssets.ReadFile("webassets/abyss_set_pity.html")
	if err != nil {
		t.Fatal(err)
	}
	styles, err := webAssets.ReadFile("webassets/abyss_set_pity.css")
	if err != nil {
		t.Fatal(err)
	}
	page, err := webAssets.ReadFile("webassets/abyss.html")
	if err != nil {
		t.Fatal(err)
	}
	source := string(templateBody) + string(styles) + string(page)
	for _, token := range []string{
		`define "abyss-set-pity"`, `role="progressbar"`, `aria-valuenow=`,
		`unidentified gear item(s)`, `background: #0b1220`, `@media (forced-colors: active)`,
		`@media (max-width: 640px)`, `template "abyss-set-pity" .SetPityPanel`,
		`/static/abyss_set_pity.css`,
	} {
		if !strings.Contains(source, token) {
			t.Errorf("set-pity presentation missing %q", token)
		}
	}
}
