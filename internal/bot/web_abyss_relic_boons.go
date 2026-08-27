package bot

import "ts3news/internal/content"

const abyssRunFlagBoonDraftDepth = "boon_draft_depth"

type abyssRunRelic struct {
	ID     int64  `json:"id"`
	Key    string `json:"key"`
	Name   string `json:"name"`
	Icon   string `json:"icon"`
	Effect string `json:"effect"`
}

var abyssRunRelics = []abyssRunRelic{
	{ID: 1, Key: "ember_lens", Name: "Ember Lens", Icon: "◈", Effect: "+12% STR this run"},
	{ID: 2, Key: "bastion_seal", Name: "Bastion Seal", Icon: "⬢", Effect: "+12% DEF this run"},
	{ID: 3, Key: "tidal_heart", Name: "Tidal Heart", Icon: "◆", Effect: "+15% maximum HP this run"},
	{ID: 4, Key: "gale_spur", Name: "Gale Spur", Icon: "✦", Effect: "+12% SPD this run"},
}

type abyssRunBoon struct {
	ID     int64  `json:"id"`
	Key    string `json:"key"`
	Name   string `json:"name"`
	Icon   string `json:"icon"`
	Effect string `json:"effect"`
	Stacks int64  `json:"stacks,omitempty"`
}

var abyssRunBoons = []abyssRunBoon{
	{ID: 1, Key: "giants_favor", Name: "Giant's Favor", Icon: "⚔", Effect: "+8% STR per stack"},
	{ID: 2, Key: "iron_oath", Name: "Iron Oath", Icon: "⬡", Effect: "+8% DEF per stack"},
	{ID: 3, Key: "quickblood", Name: "Quickblood", Icon: "➶", Effect: "+8% SPD per stack"},
	{ID: 4, Key: "deep_well", Name: "Deep Well", Icon: "♥", Effect: "+10% maximum HP per stack"},
	{ID: 5, Key: "fortune_thread", Name: "Fortune Thread", Icon: "◇", Effect: "+10 LCK per stack"},
	{ID: 6, Key: "arcanum", Name: "Arcanum", Icon: "✧", Effect: "+8% INT per stack"},
}

type abyssBoonDraftView struct {
	Pending bool           `json:"pending"`
	Depth   int            `json:"depth"`
	Options []abyssRunBoon `json:"options"`
}

type abyssRunAdvance struct {
	Relic         *abyssRunRelic     `json:"relic,omitempty"`
	Draft         abyssBoonDraftView `json:"draft"`
	StoryComplete bool               `json:"story_complete,omitempty"`
}

func abyssRunRelicFlag(id int64) string { return "run_relic_" + itoa(int(id)) }

func abyssRunBoonFlag(id int64) string { return "run_boon_" + itoa(int(id)) }

func abyssRunRelicByID(id int64) (abyssRunRelic, bool) {
	for _, relic := range abyssRunRelics {
		if relic.ID == id {
			return relic, true
		}
	}
	return abyssRunRelic{}, false
}

func abyssRunBoonByID(id int64) (abyssRunBoon, bool) {
	for _, boon := range abyssRunBoons {
		if boon.ID == id {
			return boon, true
		}
	}
	return abyssRunBoon{}, false
}

func abyssBoonDraftFromFlags(flags map[string]int64) abyssBoonDraftView {
	depth := int(flags[abyssRunFlagBoonDraftDepth])
	view := abyssBoonDraftView{
		Depth:   max(depth, 0),
		Options: []abyssRunBoon{},
	}
	if depth <= 0 {
		return view
	}
	offset := max(0, depth/5-1) % len(abyssRunBoons)
	for _, step := range []int{0, 2, 4, 1, 3, 5} {
		boon := abyssRunBoons[(offset+step)%len(abyssRunBoons)]
		boon.Stacks = flags[abyssRunBoonFlag(boon.ID)]
		if boon.Stacks >= 3 {
			continue
		}
		view.Options = append(view.Options, boon)
		if len(view.Options) == 3 {
			break
		}
	}
	view.Pending = len(view.Options) > 0
	if !view.Pending {
		view.Depth = 0
	}
	return view
}

func abyssRunChoicePending(flags map[string]int64) bool {
	return abyssBoonDraftFromFlags(flags).Pending
}

func advanceAbyssRunIdentity(flags map[string]int64, depth int) abyssRunAdvance {
	advance := abyssRunAdvance{Draft: abyssBoonDraftFromFlags(flags)}
	if depth > 0 && depth%4 == 0 {
		start := (depth/4 - 1) % len(abyssRunRelics)
		for step := range len(abyssRunRelics) {
			relic := abyssRunRelics[(start+step)%len(abyssRunRelics)]
			if flags[abyssRunRelicFlag(relic.ID)] > 0 {
				continue
			}
			flags[abyssRunRelicFlag(relic.ID)] = 1
			copy := relic
			advance.Relic = &copy
			break
		}
	}
	if depth > 0 && depth%5 == 0 {
		flags[abyssRunFlagBoonDraftDepth] = int64(depth)
		advance.Draft = abyssBoonDraftFromFlags(flags)
		if !advance.Draft.Pending {
			flags[abyssRunFlagBoonDraftDepth] = 0
		}
	}
	if flags[abyssRunFlagStoryCampaign] == 1 && depth >= len(abyssStoryCampaign) {
		flags[abyssRunFlagStoryComplete] = 1
		advance.StoryComplete = true
	}
	return advance
}

func applyAbyssRunIdentityBuild(user *UserInCombat, flags map[string]int64) {
	for _, relic := range abyssRunRelics {
		if flags[abyssRunRelicFlag(relic.ID)] == 0 {
			continue
		}
		switch relic.Key {
		case "ember_lens":
			user.Stats.STR += user.Stats.STR * 12 / 100
		case "bastion_seal":
			user.Stats.DEF += user.Stats.DEF * 12 / 100
		case "tidal_heart":
			user.Stats.HP += user.Stats.HP * 15 / 100
		case "gale_spur":
			user.Stats.SPD += user.Stats.SPD * 12 / 100
		}
	}
	for _, boon := range abyssRunBoons {
		stacks := min(flags[abyssRunBoonFlag(boon.ID)], int64(3))
		if stacks <= 0 {
			continue
		}
		switch boon.Key {
		case "giants_favor":
			user.Stats.STR += user.Stats.STR * int(stacks) * 8 / 100
		case "iron_oath":
			user.Stats.DEF += user.Stats.DEF * int(stacks) * 8 / 100
		case "quickblood":
			user.Stats.SPD += user.Stats.SPD * int(stacks) * 8 / 100
		case "deep_well":
			user.Stats.HP += user.Stats.HP * int(stacks) * 10 / 100
		case "fortune_thread":
			user.Stats.LCK += int(stacks) * 10
		case "arcanum":
			user.Stats.INT += user.Stats.INT * int(stacks) * 8 / 100
		}
	}
}

func abyssRunRelicViews(flags map[string]int64) []abyssRunRelic {
	views := []abyssRunRelic{}
	for _, relic := range abyssRunRelics {
		if flags[abyssRunRelicFlag(relic.ID)] > 0 {
			views = append(views, relic)
		}
	}
	return views
}

func abyssRunBoonViews(flags map[string]int64) []abyssRunBoon {
	views := []abyssRunBoon{}
	for _, boon := range abyssRunBoons {
		boon.Stacks = flags[abyssRunBoonFlag(boon.ID)]
		if boon.Stacks > 0 {
			views = append(views, boon)
		}
	}
	return views
}

func abyssRunIdentityCombatLog(flags map[string]int64) []string {
	logs := []string{}
	if relics := abyssRunRelicViews(flags); len(relics) > 0 {
		logs = append(logs, "[color=#62d6c5]◈ Run relics resonate: "+itoa(len(relics))+" temporary passives active.[/color]")
	}
	if boons := abyssRunBoonViews(flags); len(boons) > 0 {
		logs = append(logs, "[color=#efbd59]✦ Drafted boons answer: "+itoa(len(boons))+" run buffs active.[/color]")
	}
	return logs
}

func clearAbyssRunIdentityFlags(flags map[string]int64) bool {
	keys := []string{
		abyssRunFlagStoryCampaign,
		abyssRunFlagStoryComplete,
		abyssRunFlagBiomeChoice,
		abyssRunFlagBiomeUntil,
		abyssRunFlagBiomeSelectedAt,
		abyssRunFlagBoonDraftDepth,
	}
	for _, relic := range abyssRunRelics {
		keys = append(keys, abyssRunRelicFlag(relic.ID))
	}
	for _, boon := range abyssRunBoons {
		keys = append(keys, abyssRunBoonFlag(boon.ID))
	}
	removed := false
	for _, key := range keys {
		if _, exists := flags[key]; exists {
			delete(flags, key)
			removed = true
		}
	}
	return removed
}

func abyssRunIdentityStats(flags map[string]int64) content.Stats {
	user := UserInCombat{Stats: content.Stats{HP: 100, STR: 100, DEF: 100, SPD: 100, INT: 100}}
	applyAbyssRunIdentityBuild(&user, flags)
	return user.Stats
}
