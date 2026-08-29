package content

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

const (
	PixelArtColumns = 14
	PixelArtRows    = 12
)

// PixelArtEntry is one permanent visual identity in the game catalog. Page,
// Column and Row form a collision-free coordinate assigned from the stable Key.
type PixelArtEntry struct {
	Key     string
	Name    string
	Kind    string
	Family  string
	Variant string
	Element string
	Rarity  string
	Page    int
	Column  int
	Row     int
	Asset   string
}

var (
	pixelArtOnce  sync.Once
	pixelArtList  []PixelArtEntry
	pixelArtByKey map[string]PixelArtEntry
)

// GearPixelArtFamily returns the semantic atlas family for a gear slot.
func GearPixelArtFamily(slot GearSlot) string {
	switch slot {
	case SlotOffHand:
		return "offhands"
	case SlotRanged:
		return "ranged"
	case SlotRelic:
		return "relics"
	case SlotArtifact:
		return "artifacts"
	case SlotSoul:
		return "souls"
	case SlotAura:
		return "auras"
	case SlotCharm:
		return "charms"
	case SlotMount:
		return "mounts"
	case SlotCompanion:
		return "companions"
	case SlotPet1, SlotPet2:
		return "pets"
	case SlotEmblem1, SlotEmblem2:
		return "emblems"
	case SlotBanner:
		return "banners"
	case SlotTotem:
		return "totems"
	default:
		return "items"
	}
}

func buildPixelArtCatalog() {
	initSkills()
	initUltimateSkills()
	initMobs()

	entries := make([]PixelArtEntry, 0, len(allSkills)+len(allUltimateSkills)+len(allGear)+len(baseMobs)+64)
	for _, gear := range GearAppearanceCatalog() {
		entries = append(entries, PixelArtEntry{
			Key: "item:" + gear.ID, Name: gear.Name, Kind: "gear",
			Family: GearPixelArtFamily(gear.Slot), Variant: string(gear.Slot),
			Element: string(gear.Element), Rarity: gear.Rarity.String(),
		})
	}
	consumables := append(append([]Consumable(nil), allConsumables...), abyssExclusiveConsumables...)
	for _, consumable := range consumables {
		entries = append(entries, PixelArtEntry{
			Key: "item:" + consumable.ID, Name: consumable.Name, Kind: "consumable",
			Family: "items", Variant: string(consumable.Type),
		})
	}
	for index, artifact := range corruptedArtifacts {
		entries = append(entries, PixelArtEntry{
			Key:  fmt.Sprintf("item:artifact:%d:%s", index, artifact.Name),
			Name: artifact.Name, Kind: "artifact", Family: "artifacts", Variant: "Corrupted",
		})
	}
	for _, skill := range allSkills {
		entries = append(entries, PixelArtEntry{
			Key: "skill:" + skill.ID, Name: skill.Name, Kind: "skill", Family: "skills",
			Variant: string(skill.Type), Element: string(skill.Element), Rarity: skill.Rarity.String(),
		})
	}
	for _, ultimate := range allUltimateSkills {
		entries = append(entries, PixelArtEntry{
			Key: "ultimate:" + ultimate.ID, Name: ultimate.Name, Kind: "ultimate",
			Family: "skills", Variant: "Ultimate", Rarity: ultimate.Rarity.String(),
		})
	}
	seenMobs := make(map[string]struct{}, len(baseMobs)+len(TreasureGoblinNames))
	for index, mob := range AbyssMobCatalog() {
		if _, exists := seenMobs[mob.Name]; exists {
			continue
		}
		seenMobs[mob.Name] = struct{}{}
		key := fmt.Sprintf("monster:%03d", index)
		family := "creatures"
		if mob.Type == MobBoss || mob.Type == MobLegendary {
			family = "bosses"
		}
		entries = append(entries, PixelArtEntry{
			Key: key, Name: mob.Name, Kind: "monster", Family: family,
			Variant: string(mob.Type), Element: string(mob.Element),
		})
	}
	for _, mobType := range []MobType{MobCommon, MobEliteMinion, MobElite, MobMiniboss, MobBoss, MobLegendary} {
		entries = append(entries, PixelArtEntry{
			Key: "pet-type:" + string(mobType), Name: string(mobType) + " companion",
			Kind: "pet", Family: "pets", Variant: string(mobType),
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Family != entries[j].Family {
			return entries[i].Family < entries[j].Family
		}
		return entries[i].Key < entries[j].Key
	})

	byFamily := make(map[string]int, 17)
	byKey := make(map[string]PixelArtEntry, len(entries))
	deduplicated := entries[:0]
	for _, entry := range entries {
		if entry.Key == "" {
			continue
		}
		if _, exists := byKey[entry.Key]; exists {
			continue
		}
		index := byFamily[entry.Family]
		entry.Page = index / (PixelArtColumns * PixelArtRows)
		cell := index % (PixelArtColumns * PixelArtRows)
		entry.Column = cell % PixelArtColumns
		entry.Row = cell / PixelArtColumns
		entry.Asset = fmt.Sprintf("/static/abyss_catalog_%s_p%02d.png", entry.Family, entry.Page)
		byFamily[entry.Family]++
		byKey[entry.Key] = entry
		deduplicated = append(deduplicated, entry)
	}
	pixelArtList = append([]PixelArtEntry(nil), deduplicated...)
	pixelArtByKey = byKey
}

// PixelArtCatalog returns a detached catalog sorted by family then identity.
func PixelArtCatalog() []PixelArtEntry {
	pixelArtOnce.Do(buildPixelArtCatalog)
	return append([]PixelArtEntry(nil), pixelArtList...)
}

// PixelArtByKey resolves a stable catalog identity to its exact generated cell.
func PixelArtByKey(key string) (PixelArtEntry, bool) {
	pixelArtOnce.Do(buildPixelArtCatalog)
	entry, ok := pixelArtByKey[key]
	return entry, ok
}

// MonsterPixelArtKey resolves runtime affix prefixes back to the authored
// monster identity. Exact names win; otherwise the longest contained catalog
// name is used so variants such as "Nemesis: Frost Lich" keep the lich art.
func MonsterPixelArtKey(name string) string {
	direct := "monster:" + name
	if _, ok := PixelArtByKey(direct); ok {
		return direct
	}
	lowerName := strings.ToLower(name)
	bestKey, bestLength := "", 0
	for _, entry := range PixelArtCatalog() {
		if entry.Kind != "monster" || len(entry.Name) <= bestLength {
			continue
		}
		if strings.Contains(lowerName, strings.ToLower(entry.Name)) {
			bestKey, bestLength = entry.Key, len(entry.Name)
		}
	}
	return bestKey
}

// PixelArtKeyByName resolves the exact name of a catalog entity of one kind.
// It is intended for legacy payloads which predate storing stable catalog IDs.
func PixelArtKeyByName(kind, name string) string {
	for _, entry := range PixelArtCatalog() {
		if entry.Kind == kind && entry.Name == name {
			return entry.Key
		}
	}
	return ""
}
