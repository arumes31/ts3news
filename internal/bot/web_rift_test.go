package bot

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"ts3news/internal/content"
	"ts3news/internal/rift"
)

func TestRiftLootCapsMatchMissionPreviews(t *testing.T) {
	for room, expected := range []content.Rarity{content.RarityEpic, content.RarityEpic, content.RarityLegendary} {
		if rift.LootRarityCap(room) != expected {
			t.Fatalf("tier %d cap changed", room+1)
		}
		for _, level := range rift.Campaign() {
			if level.Rooms[room].LootRarityCeiling != expected.String() {
				t.Fatalf("mission %d tier %d preview mismatch", level.ID, room+1)
			}
		}
		available := false
		for _, gear := range content.AbyssGearCatalog() {
			if gear.Rarity == expected {
				available = true
			}
		}
		if !available {
			t.Fatalf("tier %d ceiling has no catalog gear", room+1)
		}
		for range 100 {
			gear, err := rollRiftGear(room, time.Unix(100, 0))
			if err != nil {
				t.Fatal(err)
			}
			if gear.Rarity > expected {
				t.Fatalf("tier %d rolled above ceiling: %v", room+1, gear.Rarity)
			}
		}
	}
}

func TestRiftBankAtomicAndReplaySafe(t *testing.T) {
	for _, scenario := range []string{"success", "duplicate", "state failure", "old epoch", "wrong run", "fighting", "advance", "advance duplicate", "advance failure"} {
		t.Run(scenario, func(t *testing.T) {
			database, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer database.Close()
			run := rift.NewRun("run", rift.Build{Name: "Delver", HP: 200}, time.Unix(100, 0))
			advance := strings.HasPrefix(scenario, "advance")
			if advance {
				run = rift.NewRunAtLevel("run", rift.Build{Name: "Delver", HP: 200}, time.Unix(100, 0), riftMobCatalog(time.Unix(100, 0)), 10)
				run.Room = 2
			}
			run.Status = "cleared"
			run.Epoch = "2"
			run.Revision = 4
			run.Drops = []rift.Drop{{ID: "drop", Gold: 30, Collected: true, Gear: &content.Gear{ID: "ABYSS_TEST", Name: "Test Blade", MaxDurability: 80}}}
			request := riftRequest{Kind: "exit", RunID: "run", Revision: 5, RequestID: "checkpoint-request-1"}
			if advance {
				request.Kind = "advance"
			}
			if scenario == "duplicate" || scenario == "advance duplicate" {
				request.Revision = 4
				run.Drops[0].Banked = true
				run.Status = "banked"
				run.BankedGold = 30
				if advance {
					run.Status = "cleared"
					run.FinishCheckpoint("advance", riftMobCatalog(time.Unix(100, 0)))
				}
			}
			if scenario == "old epoch" {
				run.Epoch = "1"
			}
			if scenario == "wrong run" {
				request.RunID = "someone-else"
			}
			if scenario == "fighting" {
				run.Status = "fighting"
			}
			data, err := json.Marshal(run)
			if err != nil {
				t.Fatal(err)
			}
			mock.ExpectBegin()
			mock.ExpectQuery("SELECT client_uid FROM users").WithArgs("owner").WillReturnRows(sqlmock.NewRows([]string{"client_uid"}).AddRow("owner"))
			mock.ExpectQuery("SELECT COALESCE").WillReturnRows(sqlmock.NewRows([]string{"epoch"}).AddRow("2"))
			mock.ExpectQuery("SELECT value FROM app_meta").WithArgs("rift_brawl:owner").WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(string(data)))
			if scenario == "success" || scenario == "state failure" || scenario == "advance" || scenario == "advance failure" {
				mock.ExpectExec("SELECT set_config").WithArgs("rift_brawl", request.RequestID, "run", "").WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectExec("INSERT INTO user_inventory").WithArgs("owner", "ABYSS_TEST", 80, sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectExec("UPDATE users SET gold").WithArgs(int64(30), "owner").WillReturnResult(sqlmock.NewResult(0, 1))
				write := mock.ExpectExec("INSERT INTO app_meta").WithArgs("rift_brawl:owner", sqlmock.AnyArg())
				if scenario == "state failure" || scenario == "advance failure" {
					write.WillReturnError(errors.New("disk unavailable"))
					mock.ExpectRollback()
				} else {
					write.WillReturnResult(sqlmock.NewResult(0, 1))
					mock.ExpectCommit()
				}
			} else {
				mock.ExpectRollback()
			}
			out, err := (&Bot{DB: database}).updateRift(context.Background(), "owner", request, rift.Build{}, time.Unix(101, 0))
			if scenario == "success" || scenario == "duplicate" || scenario == "advance" || scenario == "advance duplicate" {
				if err != nil || out.BankedGold != 30 {
					t.Fatalf("bank failed: %v %+v", err, out)
				}
				if advance && (out.Level.ID != 11 || out.Room != 0 || out.Status != "fighting" || len(out.CompletedLevels) != 1 || out.CompletedLevels[0] != 10) {
					t.Fatalf("advance lost persisted state: %+v", out)
				}
			} else if err == nil {
				t.Fatal("invalid or failed settlement accepted")
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestRiftRejectsUntrustedRequestsBeforeDatabase(t *testing.T) {
	for _, scenario := range []string{"origin", "cross-site", "content-type", "forged hp", "movement", "oversize", "method", "invalid mission", "negative mission"} {
		t.Run(scenario, func(t *testing.T) {
			body := `{"kind":"step","request_id":"test-request-123456","input":{"x":0}}`
			if scenario == "invalid mission" {
				body = `{"kind":"start","request_id":"test-request-123456","level_id":101}`
			}
			if scenario == "negative mission" {
				body = `{"kind":"start","request_id":"test-request-123456","level_id":-1}`
			}
			if scenario == "forged hp" {
				body = `{"kind":"step","request_id":"test-request-123456","hp":999}`
			}
			if scenario == "movement" {
				body = `{"kind":"step","request_id":"test-request-123456","input":{"x":999}}`
			}
			if scenario == "oversize" {
				body = strings.Repeat("x", 5000)
			}
			method := http.MethodPost
			if scenario == "method" {
				method = http.MethodDelete
			}
			req := httptest.NewRequest(method, "https://game.test/api/abyss/rift", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			if scenario == "origin" {
				req.Header.Set("Origin", "https://attacker.test")
			}
			if scenario == "cross-site" {
				req.Header.Set("Sec-Fetch-Site", "cross-site")
			}
			if scenario == "content-type" {
				req.Header.Set("Content-Type", "text/plain")
			}
			w := httptest.NewRecorder()
			(&WebServer{}).handleRiftAPI(w, req, "owner")
			if w.Code < 400 {
				t.Fatalf("untrusted request accepted: %d", w.Code)
			}
		})
	}
}

func TestRiftStartRetainsCampaignProgress(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	old := rift.NewRunAtLevel("old", rift.Build{HP: 200}, time.Unix(100, 0), riftMobCatalog(time.Unix(100, 0)), 10)
	old.Status = "complete"
	old.Epoch = "2"
	old.CompletedLevels = []int{1, 10}
	old.BankedGold = 300
	data, err := json.Marshal(old)
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT client_uid FROM users").WithArgs("owner").WillReturnRows(sqlmock.NewRows([]string{"client_uid"}).AddRow("owner"))
	mock.ExpectQuery("SELECT COALESCE").WillReturnRows(sqlmock.NewRows([]string{"epoch"}).AddRow("2"))
	mock.ExpectQuery("SELECT value FROM app_meta").WithArgs("rift_brawl:owner").WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(string(data)))
	mock.ExpectExec("INSERT INTO app_meta").WithArgs("rift_brawl:owner", sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	run, err := (&Bot{DB: database}).updateRift(context.Background(), "owner", riftRequest{Kind: "start", RunID: "old", RequestID: "new-mission-request", LevelID: 42}, rift.Build{HP: 200}, time.Unix(101, 0))
	if err != nil {
		t.Fatal(err)
	}
	if run.Level.ID != 42 || len(run.CompletedLevels) != 2 || run.CompletedLevels[1] != 10 || run.BankedGold != 0 || run.ID == "old" {
		t.Fatalf("incorrect new expedition: %+v", run)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRiftUsesCharacterAndOwnedSkills(t *testing.T) {
	base := UserInCombat{Stats: content.Stats{HP: 300, STR: 50, DEF: 20}, AbyssSubclass: "berserker", Skills: []content.Skill{{ID: "owned", Name: "Owned Strike", Type: content.SkillPhysical, Power: 2, ManaCost: 20}}}
	build := riftBuildFromUser(base, "Known player", 9)
	if build.Name != "Known player" || build.Class != "berserker" || len(build.Skills) != 1 || build.Skills[0].ID != "owned" {
		t.Fatalf("lost character identity: %+v", build)
	}
	base.Stats.STR = 500
	if riftBuildFromUser(base, "Known player", 9).Damage <= build.Damage {
		t.Fatal("gear stats did not affect action damage")
	}
}

func TestRiftLootStaysWithinStageRarity(t *testing.T) {
	for room := 0; room < 3; room++ {
		for i := 0; i < 30; i++ {
			gear, err := rollRiftGear(room, time.Now())
			if err != nil {
				t.Fatal(err)
			}
			cap := content.RarityEpic
			if room == 2 {
				cap = content.RarityLegendary
			}
			if !content.IsAbyssGearID(gear.ID) || gear.Rarity > cap || gear.FoundAt == "" {
				t.Fatalf("invalid stage loot: %+v", gear)
			}
		}
	}
}

func TestRiftReadExpiresObsoleteExpedition(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	run := rift.NewRun("old-run", rift.Build{HP: 200}, time.Now())
	run.Epoch = "old"
	run.Gold = 99
	encoded, err := json.Marshal(run)
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectQuery("SELECT value FROM app_meta").WithArgs("rift_brawl:owner").WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(string(encoded)))
	mock.ExpectQuery("SELECT COALESCE").WillReturnRows(sqlmock.NewRows([]string{"epoch"}).AddRow("new"))
	got, err := loadRift(context.Background(), database, "owner")
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "expired" || got.ID != "old-run" || got.Gold != 0 {
		t.Fatalf("cannot recover expired expedition: %+v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRiftMapsEveryAbyssSubclass(t *testing.T) {
	for _, class := range content.AbyssClasses() {
		for _, sub := range class.Subclasses {
			t.Run(sub.ID, func(t *testing.T) {
				u := UserInCombat{AbyssClass: class.ID, AbyssSubclass: sub.ID, Stats: content.Stats{HP: 500, STR: 80, INT: 90, DEF: 60}, Skills: content.AbyssClassSkills(sub.ID)}
				build := riftBuildFromUser(u, "Delver", 20)
				if build.Class != sub.ID || build.BaseClass != class.ID || build.Resource != sub.Resource || len(build.Signatures) != 2 || len(build.Skills) != 0 {
					t.Fatalf("subclass mapping lost: %+v", build)
				}
				if build.Signatures[0].Role != "builder" || build.Signatures[1].Role != "finisher" {
					t.Fatal("signature roles lost")
				}
			})
		}
	}
}
