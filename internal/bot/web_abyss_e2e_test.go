//go:build e2e

package bot

import (
	"mime"
	"net/http"
	"path"
	"strings"
	"testing"

	"ts3news/internal/content"
	"ts3news/internal/i18n"
)

func TestAbyssE2EServer(t *testing.T) {
	if err := i18n.InitWithLocale(i18n.LocaleEnUS); err != nil {
		t.Fatal(err)
	}
	server, err := NewWebServer(nil)
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("/static/", func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/static/")
		if name == "" || path.Clean(name) != name {
			http.NotFound(w, r)
			return
		}
		ServeAsset(
			w,
			r,
			"webassets/"+name,
			mime.TypeByExtension(path.Ext(name)),
		)
	})
	mux.HandleFunc("/abyss", func(w http.ResponseWriter, r *http.Request) {
		fixture := abyssGoldenFixture(r.URL.Query().Get("active") == "1")
		if r.URL.Query().Get("gear") == "1" {
			equipped := map[content.GearSlot]content.Gear{
				content.SlotMainHand: {
					ID: "TEST_BLADE", Name: "Cinder Test Blade", Slot: content.SlotMainHand,
					Element: content.ElementFire, Rune: string(content.ElementFire),
					Rarity: content.RarityEpic, MaxDurability: 100,
					Stats: content.Stats{STR: 60, INT: 20},
				},
				content.SlotOffHand: {
					ID: "TEST_FOCUS", Name: "Tideglass Focus", Slot: content.SlotOffHand,
					Rarity: content.RarityEpic, MaxDurability: 80,
					Stats: content.Stats{STR: 40, INT: 60},
				},
			}
			contributions := abyssGearDamageContributions(equipped)
			views := make([]gearView, 0, len(equipped))
			for _, slot := range []content.GearSlot{content.SlotMainHand, content.SlotOffHand} {
				view := toGearView(slot, equipped[slot])
				view.Damage = contributions[slot]
				view.Durability = view.MaxDurability
				views = append(views, view)
			}
			fixture["Equipped"] = views
			fixture["ForgeWorkbenchEnabled"] = true
			fixture["ForgeOperations"] = abyssForgeOperations()
			fixture["ForgeWorkbench"] = abyssForgeWorkbenchData{
				SchemaVersion: 1,
				StatPresets:   map[string]map[string]int{"balanced": {"STR": 100, "INT": 100}},
				QueuePolicies: []string{"stop"},
				MaterialFlow:  map[string][]int64{},
				CraftCap:      10,
				PresetSlots:   1,
				CosmeticTheme: "apprentice",
			}
		}
		room := r.URL.Query().Get("room")
		if room == abyssForgeFloorType || room == abyssEventChainType || room == abyssCartographerEventType {
			run := fixture["Run"].(abyssRun)
			run.FloorType = "event"
			run.EventState = prepareAbyssEventForDepth(`{"type":"`+room+`"}`, run.Depth)
			fixture["Run"] = run
		}
		if err := server.tmpl.ExecuteTemplate(w, "abyss", fixture); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
	mux.HandleFunc("/abyss/ops", func(w http.ResponseWriter, _ *http.Request) {
		if err := server.tmpl.ExecuteTemplate(w, "abyssOps", map[string]any{"Title": "Abyss Operations", "Nav": "ops"}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
	t.Log("Abyss E2E fixture listening on http://127.0.0.1:18082")
	if err := http.ListenAndServe("127.0.0.1:18082", mux); err != nil {
		t.Fatal(err)
	}
}
