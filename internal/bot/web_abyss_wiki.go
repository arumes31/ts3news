package bot

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"ts3news/internal/content"
)

type abyssWikiEntryView struct {
	ID          string
	Name        string
	Icon        string
	Kind        string
	Meta        string
	Description string
	Search      string
}

type abyssWikiCategoryView struct {
	Key     string
	Label   string
	Icon    string
	Entries []abyssWikiEntryView
}

type abyssWikiView struct {
	Categories []abyssWikiCategoryView
	Total      int
}

var abyssDailyAffixDescriptions = map[string]string{
	"double_hazards":       "Floor hazard damage is doubled.",
	"zero_durability_loss": "Equipped gear takes no durability loss.",
	"enraged_mobs":         "Every enemy enters combat enraged.",
	"glass_cannon":         "Floors are 30% deadlier and award 30% more cache.",
	"gold_rush":            "Every cleared floor awards double cache.",
	"iron_skin":            "Delvers take 30% less direct damage but earn 10% less cache.",
	"bloodlust":            "A killing blow restores 20% of the delver's maximum HP.",
	"execute":              "Attacks deal 50% more damage below 30% enemy HP, with 10% less cache.",
	"vampiric_mobs":        "Enemies heal for 15% of damage dealt and award 15% more cache.",
}

func abyssWikiCatalog() abyssWikiView {
	categories := []abyssWikiCategoryView{
		{Key: "gear", Label: "Gear", Icon: "⚔", Entries: abyssWikiGearEntries()},
		{Key: "mobs", Label: "Monsters", Icon: "☠", Entries: abyssWikiMobEntries()},
		{Key: "pacts", Label: "Pacts", Icon: "⚖", Entries: abyssWikiPactEntries()},
		{Key: "affixes", Label: "Affixes", Icon: "✦", Entries: abyssWikiAffixEntries()},
	}
	view := abyssWikiView{Categories: categories}
	for _, category := range categories {
		view.Total += len(category.Entries)
	}
	return view
}

func abyssWikiGearEntries() []abyssWikiEntryView {
	catalog := content.AbyssGearCatalog()
	sort.Slice(catalog, func(i, j int) bool {
		if catalog[i].Slot != catalog[j].Slot {
			return catalog[i].Slot < catalog[j].Slot
		}
		return catalog[i].Name < catalog[j].Name
	})
	entries := make([]abyssWikiEntryView, 0, len(catalog))
	for _, gear := range catalog {
		description := gear.Lore
		if description == "" {
			description = content.ItemEffectDescription(gear.Special)
		}
		if description == "" {
			description = "An authored Abyss equipment catalog entry."
		}
		stats := abyssWikiStats(gear.Stats)
		meta := fmt.Sprintf(
			"%s · %s · CR %.1f · %s",
			gear.Slot,
			gear.Rarity.String(),
			gear.CombatRating(),
			stats,
		)
		search := strings.Join([]string{
			gear.ID,
			gear.Name,
			string(gear.Slot),
			gear.Rarity.String(),
			string(gear.Special),
			gear.EffectiveSetID(),
			stats,
		}, " ")
		entries = append(entries, abyssWikiEntryView{
			ID:          gear.ID,
			Name:        gear.Name,
			Icon:        content.SlotIcon(gear.Slot),
			Kind:        gear.Rarity.String(),
			Meta:        meta,
			Description: description,
			Search:      strings.ToLower(search),
		})
	}
	return entries
}

func abyssWikiMobEntries() []abyssWikiEntryView {
	catalog := content.AbyssMobCatalog()
	sort.Slice(catalog, func(i, j int) bool {
		if catalog[i].Type != catalog[j].Type {
			return catalog[i].Type < catalog[j].Type
		}
		return catalog[i].Name < catalog[j].Name
	})
	entries := make([]abyssWikiEntryView, 0, len(catalog))
	for _, mob := range catalog {
		stats := abyssWikiStats(mob.Stats)
		meta := fmt.Sprintf("%s · base XP %d · %s", mob.Type, mob.RewardXP, stats)
		search := strings.Join([]string{mob.Name, string(mob.Type), stats}, " ")
		entries = append(entries, abyssWikiEntryView{
			ID:          strings.ToLower(strings.ReplaceAll(mob.Name, " ", "_")),
			Name:        mob.Name,
			Icon:        "☠",
			Kind:        string(mob.Type),
			Meta:        meta,
			Description: "Base encounter template; floor level and difficulty scale these values at spawn time.",
			Search:      strings.ToLower(search),
		})
	}
	return entries
}

func abyssWikiPactEntries() []abyssWikiEntryView {
	entries := make([]abyssWikiEntryView, 0, len(abyssPactCatalog))
	for _, pact := range abyssPactCatalog {
		meta := fmt.Sprintf(
			"+%d%% cache · danger ×%.2f",
			int(math.Round(pact.Reward*100)),
			pact.Danger,
		)
		search := strings.Join([]string{pact.Key, pact.Label, pact.Desc, meta}, " ")
		entries = append(entries, abyssWikiEntryView{
			ID:          pact.Key,
			Name:        pact.Label,
			Icon:        "⚖",
			Kind:        "Challenge pact",
			Meta:        meta,
			Description: pact.Desc,
			Search:      strings.ToLower(search),
		})
	}
	return entries
}

func abyssWikiAffixEntries() []abyssWikiEntryView {
	entries := make([]abyssWikiEntryView, 0, len(abyssDailyMods)+32)
	for _, key := range abyssDailyMods {
		label := abyssDailyAffixLabel(key)
		description := abyssDailyAffixDescriptions[key]
		entries = append(entries, abyssWikiEntryView{
			ID:          key,
			Name:        label,
			Icon:        "◈",
			Kind:        "Daily affix",
			Meta:        "Daily run modifier",
			Description: description,
			Search:      strings.ToLower(key + " " + label + " " + description),
		})
	}
	for _, effect := range content.ItemEffectCatalog() {
		description := content.ItemEffectDescription(effect)
		entries = append(entries, abyssWikiEntryView{
			ID:          string(effect),
			Name:        string(effect),
			Icon:        "◆",
			Kind:        "Item affix",
			Meta:        "Gear combat effect",
			Description: description,
			Search:      strings.ToLower(string(effect) + " " + description),
		})
	}
	for _, effect := range content.MobEffectCatalog() {
		entries = append(entries, abyssWikiEntryView{
			ID:          effect.Key,
			Name:        abyssWikiLabel(effect.Key),
			Icon:        effect.Icon,
			Kind:        "Monster affix",
			Meta:        effect.Tone,
			Description: effect.Description,
			Search: strings.ToLower(strings.Join([]string{
				effect.Key,
				effect.Tone,
				effect.Description,
			}, " ")),
		})
	}
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].Kind != entries[j].Kind {
			return entries[i].Kind < entries[j].Kind
		}
		return entries[i].Name < entries[j].Name
	})
	return entries
}

func abyssWikiStats(stats content.Stats) string {
	values := []struct {
		label string
		value int
	}{
		{label: "HP", value: stats.HP},
		{label: "MNA", value: stats.MNA},
		{label: "STR", value: stats.STR},
		{label: "DEF", value: stats.DEF},
		{label: "SPD", value: stats.SPD},
		{label: "CRT", value: stats.CRT},
		{label: "DGE", value: stats.DGE},
		{label: "LCK", value: stats.LCK},
		{label: "INT", value: stats.INT},
		{label: "STA", value: stats.STA},
	}
	parts := make([]string, 0, len(values))
	for _, value := range values {
		if value.value != 0 {
			parts = append(parts, fmt.Sprintf("%s %+d", value.label, value.value))
		}
	}
	if len(parts) == 0 {
		return "no base combat stats"
	}
	return strings.Join(parts, " · ")
}

func abyssWikiLabel(key string) string {
	label := strings.ReplaceAll(key, "-", " ")
	if label == "" {
		return "Unknown"
	}
	return strings.ToUpper(label[:1]) + label[1:]
}
