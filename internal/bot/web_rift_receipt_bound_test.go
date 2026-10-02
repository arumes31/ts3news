package bot

import (
	"reflect"
	"strconv"
	"testing"

	"ts3news/internal/rift"
)

func TestRiftDecodeBoundsReceiptPresentation(t *testing.T) {
	r := riftVitalsFixture()
	r.BankedItems = nil
	r.BankedLoot = nil
	for i := 0; i < 5000; i++ {
		name := "Gear " + strconv.Itoa(i)
		r.BankedItems = append(r.BankedItems, name)
		r.BankedLoot = append(r.BankedLoot, rift.BankedLoot{Name: name, Mission: 1, Tier: 1, Origin: "Kept origin"})
	}
	r.Gold, r.BankedGold = 17, 23
	saved, err := encodeRift(r)
	if err != nil {
		t.Fatal(err)
	}
	got, err := decodeRift(saved)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.BankedItems) != 200 || len(got.BankedLoot) != 200 || cap(got.BankedItems) > 200 || cap(got.BankedLoot) > 200 {
		t.Fatalf("unbounded receipt: %d items, %d metadata", len(got.BankedItems), len(got.BankedLoot))
	}
	if got.TotalBankedItems() != 5000 || got.Gold != 17 || got.BankedGold != 23 {
		t.Fatal("trim changed confirmed rewards")
	}
	if !reflect.DeepEqual(got.BankedItems, r.BankedItems[4800:]) || !reflect.DeepEqual(got.BankedLoot, r.BankedLoot[4800:]) {
		t.Fatal("newest receipt metadata changed")
	}
	again, err := encodeRift(got)
	if err != nil {
		t.Fatal(err)
	}
	twice, err := decodeRift(again)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, twice) {
		t.Fatal("receipt migration not idempotent")
	}
}

func TestRiftDecodeRejectsInvalidReceiptTotals(t *testing.T) {
	for _, total := range []int{-1, 1, 9007199254740992} {
		r := riftVitalsFixture()
		r.BankedItems = []string{"one", "two"}
		r.BankedItemsTotal = total
		saved, err := encodeRift(r)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = decodeRift(saved); err == nil {
			t.Fatalf("invalid receipt total accepted: %d", total)
		}
	}
}
