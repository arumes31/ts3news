package bot

import "ts3news/internal/content"

const abyssSetPityMilestone = 4

type abyssSetPityProgressView struct {
	ID        string
	Name      string
	Icon      string
	Owned     int
	Required  int
	Percent   int
	Remaining int
	Active    bool
	Complete  bool
}

type abyssSetPityPanelView struct {
	Sets        []abyssSetPityProgressView
	HiddenItems int
	ChancePct   int
}

func abyssSetPityPanel(
	equipped map[content.GearSlot]content.Gear,
	inventory []gearView,
	escrow []runLootRow,
) abyssSetPityPanelView {
	owned := make(map[string]bool)
	hidden := 0
	for _, gear := range equipped {
		if gear.Unidentified {
			hidden++
			continue
		}
		owned[gear.ID] = true
	}
	for _, gear := range inventory {
		if gear.Unidentified {
			hidden++
			continue
		}
		if gear.ID != "" {
			owned[gear.ID] = true
		}
	}
	for _, gear := range escrow {
		if gear.Unidentified {
			hidden++
			continue
		}
		if gear.GearID != "" {
			owned[gear.GearID] = true
		}
	}

	sets := []struct {
		id, name, icon string
	}{
		{id: "predator", name: "Predator", icon: "🐺"},
		{id: "warden", name: "Warden", icon: "🛡️"},
	}
	view := abyssSetPityPanelView{
		Sets:        make([]abyssSetPityProgressView, 0, len(sets)),
		HiddenItems: hidden,
		ChancePct:   int(abyssSetPityChance * 100),
	}
	for _, set := range sets {
		count := 0
		for _, gear := range content.AbyssSetCatalog(set.id) {
			if owned[gear.ID] {
				count++
			}
		}
		progress := min(count, abyssSetPityMilestone)
		view.Sets = append(view.Sets, abyssSetPityProgressView{
			ID: set.id, Name: set.name, Icon: set.icon,
			Owned: count, Required: abyssSetPityMilestone,
			Percent:   progress * 100 / abyssSetPityMilestone,
			Remaining: max(0, abyssSetPityMilestone-count),
			Active:    count == abyssSetPityMilestone-1,
			Complete:  count >= abyssSetPityMilestone,
		})
	}
	return view
}
