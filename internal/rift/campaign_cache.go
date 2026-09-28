package rift

import "slices"

// Owned by this package and never returned directly or mutated after startup.
var campaignDefinitions = buildCampaign()

// Campaign returns detached authored definitions. Mutable room state belongs
// to callers; mission selection copies only its chosen definition.
func Campaign() []Level {
	levels := make([]Level, len(campaignDefinitions))
	for i, level := range campaignDefinitions {
		levels[i] = cloneCampaignLevel(level)
	}
	return levels
}

func cloneCampaignLevel(level Level) Level {
	level.Rooms = slices.Clone(level.Rooms)
	for i := range level.Rooms {
		room := &level.Rooms[i]
		// Translations initialize after package-level campaign authoring.
		room.LootRarityCeiling = LootRarityCap(i).String()
		room.Obstacles = slices.Clone(room.Obstacles)
		room.HighCover = slices.Clone(room.HighCover)
		room.Hazards = slices.Clone(room.Hazards)
		room.WindGusts = slices.Clone(room.WindGusts)
		room.WaterCurrents = slices.Clone(room.WaterCurrents)
		room.SteamVents = slices.Clone(room.SteamVents)
		room.Platforms = slices.Clone(room.Platforms)
		room.DropEdges = slices.Clone(room.DropEdges)
		room.Cover = slices.Clone(room.Cover)
		if room.RestAlcove != nil {
			rest := *room.RestAlcove
			room.RestAlcove = &rest
		}
		if room.WaveGate != nil {
			gate := *room.WaveGate
			room.WaveGate = &gate
		}
		if room.Exit != nil {
			exit := *room.Exit
			room.Exit = &exit
		}
		if room.Entrance != nil {
			entry := *room.Entrance
			room.Entrance = &entry
		}
		if room.HazardSwitch != nil {
			s := *room.HazardSwitch
			room.HazardSwitch = &s
		}
		if room.Encounter != nil {
			encounter := *room.Encounter
			room.Encounter = &encounter
		}
	}
	return level
}
