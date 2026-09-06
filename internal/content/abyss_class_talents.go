package content

import "fmt"

type AbyssTalent struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Tier        int     `json:"tier"`
	Branch      int     `json:"branch"`
	Effect      string  `json:"effect"`
	Amount      float64 `json:"amount"`
	Description string  `json:"description"`
	Art         string  `json:"art"`
}
type AbyssTalentTree struct {
	ID       string        `json:"id"`
	Class    string        `json:"class"`
	Subclass string        `json:"subclass"`
	Nodes    []AbyssTalent `json:"nodes"`
	Budget   int           `json:"budget"`
}

var AbyssClassPointFloors = [...]int64{1, 5, 15, 35, 75, 575, 1325, 2325, 3825, 5825, 8325, 11325, 14825, 18825, 23825}

func AbyssClassPoints(xp int64) int {
	n := 0
	for _, f := range AbyssClassPointFloors {
		if xp < f*1000 {
			break
		}
		n++
	}
	return n
}
func AbyssClassByID(id string) (AbyssClass, bool) {
	for _, c := range AbyssClasses() {
		if c.ID == id {
			return c, true
		}
	}
	return AbyssClass{}, false
}

// Effects describe real resolver inputs. Class and subclass trees share combat
// rules, while names and artwork carry each discipline's identity.
func AbyssTalents(id string) AbyssTalentTree {
	tree := AbyssTalentTree{ID: id, Budget: 5}
	name := ""
	primary := "STR"
	if c, ok := AbyssClassByID(id); ok {
		tree.Class = c.ID
		name = c.Name
		if id == "arcanist" || id == "warden" || id == "artificer" {
			primary = "INT"
		}
	}
	if sub, ok := AbyssSubclassByID(id); ok {
		tree.Class = sub.ClassID
		tree.Subclass = id
		tree.Budget = 10
		name = sub.Name
		primary = sub.Scaling
	}
	if name == "" {
		return tree
	}
	type spec struct {
		effect             string
		amount             float64
		title, description string
	}
	rows := [][]spec{
		{{"primary", .05, "Discipline", "+5% " + primary + "."}, {"health", .08, "Vitality", "+8% maximum HP."}, {"mana", .10, "Reservoir", "+10% maximum mana."}},
		{{"builder_power", .12, "Opening Strike", "Signature builder has +12% power."}, {"builder_shield", .25, "Guarded Opening", "Signature builder adds a self barrier equal to 25% DEF; total barriers cap at 50% maximum HP."}, {"builder_heal", .02, "Renewing Opening", "Signature builder heals the caster for 2% maximum HP."}},
		{{"finisher_power", .15, "Decisive Force", "Signature finisher has +15% power."}, {"penetration", .08, "Armor Breach", "Signature attacks ignore an additional 8% defense."}, {"mana_discount", 3, "Measured Casting", "Signature actions cost 3 less mana, minimum 5."}},
		{{"resource_power", .05, "Stored Might", "Charged finishers gain an additional 5% power per resource spent."}, {"finisher_heal", .03, "Second Wind", "Signature finishers heal the caster for 3% maximum HP."}, {"regen", 3, "Steady Recovery", "Recover 3 additional mana at the start of each round."}},
		{{"low_health", .12, "Last Defiance", "Signature attacks have +12% power below 50% caster HP."}, {"marked_power", .10, "Exploit Opening", "Signature finishers have +10% power against the builder's marked target."}, {"defense", .08, "Firm Footing", "+8% DEF."}},
	}
	if tree.Subclass != "" {
		rows = append(rows, []spec{
			{"cap_burst", .35, "Cataclysm", "Signature finisher has +35% power and costs one extra cooldown round."},
			{"cap_aegis", .20, "Living Bastion", "Charged finishers grant a self barrier equal to 20% maximum HP, but finisher power is reduced by 15%. Total barriers cap at 50% maximum HP."},
			{"cap_siphon", .10, "Endless Renewal", "Charged finishers heal 10% maximum HP but cost 10 additional mana."},
		})
	}
	for tier, row := range rows {
		for branch, entry := range row {
			nodeID := fmt.Sprintf("%s_t%d_%d", id, tier+1, branch+1)
			tree.Nodes = append(tree.Nodes, AbyssTalent{ID: nodeID, Name: name + " · " + entry.title, Tier: tier, Branch: branch, Effect: entry.effect, Amount: entry.amount, Description: entry.description, Art: "/static/abyss_talents/" + nodeID + ".svg"})
		}
	}
	return tree
}

// Foundation actions are earned at the first and third class point. The final
// foundation tier opens the empowered subclass's existing signature pair.
func AbyssFoundationStyle(id string) (AbyssSubclass, bool) {
	c, ok := AbyssClassByID(id)
	if !ok {
		return AbyssSubclass{}, false
	}
	styles := map[string][3]string{
		"warrior": {"Momentum", "Training Strike", "Driving Blow"}, "ranger": {"Aim", "Tracking Shot", "Aimed Release"},
		"arcanist": {"Spark", "Arcane Spark", "Arcane Pulse"}, "warden": {"Spirit", "Spirit Thorn", "Verdant Surge"},
		"reaver": {"Hunger", "Dusk Cut", "Dusk Rend"}, "artificer": {"Energy", "Charged Bolt", "Overload"},
	}
	v := styles[id]
	primary := "STR"
	if id == "arcanist" || id == "warden" || id == "artificer" {
		primary = "INT"
	}
	return AbyssSubclass{ID: id, ClassID: id, Name: c.Name, Resource: v[0], Builder: v[1], Finisher: v[2], Scaling: primary, Sequence: "Use " + v[1] + " to build " + v[0] + ", then spend it with " + v[2] + ".", Stats: []string{primary, "HP", "MNA"}, Gear: c.Description, Buffs: "Support your primary stat and mana recovery."}, true
}
func AbyssCombatStyle(id string) (AbyssSubclass, bool) {
	if s, ok := AbyssSubclassByID(id); ok {
		return s, true
	}
	return AbyssFoundationStyle(id)
}
