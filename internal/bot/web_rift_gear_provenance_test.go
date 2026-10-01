package bot

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"ts3news/internal/content"
	"ts3news/internal/rift"
)

type riftGearProvenanceCheck struct {
	origin, at string
	cap        content.Rarity
}

func (check riftGearProvenanceCheck) Match(value driver.Value) bool {
	raw, ok := value.(string)
	if !ok {
		return false
	}
	var gear content.Gear
	return json.Unmarshal([]byte(raw), &gear) == nil && content.IsAbyssGearID(gear.ID) && gear.FoundBoss == check.origin && gear.FoundAt == check.at && gear.Rarity <= check.cap
}

func TestRiftGearProvenanceThroughRewardSaveAndInventory(t *testing.T) {
	now := time.Date(2026, 9, 24, 16, 30, 0, 0, time.FixedZone("author", 7200))
	for _, level := range rift.Campaign() {
		for room := range rift.Rooms {
			t.Run(fmt.Sprintf("mission-%d-tier-%d", level.ID, room+1), func(t *testing.T) {
				database, mock, err := sqlmock.New()
				if err != nil {
					t.Fatal(err)
				}
				defer database.Close()
				run := rift.NewRunAtLevel("provenance", rift.Build{HP: 100}, now, riftMobCatalog(now), level.ID)
				run.Room = room
				if len(level.Rooms[room].FragileFloor) > 0 {
					run.RoomObjective = &rift.RoomObjective{Kind: "survive_waves"}
					for _, box := range level.Rooms[room].FragileFloor {
						run.RoomObjective.FloorSegments = append(run.RoomObjective.FloorSegments, rift.WaveFloorSegment{Obstacle: box})
					}
				} else {
					run.RoomObjective = nil
				}
				run.Epoch = "2"
				run.Revision = 1
				run.Drops = []rift.Drop{{ID: "enemy-origin", Mission: level.ID, Tier: room + 1, NeedsGear: true, Collected: true}}
				raw, err := encodeRift(run)
				if err != nil {
					t.Fatal(err)
				}
				origin := fmt.Sprintf("Rift Brawl: %s · Tier %d", level.Name, room+1)
				foundAt := "2026-09-24T14:30:00Z"
				mock.ExpectBegin()
				mock.ExpectQuery("SELECT client_uid FROM users").WithArgs("owner").WillReturnRows(sqlmock.NewRows([]string{"client_uid"}).AddRow("owner"))
				mock.ExpectQuery("SELECT COALESCE").WillReturnRows(sqlmock.NewRows([]string{"epoch"}).AddRow("2"))
				mock.ExpectQuery("SELECT value FROM app_meta").WithArgs("rift_brawl:owner").WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(raw))
				mock.ExpectExec("INSERT INTO app_meta").WithArgs("rift_brawl:owner", riftSnapshotCheck(func(saved *rift.Run) bool {
					if len(saved.Drops) != 1 {
						return false
					}
					d := saved.Drops[0]
					return d.Gear != nil && d.ID == "enemy-origin" && d.Mission == level.ID && d.Tier == room+1 && d.Gear.FoundBoss == origin && d.Gear.FoundAt == foundAt
				})).WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectCommit()
				saved, err := (&Bot{DB: database}).updateRift(context.Background(), "owner", riftRequest{Kind: "pause", RunID: run.ID, Revision: 2, RequestID: "roll"}, rift.Build{}, now)
				if err != nil {
					t.Fatal(err)
				}
				gear := saved.Drops[0].Gear
				saved.Objectives = nil
				mock.ExpectBegin()
				mock.ExpectExec("SELECT set_config").WithArgs("rift_brawl", "bank", run.ID, "").WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectExec("INSERT INTO user_inventory").WithArgs("owner", gear.ID, gear.MaxDurability, riftGearProvenanceCheck{origin, foundAt, rift.LootRarityCap(room)}).WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
				tx, err := database.Begin()
				if err != nil {
					t.Fatal(err)
				}
				if err = bankRift(context.Background(), tx, "owner", "bank", saved); err != nil {
					t.Fatal(err)
				}
				if err = tx.Commit(); err != nil {
					t.Fatal(err)
				}
				encoded, err := encodeRift(saved)
				if err != nil {
					t.Fatal(err)
				}
				restored, err := decodeRift(encoded)
				if err != nil {
					t.Fatal(err)
				}
				want := rift.BankedLoot{Name: gear.Name, Rarity: int(gear.Rarity), Mission: level.ID, Tier: room + 1, Origin: origin, FoundAt: foundAt}
				if len(restored.BankedLoot) != 1 || restored.BankedLoot[0] != want {
					t.Fatalf("receipt provenance changed: %+v", restored.BankedLoot)
				}
				if err = mock.ExpectationsWereMet(); err != nil {
					t.Fatal(err)
				}
			})
		}
	}
}
