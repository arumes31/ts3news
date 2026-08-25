package content

import "testing"

func TestAbyssBiomeForAffinityWeightsMatchingBiome(t *testing.T) {
	t.Parallel()

	for _, depth := range []int{1, 15, 40} {
		weight := AbyssBiomeWeight(depth, "fire")
		fire, other := 0, 0
		for roll := 0; roll < weight; roll++ {
			if biome := AbyssBiomeForAffinity(depth, "fire", roll); biome.Affinity == "fire" {
				fire++
			} else {
				other++
			}
		}
		if fire != 3 {
			t.Fatalf("depth %d fire selections = %d, want 3", depth, fire)
		}
		if other != len(abyssBiomePool(depth))-1 {
			t.Fatalf("depth %d other selections = %d, want %d", depth, other, len(abyssBiomePool(depth))-1)
		}
	}
}

func TestAbyssBiomeForAffinityStaysInDepthPool(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		depth int
		name  string
	}{
		{depth: 1, name: "Cinder-Choked"},
		{depth: 15, name: "Bloodrust"},
		{depth: 40, name: "Pyre-Eternal"},
	} {
		found := false
		for roll := 0; roll < AbyssBiomeWeight(test.depth, "fire"); roll++ {
			if AbyssBiomeForAffinity(test.depth, "fire", roll).Name == test.name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("depth %d never selected %q", test.depth, test.name)
		}
	}
}

func TestAbyssBiomeForAffinityNormalizesNegativeRoll(t *testing.T) {
	t.Parallel()
	if got := AbyssBiomeForAffinity(1, "fire", -1); got.Name == "" {
		t.Fatal("negative roll returned an empty biome")
	}
}
