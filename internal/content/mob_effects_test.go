package content

import "testing"

func TestEveryMobEffectHasStablePresentationMetadata(t *testing.T) {
	t.Parallel()
	effects := []MobEffect{
		EffectEnraged,
		EffectArmored,
		EffectFleet,
		EffectPoisoned,
		EffectWeakened,
		EffectBlinded,
		EffectRegen,
		EffectSilenced,
	}
	seen := make(map[string]bool, len(effects))
	for _, effect := range effects {
		info, ok := MobEffectDetails(effect)
		if !ok {
			t.Errorf("effect %q has no presentation metadata", effect)
			continue
		}
		if info.Key == "" || info.Icon == "" || info.Description == "" || info.Tone == "" {
			t.Errorf("effect %q has incomplete metadata: %+v", effect, info)
		}
		if seen[info.Key] {
			t.Errorf("effect %q reuses key %q", effect, info.Key)
		}
		seen[info.Key] = true
	}
}

func TestUnknownMobEffectHasNoPresentationMetadata(t *testing.T) {
	t.Parallel()
	if _, ok := MobEffectDetails(MobEffect("untrusted")); ok {
		t.Fatal("unknown mob effect unexpectedly has presentation metadata")
	}
}
