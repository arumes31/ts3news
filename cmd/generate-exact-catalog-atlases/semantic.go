package main

import (
	"image"
	"strings"

	"ts3news/internal/content"
)

// These coordinates refer to visually reviewed source art, not an ID hash.
// Hash selection remains appropriate only among generic, same-slot variants.
func semanticCell(entry content.PixelArtEntry) (image.Point, bool) {
	name := strings.ToLower(entry.Name)
	has := func(words ...string) bool {
		for _, word := range words {
			if strings.Contains(name, word) {
				return true
			}
		}
		return false
	}
	cell := func(x, y int) (image.Point, bool) { return image.Pt(x, y), true }
	switch entry.Family {
	case "ranged":
		if has("repeater", "crossbow") {
			return cell(0, 1)
		}
		if has("quiver", "arrow") {
			return cell(4, 9)
		}
		return cell(0, 0)
	case "offhands":
		if has("mirror", "voidglass") {
			return cell(2, 0)
		}
		if has("divine", "aegis", "ward", "wall", "shield", "verdict", "bastion", "bulwark", "sentinel") {
			return cell(1, 1)
		}
	case "auras":
		switch {
		case has("blood"):
			return cell(13, 1)
		case has("fire", "rekindl", "ember"):
			return cell(0, 0)
		case has("drown", "tide", "gale"):
			return cell(9, 0)
		case has("sun", "faith", "halo", "cleans"):
			return cell(5, 0)
		}
	case "relics":
		switch {
		case has("grail"):
			return cell(7, 0)
		case has("urn"):
			return cell(4, 0)
		case has("heart"):
			return cell(3, 0)
		case has("void", "essence"):
			return cell(5, 0)
		case has("reliquary"):
			return cell(0, 0)
		}
	case "charms":
		if has("blood") {
			return cell(0, 0)
		}
		if has("star") {
			return cell(0, 1)
		}
	}
	if entry.Family == "skills" {
		// The action takes precedence over its elemental adjective.
		switch {
		case has("basic attack"):
			return cell(0, 2)
		case has("defend"):
			return cell(1, 1)
		case has("focus target"):
			return cell(9, 1)
		case has("free-for-all"):
			return cell(0, 4)
		case has("heal", "mend", "restor", "rejuven"):
			return cell(0, 1)
		case has("shield", "guard", "ward", "barrier", "aegis"):
			return cell(1, 1)
		case has("drain", "leech", "siphon", "absorb"):
			return cell(11, 0)
		case has("curse", "hex", "terror", "terrify", "poison"):
			return cell(10, 0)
		case has("sunder", "quake", "seismic"):
			return cell(3, 0)
		case has("pack assault"):
			return cell(0, 4)
		case has("mark", "sighting"):
			return cell(9, 1)
		case has("time", "temporal"):
			return cell(4, 1)
		case has("rune", "runic"):
			return cell(2, 1)
		case has("volatile", "mixture"):
			return cell(5, 0)
		case has("catalytic"):
			return cell(0, 0)
		case has("summon"):
			return cell(0, 3)
		case has("silenc", "blind", "stun", "paralyz"):
			return cell(12, 1)
		case has("arrow", "shot", "volley", "barrage"):
			return cell(2, 2)
		case has("fiery", "fire", "flam", "inciner", "ignit", "burn", "ember"):
			return cell(0, 0)
		case has("icy", "ice", "frost", "freez"):
			return cell(1, 0)
		case has("storm", "electr", "lightning", "thunder"):
			return cell(2, 0)
		case has("wind", "air", "gale"):
			return cell(4, 0)
		case has("toxic", "venom", "corrupt"):
			return cell(5, 0)
		case has("shadow", "void", "dark", "annihilat"):
			return cell(9, 0)
		case has("blood", "rage", "ravag", "crimson"):
			return cell(11, 0)
		case has("holy", "divin", "puri", "bless", "inspir"):
			return cell(8, 0)
		case has("arcane", "spark", "magic", "channel", "transcend"):
			return cell(2, 1)
		case has("earth", "crush", "smash", "pulver", "shatter"):
			return cell(3, 0)
		case has("punch", "bash"):
			return cell(5, 2)
		case has("slash", "strike", "thrust", "cleav", "rend", "shred", "pierc", "onslaught"):
			return cell(0, 2)
		default:
			return cell(3, 1) // neutral magical energy, never an unrelated item/creature
		}
	}
	if entry.Family != "items" {
		return image.Point{}, false
	}
	if entry.Kind == "consumable" {
		switch {
		case has("scroll"):
			return cell(0, 7)
		case has("charm"):
			return cell(3, 5)
		case has("mana", "intellect"):
			return cell(1, 6)
		case has("rejuven", "revive"):
			return cell(3, 6)
		case has("strength"):
			return cell(10, 6)
		case has("skin"):
			return cell(11, 6)
		case has("luck"):
			return cell(4, 6)
		case has("speed"):
			return cell(2, 6)
		case has("suppress"):
			return cell(5, 6)
		default:
			return cell(0, 6)
		}
	}
	switch strings.ToLower(entry.Variant) {
	case "mainhand":
		switch {
		case has("hammer"):
			return cell(9, 0)
		case has("crossbow", "repeater"):
			return cell(11, 0)
		case has("bow"):
			return cell(10, 0)
		case has("staff"):
			return cell(12, 0)
		case has("axe", "cleaver"):
			return cell(6, 0)
		case has("sword", "claymore", "blade", "edge"):
			return cell(0, 0)
		}
	case "head":
		switch {
		case has("crown", "tiara", "diadem"):
			return cell(10, 1)
		case has("hood", "bandana"):
			return cell(8, 1)
		case has("helm"):
			return cell(4, 1)
		}
	case "back":
		return cell(4, 2)
	case "chest":
		if has("robe", "vestment") {
			return cell(10, 2)
		}
		if has("harness", "leather") {
			return cell(8, 2)
		}
		return cell(0, 2)
	case "trinket1", "trinket2":
		switch {
		case has("key"):
			return cell(1, 9)
		case has("watch", "chrono"):
			return cell(10, 5)
		case has("orb", "battery"):
			return cell(12, 11)
		case has("book", "tome"):
			return cell(2, 5)
		default:
			return cell(3, 5)
		}
	}
	return image.Point{}, false
}

type iconSource struct {
	asset                      string
	column, row, columns, rows int
	// Combat sheets have measured nonuniform row boundaries.
	rowBounds []int
}

func (s iconSource) bounds(size image.Rectangle) image.Rectangle {
	x0, x1 := s.column*size.Dx()/s.columns, (s.column+1)*size.Dx()/s.columns
	y0, y1 := s.row*size.Dy()/s.rows, (s.row+1)*size.Dy()/s.rows
	if len(s.rowBounds) > 0 {
		y0, y1 = s.rowBounds[s.row], s.rowBounds[s.row+1]
	}
	return image.Rect(x0, y0, x1, y1)
}

func specialSource(entry content.PixelArtEntry) (iconSource, bool) {
	name, slot := strings.ToLower(entry.Name), strings.ToLower(entry.Variant)
	supplement := func(column, row int) (iconSource, bool) {
		return iconSource{asset: "abyss_equipment_supplement_v1.png", column: column, row: row, columns: 4, rows: 4}, true
	}
	if entry.Kind == "monster" {
		return monsterSource(name)
	}
	if entry.Kind == "artifact" {
		if strings.Contains(name, "soul") {
			return iconSource{asset: "abyss_atlas_souls.png", column: 2, columns: 14, rows: 12}, true
		}
		if strings.Contains(name, "heart") {
			return iconSource{asset: "abyss_atlas_skills.png", row: 7, columns: 14, rows: 12}, true
		}
		return iconSource{asset: "abyss_atlas_artifacts.png", column: 2, columns: 14, rows: 12}, true
	}
	if entry.Kind == "consumable" {
		if strings.Contains(name, "repair") {
			return supplement(1, 3)
		}
		if strings.Contains(name, "feather") {
			return supplement(2, 3)
		}
	}
	if entry.Kind != "gear" {
		return iconSource{}, false
	}
	switch slot {
	case "legs":
		if strings.Contains(name, "wrap") {
			return supplement(1, 0)
		}
		return supplement(0, 0)
	case "wrists":
		return supplement(2, 0)
	case "shoulders":
		return supplement(0, 1)
	case "waist":
		if strings.Contains(name, "pouch") {
			return supplement(0, 3)
		}
	case "mainhand", "offhand", "ranged":
		for _, weapon := range []struct {
			word        string
			column, row int
		}{
			{"dagger", 0, 2}, {"spear", 2, 1}, {"scythe", 3, 1},
			{"wand", 1, 2}, {"mace", 2, 2}, {"scepter", 3, 2},
		} {
			if strings.Contains(name, weapon.word) {
				return supplement(weapon.column, weapon.row)
			}
		}
	case "relic":
		if strings.Contains(name, "spores") {
			return iconSource{asset: "abyss_atlas_skills.png", column: 5, row: 8, columns: 14, rows: 12}, true
		}
		if strings.Contains(name, "doll") {
			return supplement(3, 3)
		}
	}
	return iconSource{}, false
}

// Monster portraits use the same authored anatomy as their combat idle pose.
// Prefixes (Ghostly Rat, Undead Wolf) never replace the underlying species.
func monsterSource(name string) (iconSource, bool) {
	name = strings.ReplaceAll(name, "_", " ")
	roles := []int{0, 158, 318, 476, 638, 783, 924, 1086, 1254}
	creatures := []int{0, 144, 298, 441, 617, 789, 947, 1076, 1254}
	bestiary := []int{0, 143, 281, 435, 591, 758, 900, 1056, 1254}
	bosses := []int{0, 155, 312, 466, 625, 786, 941, 1085, 1254}
	for _, rule := range []struct {
		word, atlas string
		row         int
		bounds      []int
	}{
		{"gorgoroth", "bosses", 0, bosses}, {"malakor", "bosses", 1, bosses},
		{"azazoth", "bosses", 2, bosses}, {"abyssus", "bosses", 3, bosses},
		{"scribe without eyes", "bosses", 4, bosses}, {"mnemos", "bosses", 5, bosses},
		{"abyss that remembers", "bosses", 6, bosses},
		{"rat", "bestiary", 0, bestiary}, {"bat", "bestiary", 1, bestiary},
		{"zombie", "bestiary", 2, bestiary}, {"skeleton", "bestiary", 3, bestiary},
		{"troll", "bestiary", 4, bestiary}, {"kraken", "bestiary", 5, bestiary},
		{"void lord", "bestiary", 6, bestiary}, {"chronos", "bestiary", 7, bestiary},
		{"slime", "creatures", 0, creatures}, {"spider", "creatures", 1, creatures},
		{"goblin", "creatures", 2, creatures}, {"orc", "creatures", 7, creatures},
		{"behemoth", "creatures", 7, creatures},
		{"wolf", "roles", 5, roles}, {"lich", "roles", 6, roles}, {"dragon", "roles", 7, roles},
		{"assassin", "roles", 4, roles}, {"guard", "roles", 0, roles}, {"knight", "roles", 0, roles},
		{"gatekeeper", "bosses", 7, bosses},
	} {
		if strings.Contains(name, rule.word) {
			return iconSource{asset: "abyss_combat_" + rule.atlas + "_v2.png", row: rule.row, columns: 8, rows: 8, rowBounds: rule.bounds}, true
		}
	}
	return iconSource{}, false
}
