//go:build e2e

package bot

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"ts3news/internal/content"
)

// This opt-in fixture exercises the production combat engine and HTTP handlers.
// Entry, rewards, banking and revival are fixture orchestration. The deliberately
// small SQL double records committed live-session state; it is not PostgreSQL or
// evidence of production inventory/economy transaction correctness.
type abyssTransportFixture struct {
	mu                    sync.Mutex
	server                *WebServer
	store                 *abyssTransportStore
	uid                   string
	active                bool
	depth, hp             int
	gold, escrow          int64
	fights                int
	defeat                bool
	streamFailures        int
	streamRequests, polls int
}

func registerAbyssTransportFixture(t *testing.T, mux *http.ServeMux, templates *WebServer) {
	var fixtures sync.Map
	lookup := func(r *http.Request) *abyssTransportFixture {
		cookie, err := r.Cookie("abyss_transport_fixture")
		if err != nil {
			return nil
		}
		value, ok := fixtures.Load(cookie.Value)
		if !ok {
			return nil
		}
		return value.(*abyssTransportFixture)
	}
	mux.HandleFunc("/abyss/transport", func(w http.ResponseWriter, r *http.Request) {
		f := lookup(r)
		if f == nil {
			id, err := newAbyssLiveSessionID()
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			store := &abyssTransportStore{states: map[string]string{}}
			db := sql.OpenDB(abyssTransportConnector{store})
			// Production option loading may read the run loadout while its
			// consumable rows are still open; allow the nested read a connection.
			db.SetMaxOpenConns(4)
			t.Cleanup(func() { _ = db.Close() })
			f = &abyssTransportFixture{server: &WebServer{bot: &Bot{DB: db}}, store: store, uid: id, hp: 1000, gold: 5000}
			fixtures.Store(id, f)
			http.SetCookie(w, &http.Cookie{Name: "abyss_transport_fixture", Value: id, Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode})
		}
		f.mu.Lock()
		data := abyssGoldenFixture(f.active)
		run := data["Run"].(abyssRun)
		run.Depth, run.Escrow = f.depth, f.escrow
		data["Run"] = run
		user := data["U"].(*webUser)
		user.UID, user.CurrentHP, user.MaxHP, user.Gold = f.uid, f.hp, 1000, f.gold
		f.mu.Unlock()
		if err := templates.tmpl.ExecuteTemplate(w, "abyss", data); err != nil {
			http.Error(w, err.Error(), 500)
		}
	})
	register := func(path string, handler func(*abyssTransportFixture, http.ResponseWriter, *http.Request)) {
		mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			f := lookup(r)
			if f == nil {
				writeJSON(w, map[string]any{"ok": false, "error": "no active combat"})
				return
			}
			handler(f, w, r)
		})
	}
	register("/api/abyss/combat/state", func(f *abyssTransportFixture, w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		f.polls++
		f.mu.Unlock()
		f.server.handleAbyssCombatState(w, r, f.uid)
	})
	register("/api/abyss/combat/events", func(f *abyssTransportFixture, w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		f.streamRequests++
		fail := f.streamFailures > 0
		if fail {
			f.streamFailures--
		}
		f.mu.Unlock()
		if fail {
			http.Error(w, "fixture streaming outage", http.StatusServiceUnavailable)
			return
		}
		f.server.handleAbyssCombatEvents(w, r, f.uid)
	})
	register("/api/abyss/combat/action", func(f *abyssTransportFixture, w http.ResponseWriter, r *http.Request) {
		f.server.handleAbyssCombatAction(w, r, f.uid)
	})
	register("/api/abyss/combat/ready", func(f *abyssTransportFixture, w http.ResponseWriter, r *http.Request) {
		f.server.handleAbyssCombatReady(w, r, f.uid)
	})
	register("/api/abyss/combat/time_bank", func(f *abyssTransportFixture, w http.ResponseWriter, r *http.Request) {
		f.server.handleAbyssCombatTimeBank(w, r, f.uid)
	})
	register("/api/abyss/transport/control", func(f *abyssTransportFixture, w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		if r.Method == http.MethodPost {
			var control struct {
				Defeat         bool `json:"defeat"`
				StreamFailures int  `json:"stream_failures"`
			}
			if readJSON(r, &control) != nil {
				http.Error(w, "bad request", 400)
				return
			}
			f.defeat, f.streamFailures = control.Defeat, control.StreamFailures
		}
		f.store.mu.Lock()
		states := map[string]string{}
		for k, v := range f.store.states {
			states[k] = v
		}
		commits := f.store.commits
		f.store.mu.Unlock()
		writeJSON(w, map[string]any{"ok": true, "active": f.active, "depth": f.depth, "hp": f.hp, "gold": f.gold, "escrow": f.escrow, "fights": f.fights, "stream_requests": f.streamRequests, "polls": f.polls, "commits": commits, "session_states": states})
	})
	register("/api/abyss/enter", func(f *abyssTransportFixture, w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		if !f.active {
			f.active, f.depth, f.hp, f.escrow = true, 0, 1000, 0
		}
		writeJSON(w, map[string]any{"ok": true, "free_entry": true})
	})
	register("/api/abyss/descend", func(f *abyssTransportFixture, w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		if !f.active || f.hp <= 0 {
			f.mu.Unlock()
			writeJSON(w, map[string]any{"ok": false, "error": "enter or recover first"})
			return
		}
		if c, ok := f.server.liveCombatForUID(f.uid); ok && c.isActive() {
			f.mu.Unlock()
			writeJSON(w, map[string]any{"ok": true, "live_combat": true, "state": c.snapshotFor(f.uid)})
			return
		}
		c := f.startFightLocked()
		f.mu.Unlock()
		// Wait only for engine setup so the response includes legal targets/options.
		for range 200 {
			if c.snapshotFor(f.uid).Phase != "starting" {
				break
			}
			time.Sleep(5 * time.Millisecond)
		}
		writeJSON(w, map[string]any{"ok": true, "live_combat": true, "state": c.snapshotFor(f.uid)})
	})
	register("/api/abyss/bank", func(f *abyssTransportFixture, w http.ResponseWriter, r *http.Request) {
		var request struct {
			Preview bool `json:"preview"`
		}
		_ = readJSON(r, &request)
		f.mu.Lock()
		defer f.mu.Unlock()
		if request.Preview {
			writeJSON(w, map[string]any{"ok": true, "escrow": f.escrow, "source_escrow": f.escrow, "payout": f.escrow, "loot_count": 0})
			return
		}
		banked := f.escrow
		f.gold += banked
		f.escrow = 0
		f.active = false
		writeJSON(w, map[string]any{"ok": true, "banked": banked, "gold": f.gold, "depth": f.depth, "tokens": 12, "mult": 1})
	})
	register("/api/abyss/revive", func(f *abyssTransportFixture, w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		f.hp = 1000
		f.defeat = false
		writeJSON(w, map[string]any{"ok": true, "victory": true, "hp": f.hp, "max_hp": 1000, "gold": f.gold, "tokens": 12, "escrow": f.escrow, "logs": []string{}, "loot": []any{}, "dura": []any{}, "timeline": []any{}, "consumables": []any{}})
	})
}

func (f *abyssTransportFixture) startFightLocked() *abyssLiveCombat {
	f.fights++
	id := fmt.Sprintf("transport-%s-%d", f.uid, f.fights)
	c := &abyssLiveCombat{server: f.server, id: id, ownerUID: f.uid, participants: map[string]bool{f.uid: true}, tactics: map[string]string{f.uid: "balanced"}, options: map[string][]abyssLiveOption{}, queued: map[string]abyssLiveAction{}, idempotency: map[string]abyssLiveIdempotency{}, phase: "starting", pauseMode: "fast", createdAt: time.Now(), randomSeed: [2]uint64{7, 11}}
	c.ensureSocialLocked()
	f.server.liveCombats.Store(id, c)
	f.server.liveCombatByUID.Store(f.uid, id)
	user := UserInCombat{UID: f.uid, Nickname: "Transport Delver", Level: 10, Stats: content.Stats{HP: 1000, STR: 200, DEF: 100, SPD: 100}, CurrentHP: f.hp, EscrowLoot: true, IsClone: true, shadow: true, live: c}
	mob := &content.Mob{Name: "Goblin", Level: 1, Type: content.MobCommon, Stats: content.Stats{HP: 1200, STR: 12, DEF: 5, SPD: 1}, MaxHP: 1200, CurrentHP: 1200}
	if f.defeat {
		user.CurrentHP = 1
		user.Stats.STR = 1
		mob.Stats.STR = 10000
		mob.Stats.SPD = 1000
	}
	go func() {
		users := []UserInCombat{user}
		logs, _, won, _, timeline, _ := f.server.bot.resolveChannelCombatDetailedWithRandom(users, []*content.Mob{mob}, 10, 1, content.Zone{Difficulty: 1}, rand.New(rand.NewPCG(7, 11)))
		f.mu.Lock()
		f.depth++
		f.hp = max(0, users[0].CurrentHP)
		if won {
			f.escrow += 100
		}
		depth, hp, gold, escrow := f.depth, f.hp, f.gold, f.escrow
		f.mu.Unlock()
		c.complete(map[string]any{"ok": true, "victory": won, "depth": depth, "hp": hp, "max_hp": 1000, "gold": gold, "tokens": 12, "escrow": escrow, "bonus": 100, "risk": 10, "logs": logs, "loot": []any{}, "dura": []any{}, "timeline": timeline, "consumables": []any{}, "can_revive": true, "revive_chance_pct": 100})
	}()
	return c
}

// Only UPDATE live session state is retained. Other reads return empty rows and
// bookkeeping writes are discarded. Commit is the visibility boundary.
type abyssTransportStore struct {
	mu      sync.Mutex
	states  map[string]string
	commits int
}
type abyssTransportConnector struct{ store *abyssTransportStore }

func (c abyssTransportConnector) Connect(context.Context) (driver.Conn, error) {
	return &abyssTransportConn{store: c.store}, nil
}
func (c abyssTransportConnector) Driver() driver.Driver { return abyssTransportDriver{} }

type abyssTransportDriver struct{}

func (abyssTransportDriver) Open(string) (driver.Conn, error) {
	return nil, fmt.Errorf("use connector")
}

type abyssTransportConn struct {
	store   *abyssTransportStore
	pending map[string]string
}

func (*abyssTransportConn) Prepare(string) (driver.Stmt, error) {
	return nil, fmt.Errorf("prepared statements unsupported in transport fixture")
}
func (*abyssTransportConn) Close() error { return nil }
func (c *abyssTransportConn) Begin() (driver.Tx, error) {
	c.pending = map[string]string{}
	return c, nil
}
func (c *abyssTransportConn) Commit() error {
	c.store.mu.Lock()
	defer c.store.mu.Unlock()
	for k, v := range c.pending {
		c.store.states[k] = v
	}
	c.store.commits++
	c.pending = nil
	return nil
}
func (c *abyssTransportConn) Rollback() error { c.pending = nil; return nil }
func (c *abyssTransportConn) ExecContext(_ context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	if strings.Contains(query, "UPDATE abyss_combat_sessions") && len(args) == 7 {
		state, id := args[5].Value.(string), args[6].Value.(string)
		var decoded abyssLivePersistedState
		if err := json.Unmarshal([]byte(state), &decoded); err != nil {
			return nil, err
		}
		if c.pending == nil {
			return nil, fmt.Errorf("session update requires transaction")
		}
		c.pending[id] = state
	}
	return driver.RowsAffected(1), nil
}
func (*abyssTransportConn) QueryContext(context.Context, string, []driver.NamedValue) (driver.Rows, error) {
	return abyssTransportRows{}, nil
}

type abyssTransportRows struct{}

func (abyssTransportRows) Columns() []string         { return []string{"value"} }
func (abyssTransportRows) Close() error              { return nil }
func (abyssTransportRows) Next([]driver.Value) error { return io.EOF }
