package rift

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"
	"ts3news/internal/content"
)

func mutateCampaignValue(value reflect.Value) {
	switch value.Kind() {
	case reflect.Pointer:
		if !value.IsNil() {
			mutateCampaignValue(value.Elem())
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < value.Len(); i++ {
			mutateCampaignValue(value.Index(i))
		}
	case reflect.Struct:
		for i := 0; i < value.NumField(); i++ {
			mutateCampaignValue(value.Field(i))
		}
	case reflect.String:
		value.SetString("mutated")
	case reflect.Int:
		value.SetInt(value.Int() + 1)
	case reflect.Float64:
		value.SetFloat(value.Float() + 1)
	case reflect.Bool:
		value.SetBool(!value.Bool())
	}
}

func TestCampaignDefinitionsAreDetachedFromEveryMutableField(t *testing.T) {
	original := Campaign()
	baseline, err := json.Marshal(original)
	if err != nil {
		t.Fatal(err)
	}
	other := Campaign()
	mutateCampaignValue(reflect.ValueOf(other))
	unchanged, err := json.Marshal(original)
	if err != nil {
		t.Fatal(err)
	}
	current, err := json.Marshal(Campaign())
	if err != nil {
		t.Fatal(err)
	}
	if string(baseline) != string(unchanged) || string(baseline) != string(current) {
		t.Fatal("campaign callers share mutable content")
	}
	for id := 1; id <= LevelCount; id++ {
		run := NewRunAtLevel("copy", Build{HP: 100}, time.Unix(100, 0), nil, id)
		mutateCampaignValue(reflect.ValueOf(run.Level))
	}
	current, err = json.Marshal(Campaign())
	if err != nil {
		t.Fatal(err)
	}
	if string(baseline) != string(current) {
		t.Fatal("run arena mutation leaked into campaign")
	}
}

func BenchmarkCampaignMissionSelection(b *testing.B) {
	catalog := content.AbyssMobCatalog()
	now := time.Unix(100, 0)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		NewRunAtLevel("bench", Build{HP: 100}, now, catalog, 42)
	}
}
func BenchmarkCampaignPublicCopy(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		Campaign()
	}
}
