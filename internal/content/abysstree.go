package content

// The Abyss Skill Web: a Path-of-Exile-style passive tree with exactly 1000
// allocatable nodes arranged as a radial web around a free root. Six archetype
// sectors (War, Vitality, Shadow, Arcane, Fortune, Void) each span six angular
// lanes across 27 rings; sparse lateral links inside every ring and bridge
// notables between sectors create many alternative paths to any node.
//
// The web is generated deterministically (fixed-seed PCG) so node IDs, layout
// and effects are stable across restarts and identical for every player.

import (
	"fmt"
	"hash/fnv"
	"math"
	"math/rand/v2"
	"sort"
	"sync"
	"time"
)

// TreeNode is one allocatable node of the Abyss skill web.
type TreeNode struct {
	ID     int                `json:"id"`
	Ring   int                `json:"ring"`
	Slot   int                `json:"slot"`
	Sector int                `json:"sector"`
	Type   string             `json:"type"` // small | notable | keystone | bridge
	Name   string             `json:"name"`
	Desc   string             `json:"desc"`
	Stats  Stats              `json:"stats"`
	Pct    map[string]float64 `json:"pct,omitempty"`
	X      float64            `json:"x"`
	Y      float64            `json:"y"`
}

// TreeBonus is the summed effect of a set of allocated nodes.
type TreeBonus struct {
	Stats Stats
	Pct   map[string]float64
}

// ApplyCombatPct applies the combat %-multipliers (str_pct/hp_pct/spd_pct/
// int_pct) to a stat block; economy keys are consumed by their own hooks.
func (tb TreeBonus) ApplyCombatPct(s Stats) Stats {
	if v := tb.Pct["str_pct"]; v != 0 {
		s.STR = int(float64(s.STR) * (1 + v))
	}
	if v := tb.Pct["hp_pct"]; v != 0 {
		s.HP = int(float64(s.HP) * (1 + v))
	}
	if v := tb.Pct["spd_pct"]; v != 0 {
		s.SPD = int(float64(s.SPD) * (1 + v))
	}
	if v := tb.Pct["int_pct"]; v != 0 {
		s.INT = int(float64(s.INT) * (1 + v))
	}

	// Unique Keystones Conversion modifiers
	if v := tb.Pct["str_to_spd"]; v != 0 {
		converted := int(float64(s.STR) * v)
		s.SPD += converted
		s.STR -= converted
	}
	if v := tb.Pct["hp_to_def"]; v != 0 {
		converted := int(float64(s.HP) * v)
		s.DEF += converted / 10
		s.HP -= converted
	}
	if v := tb.Pct["spd_to_dge"]; v != 0 {
		converted := int(float64(s.SPD) * v)
		s.DGE += converted
		s.SPD -= converted
	}
	if v := tb.Pct["int_to_mna"]; v != 0 {
		converted := int(float64(s.INT) * v)
		s.MNA += converted * 5
		s.INT -= converted
	}

	// Crown Keystone: Limit Break (Item 73)
	if v := tb.Pct["limit_break"]; v != 0 {
		mult := 1.0 + v
		s.STR = int(float64(s.STR) * mult)
		s.HP = int(float64(s.HP) * mult)
		s.DEF = int(float64(s.DEF) * mult)
		s.SPD = int(float64(s.SPD) * mult)
		s.INT = int(float64(s.INT) * mult)
	}

	return s
}

// AbyssTreeData is the whole generated web: nodes plus adjacency. Node 0 is
// the virtual root: never allocatable, always counted as allocated.
type AbyssTreeData struct {
	Nodes []TreeNode    // 1000 nodes, IDs 1..1000
	Adj   map[int][]int // undirected adjacency, includes root (0) edges
	byID  map[int]*TreeNode
}

// Node returns a node by ID (nil for the root or unknown IDs).
func (t *AbyssTreeData) Node(id int) *TreeNode { return t.byID[id] }

// BonusFor sums the stats and percent effects of the given allocated node IDs.
func (t *AbyssTreeData) BonusFor(ids []int) TreeBonus {
	tb := TreeBonus{Pct: map[string]float64{}}
	allocated := map[int]bool{}
	var uniqueIDs []int
	for _, id := range ids {
		if !allocated[id] {
			allocated[id] = true
			uniqueIDs = append(uniqueIDs, id)
		}
	}
	for _, id := range uniqueIDs {
		n := t.byID[id]
		if n == nil {
			continue
		}
		tb.Stats = tb.Stats.Add(n.Stats)
		for k, v := range n.Pct {
			tb.Pct[k] += v
		}

		// Jewel Sockets adjacent modifiers (Item 34)
		if n.Type == "socket" {
			for _, neighborID := range t.Adj[n.ID] {
				if allocated[neighborID] {
					tb.Stats.HP += 15
					tb.Stats.STR += 5
					tb.Stats.INT += 5
				}
			}
		}

		// Dynamic temporal day-of-week node (Item 39)
		if n.Name == "⏳ Temporal Shift" {
			weekday := time.Now().Weekday()
			if weekday == time.Saturday || weekday == time.Sunday {
				tb.Pct["gold_find"] += 0.15
			} else {
				tb.Pct["xp_gain"] += 0.10
			}
		}
	}
	return tb
}

const (
	treeSectors = 6
	treeLanes   = 6 // angular lanes per sector
	treeRings   = 26
	treeSlots   = treeSectors * treeLanes // 36 angular slots per ring

	// 936 grid + 40 keystones + 24 bridges = exactly 1000 nodes.
	treeGridNodes  = treeRings * treeSlots // 936, IDs 1..936
	treeKeystoneN  = 40                    // IDs 937..976
	treeFirstKeyID = treeGridNodes + 1
)

// Skill-granting tree nodes: allocating one of these adds an active skill to
// the player's combat kit. The IDs live here, next to the tree definition, so
// the combat loop doesn't hardcode raw grid IDs (see getSkills in the bot).
const (
	NodeSkillEarthquake   = 588 // grants S_EQ; grid ring 17, slot 11 → "🔮 Spellweaver (Earthquake)"
	NodeSkillArcaneShield = 466 // grants S_AS; grid ring 13, slot 33 → "🧪 Alchemist's Ritual (Arcane Shield)"
)

var treeSectorNames = [treeSectors]string{"War", "Vitality", "Shadow", "Arcane", "Fortune", "Void"}
var treeSectorIcons = [treeSectors]string{"⚔️", "❤️", "🌫️", "🔮", "🍀", "🕳️"}

// treeRingAdjectives is a 26-step power ladder, one word per ring. Combined
// with a per-lane noun it makes every grid node's name provably unique:
// (ring adjective) × (lane noun) collide for no two nodes.
var treeRingAdjectives = [treeRings]string{
	"Faint", "Dim", "Pale", "Soft", "Quiet", "Keen", "Steady", "Bold",
	"Fierce", "Grim", "Wild", "Deep", "Dire", "Savage", "Ancient", "Umbral",
	"Raging", "Hallowed", "Sovereign", "Mythic", "Eternal", "Primal", "Zenith",
	"Transcendent", "Apex", "Absolute",
}

// treeLaneNouns holds six thematic nouns per sector — 36 in total, all
// distinct, so names never collide across sectors either.
var treeLaneNouns = [treeSectors][treeLanes]string{
	{"Edge", "Cleave", "Warcry", "Onslaught", "Rampage", "Bloodlust"},
	{"Heartwood", "Lifeblood", "Bastion", "Vigor", "Stoneflesh", "Endurance"},
	{"Quickstep", "Phantom Stride", "Veil", "Ambush", "Silhouette", "Fleetfoot"},
	{"Sigilweave", "Manaflow", "Runeglow", "Mindspark", "Spellthread", "Aether"},
	{"Windfall", "Gilding", "Serendipity", "Coinflip", "Prospect", "Goldvein"},
	{"Umbra", "Nadir", "Voidtouch", "Abyssal Whisper", "Dark Communion", "Hollow"},
}

// treeBridgeTiers names each bridge ring so pacts on the same boundary stay
// unique ("First Pact of…", "Second Pact of…", …).
var treeBridgeTiers = map[int]string{7: "First", 14: "Second", 21: "Third", treeRings: "Final"}

var (
	abyssTreeOnce sync.Once
	abyssTreeData *AbyssTreeData
)

// AbyssTree returns the generated skill web (built once, deterministic).
func AbyssTree() *AbyssTreeData {
	abyssTreeOnce.Do(func() { abyssTreeData = buildAbyssTree() })
	return abyssTreeData
}

// gridID maps (ring 1..26, slot 0..35) to node IDs 1..936.
func gridID(ring, slot int) int { return 1 + (ring-1)*treeSlots + slot }

// treeKeystoneDef declares one keystone: 36 rim keystones cap every lane of
// every sector (index = sector*6 + lane); 4 crown keystones sit beyond the rim
// at sector boundaries and require an adjacent rim keystone to reach.
type treeKeystoneDef struct {
	name string
	pct  map[string]float64
}

// treeRimKeystones — one per lane, sector-major order (War, Vitality, Shadow,
// Arcane, Fortune, Void × 6 lanes each). Every name is unique.
var treeRimKeystones = [treeSlots]treeKeystoneDef{
	// ⚔️ War
	{"Juggernaut", map[string]float64{"str_pct": 0.25, "spd_pct": -0.10}},
	{"Berserker's Oath", map[string]float64{"str_pct": 0.15, "str_to_spd": 0.15}},
	{"Warbringer", map[string]float64{"str_pct": 0.10, "xp_gain": 0.05}},
	{"Executioner", map[string]float64{"str_pct": 0.12, "gold_find": 0.06}},
	{"Bloodforged", map[string]float64{"str_pct": 0.10, "material_yield": 0.10}},
	{"Colossus Slayer", map[string]float64{"str_pct": 0.08, "loot_find": 0.08}},
	// ❤️ Vitality
	{"Colossus", map[string]float64{"hp_pct": 0.30}},
	{"Undying", map[string]float64{"hp_pct": 0.20, "xp_gain": 0.05}},
	{"Ironroot", map[string]float64{"hp_pct": 0.15, "material_yield": 0.10}},
	{"Heartspring", map[string]float64{"hp_pct": 0.12, "escrow_bonus": 0.08}},
	{"Titan's Blood", map[string]float64{"hp_pct": 0.18, "hp_to_def": 0.10}},
	{"Warden's Resolve", map[string]float64{"hp_pct": 0.10, "token_gain": 0.10}},
	// 🌫️ Shadow
	{"Phantom", map[string]float64{"spd_pct": 0.20}},
	{"Ghostwalk", map[string]float64{"spd_pct": 0.12, "spd_to_dge": 0.15}},
	{"Night's Edge", map[string]float64{"spd_pct": 0.10, "gold_find": 0.10}},
	{"Umbral Dance", map[string]float64{"spd_pct": 0.15, "hp_pct": -0.05}},
	{"Silent Fortune", map[string]float64{"spd_pct": 0.08, "escrow_bonus": 0.10}},
	{"Shadowbroker", map[string]float64{"spd_pct": 0.06, "token_gain": 0.12}},
	// 🔮 Arcane
	{"Archmage", map[string]float64{"int_pct": 0.25}},
	{"Mindstorm", map[string]float64{"int_pct": 0.15, "xp_gain": 0.10}},
	{"Runebinder", map[string]float64{"int_pct": 0.12, "material_yield": 0.12}},
	{"Aetherflow", map[string]float64{"int_pct": 0.15, "str_pct": -0.05}},
	{"Scryer's Insight", map[string]float64{"int_pct": 0.10, "loot_find": 0.10}},
	{"Manaforged", map[string]float64{"int_pct": 0.12, "int_to_mna": 0.20}},
	// 🍀 Fortune
	{"Midas", map[string]float64{"gold_find": 0.15, "loot_find": 0.05}},
	{"Alchemy of the Soul", map[string]float64{"xp_to_gold": 0.50, "gold_find": 0.10}},
	{"Treasure Sense", map[string]float64{"loot_find": 0.15}},
	{"Golden Tongue", map[string]float64{"gold_find": 0.10, "token_gain": 0.10}},
	{"Prospector", map[string]float64{"material_yield": 0.12, "gold_find": 0.08}},
	{"Fate's Favor", map[string]float64{"loot_find": 0.08, "xp_gain": 0.08}},
	// 🕳️ Void
	{"Voidheart", map[string]float64{"escrow_bonus": 0.10, "token_gain": 0.10}},
	{"Abyssal Pact", map[string]float64{"escrow_bonus": 0.15, "hp_pct": -0.05}},
	{"Deep Hunger", map[string]float64{"loot_find": 0.12, "gold_find": -0.05}},
	{"Nadir Crown", map[string]float64{"token_gain": 0.12, "xp_gain": 0.06}},
	{"Hollow King", map[string]float64{"escrow_bonus": 0.10, "material_yield": 0.10}},
	{"Whisper of the Depths", map[string]float64{"xp_gain": 0.10, "loot_find": 0.08}},
}

// treeCrownKeystones sit beyond the rim at the first four sector boundaries,
// reachable only through an adjacent rim keystone — the web's summit picks.
var treeCrownKeystones = [4]treeKeystoneDef{
	{"👑 Limit Break", map[string]float64{"str_pct": 0.20, "hp_pct": 0.20, "limit_break": 0.15}},
	{"👑 The Thousandth Star", map[string]float64{"str_pct": 0.05, "hp_pct": 0.05, "spd_pct": 0.05, "int_pct": 0.05}},
	{"👑 Duskbringer", map[string]float64{"spd_pct": 0.10, "int_pct": 0.10}},
	{"👑 Deepmiser", map[string]float64{"gold_find": 0.10, "escrow_bonus": 0.10}},
}

// pctBrief renders a keystone's percent effects deterministically (sorted keys).
func pctBrief(pct map[string]float64) string {
	keys := make([]string, 0, len(pct))
	for k := range pct {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := ""
	for _, k := range keys {
		if out != "" {
			out += " · "
		}
		out += fmt.Sprintf("%+.0f%% %s", pct[k]*100, treePctLabel(k))
	}
	return out
}

func buildAbyssTree() *AbyssTreeData {
	t := &AbyssTreeData{Adj: map[int][]int{}, byID: map[int]*TreeNode{}}
	addEdge := func(a, b int) {
		t.Adj[a] = append(t.Adj[a], b)
		t.Adj[b] = append(t.Adj[b], a)
	}

	// --- 972 grid nodes -----------------------------------------------------
	for ring := 1; ring <= treeRings; ring++ {
		for slot := 0; slot < treeSlots; slot++ {
			sector := slot / treeLanes
			id := gridID(ring, slot)
			notable := ring%7 == 0 && slot%3 == 1
			n := TreeNode{
				ID: id, Ring: ring, Slot: slot, Sector: sector,
				Type:  "small",
				Stats: treeSmallStats(sector, ring, slot, id),
			}

			// Intercept custom Jewel Sockets and Special Notables (Item 34, 36, 37, 39, 40, 45)
			isCustom := false
			if ring == 12 && slot%6 == 3 {
				n.Type = "socket"
				n.Stats = Stats{}
				isCustom = true
			} else if ring == 18 && slot == 20 {
				n.Type = "notable"
				n.Stats = n.Stats.Scaled(3)
				n.Pct = map[string]float64{"ult_cooldown": 0.10, "ult_damage": 0.15}
				isCustom = true
			} else if ring == 16 && slot == 16 {
				n.Type = "notable"
				n.Stats = n.Stats.Scaled(3)
				n.Pct = map[string]float64{"def_to_lifesteal": 0.01}
				isCustom = true
			} else if ring == 15 && slot == 27 {
				n.Type = "notable"
				n.Stats = n.Stats.Scaled(3)
				n.Pct = map[string]float64{} // Dynamic temporal shift
				isCustom = true
			} else if ring == 14 && slot == 9 {
				n.Type = "notable"
				n.Stats = n.Stats.Scaled(3)
				n.Pct = map[string]float64{"pet_betrayal_reduce": 0.02, "pet_damage_pct": 0.20}
				isCustom = true
			} else if ring == 18 && slot == 4 {
				n.Type = "notable"
				n.Stats = n.Stats.Scaled(3)
				n.Pct = map[string]float64{"stun_immunity": 1.0}
				isCustom = true
			} else if ring == 17 && slot == 11 {
				n.Type = "notable"
				n.Stats = n.Stats.Scaled(3)
				n.Pct = map[string]float64{"skill_damage": 0.20, "skill_mana_cost": 0.25}
				isCustom = true
			} else if ring == 13 && slot == 33 {
				n.Type = "notable"
				n.Stats = n.Stats.Scaled(3)
				n.Pct = map[string]float64{"consumable_save": 0.25}
				isCustom = true
			} else if ring == 21 && slot == 2 {
				n.Type = "notable"
				n.Stats = n.Stats.Scaled(3)
				n.Pct = map[string]float64{"skill_damage": 0.30, "hp_pct": -0.15}
				isCustom = true
			} else if ring == 22 && slot == 14 {
				n.Type = "notable"
				n.Stats = n.Stats.Scaled(3)
				n.Pct = map[string]float64{} // Specialist's Harmony
				isCustom = true
			} else if ring == 23 && slot == 25 {
				n.Type = "notable"
				n.Stats = n.Stats.Scaled(3)
				n.Pct = map[string]float64{} // Abyssal Attunement
				isCustom = true
			} else if ring == 24 && slot == 5 {
				n.Type = "notable"
				n.Stats = n.Stats.Scaled(3)
				n.Pct = map[string]float64{} // Prestige Focus
				isCustom = true
			} else if ring == 25 && slot == 18 {
				n.Type = "notable"
				n.Stats = n.Stats.Scaled(3)
				n.Pct = map[string]float64{} // Set Resonance
				isCustom = true
			} else if ring == 26 && slot == 31 {
				n.Type = "notable"
				n.Stats = n.Stats.Scaled(3)
				n.Pct = map[string]float64{} // Elemental Transmutation
				isCustom = true
			} else if ring == 20 && slot == 12 {
				n.Type = "notable"
				n.Stats = n.Stats.Scaled(3)
				n.Pct = map[string]float64{"xp_gain": 0.15} // Victor's Trophy
				isCustom = true
			}

			if !isCustom && notable {
				n.Type = "notable"
				n.Stats = n.Stats.Scaled(3)
				
				primaryPctKey := treeSectorPctKey(sector, slot)
				primaryPctVal := treeNotablePct(sector, ring)
				n.Pct = map[string]float64{primaryPctKey: primaryPctVal}
				
				rSec := rand.New(rand.NewPCG(uint64(id), 999))
				allKeys := []string{"str_pct", "hp_pct", "spd_pct", "int_pct", "gold_find", "loot_find", "escrow_bonus", "xp_gain", "token_gain", "material_yield"}
				rSec.Shuffle(len(allKeys), func(i, j int) { allKeys[i], allKeys[j] = allKeys[j], allKeys[i] })
				
				var secKey string
				for _, k := range allKeys {
					if k != primaryPctKey {
						secKey = k
						break
					}
				}
				secVal := primaryPctVal * (0.2 + 0.3*rSec.Float64())
				secVal = math.Round(secVal*1000) / 1000
				if secVal > 0.001 {
					n.Pct[secKey] = secVal
				}
			}
			n.Name, n.Desc = treeNodeText(&n)
			polar(&n)
			t.Nodes = append(t.Nodes, n)
		}
	}

	// --- 40 keystones ---------------------------------------------------------
	// 36 rim keystones: every lane of every sector ends in one, connected to
	// its lane's outermost grid node. IDs 937..972.
	for slot := 0; slot < treeSlots; slot++ {
		sec := slot / treeLanes
		ks := treeRimKeystones[slot]
		id := treeFirstKeyID + slot
		n := TreeNode{
			ID: id, Ring: treeRings + 1, Slot: slot, Sector: sec,
			Type: "keystone", Name: treeSectorIcons[sec] + " " + ks.name,
			Desc: pctBrief(ks.pct), Pct: ks.pct,
		}
		polar(&n)
		t.Nodes = append(t.Nodes, n)
		addEdge(id, gridID(treeRings, slot))
	}
	// 4 crown keystones beyond the rim at the first four sector boundaries,
	// each reachable only through its two adjacent rim keystones. IDs 973..976.
	for b := 0; b < 4; b++ {
		ks := treeCrownKeystones[b]
		leftSlot := b*treeLanes + (treeLanes - 1)
		rightSlot := (leftSlot + 1) % treeSlots
		id := treeFirstKeyID + treeSlots + b
		n := TreeNode{
			ID: id, Ring: treeRings + 2, Slot: leftSlot, Sector: b,
			Type: "keystone", Name: ks.name,
			Desc: pctBrief(ks.pct), Pct: ks.pct,
		}
		n.X, n.Y = polarXY(float64(treeRings)+2.2, float64(leftSlot)+0.5)
		t.Nodes = append(t.Nodes, n)
		addEdge(id, treeFirstKeyID+leftSlot)
		addEdge(id, treeFirstKeyID+rightSlot)
	}

	// --- 24 bridge notables between sectors ---------------------------------
	// Four per boundary at rings 7/14/21/26 — 24 total, bringing the web to
	// exactly 1000 nodes (936 grid + 40 keystones + 24 bridges).
	bridgeID := treeFirstKeyID + treeKeystoneN - 1 // 976; bridges take 977..1000
	addBridge := func(boundary, ring int) {
		bridgeID++
		leftSlot := boundary*treeLanes + (treeLanes - 1)
		rightSlot := (boundary*treeLanes + treeLanes) % treeSlots
		n := TreeNode{
			ID: bridgeID, Ring: ring, Slot: leftSlot, Sector: boundary,
			Type: "bridge",
			Name: fmt.Sprintf("🌉 %s Pact of %s and %s", treeBridgeTiers[ring], treeSectorNames[boundary], treeSectorNames[(boundary+1)%treeSectors]),
			Desc: "A bridge between disciplines: bonuses of both neighboring paths.",
			Stats: treeSmallStats(boundary, ring, leftSlot, bridgeID).
				Add(treeSmallStats((boundary+1)%treeSectors, ring, rightSlot, bridgeID+10000)).Scaled(2),
			Pct: map[string]float64{
				treeSectorPctKey(boundary, leftSlot):       math.Round(treeNotablePct(boundary, ring)*0.5*1000) / 1000,
				treeSectorPctKey((boundary+1)%treeSectors, rightSlot): math.Round(treeNotablePct((boundary+1)%treeSectors, ring)*0.5*1000) / 1000,
			},
		}
		if bridgeID == 999 {
			n.Name = "🌌 Secret Sanctuary"
			n.Desc = "Secret Sanctuary: Grants +20% loot find and +10% token gain."
			n.Pct = map[string]float64{"loot_find": 0.20, "token_gain": 0.10}
		}
		// Sits visually between the two sectors, half a slot outward.
		n.X, n.Y = polarXY(float64(ring)+0.5, float64(leftSlot)+0.5)
		t.Nodes = append(t.Nodes, n)
		addEdge(bridgeID, gridID(ring, leftSlot))
		addEdge(bridgeID, gridID(ring, rightSlot))
	}
	for b := 0; b < treeSectors; b++ {
		for _, ring := range []int{7, 14, 21, treeRings} {
			addBridge(b, ring)
		}
	}

	// --- Grid edges ----------------------------------------------------------
	// Root → two center lanes of every sector (12 starting nodes).
	for sec := 0; sec < treeSectors; sec++ {
		addEdge(0, gridID(1, sec*treeLanes+2))
		addEdge(0, gridID(1, sec*treeLanes+3))
	}
	for ring := 1; ring <= treeRings; ring++ {
		for slot := 0; slot < treeSlots; slot++ {
			rel := slot % treeLanes
			sec := slot / treeLanes

			// Radial spokes based on branch active states (with Ring 10 Choke Bottlenecks)
			if ring < treeRings {
				chokeSlot := sec*treeLanes + 2
				chokeID := gridID(10, chokeSlot)

				if ring == 9 {
					// All slots in Ring 9 connect only to the Ring 10 choke node
					addEdge(gridID(9, slot), chokeID)
				} else if ring == 10 {
					// Connect Ring 10 choke node to all other slots in Ring 10
					if slot == chokeSlot {
						for s := sec * treeLanes; s < (sec+1)*treeLanes; s++ {
							if s != chokeSlot {
								addEdge(chokeID, gridID(10, s))
							}
						}
					}
					// Normal radial propagation from Ring 10 to Ring 11: every
					// lane (rel 0..5) propagates, so this is unconditional.
					addEdge(gridID(10, slot), gridID(11, slot))
				} else {
					// Normal radial spoke propagation
					if rel == 2 || rel == 3 {
						addEdge(gridID(ring, slot), gridID(ring+1, slot))
					} else if rel == 1 || rel == 4 {
						if ring >= 4 {
							addEdge(gridID(ring, slot), gridID(ring+1, slot))
						}
					} else if rel == 0 || rel == 5 {
						if ring >= 9 {
							addEdge(gridID(ring, slot), gridID(ring+1, slot))
						}
					}
				}
			}

			// Lateral links to connect sub-branches below split points
			if rel == 1 {
				if ring < 4 {
					addEdge(gridID(ring, slot), gridID(ring, slot+1))
				} else if ring == 4 {
					addEdge(gridID(ring, slot), gridID(ring, slot+1))
				}
			} else if rel == 0 {
				if ring < 9 {
					addEdge(gridID(ring, slot), gridID(ring, slot+1))
				} else if ring == 9 {
					addEdge(gridID(ring, slot), gridID(ring, slot+1))
				}
			} else if rel == 4 {
				if ring < 4 {
					addEdge(gridID(ring, slot), gridID(ring, slot-1))
				} else if ring == 4 {
					addEdge(gridID(ring, slot), gridID(ring, slot-1))
				}
			} else if rel == 5 {
				if ring < 9 {
					addEdge(gridID(ring, slot), gridID(ring, slot-1))
				} else if ring == 9 {
					addEdge(gridID(ring, slot), gridID(ring, slot-1))
				}
			}

			// Organic maze-like broken lateral connections. Kept sparse (0.02) so a
			// few extra sideways links add texture without webbing the whole ring.
			rEdge := rand.New(rand.NewPCG(uint64(ring*1000+slot), 555))
			if slot%treeLanes != treeLanes-1 && rEdge.Float64() < 0.02 {
				alreadyConnected := false
				for _, neighbor := range t.Adj[gridID(ring, slot)] {
					if neighbor == gridID(ring, slot+1) {
						alreadyConnected = true
						break
					}
				}
				if !alreadyConnected {
					addEdge(gridID(ring, slot), gridID(ring, slot+1))
				}
			}
		}
	}

	// Add 5 chaotic cross-sector portals. These are long edges that cross the
	// whole web, so a handful reads as "occasional shortcut"; a dozen turned the
	// centre into a tangle. Endpoints are grid nodes only: keystones and bridges
	// are gated progression rewards and must not gain cross-sector shortcuts.
	rChaos := rand.New(rand.NewPCG(888, 999))
	for i := 0; i < 5; {
		nA := t.Nodes[rChaos.IntN(len(t.Nodes))]
		nB := t.Nodes[rChaos.IntN(len(t.Nodes))]
		nodeA := nA.ID
		nodeB := nB.ID
		// Ensure they are grid nodes (not root, keystone or bridge), not
		// identical, in different sectors, and not already connected
		if nodeA > 0 && nodeB > 0 && nodeA < treeFirstKeyID && nodeB < treeFirstKeyID && nodeA != nodeB && nA.Sector != nB.Sector {
			alreadyConnected := false
			for _, neighbor := range t.Adj[nodeA] {
				if neighbor == nodeB {
					alreadyConnected = true
					break
				}
			}
			if !alreadyConnected {
				addEdge(nodeA, nodeB)
				i++
			}
		}
	}

	// --- 100 signature nodes: the outer "Ascendant Rim" (IDs 1001..1100) -------
	// Build-defining notables past the keystone rim. Each hangs off a keystone so
	// it is reachable only after deep investment, and is Type "notable" (cost 2)
	// so the keystone count stays 40. These grow the web beyond its original 1000
	// nodes, so LayoutHash changes → a one-time free respec for every player.
	sigNodes := []struct {
		name string
		key  string
		val  float64
	}{
		// str_pct — Wrath
		{"Wrath of the First Blade", "str_pct", 0.05}, {"Fury Unchained", "str_pct", 0.06}, {"Berserker's Ascendance", "str_pct", 0.07}, {"Titanbreaker", "str_pct", 0.08}, {"Warlord's Dominion", "str_pct", 0.09}, {"Hand of Ruin", "str_pct", 0.10}, {"Godsbane Ascendant", "str_pct", 0.11}, {"The Crimson Apex", "str_pct", 0.12}, {"Wrathforged Crown", "str_pct", 0.13}, {"Avatar of Slaughter", "str_pct", 0.14},
		// hp_pct — Vitality
		{"Heart of the Mountain", "hp_pct", 0.05}, {"Enduring Colossus", "hp_pct", 0.06}, {"Unbreaking Vigor", "hp_pct", 0.07}, {"Bulwark Eternal", "hp_pct", 0.08}, {"Lifewell Ascendant", "hp_pct", 0.09}, {"Bastion of Flesh", "hp_pct", 0.10}, {"Immortal Coil", "hp_pct", 0.11}, {"The Living Fortress", "hp_pct", 0.12}, {"Worldheart Vitality", "hp_pct", 0.13}, {"Avatar of Endurance", "hp_pct", 0.14},
		// spd_pct — Alacrity
		{"Whisper of the Gale", "spd_pct", 0.05}, {"Fleetfoot Ascendance", "spd_pct", 0.06}, {"Tempest Cadence", "spd_pct", 0.07}, {"Blur of the Duelist", "spd_pct", 0.08}, {"Stormstride", "spd_pct", 0.09}, {"Quicksilver Apex", "spd_pct", 0.10}, {"The Unseen Step", "spd_pct", 0.11}, {"Windsworn Ascendant", "spd_pct", 0.12}, {"Lightning Reflex", "spd_pct", 0.13}, {"Avatar of Alacrity", "spd_pct", 0.14},
		// int_pct — Arcana
		{"Spark of Genius", "int_pct", 0.05}, {"Arcane Ascendance", "int_pct", 0.06}, {"Mind of the Deep", "int_pct", 0.07}, {"Eldritch Insight", "int_pct", 0.08}, {"Weaver of Secrets", "int_pct", 0.09}, {"Oracle's Apex", "int_pct", 0.10}, {"The Boundless Mind", "int_pct", 0.11}, {"Astral Ascendant", "int_pct", 0.12}, {"Cosmic Intellect", "int_pct", 0.13}, {"Avatar of Arcana", "int_pct", 0.14},
		// skill_damage — Spellfury
		{"Kindled Focus", "skill_damage", 0.06}, {"Spellfury Rising", "skill_damage", 0.07}, {"Runic Overload", "skill_damage", 0.08}, {"Cinderweave Mastery", "skill_damage", 0.09}, {"Voidcaller's Edge", "skill_damage", 0.10}, {"Arclight Apex", "skill_damage", 0.11}, {"The Searing Verdict", "skill_damage", 0.12}, {"Stormcaller Ascendant", "skill_damage", 0.13}, {"Ruinous Incantation", "skill_damage", 0.14}, {"Avatar of Spellfury", "skill_damage", 0.15},
		// xp_gain — Enlightenment
		{"Seeker's Diligence", "xp_gain", 0.03}, {"Scholar's Ascendance", "xp_gain", 0.04}, {"Path of Wisdom", "xp_gain", 0.05}, {"Enlightened Delver", "xp_gain", 0.06}, {"Runescribe's Insight", "xp_gain", 0.07}, {"Ascendant Erudition", "xp_gain", 0.08}, {"The Long Study", "xp_gain", 0.09}, {"Mind of Ages", "xp_gain", 0.10}, {"Sage of the Abyss", "xp_gain", 0.11}, {"Avatar of Enlightenment", "xp_gain", 0.12},
		// gold_find — Avarice
		{"Prospector's Eye", "gold_find", 0.05}, {"Gilded Ascendance", "gold_find", 0.06}, {"Coinseeker's Boon", "gold_find", 0.08}, {"Hoarder's Instinct", "gold_find", 0.09}, {"Midas Reach", "gold_find", 0.10}, {"Vault-Warden's Apex", "gold_find", 0.11}, {"The Golden Verdict", "gold_find", 0.12}, {"Dragonhoard Ascendant", "gold_find", 0.14}, {"Avaricious Crown", "gold_find", 0.15}, {"Avatar of Avarice", "gold_find", 0.16},
		// ult_damage — Cataclysm
		{"Rising Cataclysm", "ult_damage", 0.05}, {"Doombringer's Ascendance", "ult_damage", 0.06}, {"Apocalypse Engine", "ult_damage", 0.07}, {"Worldender's Edge", "ult_damage", 0.08}, {"Ruin Unleashed", "ult_damage", 0.09}, {"Annihilation Apex", "ult_damage", 0.10}, {"The Final Word", "ult_damage", 0.12}, {"Cataclysm Ascendant", "ult_damage", 0.13}, {"Starfall Verdict", "ult_damage", 0.14}, {"Avatar of Cataclysm", "ult_damage", 0.15},
		// def_to_lifesteal — Sanguine
		{"Sanguine Trickle", "def_to_lifesteal", 0.005}, {"Bloodward Ascendance", "def_to_lifesteal", 0.006}, {"Ironblood Pact", "def_to_lifesteal", 0.007}, {"Crimson Aegis", "def_to_lifesteal", 0.008}, {"Leeching Bulwark", "def_to_lifesteal", 0.009}, {"Vampiric Apex", "def_to_lifesteal", 0.010}, {"The Drinking Wall", "def_to_lifesteal", 0.011}, {"Sanguine Ascendant", "def_to_lifesteal", 0.012}, {"Hemocryst Bastion", "def_to_lifesteal", 0.013}, {"Avatar of Sanguinity", "def_to_lifesteal", 0.014},
		// ult_cooldown — Tempo
		{"Quickened Resolve", "ult_cooldown", 0.04}, {"Momentum Ascendance", "ult_cooldown", 0.05}, {"Relentless Cadence", "ult_cooldown", 0.06}, {"Everready Instinct", "ult_cooldown", 0.07}, {"Chrono-Attunement", "ult_cooldown", 0.08}, {"Tempo Apex", "ult_cooldown", 0.09}, {"The Endless Encore", "ult_cooldown", 0.10}, {"Timebender Ascendant", "ult_cooldown", 0.11}, {"Perpetual Onslaught", "ult_cooldown", 0.12}, {"Avatar of Tempo", "ult_cooldown", 0.13},
	}
	for k, sg := range sigNodes {
		id := 1000 + 1 + k // 1001..1100
		slot := k % treeSlots
		n := TreeNode{
			ID: id, Ring: treeRings + 2 + k/treeSlots, Slot: slot, Sector: slot / treeLanes,
			Type: "notable",
			Name: "✦ " + sg.name,
			Desc: fmt.Sprintf("A signature Ascendant power (%+.1f%% %s).", sg.val*100, sg.key),
			Pct:  map[string]float64{sg.key: sg.val},
		}
		polar(&n)
		t.Nodes = append(t.Nodes, n)
		addEdge(id, treeFirstKeyID+(k%treeKeystoneN)) // hang off a keystone → reachable
	}

	for i := range t.Nodes {
		t.byID[t.Nodes[i].ID] = &t.Nodes[i]
	}
	return t
}

// Cost is the node's skill-point price: bigger nodes cost more. Small grid
// nodes and sockets cost 1, notables and bridges 2, keystones 3.
func (n *TreeNode) Cost() int {
	switch n.Type {
	case "keystone":
		return 3
	case "notable", "bridge":
		return 2
	}
	return 1
}

// LayoutHash fingerprints the generated web: node identities, types, point
// costs, effect values and every edge. Any code change that alters the layout
// or balance changes the hash; the bot compares it against the stored value at
// startup and grants a free full respec to every player on mismatch. Names and
// descriptions are deliberately excluded — cosmetic edits must not wipe trees.
func (t *AbyssTreeData) LayoutHash() string {
	h := fnv.New64a()
	for i := range t.Nodes {
		n := &t.Nodes[i]
		_, _ = fmt.Fprintf(h, "n:%d:%s:%d:%d:%d:%s;", n.ID, n.Type, n.Ring, n.Slot, n.Cost(), statsBrief(n.Stats))
		pctKeys := make([]string, 0, len(n.Pct))
		for k := range n.Pct {
			pctKeys = append(pctKeys, k)
		}
		sort.Strings(pctKeys)
		for _, k := range pctKeys {
			_, _ = fmt.Fprintf(h, "p:%s=%g;", k, n.Pct[k])
		}
	}
	ids := make([]int, 0, len(t.Adj))
	for id := range t.Adj {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	for _, id := range ids {
		nbs := append([]int(nil), t.Adj[id]...)
		sort.Ints(nbs)
		for _, nb := range nbs {
			if id < nb {
				_, _ = fmt.Fprintf(h, "e:%d-%d;", id, nb)
			}
		}
	}
	return fmt.Sprintf("%016x", h.Sum64())
}

// SpentPoints sums the skill-point cost of the allocated node IDs. Unknown
// IDs (rows saved under an older layout) count as 1 so they never panic.
func (t *AbyssTreeData) SpentPoints(alloc []int) int {
	spent := 0
	for _, id := range alloc {
		if n := t.Node(id); n != nil {
			spent += n.Cost()
		} else {
			spent++
		}
	}
	return spent
}

// polar fills a node's layout position from its ring/slot.
func polar(n *TreeNode) { n.X, n.Y = polarXY(float64(n.Ring), float64(n.Slot)) }

func polarXY(ring, slot float64) (float64, float64) {
	if ring <= 5 {
		// Fermat spiral: N = (ring - 1) * 36 + slot
		n := (ring-1)*36 + slot
		goldenAngle := 137.507764 * math.Pi / 180.0
		theta := n * goldenAngle
		
		// Scaled radius for rings 1 to 5
		radius := 14.0 * math.Sqrt(n) + 20.0
		
		// Add tiny organic jitter
		seedVal := uint64(math.Round(ring)*1000 + math.Round(slot))
		r := rand.New(rand.NewPCG(seedVal, 777))
		radius += float64(r.IntN(5)-2)
		theta += (r.Float64()*0.02 - 0.01)

		return math.Round(radius * math.Cos(theta)), math.Round(radius * math.Sin(theta))
	}

	radius := 60 + ring*34
	
	// Determine sector and slot index within sector
	slotInt := int(math.Round(slot)) % treeSlots
	if slotInt < 0 {
		slotInt += treeSlots
	}
	sector := slotInt / treeLanes
	rel := slotInt % treeLanes

	// Base center angle for the sector
	centerAngle := (float64(sector)*6.0 + 2.5) / 36.0 * 2.0 * math.Pi - math.Pi/2.0
	spanHalf := math.Pi / 6.0 // 30 degrees (half of sector span)

	// Calculate fraction from sector center (-1.0 to 1.0)
	var fraction float64
	switch rel {
	case 2:
		fraction = -0.15
	case 3:
		fraction = 0.15
	case 1:
		if ring < 4.0 {
			fraction = -0.22
		} else {
			fraction = -0.22 - 0.38*(ring-4.0)/22.0 // branches out to -0.6
		}
	case 0:
		var f1 float64
		if ring < 4.0 {
			f1 = -0.22
		} else {
			f1 = -0.22 - 0.38*(ring-4.0)/22.0
		}
		if ring < 9.0 {
			fraction = f1 - 0.07
		} else {
			f1At9 := -0.22 - 0.38*(5.0)/22.0
			fraction = (f1At9 - 0.07) - 0.57*(ring-9.0)/17.0 // branches out to -0.95
		}
	case 4:
		if ring < 4.0 {
			fraction = 0.22
		} else {
			fraction = 0.22 + 0.38*(ring-4.0)/22.0 // branches out to 0.6
		}
	case 5:
		var f4 float64
		if ring < 4.0 {
			f4 = 0.22
		} else {
			f4 = 0.22 + 0.38*(ring-4.0)/22.0
		}
		if ring < 9.0 {
			fraction = f4 + 0.07
		} else {
			f4At9 := 0.22 + 0.38*(5.0)/22.0
			fraction = (f4At9 + 0.07) + 0.57*(ring-9.0)/17.0 // branches out to 0.95
		}
	default:
		// Fallback to straight line
		fraction = (float64(rel) - 2.5) / 3.0
	}

	angle := centerAngle + fraction*spanHalf

	// Seed PCG generator deterministically per position to add organic jitter
	seedVal := uint64(math.Round(ring)*1000 + math.Round(slot))
	r := rand.New(rand.NewPCG(seedVal, 777))

	// Jitter radius by [-6..6] pixels
	jitterRadius := radius + float64(r.IntN(13)-6)
	// Jitter angle by [-0.025..0.025] radians
	jitterAngle := angle + (r.Float64()*0.05 - 0.025)

	return math.Round(jitterRadius * math.Cos(jitterAngle)), math.Round(jitterRadius * math.Sin(jitterAngle))
}

// treeSmallStats is the flat stat block of a small node, scaling with depth
// into the web and flavored by its sector.
func treeSmallStats(sector, ring, slot, nodeID int) Stats {
	// Seed the random generator uniquely for this node ID to ensure stable, unique stats
	r := rand.New(rand.NewPCG(uint64(nodeID), 42))

	grow := 1 + ring/3
	
	var s Stats
	
	// 1. Primary stats based on sector
	switch sector {
	case 0: // War: Strength-focused
		s.STR = 2 + ring/2 + r.IntN(3)
		if r.IntN(2) == 0 {
			s.CRT = 1 + ring/8
		}
	case 1: // Vitality: HP/Stamina-focused
		s.HP = 10 + 5*ring + r.IntN(10)
		s.STA = grow + r.IntN(2)
	case 2: // Shadow: Speed/Dodge-focused
		s.SPD = 2 + ring/2 + r.IntN(3)
		if r.IntN(2) == 0 {
			s.DGE = 1 + ring/8
		}
	case 3: // Arcane: Intellect/Mana-focused
		s.INT = 2 + ring/2 + r.IntN(3)
		s.MNA = 5 + 3*ring + r.IntN(5)
	case 4: // Fortune: Luck-focused
		s.LCK = 2 + ring/2 + r.IntN(3)
		if r.IntN(2) == 0 {
			s.CRT = 1 + ring/8
		}
	default: // Void: Balanced stats
		s.HP = 5 + 3*ring + r.IntN(5)
		s.STR = 1 + ring/4
		s.DEF = 1 + ring/4 + r.IntN(2)
		s.SPD = 1 + ring/4
	}

	// 2. Add a unique secondary stat to ensure no two nodes are identical
	statPool := []string{"HP", "MNA", "STR", "DEF", "SPD", "LCK", "INT", "STA", "CRT", "DGE"}
	secIndex := r.IntN(len(statPool))
	secVal := 1 + r.IntN(grow+1)
	
	switch statPool[secIndex] {
	case "HP":
		if s.HP == 0 { s.HP = secVal * 4 }
	case "MNA":
		if s.MNA == 0 { s.MNA = secVal * 2 }
	case "STR":
		if s.STR == 0 { s.STR = secVal }
	case "DEF":
		if s.DEF == 0 { s.DEF = secVal }
	case "SPD":
		if s.SPD == 0 { s.SPD = secVal }
	case "LCK":
		if s.LCK == 0 { s.LCK = secVal }
	case "INT":
		if s.INT == 0 { s.INT = secVal }
	case "STA":
		if s.STA == 0 { s.STA = secVal }
	case "CRT":
		if s.CRT == 0 { s.CRT = secVal }
	case "DGE":
		if s.DGE == 0 { s.DGE = secVal }
	}

	// 3. Add a small tertiary raw stat if deep enough (ring > 5)
	if ring > 5 {
		tertIndex := (secIndex + 1 + r.IntN(len(statPool)-1)) % len(statPool)
		tertVal := 1 + r.IntN(2)
		switch statPool[tertIndex] {
		case "HP": s.HP += tertVal * 3
		case "MNA": s.MNA += tertVal * 2
		case "STR": s.STR += tertVal
		case "DEF": s.DEF += tertVal
		case "SPD": s.SPD += tertVal
		case "LCK": s.LCK += tertVal
		case "INT": s.INT += tertVal
		case "STA": s.STA += tertVal
		case "CRT": s.CRT += tertVal
		case "DGE": s.DGE += tertVal
		}
	}

	return s
}

// treeSectorPctKey picks which percent effect a sector's notables carry.
func treeSectorPctKey(sector, slot int) string {
	switch sector {
	case 0:
		return "str_pct"
	case 1:
		return "hp_pct"
	case 2:
		return "spd_pct"
	case 3:
		return "int_pct"
	case 4:
		if slot%2 == 0 {
			return "gold_find"
		}
		return "loot_find"
	default: // Void rotates through the economy effects
		switch slot % 4 {
		case 0:
			return "escrow_bonus"
		case 1:
			return "xp_gain"
		case 2:
			return "token_gain"
		default:
			return "material_yield"
		}
	}
}

// treeNotablePct sizes a notable's percent effect by how deep it sits.
func treeNotablePct(sector, ring int) float64 {
	base := 0.01 + float64(ring)*0.001
	if sector <= 3 { // combat multipliers run a little hotter
		base = 0.02 + float64(ring)*0.002
	}
	return math.Round(base*1000) / 1000
}

// treeNodeText builds the display name and effect description of a grid node.
// Names are unique by construction: the ring picks the adjective, the lane
// picks the noun, and no two grid nodes share both. Notables carry a ★ marker.
func treeNodeText(n *TreeNode) (string, string) {
	// Jewel Sockets and custom notables names and descriptions (Item 34, 36, 37, 39, 40, 45)
	if n.Ring == 18 && n.Slot == 20 {
		return "⚡ Overcharged Core", "Reduces Ultimate skill cooldown by 10%, and increases Ultimate skill damage by 15%."
	}
	if n.Ring == 16 && n.Slot == 16 {
		return "🩸 Souldrinker", "Converts defensive stats into Lifesteal: +1% of DEF added as Lifesteal."
	}
	if n.Ring == 15 && n.Slot == 27 {
		return "⏳ Temporal Shift", "Grants +15% gold find on weekends, and +10% floor XP on weekdays."
	}
	if n.Ring == 14 && n.Slot == 9 {
		return "🐾 Beastmaster's Command (Synergy)", "Reduces pet betrayal chance by 2% and increases pet attack damage by 20%. Also adds +8 max HP per allocated Void-sector node."
	}
	if n.Ring == 18 && n.Slot == 4 {
		return "🛡️ Unbreakable (Synergy)", "Grants immunity to stun effects in combat. Also adds +2 STR per allocated Arcane-sector node."
	}
	if n.Ring == 17 && n.Slot == 11 {
		return "🔮 Spellweaver (Earthquake)", "Skill casts deal +20% damage and cost 25% less mana. Unlocks the Earthquake physical combat skill."
	}
	if n.Ring == 13 && n.Slot == 33 {
		return "🧪 Alchemist's Ritual (Arcane Shield)", "25% chance after each fight that your consumables lose no charge. Unlocks the Arcane Shield magical combat skill."
	}
	if n.Ring == 21 && n.Slot == 2 {
		return "💀 Reckless Abandon", "Grants +30% Skill damage, but reduces maximum HP by 15%."
	}
	if n.Ring == 22 && n.Slot == 14 {
		return "🎭 Specialist's Harmony", "Grants passive bonuses based on active Specialization (+15% max HP for Warden, +15% STR for Delver, +25% Gold Find for Plunderer)."
	}
	if n.Ring == 23 && n.Slot == 25 {
		return "🌀 Abyssal Attunement", "Grants +0.5% STR and +0.5% max HP per Abyss depth level reached (based on your best record)."
	}
	if n.Ring == 24 && n.Slot == 5 {
		return "🎖️ Prestige Focus", "Multiplies all passive node bonuses in the War discipline sector by +10% per prestige level."
	}
	if n.Ring == 25 && n.Slot == 18 {
		return "💎 Set Resonance", "Grants +5% to all base attributes (STR, INT, SPD, max HP) per active equipped gear set bonus tier."
	}
	if n.Ring == 26 && n.Slot == 31 {
		return "🧪 Elemental Transmutation", "Converts 50% of your total physical damage modifier (STR %) into magical damage modifier (INT %)."
	}
	if n.Ring == 20 && n.Slot == 12 {
		return "🏆 Victor's Trophy", "🔒 Requires depth record of 25+ to allocate. Grants +15% to character experience gain."
	}
	if n.Type == "socket" {
		sectors := []string{"War", "Vitality", "Shadow", "Arcane", "Fortune", "Void"}
		secName := "Unknown"
		if n.Sector >= 0 && n.Sector < len(sectors) {
			secName = sectors[n.Sector]
		}
		return fmt.Sprintf("💎 Jewel Socket (%s)", secName), "Can slot a Jewel to modify adjacent nodes. Slot a Ruby (+15 STR), Sapphire (+15 INT), or Topaz (+15 SPD) to allocated neighbors."
	}

	star := ""
	if n.Type == "notable" {
		star = "★ "
	}
	name := fmt.Sprintf("%s %s%s %s", treeSectorIcons[n.Sector], star, treeRingAdjectives[n.Ring-1], treeLaneNouns[n.Sector][n.Slot%treeLanes])
	desc := statsBrief(n.Stats)
	for k, v := range n.Pct {
		desc += fmt.Sprintf(" · +%.1f%% %s", v*100, treePctLabel(k))
	}
	return name, desc
}

// treePctLabel is the player-facing label of a percent effect key.
func treePctLabel(k string) string {
	switch k {
	case "str_pct":
		return "STR (Abyss)"
	case "hp_pct":
		return "max HP (Abyss)"
	case "spd_pct":
		return "SPD (Abyss)"
	case "int_pct":
		return "INT (Abyss)"
	case "gold_find":
		return "gold from drops"
	case "loot_find":
		return "loot find"
	case "escrow_bonus":
		return "escrow floor bonus"
	case "xp_gain":
		return "floor XP"
	case "token_gain":
		return "tokens on bank"
	case "material_yield":
		return "crafting materials"
	case "str_to_spd":
		return "STR converted to SPD"
	case "hp_to_def":
		return "HP converted to DEF"
	case "spd_to_dge":
		return "SPD converted to DGE"
	case "int_to_mna":
		return "INT converted to MNA"
	case "xp_to_gold":
		return "floor XP converted to Gold"
	case "ult_cooldown":
		return "Ultimate Cooldown reduction"
	case "ult_damage":
		return "Ultimate Skill damage"
	case "def_to_lifesteal":
		return "DEF converted to Lifesteal"
	case "pet_betrayal_reduce":
		return "pet betrayal reduction"
	case "pet_damage_pct":
		return "pet attack damage"
	case "stun_immunity":
		return "Stun Immunity"
	case "limit_break":
		return "Limit Break"
	case "skill_damage":
		return "Skill damage"
	case "skill_mana_cost":
		return "Skill mana cost reduction"
	case "consumable_save":
		return "chance consumables keep their charge"
	}
	return k
}

// statsBrief renders the non-zero stats of a node compactly ("+4 STR · +1 CRT").
func statsBrief(s Stats) string {
	out := ""
	add := func(v int, label string) {
		if v != 0 {
			if out != "" {
				out += " · "
			}
			out += fmt.Sprintf("%+d %s", v, label)
		}
	}
	add(s.HP, "HP")
	add(s.MNA, "MNA")
	add(s.STR, "STR")
	add(s.DEF, "DEF")
	add(s.SPD, "SPD")
	add(s.LCK, "LCK")
	add(s.INT, "INT")
	add(s.STA, "STA")
	add(s.CRT, "CRT")
	add(s.DGE, "DGE")
	return out
}
