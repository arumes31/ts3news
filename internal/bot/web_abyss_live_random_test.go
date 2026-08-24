package bot

import (
	"reflect"
	"testing"

	"ts3news/internal/content"
)

func TestAbyssLiveRandomSameSeedProducesSameSequence(t *testing.T) {
	first := &abyssLiveCombat{randomSeed: [2]uint64{11, 22}}
	second := &abyssLiveCombat{randomSeed: [2]uint64{11, 22}}

	for draw := 0; draw < 32; draw++ {
		if got, want := first.IntN(10_000), second.IntN(10_000); got != want {
			t.Fatalf("draw %d differs for equal seeds: %d != %d", draw, got, want)
		}
	}
	if first.randomDrawCount() != 32 || second.randomDrawCount() != 32 {
		t.Fatalf(
			"draw counts = %d and %d, want 32",
			first.randomDrawCount(),
			second.randomDrawCount(),
		)
	}
}

func TestAbyssLiveRandomMakesEnemyPlansReproducible(t *testing.T) {
	users := []activeUser{
		{u: &UserInCombat{UID: "front", Nickname: "Front", CurrentHP: 100, Position: content.PositionFrontline}},
		{u: &UserInCombat{UID: "back", Nickname: "Back", CurrentHP: 100, Position: content.PositionBackline}},
	}
	mobs := []*content.Mob{{
		Name:   "Caster",
		Type:   content.MobTreasureGoblin,
		Stats:  content.Stats{HP: 100, SPD: 20},
		Spells: []content.Skill{{Name: "Bolt"}, {Name: "Nova"}},
	}}
	first := &abyssLiveCombat{randomSeed: [2]uint64{101, 202}}
	second := &abyssLiveCombat{randomSeed: [2]uint64{101, 202}}

	firstPlans := planLiveEnemyIntentsWithRandom(3, users, mobs, nil, first)
	secondPlans := planLiveEnemyIntentsWithRandom(3, users, mobs, nil, second)
	if !reflect.DeepEqual(firstPlans, secondPlans) {
		t.Fatalf("enemy plans differ for equal seeds:\nfirst: %+v\nsecond: %+v", firstPlans, secondPlans)
	}
}
