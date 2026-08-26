package bot

import (
	"encoding/json"
	"testing"
)

func TestPlanAbyssCartographerRouteHonorsForcedFloorsAndEventCadence(t *testing.T) {
	t.Parallel()

	rolled := []string{"rest", "combat"}
	rollIndex := 0
	route := planAbyssCartographerRoute(
		abyssCartographerPlanInput{
			Depth:          12,
			LastRestDepth:  10,
			NextEventDepth: 14,
		},
		func() string {
			floor := rolled[rollIndex]
			rollIndex++
			return floor
		},
		func() int { return 3 },
	)

	want := []abyssCartographerFloor{
		{Depth: 13, Type: "rest"},
		{Depth: 14, Type: "event"},
		{Depth: 15, Type: "combat"},
		{Depth: 16, Type: "combat"},
		{Depth: 17, Type: "event"},
	}
	if len(route.Floors) != len(want) {
		t.Fatalf("floors = %#v", route.Floors)
	}
	for i := range want {
		if route.Floors[i] != want[i] {
			t.Fatalf("floor %d = %#v, want %#v", i, route.Floors[i], want[i])
		}
	}
	if route.NextEventDepth != 20 {
		t.Fatalf("next event depth = %d, want 20", route.NextEventDepth)
	}
}

func TestPlanAbyssCartographerRouteAppliesPactRules(t *testing.T) {
	t.Parallel()

	route := planAbyssCartographerRoute(
		abyssCartographerPlanInput{
			Depth:          1,
			LastRestDepth:  0,
			NextEventDepth: 9,
			Pacts:          []string{"deep_drums", "famine"},
		},
		func() string { return "rest" },
		func() int { return 4 },
	)

	for _, floor := range route.Floors {
		if floor.Type != "combat" {
			t.Fatalf("famine route contains %#v", floor)
		}
	}
	if route.Floors[1].Depth != 3 {
		t.Fatalf("deep-drums boss floor missing: %#v", route.Floors)
	}
}

func TestAbyssCartographerRouteViewDropsTraversedFloors(t *testing.T) {
	t.Parallel()

	route := abyssCartographerRoute{Floors: []abyssCartographerFloor{
		{Depth: 21, Type: "combat"},
		{Depth: 22, Type: "event"},
		{Depth: 23, Type: "invalid"},
	}}
	view := buildAbyssCartographerRouteView(route, 21)
	if !view.Active || view.Remaining != 2 || len(view.Floors) != 2 {
		t.Fatalf("view = %+v", view)
	}
	if view.Floors[0].Type != "event" || view.Floors[1].Type != "combat" {
		t.Fatalf("normalized view = %+v", view)
	}
	if floorType, ok := abyssCartographerFloorAt(route, 23); !ok || floorType != "combat" {
		t.Fatalf("floor lookup = %q, %v", floorType, ok)
	}
}

func TestAbyssCartographerMapCostScalesFromMinimum(t *testing.T) {
	t.Parallel()

	if got := abyssCartographerMapCost(0); got != 250 {
		t.Fatalf("floor zero cost = %d, want 250", got)
	}
	if shallow, deep := abyssCartographerMapCost(10), abyssCartographerMapCost(50); deep <= shallow {
		t.Fatalf("cost did not scale: shallow %d, deep %d", shallow, deep)
	}
}

func TestPrepareAbyssCartographerEventPostsServerPrice(t *testing.T) {
	t.Parallel()

	var state struct {
		Type        string `json:"type"`
		Depth       int    `json:"depth"`
		Price       int64  `json:"price"`
		RouteLength int    `json:"route_length"`
	}
	raw := prepareAbyssEventForDepth(`{"type":"lost_cartographer"}`, 20)
	if err := json.Unmarshal([]byte(raw), &state); err != nil {
		t.Fatal(err)
	}
	if state.Type != abyssCartographerEventType || state.Depth != 20 ||
		state.Price != abyssCartographerMapCost(20) || state.RouteLength != abyssCartographerRouteLength {
		t.Fatalf("prepared state = %+v", state)
	}
}
