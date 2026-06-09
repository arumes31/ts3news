package content

import (
	"strings"
	"testing"
)

func TestGetRandomZone(t *testing.T) {
	for i := 0; i < 100; i++ {
		z := GetRandomZone(10, 50.5)
		if z.Name == "" {
			t.Error("Zone name should not be empty")
		}
		if z.Difficulty <= 0 {
			t.Errorf("Zone %q has non-positive difficulty: %f", z.Name, z.Difficulty)
		}
		if len(z.Effects) == 0 {
			t.Errorf("Zone %q should have at least one effect", z.Name)
		}
	}
}

func TestZoneDisplay(t *testing.T) {
	z := Zone{
		Name: "Test Zone",
		Difficulty: 1.2,
		Effects: []ZoneEffect{
			{Name: "Effect1", Type: ZoneBuff},
		},
	}
	d := z.Display()
	if !strings.Contains(d, "Test Zone") || !strings.Contains(d, "1.20") || !strings.Contains(d, "Effect1") {
		t.Errorf("Display() = %q", d)
	}
}
