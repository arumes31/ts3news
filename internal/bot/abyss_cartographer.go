package bot

const (
	abyssCartographerEventType   = "lost_cartographer"
	abyssCartographerRouteLength = 5
)

type abyssCartographerFloor struct {
	Depth int    `json:"depth"`
	Type  string `json:"type"`
}

type abyssCartographerRoute struct {
	Floors         []abyssCartographerFloor `json:"floors"`
	NextEventDepth int                      `json:"next_event_depth"`
}

type abyssCartographerFloorView struct {
	Depth int    `json:"depth"`
	Type  string `json:"type"`
	Label string `json:"label"`
	Icon  string `json:"icon"`
}

type abyssCartographerRouteView struct {
	Active    bool                         `json:"active"`
	Remaining int                          `json:"remaining"`
	Floors    []abyssCartographerFloorView `json:"floors"`
}

type abyssCartographerPlanInput struct {
	Depth          int
	LastRestDepth  int
	NextEventDepth int
	Pacts          []string
}

func abyssCartographerMapCost(depth int) int64 {
	return max(int64(250), int64(max(depth, 0)+1)*75)
}

func planAbyssCartographerRoute(
	input abyssCartographerPlanInput,
	rollFloor func() string,
	rollGap func() int,
) abyssCartographerRoute {
	nextEventDepth := input.NextEventDepth
	if nextEventDepth <= input.Depth {
		nextEventDepth = input.Depth + boundedAbyssEventGap(rollGap())
	}

	lastRestDepth := input.LastRestDepth
	floors := make([]abyssCartographerFloor, 0, abyssCartographerRouteLength)
	for offset := 1; offset <= abyssCartographerRouteLength; offset++ {
		depth := input.Depth + offset
		var floorType string
		switch {
		case depth%abyssBossEvery == 0 || abyssPactBossFloor(input.Pacts, depth):
			floorType = "combat"
		case abyssRestFloorDue(lastRestDepth, depth) && abyssPactAllowsRest(input.Pacts):
			floorType = "rest"
			lastRestDepth = depth
		case depth >= nextEventDepth:
			floorType = "event"
			nextEventDepth = depth + boundedAbyssEventGap(rollGap())
		case abyssHasPact(input.Pacts, "famine"):
			floorType = "combat"
		default:
			floorType = normalizeAbyssMappedFloor(rollFloor())
			if floorType == "rest" {
				lastRestDepth = depth
			}
		}
		floors = append(floors, abyssCartographerFloor{Depth: depth, Type: floorType})
	}

	return abyssCartographerRoute{Floors: floors, NextEventDepth: nextEventDepth}
}

func boundedAbyssEventGap(gap int) int {
	return min(max(gap, abyssEventGapMin), abyssEventGapMax)
}

func normalizeAbyssMappedFloor(floorType string) string {
	if floorType == "rest" {
		return floorType
	}
	return "combat"
}

func abyssCartographerFloorAt(route abyssCartographerRoute, depth int) (string, bool) {
	for _, floor := range route.Floors {
		if floor.Depth == depth {
			return normalizeAbyssMappedFloorType(floor.Type), true
		}
	}
	return "", false
}

func buildAbyssCartographerRouteView(
	route abyssCartographerRoute,
	currentDepth int,
) abyssCartographerRouteView {
	floors := make([]abyssCartographerFloorView, 0, len(route.Floors))
	for _, floor := range route.Floors {
		if floor.Depth <= currentDepth {
			continue
		}
		floorType := normalizeAbyssMappedFloorType(floor.Type)
		info := floorCandidateInfo[floorType]
		floors = append(floors, abyssCartographerFloorView{
			Depth: floor.Depth,
			Type:  floorType,
			Label: info.Label,
			Icon:  info.Icon,
		})
	}
	return abyssCartographerRouteView{
		Active:    len(floors) > 0,
		Remaining: len(floors),
		Floors:    floors,
	}
}

func normalizeAbyssMappedFloorType(floorType string) string {
	switch floorType {
	case "combat", "rest", "event":
		return floorType
	default:
		return "combat"
	}
}
