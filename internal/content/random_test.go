package content

import (
	"math/rand/v2"
	"reflect"
	"testing"
)

func TestSpawnMobGroupWithRandomIsReproducible(t *testing.T) {
	zone := Zone{
		Name:       "Replay Vault",
		Difficulty: 1.2,
		Effects: []ZoneEffect{{
			Name: "Surge",
			Type: ZoneSpecial,
		}},
	}
	first := rand.New(rand.NewPCG(77, 99))
	second := rand.New(rand.NewPCG(77, 99))

	firstGroup := SpawnMobGroupWithRandom(20, zone, 1.3, 2, false, first)
	secondGroup := SpawnMobGroupWithRandom(20, zone, 1.3, 2, false, second)
	if !reflect.DeepEqual(firstGroup, secondGroup) {
		t.Fatalf("spawned groups differ for equal seeds:\nfirst: %+v\nsecond: %+v", firstGroup, secondGroup)
	}
}
