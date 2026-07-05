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
	"math"
	"math/rand/v2"
	"sort"
	"sync"
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
	for _, id := range ids {
		n := t.byID[id]
		if n == nil {
			continue
		}
		tb.Stats = tb.Stats.Add(n.Stats)
		for k, v := range n.Pct {
			tb.Pct[k] += v
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
	{"Berserker's Oath", map[string]float64{"str_pct": 0.15, "hp_pct": -0.10}},
	{"Warbringer", map[string]float64{"str_pct": 0.10, "xp_gain": 0.05}},
	{"Executioner", map[string]float64{"str_pct": 0.12, "gold_find": 0.06}},
	{"Bloodforged", map[string]float64{"str_pct": 0.10, "material_yield": 0.10}},
	{"Colossus Slayer", map[string]float64{"str_pct": 0.08, "loot_find": 0.08}},
	// ❤️ Vitality
	{"Colossus", map[string]float64{"hp_pct": 0.30}},
	{"Undying", map[string]float64{"hp_pct": 0.20, "xp_gain": 0.05}},
	{"Ironroot", map[string]float64{"hp_pct": 0.15, "material_yield": 0.10}},
	{"Heartspring", map[string]float64{"hp_pct": 0.12, "escrow_bonus": 0.08}},
	{"Titan's Blood", map[string]float64{"hp_pct": 0.18, "spd_pct": -0.05}},
	{"Warden's Resolve", map[string]float64{"hp_pct": 0.10, "token_gain": 0.10}},
	// 🌫️ Shadow
	{"Phantom", map[string]float64{"spd_pct": 0.20}},
	{"Ghostwalk", map[string]float64{"spd_pct": 0.12, "loot_find": 0.08}},
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
	{"Manaforged", map[string]float64{"int_pct": 0.12, "escrow_bonus": 0.08}},
	// 🍀 Fortune
	{"Midas", map[string]float64{"gold_find": 0.15, "loot_find": 0.05}},
	{"Gambler's Creed", map[string]float64{"gold_find": 0.20, "escrow_bonus": -0.05}},
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
	{"👑 Worldsplitter", map[string]float64{"str_pct": 0.10, "hp_pct": 0.10}},
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
			if notable {
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
			// Radial spokes.
			if ring < treeRings {
				addEdge(gridID(ring, slot), gridID(ring+1, slot))
			}
			// Sparse lateral links inside the sector: roughly two of every
			// three neighbors connect, so rings are broken arcs with gaps —
			// forcing route choices while always leaving multiple paths.
			if slot%treeLanes != treeLanes-1 && (ring+slot)%3 != 0 {
				addEdge(gridID(ring, slot), gridID(ring, slot+1))
			}
		}
	}

	for i := range t.Nodes {
		t.byID[t.Nodes[i].ID] = &t.Nodes[i]
	}
	return t
}

// polar fills a node's layout position from its ring/slot.
func polar(n *TreeNode) { n.X, n.Y = polarXY(float64(n.Ring), float64(n.Slot)) }

func polarXY(ring, slot float64) (float64, float64) {
	radius := 60 + ring*34
	angle := (slot/float64(treeSlots))*2*math.Pi - math.Pi/2
	return math.Round(radius * math.Cos(angle)), math.Round(radius * math.Sin(angle))
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
