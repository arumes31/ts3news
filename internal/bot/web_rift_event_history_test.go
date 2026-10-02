package bot

import (
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"ts3news/internal/rift"
)

func TestRiftDecodeBoundsEventHistory(t *testing.T) {
	for _, total := range []int{1000, 5000} {
		t.Run(strconv.Itoa(total), func(t *testing.T) {
			original := rift.NewRun("event-history", rift.Build{HP: 100}, time.Now())
			original.Gold, original.BankedGold, original.Counter = 17, 23, total
			original.Events = make([]rift.Event, total)
			for i := range original.Events {
				original.Events[i] = rift.Event{ID: i + 1, Kind: "hit", X: float64(i), Y: 400, Value: 3, ActorName: "Recorded target"}
			}
			encoded, err := encodeRift(original)
			if err != nil {
				t.Fatal(err)
			}
			if total == 5000 && !strings.HasPrefix(encoded, riftCompressedSnapshotPrefix) {
				t.Fatal("large fixture did not exercise compressed storage")
			}
			restored, err := decodeRift(encoded)
			if err != nil {
				t.Fatal(err)
			}
			if len(restored.Events) != 40 || cap(restored.Events) > 40 {
				t.Fatalf("event history retains len=%d cap=%d", len(restored.Events), cap(restored.Events))
			}
			if !reflect.DeepEqual(restored.Events, original.Events[total-40:]) {
				t.Fatal("newest event metadata changed")
			}
			if restored.Counter != total || restored.Gold != 17 || restored.BankedGold != 23 {
				t.Fatal("trimming changed event identity or rewards")
			}
			restored.Land(1)
			if len(restored.Events) != 40 || restored.Events[39].ID != total+1 {
				t.Fatal("event sequence did not continue after recovery")
			}
			encoded, err = encodeRift(restored)
			if err != nil {
				t.Fatal(err)
			}
			again, err := decodeRift(encoded)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(again.Events, restored.Events) {
				t.Fatal("bounded history did not survive another save")
			}
		})
	}
}
