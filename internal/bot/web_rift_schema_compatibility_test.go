package bot

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestRiftDecodeLegacyOptionalFields(t *testing.T) {
	// Intentionally literal: constructors would silently add today's optional fields.
	legacy := `{"schema":1,"id":"legacy-v1","room":0,"status":"fighting","epoch":"old-economy","revision":7,"player":{"id":"player","x":160,"y":410,"hp":80,"max_hp":100,"mana":40},"skill_timers":{},"gold":17,"banked_gold":23,"banked_items":["Earlier sword"],"drops":[]}`
	run, err := decodeRift(legacy)
	if err != nil {
		t.Fatal(err)
	}
	if run.Level != nil || run.Practice != nil || run.History != nil || run.PauseStartedMS != nil || run.MissionDefinition != "" {
		t.Fatal("legacy absence was replaced with invented content")
	}
	if run.Schema != 1 || run.ID != "legacy-v1" || run.Epoch != "old-economy" || run.Revision != 7 || run.Gold != 17 || run.BankedGold != 23 || !reflect.DeepEqual(run.BankedItems, []string{"Earlier sword"}) {
		t.Fatal("legacy identity or rewards changed")
	}
	saved, err := encodeRift(run)
	if err != nil {
		t.Fatal(err)
	}
	again, err := decodeRift(saved)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(run, again) {
		t.Fatal("legacy decode/encode/decode is not stable")
	}
	// A future additive field is harmless to reads. Writers must not depend on an
	// older reader retaining unknown fields, because encoding drops them.
	var fields map[string]any
	if err = json.Unmarshal([]byte(legacy), &fields); err != nil {
		t.Fatal(err)
	}
	fields["optional_future_hint"] = map[string]any{"label": "new"}
	raw, err := json.Marshal(fields)
	if err != nil {
		t.Fatal(err)
	}
	withHint, err := decodeRift(string(raw))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(run, withHint) {
		t.Fatal("unknown optional hint changed known state")
	}
	for _, schema := range []int{-1, 0, 2, 999} {
		fields["schema"] = schema
		raw, err = json.Marshal(fields)
		if err != nil {
			t.Fatal(err)
		}
		if got, err := decodeRift(string(raw)); err == nil || got != nil {
			t.Fatalf("unsupported schema %d accepted", schema)
		}
	}
}
