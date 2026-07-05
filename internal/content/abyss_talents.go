package content

import "fmt"

// Generic, level-based Abyss talent trees: the Deep-Delver *extension* (Spec == "")
// and, later, the three per-spec sub-trees (Spec == delver|plunderer|warden).
//
// Unlike the original twelve Deep-Delver nodes — each of which is its own DB
// column with a hand-wired effect — these are stored generically as key→level
// (app_meta JSON, see Bot.loadAbyssTalentLevels), so the trees can grow without a
// migration per node. Each allocated *level* adds the node's Stats/Pct through the
// exact same live effect pipeline as the skill web (folded into Bot.treeBonusFor),
// so every node has a real, distinct in-game effect rather than a dead stub.

// TalentMaxLevel is the shared per-node cap. The web UI hardcodes 5 and the token
// cost formula is (level+1)*10, matching the legacy Deep-Delver nodes.
const TalentMaxLevel = 5

// Talent is one node of a generic talent tree. Only the fields the web renderer
// needs are serialized; Stats/Pct/Spec/Parent stay server-side.
//
// Parent vs Edge: Parent is the *gate* prerequisite the server enforces (a talent
// that must be level ≥1, "" = none). Edge is the *display* source the client draws
// the connecting line from — usually the same as Parent, but a spec sub-tree's
// root draws from the spec picker node (Edge) while gating only on the spec being
// active (Parent == "").
type Talent struct {
	Key       string             `json:"key"`
	Label     string             `json:"label"`
	Desc      string             `json:"desc"`
	X         int                `json:"x"`
	Y         int                `json:"y"`
	Edge      string             `json:"parent"` // client edge source
	Icon      string             `json:"icon"`
	GateDepth int                `json:"gateDepth"`
	Parent    string             `json:"-"` // server gate prerequisite ("" = none)
	Spec      string             `json:"-"` // "" = Deep-Delver; else delver|plunderer|warden
	Stats     Stats              `json:"-"`
	Pct       map[string]float64 `json:"-"`
}

// ddSpec is the per-node authoring record for the Deep-Delver extension; the
// builder turns each into a positioned, parented Talent.
type ddSpec struct {
	label string
	desc  string
	icon  string
	stats Stats
	pct   map[string]float64
}

func p1(k string, v float64) map[string]float64 { return map[string]float64{k: v} }

// deepDelverBranches: five thematic arms of ten tiers. Every node has a unique
// name and a distinct per-level effect drawn from the live Abyss effect
// vocabulary (the same keys the skill web and its custom notables already use).
var deepDelverBranches = [5]struct {
	name  string
	nodes [10]ddSpec
}{
	{"Warpath", [10]ddSpec{
		{"Whetstone Instinct", "Sharper strikes. +2% Abyss STR per level.", "crossed-swords", Stats{}, p1("str_pct", 0.02)},
		{"Iron Grip", "A surer hold on your weapon. +4 STR per level.", "broadsword", Stats{STR: 4}, nil},
		{"Savage Technique", "Brutal form. +3% skill damage per level.", "fangs", Stats{}, p1("skill_damage", 0.03)},
		{"Killer's Eye", "Find the gap. +2 CRT per level.", "crossed-swords", Stats{CRT: 2}, nil},
		{"Warmonger's Rage", "Fury compounds. +2.5% Abyss STR per level.", "dragon-head", Stats{}, p1("str_pct", 0.025)},
		{"Executioner's Might", "Finish the wounded. +3% Ultimate damage per level.", "broadsword", Stats{}, p1("ult_damage", 0.03)},
		{"Titanic Thews", "Monstrous strength. +6 STR per level.", "dragon-head", Stats{STR: 6}, nil},
		{"Ruinblade Mastery", "Every cut ruinous. +3.5% skill damage per level.", "fangs", Stats{}, p1("skill_damage", 0.035)},
		{"Avatar of War", "War made flesh. +3% Abyss STR per level.", "crossed-swords", Stats{}, p1("str_pct", 0.03)},
		{"Cataclysmic Finisher", "Your Ultimate ends worlds. +4% Ultimate damage per level.", "dragon-head", Stats{}, p1("ult_damage", 0.04)},
	}},
	{"Bulwark", [10]ddSpec{
		{"Thickened Hide", "Toughened skin. +2% Abyss max HP per level.", "checked-shield", Stats{}, p1("hp_pct", 0.02)},
		{"Ironbark Plating", "Harder to dent. +3 DEF per level.", "visored-helm", Stats{DEF: 3}, nil},
		{"Deepwell Vitality", "A deeper reserve. +30 HP per level.", "health-normal", Stats{HP: 30}, nil},
		{"Bloodward Osmosis", "Turn guard to greed. +0.3% of DEF as Lifesteal per level.", "totem", Stats{}, p1("def_to_lifesteal", 0.003)},
		{"Bastion Heart", "An unbroken core. +2.5% Abyss max HP per level.", "checked-shield", Stats{}, p1("hp_pct", 0.025)},
		{"Aegis Discipline", "Perfect guard. +5 DEF per level.", "visored-helm", Stats{DEF: 5}, nil},
		{"Stoneblood Conversion", "Vitality into armor. Convert 5% of HP into DEF per level.", "totem", Stats{}, p1("hp_to_def", 0.05)},
		{"Undying Reserve", "Refuse to fall. +45 HP per level.", "health-normal", Stats{HP: 45}, nil},
		{"Fortress Eternal", "A living wall. +3% Abyss max HP per level.", "checked-shield", Stats{}, p1("hp_pct", 0.03)},
		{"Sanguine Rampart", "The wall drinks. +0.4% of DEF as Lifesteal per level.", "totem", Stats{}, p1("def_to_lifesteal", 0.004)},
	}},
	{"Tempest", [10]ddSpec{
		{"Fleet Step", "Lighter on your feet. +2% Abyss SPD per level.", "hood", Stats{}, p1("spd_pct", 0.02)},
		{"Windrunner's Gait", "Never slowing. +4 SPD per level.", "eagle-emblem", Stats{SPD: 4}, nil},
		{"Ghostweave Reflex", "Slip the blow. +2 DGE per level.", "spectre", Stats{DGE: 2}, nil},
		{"Phantom Footwork", "Speed into evasion. Convert 5% of SPD into DGE per level.", "hood", Stats{}, p1("spd_to_dge", 0.05)},
		{"Galewrought Speed", "Riding the gale. +2.5% Abyss SPD per level.", "eagle-emblem", Stats{}, p1("spd_pct", 0.025)},
		{"Opportunist's Timing", "Strike the opening. +2 CRT per level.", "wolf-head", Stats{CRT: 2}, nil},
		{"Stormstride", "Faster than the eye. +6 SPD per level.", "eagle-emblem", Stats{SPD: 6}, nil},
		{"Untouchable", "Barely there. +3 DGE per level.", "spectre", Stats{DGE: 3}, nil},
		{"Tempest Cadence", "A blur of motion. +3% Abyss SPD per level.", "hood", Stats{}, p1("spd_pct", 0.03)},
		{"Blur of the Duelist", "Speed becomes phantom. Convert 6% of SPD into DGE per level.", "spectre", Stats{}, p1("spd_to_dge", 0.06)},
	}},
	{"Arcanum", [10]ddSpec{
		{"Kindled Mind", "The spark grows. +2% Abyss INT per level.", "spell-book", Stats{}, p1("int_pct", 0.02)},
		{"Runic Attunement", "Sharper focus. +4 INT per level.", "crystal-ball", Stats{INT: 4}, nil},
		{"Deep Manawell", "A wider channel. +15 MNA per level.", "psychic-waves", Stats{MNA: 15}, nil},
		{"Spellfury", "Louder incantations. +3% skill damage per level.", "sunbeams", Stats{}, p1("skill_damage", 0.03)},
		{"Eldritch Insight", "Forbidden clarity. +2.5% Abyss INT per level.", "crystal-ball", Stats{}, p1("int_pct", 0.025)},
		{"Chrono-Attunement", "Bend the tempo. +2% Ultimate cooldown recovery per level.", "psychic-waves", Stats{}, p1("ult_cooldown", 0.02)},
		{"Boundless Intellect", "No ceiling. +6 INT per level.", "spell-book", Stats{INT: 6}, nil},
		{"Mana Overflow", "Wit into power. Convert 10% of INT into MNA per level.", "psychic-waves", Stats{}, p1("int_to_mna", 0.10)},
		{"Astral Ascendance", "Mind of the deep. +3% Abyss INT per level.", "crystal-ball", Stats{}, p1("int_pct", 0.03)},
		{"Perpetual Casting", "The encore never ends. +3% Ultimate cooldown recovery per level.", "sunbeams", Stats{}, p1("ult_cooldown", 0.03)},
	}},
	{"Fortune", [10]ddSpec{
		{"Coin Sense", "A nose for gold. +3% gold from drops per level.", "clover", Stats{}, p1("gold_find", 0.03)},
		{"Sharp Eye", "Spot the shine. +2% loot find per level.", "rolling-dices", Stats{}, p1("loot_find", 0.02)},
		{"Quick Study", "Learn faster. +2% floor XP per level.", "spell-book", Stats{}, p1("xp_gain", 0.02)},
		{"Tribute Tithe", "Skim the offering. +3% tokens on bank per level.", "laurel-crown", Stats{}, p1("token_gain", 0.03)},
		{"Prospector's Nose", "Follow the vein. +3.5% gold from drops per level.", "open-treasure-chest", Stats{}, p1("gold_find", 0.035)},
		{"Vault Interest", "Deeper caches pay more. +2% escrow floor bonus per level.", "unique-gem", Stats{}, p1("escrow_bonus", 0.02)},
		{"Salvager's Hands", "Waste nothing. +3% crafting materials per level.", "unique-gem", Stats{}, p1("material_yield", 0.03)},
		{"Treasure Sense", "The deep hides riches. +3% loot find per level.", "rolling-dices", Stats{}, p1("loot_find", 0.03)},
		{"Scholar's Path", "Every floor a lesson. +2.5% floor XP per level.", "spell-book", Stats{}, p1("xp_gain", 0.025)},
		{"Midas Reach", "Everything turns gold. +4% gold from drops per level.", "open-treasure-chest", Stats{}, p1("gold_find", 0.04)},
	}},
}

// DeepDelverTalents is the 50-node Deep-Delver extension, laid out as five arms
// of ten tiers hanging off the legacy tree's two leaf nodes (Scavenger and
// Quartermaster) and chaining across the top, so every node is reachable and the
// paths are explicit. Gated deeper (depth 30+) than the legacy nodes.
var DeepDelverTalents = buildDeepDelver()

func buildDeepDelver() []Talent {
	const (
		colGap, baseX = 230, -460
		rowGap, baseY = 66, 620
	)
	out := make([]Talent, 0, 50)
	for c := 0; c < len(deepDelverBranches); c++ {
		br := deepDelverBranches[c]
		for r := 0; r < len(br.nodes); r++ {
			key := fmt.Sprintf("dd_%d_%d", c, r)
			var parent string
			switch {
			case r > 0:
				parent = fmt.Sprintf("dd_%d_%d", c, r-1) // vertical chain within the arm
			case c <= 2:
				parent = "scavenger" // left arms hang off the Scavenger leaf
			default:
				parent = "quartermaster" // right arms hang off the Quartermaster leaf
			}
			sp := br.nodes[r]
			out = append(out, Talent{
				Key: key, Label: sp.label, Desc: sp.desc,
				X: baseX + c*colGap, Y: baseY + r*rowGap,
				Parent: parent, Edge: parent, Icon: sp.icon, GateDepth: 30 + r*2,
				Stats: sp.stats, Pct: sp.pct,
			})
		}
	}
	return out
}

// specTalentBranches: five thematic arms of ten tiers per specialization. Each
// spec's sub-tree is unlocked only while that spec is the active one (enforced by
// Talent.Spec), so its 50 nodes deepen the exclusive 1-of-3 choice.
var specTalentBranches = map[string][5]struct {
	name  string
	nodes [10]ddSpec
}{
	"delver": {
		{"Lorekeeper", [10]ddSpec{
			{"Apprentice's Notes", "+2% floor XP per level.", "spell-book", Stats{}, p1("xp_gain", 0.02)},
			{"Studied Mind", "+2% Abyss INT per level.", "crystal-ball", Stats{}, p1("int_pct", 0.02)},
			{"Field Researcher", "+2.5% floor XP per level.", "spell-book", Stats{}, p1("xp_gain", 0.025)},
			{"Runewise", "+4 INT per level.", "crystal-ball", Stats{INT: 4}, nil},
			{"Chronicler", "+3% floor XP per level.", "spell-book", Stats{}, p1("xp_gain", 0.03)},
			{"Applied Theory", "+3% skill damage per level.", "sunbeams", Stats{}, p1("skill_damage", 0.03)},
			{"Deep Scholar", "+3.5% floor XP per level.", "spell-book", Stats{}, p1("xp_gain", 0.035)},
			{"Arcane Savant", "+3% Abyss INT per level.", "crystal-ball", Stats{}, p1("int_pct", 0.03)},
			{"Loremaster", "+4% floor XP per level.", "spell-book", Stats{}, p1("xp_gain", 0.04)},
			{"Living Archive", "+5% floor XP per level.", "legendary-star", Stats{}, p1("xp_gain", 0.05)},
		}},
		{"Trailblazer", [10]ddSpec{
			{"Pathfinder", "+2% loot find per level.", "eagle-emblem", Stats{}, p1("loot_find", 0.02)},
			{"Lucky Break", "+3 LCK per level.", "clover", Stats{LCK: 3}, nil},
			{"Cartomancer", "+2.5% loot find per level.", "eagle-emblem", Stats{}, p1("loot_find", 0.025)},
			{"Salvage Runs", "+3% gold from drops per level.", "open-treasure-chest", Stats{}, p1("gold_find", 0.03)},
			{"Deepdelver", "+3% loot find per level.", "eagle-emblem", Stats{}, p1("loot_find", 0.03)},
			{"Relic Hunter", "+3% crafting materials per level.", "unique-gem", Stats{}, p1("material_yield", 0.03)},
			{"Vaultseeker", "+3.5% loot find per level.", "open-treasure-chest", Stats{}, p1("loot_find", 0.035)},
			{"Fortune's Student", "+5 LCK per level.", "clover", Stats{LCK: 5}, nil},
			{"Trailmaster", "+4% loot find per level.", "eagle-emblem", Stats{}, p1("loot_find", 0.04)},
			{"Worldwalker", "+5% loot find per level.", "legendary-star", Stats{}, p1("loot_find", 0.05)},
		}},
		{"Combatant", [10]ddSpec{
			{"Drilled Form", "+2% Abyss STR per level.", "crossed-swords", Stats{}, p1("str_pct", 0.02)},
			{"Field Training", "+4 STR per level.", "broadsword", Stats{STR: 4}, nil},
			{"Battle Study", "+3% skill damage per level.", "fangs", Stats{}, p1("skill_damage", 0.03)},
			{"Precise Strikes", "+2 CRT per level.", "crossed-swords", Stats{CRT: 2}, nil},
			{"Veteran's Edge", "+2.5% Abyss STR per level.", "broadsword", Stats{}, p1("str_pct", 0.025)},
			{"Warlore", "+3.5% skill damage per level.", "fangs", Stats{}, p1("skill_damage", 0.035)},
			{"Hardened Delver", "+6 STR per level.", "dragon-head", Stats{STR: 6}, nil},
			{"Frontline Scholar", "+3% Abyss STR per level.", "crossed-swords", Stats{}, p1("str_pct", 0.03)},
			{"Master-at-Arms", "+4% skill damage per level.", "broadsword", Stats{}, p1("skill_damage", 0.04)},
			{"Champion Delver", "+3.5% Abyss STR per level.", "dragon-head", Stats{}, p1("str_pct", 0.035)},
		}},
		{"Seeker", [10]ddSpec{
			{"Light Pack", "+2% Abyss SPD per level.", "hood", Stats{}, p1("spd_pct", 0.02)},
			{"Quick March", "+4 SPD per level.", "eagle-emblem", Stats{SPD: 4}, nil},
			{"Sidestep", "+2 DGE per level.", "spectre", Stats{DGE: 2}, nil},
			{"Swift Descent", "+2.5% Abyss SPD per level.", "hood", Stats{}, p1("spd_pct", 0.025)},
			{"Ambusher", "+2 CRT per level.", "wolf-head", Stats{CRT: 2}, nil},
			{"Fleet Explorer", "+3% Abyss SPD per level.", "eagle-emblem", Stats{}, p1("spd_pct", 0.03)},
			{"Evasive Delver", "+3 DGE per level.", "spectre", Stats{DGE: 3}, nil},
			{"Rapid Advance", "+6 SPD per level.", "eagle-emblem", Stats{SPD: 6}, nil},
			{"Windrunner", "+3.5% Abyss SPD per level.", "hood", Stats{}, p1("spd_pct", 0.035)},
			{"Phantom Seeker", "Convert 5% of SPD into DGE per level.", "spectre", Stats{}, p1("spd_to_dge", 0.05)},
		}},
		{"Ascendant", [10]ddSpec{
			{"Second Wind", "+2% Ultimate cooldown recovery per level.", "sunbeams", Stats{}, p1("ult_cooldown", 0.02)},
			{"Focused Fury", "+3% Ultimate damage per level.", "dragon-head", Stats{}, p1("ult_damage", 0.03)},
			{"Enlightened", "+3% floor XP per level.", "spell-book", Stats{}, p1("xp_gain", 0.03)},
			{"Momentum", "+2.5% Ultimate cooldown recovery per level.", "sunbeams", Stats{}, p1("ult_cooldown", 0.025)},
			{"Deep Focus", "+2.5% Abyss INT per level.", "crystal-ball", Stats{}, p1("int_pct", 0.025)},
			{"Overwhelm", "+3.5% Ultimate damage per level.", "dragon-head", Stats{}, p1("ult_damage", 0.035)},
			{"Sage's Reward", "+3.5% floor XP per level.", "spell-book", Stats{}, p1("xp_gain", 0.035)},
			{"Relentless", "+3% Ultimate cooldown recovery per level.", "sunbeams", Stats{}, p1("ult_cooldown", 0.03)},
			{"Ascendant Might", "+3% Abyss STR per level.", "crossed-swords", Stats{}, p1("str_pct", 0.03)},
			{"Avatar of Experience", "+5% floor XP per level.", "legendary-star", Stats{}, p1("xp_gain", 0.05)},
		}},
	},
	"plunderer": {
		{"Coinseeker", [10]ddSpec{
			{"Loose Change", "+3% gold from drops per level.", "clover", Stats{}, p1("gold_find", 0.03)},
			{"Pickpocket", "+3.5% gold from drops per level.", "open-treasure-chest", Stats{}, p1("gold_find", 0.035)},
			{"Lucky Coin", "+3 LCK per level.", "clover", Stats{LCK: 3}, nil},
			{"Coin Purse", "+4% gold from drops per level.", "open-treasure-chest", Stats{}, p1("gold_find", 0.04)},
			{"Deep Pockets", "+4.5% gold from drops per level.", "open-treasure-chest", Stats{}, p1("gold_find", 0.045)},
			{"Gilded Touch", "+5% gold from drops per level.", "clover", Stats{}, p1("gold_find", 0.05)},
			{"Fortune Seeker", "+5 LCK per level.", "rolling-dices", Stats{LCK: 5}, nil},
			{"Money Nose", "+5.5% gold from drops per level.", "open-treasure-chest", Stats{}, p1("gold_find", 0.055)},
			{"Golden Greed", "+6% gold from drops per level.", "open-treasure-chest", Stats{}, p1("gold_find", 0.06)},
			{"Midas Delver", "+7% gold from drops per level.", "legendary-star", Stats{}, p1("gold_find", 0.07)},
		}},
		{"Appraiser", [10]ddSpec{
			{"Keen Eye", "+2% loot find per level.", "rolling-dices", Stats{}, p1("loot_find", 0.02)},
			{"Scrapper", "+3% crafting materials per level.", "unique-gem", Stats{}, p1("material_yield", 0.03)},
			{"Assessor", "+2.5% loot find per level.", "rolling-dices", Stats{}, p1("loot_find", 0.025)},
			{"Dismantler", "+3.5% crafting materials per level.", "unique-gem", Stats{}, p1("material_yield", 0.035)},
			{"Curator", "+3% loot find per level.", "rolling-dices", Stats{}, p1("loot_find", 0.03)},
			{"Refiner", "+4% crafting materials per level.", "unique-gem", Stats{}, p1("material_yield", 0.04)},
			{"Gemcutter", "+3.5% loot find per level.", "unique-gem", Stats{}, p1("loot_find", 0.035)},
			{"Reclaimer", "+4.5% crafting materials per level.", "unique-gem", Stats{}, p1("material_yield", 0.045)},
			{"Master Appraiser", "+4% loot find per level.", "rolling-dices", Stats{}, p1("loot_find", 0.04)},
			{"Hoard Sense", "+5% loot find per level.", "open-treasure-chest", Stats{}, p1("loot_find", 0.05)},
		}},
		{"Smuggler", [10]ddSpec{
			{"Quick Hands", "+2% Abyss SPD per level.", "hood", Stats{}, p1("spd_pct", 0.02)},
			{"Back Alleys", "+3% gold from drops per level.", "open-treasure-chest", Stats{}, p1("gold_find", 0.03)},
			{"Slippery", "+2 DGE per level.", "spectre", Stats{DGE: 2}, nil},
			{"Contraband", "+3.5% gold from drops per level.", "open-treasure-chest", Stats{}, p1("gold_find", 0.035)},
			{"Fence Network", "+2.5% Abyss SPD per level.", "hood", Stats{}, p1("spd_pct", 0.025)},
			{"Black Market", "+4% gold from drops per level.", "open-treasure-chest", Stats{}, p1("gold_find", 0.04)},
			{"Untraceable", "+3 DGE per level.", "spectre", Stats{DGE: 3}, nil},
			{"Kingpin", "+4.5% gold from drops per level.", "open-treasure-chest", Stats{}, p1("gold_find", 0.045)},
			{"Ghost Runner", "+3% Abyss SPD per level.", "hood", Stats{}, p1("spd_pct", 0.03)},
			{"Shadow Broker", "+5% gold from drops per level.", "spectre", Stats{}, p1("gold_find", 0.05)},
		}},
		{"Banker", [10]ddSpec{
			{"Petty Cash", "+2% escrow floor bonus per level.", "laurel-crown", Stats{}, p1("escrow_bonus", 0.02)},
			{"Tithe Collector", "+3% tokens on bank per level.", "laurel-crown", Stats{}, p1("token_gain", 0.03)},
			{"Ledger Keeper", "+2.5% escrow floor bonus per level.", "checked-shield", Stats{}, p1("escrow_bonus", 0.025)},
			{"Toll Master", "+3.5% tokens on bank per level.", "laurel-crown", Stats{}, p1("token_gain", 0.035)},
			{"Vault Interest", "+3% escrow floor bonus per level.", "checked-shield", Stats{}, p1("escrow_bonus", 0.03)},
			{"Tribute Baron", "+4% tokens on bank per level.", "laurel-crown", Stats{}, p1("token_gain", 0.04)},
			{"Compound Interest", "+3.5% escrow floor bonus per level.", "checked-shield", Stats{}, p1("escrow_bonus", 0.035)},
			{"Mint Warden", "+4.5% tokens on bank per level.", "laurel-crown", Stats{}, p1("token_gain", 0.045)},
			{"Deep Vault", "+4% escrow floor bonus per level.", "checked-shield", Stats{}, p1("escrow_bonus", 0.04)},
			{"Reserve Bank", "+5% escrow floor bonus per level.", "legendary-star", Stats{}, p1("escrow_bonus", 0.05)},
		}},
		{"Tycoon", [10]ddSpec{
			{"Investor", "+4.5% gold from drops per level.", "open-treasure-chest", Stats{}, p1("gold_find", 0.045)},
			{"Speculator", "+4% tokens on bank per level.", "laurel-crown", Stats{}, p1("token_gain", 0.04)},
			{"Financier", "+3% escrow floor bonus per level.", "checked-shield", Stats{}, p1("escrow_bonus", 0.03)},
			{"Magnate", "+5% gold from drops per level.", "open-treasure-chest", Stats{}, p1("gold_find", 0.05)},
			{"Robber Baron", "+3.5% loot find per level.", "rolling-dices", Stats{}, p1("loot_find", 0.035)},
			{"Mogul", "+4.5% tokens on bank per level.", "laurel-crown", Stats{}, p1("token_gain", 0.045)},
			{"Empire Builder", "+5.5% gold from drops per level.", "open-treasure-chest", Stats{}, p1("gold_find", 0.055)},
			{"Plutocrat", "+4% crafting materials per level.", "unique-gem", Stats{}, p1("material_yield", 0.04)},
			{"Gold Dragon", "+4% escrow floor bonus per level.", "dragon-head", Stats{}, p1("escrow_bonus", 0.04)},
			{"Avatar of Avarice", "+8% gold from drops per level.", "legendary-star", Stats{}, p1("gold_find", 0.08)},
		}},
	},
	"warden": {
		{"Ironhide", [10]ddSpec{
			{"Tough Skin", "+2% Abyss max HP per level.", "health-normal", Stats{}, p1("hp_pct", 0.02)},
			{"Hearty", "+30 HP per level.", "health-normal", Stats{HP: 30}, nil},
			{"Thick Hide", "+2.5% Abyss max HP per level.", "checked-shield", Stats{}, p1("hp_pct", 0.025)},
			{"Robust", "+40 HP per level.", "health-normal", Stats{HP: 40}, nil},
			{"Ironhide", "+3% Abyss max HP per level.", "checked-shield", Stats{}, p1("hp_pct", 0.03)},
			{"Stalwart", "+50 HP per level.", "health-normal", Stats{HP: 50}, nil},
			{"Bulwark Body", "+3.5% Abyss max HP per level.", "checked-shield", Stats{}, p1("hp_pct", 0.035)},
			{"Unyielding", "+60 HP per level.", "health-normal", Stats{HP: 60}, nil},
			{"Mountainous", "+4% Abyss max HP per level.", "checked-shield", Stats{}, p1("hp_pct", 0.04)},
			{"Worldheart", "+5% Abyss max HP per level.", "legendary-star", Stats{}, p1("hp_pct", 0.05)},
		}},
		{"Sentinel", [10]ddSpec{
			{"Braced", "+3 DEF per level.", "visored-helm", Stats{DEF: 3}, nil},
			{"Shieldwall", "+4 DEF per level.", "visored-helm", Stats{DEF: 4}, nil},
			{"Bloodguard", "+0.3% of DEF as Lifesteal per level.", "totem", Stats{}, p1("def_to_lifesteal", 0.003)},
			{"Ironclad", "+5 DEF per level.", "visored-helm", Stats{DEF: 5}, nil},
			{"Stoneblood", "Convert 5% of HP into DEF per level.", "totem", Stats{}, p1("hp_to_def", 0.05)},
			{"Aegis", "+6 DEF per level.", "visored-helm", Stats{DEF: 6}, nil},
			{"Leechplate", "+0.4% of DEF as Lifesteal per level.", "totem", Stats{}, p1("def_to_lifesteal", 0.004)},
			{"Impenetrable", "+7 DEF per level.", "visored-helm", Stats{DEF: 7}, nil},
			{"Living Armor", "Convert 6% of HP into DEF per level.", "totem", Stats{}, p1("hp_to_def", 0.06)},
			{"Sanguine Wall", "+0.5% of DEF as Lifesteal per level.", "legendary-star", Stats{}, p1("def_to_lifesteal", 0.005)},
		}},
		{"Lifeward", [10]ddSpec{
			{"Field Medic", "+25 HP per level.", "totem", Stats{HP: 25}, nil},
			{"Recovery", "+2% Abyss max HP per level.", "health-normal", Stats{}, p1("hp_pct", 0.02)},
			{"Siphon", "+0.3% of DEF as Lifesteal per level.", "totem", Stats{}, p1("def_to_lifesteal", 0.003)},
			{"Resilient", "+35 HP per level.", "health-normal", Stats{HP: 35}, nil},
			{"Mending", "+2.5% Abyss max HP per level.", "health-normal", Stats{}, p1("hp_pct", 0.025)},
			{"Endurance", "+3 STA per level.", "totem", Stats{STA: 3}, nil},
			{"Second Life", "+45 HP per level.", "health-normal", Stats{HP: 45}, nil},
			{"Vampiric Guard", "+0.4% of DEF as Lifesteal per level.", "totem", Stats{}, p1("def_to_lifesteal", 0.004)},
			{"Regenerator", "+3% Abyss max HP per level.", "health-normal", Stats{}, p1("hp_pct", 0.03)},
			{"Undying Ward", "+60 HP per level.", "legendary-star", Stats{HP: 60}, nil},
		}},
		{"Guardian", [10]ddSpec{
			{"Watchful", "+2 DGE per level.", "checked-shield", Stats{DGE: 2}, nil},
			{"Guard Stance", "+4 DEF per level.", "visored-helm", Stats{DEF: 4}, nil},
			{"Deflect", "+2 DGE per level.", "spectre", Stats{DGE: 2}, nil},
			{"Warder", "+2% Abyss max HP per level.", "checked-shield", Stats{}, p1("hp_pct", 0.02)},
			{"Evade", "+3 DGE per level.", "spectre", Stats{DGE: 3}, nil},
			{"Protector", "+5 DEF per level.", "visored-helm", Stats{DEF: 5}, nil},
			{"Tireless", "+3 STA per level.", "totem", Stats{STA: 3}, nil},
			{"Untouchable Guard", "+3 DGE per level.", "spectre", Stats{DGE: 3}, nil},
			{"Sentry", "+2.5% Abyss max HP per level.", "checked-shield", Stats{}, p1("hp_pct", 0.025)},
			{"Phantom Guardian", "+4 DGE per level.", "legendary-star", Stats{DGE: 4}, nil},
		}},
		{"Colossus", [10]ddSpec{
			{"Giant's Blood", "+2.5% Abyss max HP per level.", "health-normal", Stats{}, p1("hp_pct", 0.025)},
			{"Titan Plating", "+5 DEF per level.", "visored-helm", Stats{DEF: 5}, nil},
			{"Behemoth", "+50 HP per level.", "health-normal", Stats{HP: 50}, nil},
			{"Adamant", "Convert 6% of HP into DEF per level.", "totem", Stats{}, p1("hp_to_def", 0.06)},
			{"Fortress", "+3% Abyss max HP per level.", "checked-shield", Stats{}, p1("hp_pct", 0.03)},
			{"Bloodfort", "+0.4% of DEF as Lifesteal per level.", "totem", Stats{}, p1("def_to_lifesteal", 0.004)},
			{"Immortal Frame", "+70 HP per level.", "health-normal", Stats{HP: 70}, nil},
			{"Bastion Lord", "+8 DEF per level.", "visored-helm", Stats{DEF: 8}, nil},
			{"Living Citadel", "+4% Abyss max HP per level.", "checked-shield", Stats{}, p1("hp_pct", 0.04)},
			{"Avatar of Endurance", "+6% Abyss max HP per level.", "legendary-star", Stats{}, p1("hp_pct", 0.06)},
		}},
	},
}

// SpecTalents holds each specialization's 50-node allocatable sub-tree, keyed by
// spec id. Each node's Spec is set so it only takes effect (and is only
// allocatable) while that specialization is active.
var SpecTalents = map[string][]Talent{
	"delver":    buildSpecTree("delver"),
	"plunderer": buildSpecTree("plunderer"),
	"warden":    buildSpecTree("warden"),
}

// buildSpecTree lays out one spec's five arms of ten tiers. The first arm's root
// hangs off the spec picker node (Edge) and gates only on the spec being active
// (Parent == ""); every other node gates on its predecessor.
func buildSpecTree(spec string) []Talent {
	const (
		colGap, baseX = 180, -360
		rowGap, baseY = 56, 280
	)
	branches := specTalentBranches[spec]
	out := make([]Talent, 0, 50)
	for c := 0; c < len(branches); c++ {
		br := branches[c]
		for r := 0; r < len(br.nodes); r++ {
			key := fmt.Sprintf("sp_%s_%d_%d", spec, c, r)
			var parent, edge string
			if r > 0 {
				parent = fmt.Sprintf("sp_%s_%d_%d", spec, c, r-1) // vertical chain within the arm
				edge = parent
			} else {
				parent = "" // every column root gates on the spec being active
				edge = spec // and draws its connector from the spec picker node
			}
			sp := br.nodes[r]
			out = append(out, Talent{
				Key: key, Label: sp.label, Desc: sp.desc,
				X: baseX + c*colGap, Y: baseY + r*rowGap,
				Parent: parent, Edge: edge, Icon: sp.icon, GateDepth: r * 2,
				Spec: spec, Stats: sp.stats, Pct: sp.pct,
			})
		}
	}
	return out
}

// talentByKey indexes every generic talent (Deep-Delver + all spec sub-trees) by
// key, for O(1) validation and effect summation.
var talentByKey = buildTalentIndex()

func buildTalentIndex() map[string]Talent {
	m := make(map[string]Talent, len(DeepDelverTalents))
	for _, t := range DeepDelverTalents {
		m[t.Key] = t
	}
	for _, list := range SpecTalents {
		for _, t := range list {
			m[t.Key] = t
		}
	}
	return m
}

// TalentByKey looks up a generic talent by key.
func TalentByKey(key string) (Talent, bool) {
	t, ok := talentByKey[key]
	return t, ok
}

// TalentBonus sums the allocated generic talents into one bonus block. Spec
// sub-tree nodes only count while their specialization is the active one, so
// switching specs re-scopes their power without wiping the allocation.
func TalentBonus(levels map[string]int, spec string) TreeBonus {
	tb := TreeBonus{Pct: map[string]float64{}}
	for key, lvl := range levels {
		if lvl <= 0 {
			continue
		}
		if lvl > TalentMaxLevel {
			lvl = TalentMaxLevel
		}
		t, ok := talentByKey[key]
		if !ok {
			continue
		}
		if t.Spec != "" && t.Spec != spec {
			continue
		}
		tb.Stats = tb.Stats.Add(t.Stats.Scaled(float64(lvl)))
		for k, v := range t.Pct {
			tb.Pct[k] += v * float64(lvl)
		}
	}
	return tb
}
