package bot

import (
	"strings"

	"ts3news/internal/content"
)

const (
	abyssRunFlagStoryCampaign = "story_campaign"
	abyssRunFlagStoryComplete = "story_campaign_complete"
	abyssRunFlagBiomeChoice   = "biome_choice"
	abyssRunFlagBiomeUntil    = "biome_until"
)

type abyssBiomeContract struct {
	ID       int64  `json:"id"`
	Key      string `json:"key"`
	Name     string `json:"name"`
	Icon     string `json:"icon"`
	Affinity string `json:"affinity"`
	Promise  string `json:"promise"`
	Warning  string `json:"warning"`
}

var abyssBiomeContracts = []abyssBiomeContract{
	{ID: 1, Key: "cinder", Name: "Cinder March", Icon: "🔥", Affinity: "fire", Promise: "+volatile cache pressure", Warning: "hotter, harder floors"},
	{ID: 2, Key: "verdant", Name: "Verdant Coil", Icon: "🌿", Affinity: "nature", Promise: "+steady mastery routes", Warning: "rootbound attrition"},
	{ID: 3, Key: "tempest", Name: "Tempest Crown", Icon: "⚡", Affinity: "storm", Promise: "+fast elite encounters", Warning: "storm-wracked danger"},
	{ID: 4, Key: "void", Name: "Void Pilgrimage", Icon: "◉", Affinity: "void", Promise: "+deep-biome mastery", Warning: "the deadliest route"},
}

type abyssStoryFloor struct {
	Depth    int    `json:"depth"`
	Title    string `json:"title"`
	Subtitle string `json:"subtitle"`
	Affinity string `json:"affinity"`
	Modifier string `json:"modifier"`
}

var abyssStoryCampaign = []abyssStoryFloor{
	{Depth: 1, Title: "The Sealed Gate", Subtitle: "Break the lock that remembers your name.", Affinity: "fire", Modifier: "fragile_cache"},
	{Depth: 2, Title: "Roots Below Stone", Subtitle: "Follow the living map beneath the citadel.", Affinity: "nature", Modifier: "mirror_clone"},
	{Depth: 3, Title: "The Empty Choir", Subtitle: "Cross the nave while unseen voices count your steps.", Affinity: "spirit", Modifier: "darkness"},
	{Depth: 4, Title: "Engine of Rain", Subtitle: "Silence the stormworks before the vault floods.", Affinity: "storm", Modifier: "storm_floor"},
	{Depth: 5, Title: "The First Warden", Subtitle: "Defeat the keeper of the lower seal.", Affinity: "frost", Modifier: "iron_skin"},
	{Depth: 6, Title: "Archive of Ash", Subtitle: "Recover the chronicle from a burning library.", Affinity: "fire", Modifier: "artifact_corrupted"},
	{Depth: 7, Title: "The Drowned Road", Subtitle: "Walk where the old expedition vanished.", Affinity: "water", Modifier: "no_healing"},
	{Depth: 8, Title: "Hall of Borrowed Faces", Subtitle: "Outfight the shape the Abyss made from you.", Affinity: "void", Modifier: "mirror_clone"},
	{Depth: 9, Title: "Sovereign Stair", Subtitle: "Climb through the final guard in total darkness.", Affinity: "void", Modifier: "darkness"},
	{Depth: 10, Title: "Heart of the Descent", Subtitle: "End the authored hunt and claim the Chronicle.", Affinity: "spirit", Modifier: "enraged"},
}

func abyssBiomeContractByID(id int64) (abyssBiomeContract, bool) {
	for _, contract := range abyssBiomeContracts {
		if contract.ID == id {
			return contract, true
		}
	}
	return abyssBiomeContract{}, false
}

func abyssStoryFloorAt(depth int) (abyssStoryFloor, bool) {
	if depth < 1 || depth > len(abyssStoryCampaign) {
		return abyssStoryFloor{}, false
	}
	return abyssStoryCampaign[depth-1], true
}

func abyssStoryFloorFromFlags(flags map[string]int64, depth int) (abyssStoryFloor, bool) {
	if flags[abyssRunFlagStoryCampaign] != 1 {
		return abyssStoryFloor{}, false
	}
	return abyssStoryFloorAt(depth)
}

func abyssSelectedBiomeContract(flags map[string]int64, depth int) (abyssBiomeContract, bool) {
	if flags[abyssRunFlagBiomeUntil] < int64(depth) {
		return abyssBiomeContract{}, false
	}
	return abyssBiomeContractByID(flags[abyssRunFlagBiomeChoice])
}

func abyssBiomeForRun(
	depth int,
	seasonAffinity string,
	roll int,
	flags map[string]int64,
) (content.AbyssBiome, string) {
	if story, ok := abyssStoryFloorFromFlags(flags, depth); ok {
		return abyssBiomeForAffinity(story.Affinity, depth), story.Title
	}
	if contract, ok := abyssSelectedBiomeContract(flags, depth); ok {
		return abyssBiomeForAffinity(contract.Affinity, depth), contract.Name
	}
	weight := content.AbyssBiomeWeight(depth, seasonAffinity)
	return content.AbyssBiomeForAffinity(depth, seasonAffinity, roll%max(weight, 1)), ""
}

func abyssBiomeForAffinity(affinity string, depth int) content.AbyssBiome {
	biomes := content.AbyssBiomes()
	matches := make([]content.AbyssBiome, 0, len(biomes))
	for _, biome := range biomes {
		if strings.EqualFold(biome.Affinity, affinity) {
			matches = append(matches, biome)
		}
	}
	if len(matches) == 0 {
		return content.AbyssBiomeFor(depth)
	}
	return matches[depth%len(matches)]
}
