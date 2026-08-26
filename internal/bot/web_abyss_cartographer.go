package bot

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"net/http"
	"strconv"
)

func abyssCartographerRouteKey(uid string) string {
	return "abyss_cartographer_route_" + uid
}

func abyssNextEventDepthKey(uid string) string {
	return "abyss_next_event_depth_" + uid
}

func decodeAbyssCartographerRoute(raw string) (abyssCartographerRoute, error) {
	route := abyssCartographerRoute{Floors: []abyssCartographerFloor{}}
	if raw == "" {
		return route, nil
	}
	if err := json.Unmarshal([]byte(raw), &route); err != nil {
		return abyssCartographerRoute{}, fmt.Errorf("decode cartographer route: %w", err)
	}
	if route.Floors == nil {
		route.Floors = []abyssCartographerFloor{}
	}
	if len(route.Floors) > abyssCartographerRouteLength {
		route.Floors = route.Floors[:abyssCartographerRouteLength]
	}
	return route, nil
}

func loadAbyssCartographerRouteInTx(
	ctx context.Context,
	tx *sql.Tx,
	uid string,
) (abyssCartographerRoute, bool, error) {
	var raw string
	err := tx.QueryRowContext(
		ctx,
		"SELECT value FROM app_meta WHERE key=$1 FOR UPDATE",
		abyssCartographerRouteKey(uid),
	).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return abyssCartographerRoute{Floors: []abyssCartographerFloor{}}, false, nil
	}
	if err != nil {
		return abyssCartographerRoute{}, false, fmt.Errorf("load cartographer route: %w", err)
	}
	route, err := decodeAbyssCartographerRoute(raw)
	if err != nil {
		return abyssCartographerRoute{}, false, err
	}
	return route, true, nil
}

func loadAbyssNextEventDepthInTx(ctx context.Context, tx *sql.Tx, uid string) (int, error) {
	var raw string
	err := tx.QueryRowContext(
		ctx,
		"SELECT value FROM app_meta WHERE key=$1",
		abyssNextEventDepthKey(uid),
	).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("load next event depth: %w", err)
	}
	depth, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("decode next event depth: %w", err)
	}
	return depth, nil
}

func saveAbyssCartographerRouteInTx(
	ctx context.Context,
	tx *sql.Tx,
	uid string,
	route abyssCartographerRoute,
) error {
	payload, err := json.Marshal(route)
	if err != nil {
		return fmt.Errorf("encode cartographer route: %w", err)
	}
	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO app_meta (key, value) VALUES ($1, $2)
		 ON CONFLICT (key) DO UPDATE SET value=EXCLUDED.value`,
		abyssCartographerRouteKey(uid),
		string(payload),
	); err != nil {
		return fmt.Errorf("save cartographer route: %w", err)
	}
	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO app_meta (key, value) VALUES ($1, $2)
		 ON CONFLICT (key) DO UPDATE SET value=EXCLUDED.value`,
		abyssNextEventDepthKey(uid),
		strconv.Itoa(route.NextEventDepth),
	); err != nil {
		return fmt.Errorf("save cartographer event anchor: %w", err)
	}
	return nil
}

func clearAbyssCartographerForecastInTx(
	ctx context.Context,
	tx *sql.Tx,
	uid string,
) error {
	if _, err := tx.ExecContext(
		ctx,
		"DELETE FROM app_meta WHERE key IN ($1, $2, $3)",
		abyssCartographerRouteKey(uid),
		abyssNextEventDepthKey(uid),
		abyssEventPreviewKey(uid),
	); err != nil {
		return fmt.Errorf("clear cartographer forecast: %w", err)
	}
	return nil
}

func (b *Bot) loadAbyssCartographerRoute(uid string) (abyssCartographerRoute, bool) {
	var raw string
	err := b.DB.QueryRow(
		"SELECT value FROM app_meta WHERE key=$1",
		abyssCartographerRouteKey(uid),
	).Scan(&raw)
	if err != nil {
		return abyssCartographerRoute{Floors: []abyssCartographerFloor{}}, false
	}
	route, err := decodeAbyssCartographerRoute(raw)
	if err != nil {
		return abyssCartographerRoute{Floors: []abyssCartographerFloor{}}, false
	}
	return route, true
}

func (b *Bot) abyssCartographerRouteView(uid string, currentDepth int) abyssCartographerRouteView {
	route, ok := b.loadAbyssCartographerRoute(uid)
	if !ok {
		return abyssCartographerRouteView{Floors: []abyssCartographerFloorView{}}
	}
	return buildAbyssCartographerRouteView(route, currentDepth)
}

func (b *Bot) abyssCartographerFloor(uid string, depth int) (string, bool) {
	route, ok := b.loadAbyssCartographerRoute(uid)
	if !ok {
		return "", false
	}
	return abyssCartographerFloorAt(route, depth)
}

func (b *Bot) advanceAbyssCartographerRoute(uid string, depth int) abyssCartographerRouteView {
	tx, err := b.DB.BeginTx(context.Background(), nil)
	if err != nil {
		return abyssCartographerRouteView{Floors: []abyssCartographerFloorView{}}
	}
	defer func() { _ = tx.Rollback() }()
	route, ok, err := loadAbyssCartographerRouteInTx(context.Background(), tx, uid)
	if err != nil || !ok {
		return abyssCartographerRouteView{Floors: []abyssCartographerFloorView{}}
	}

	remaining := make([]abyssCartographerFloor, 0, len(route.Floors))
	for _, floor := range route.Floors {
		if floor.Depth > depth {
			remaining = append(remaining, floor)
		}
	}
	route.Floors = remaining
	if len(route.Floors) == 0 {
		if _, err := tx.Exec(
			"DELETE FROM app_meta WHERE key=$1",
			abyssCartographerRouteKey(uid),
		); err != nil {
			return buildAbyssCartographerRouteView(route, depth)
		}
	} else {
		payload, err := json.Marshal(route)
		if err != nil {
			return buildAbyssCartographerRouteView(route, depth)
		}
		if _, err := tx.Exec(
			"UPDATE app_meta SET value=$1 WHERE key=$2",
			string(payload),
			abyssCartographerRouteKey(uid),
		); err != nil {
			return buildAbyssCartographerRouteView(route, depth)
		}
	}
	if _, err := tx.Exec(
		`INSERT INTO app_meta (key, value) VALUES ($1, $2)
		 ON CONFLICT (key) DO UPDATE SET value=EXCLUDED.value`,
		abyssNextEventDepthKey(uid),
		strconv.Itoa(route.NextEventDepth),
	); err != nil {
		return buildAbyssCartographerRouteView(route, depth)
	}
	if err := tx.Commit(); err != nil {
		return buildAbyssCartographerRouteView(route, depth)
	}
	return buildAbyssCartographerRouteView(route, depth)
}

func (s *WebServer) handleAbyssCartographerRoom(
	w http.ResponseWriter,
	ctx context.Context,
	uid string,
	run abyssRun,
	action string,
) bool {
	var state struct {
		Type string `json:"type"`
	}
	if json.Unmarshal([]byte(run.EventState), &state) != nil || state.Type != abyssCartographerEventType {
		return false
	}
	if action != "cartographer_buy" && action != "cartographer_leave" {
		writeJSON(w, map[string]any{"ok": false, "error": "invalid cartographer choice"})
		return true
	}

	pacts := s.bot.abyssRunPacts(uid)
	tx, err := s.bot.DB.BeginTx(ctx, nil)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return true
	}
	defer func() { _ = tx.Rollback() }()

	newGold := int64(0)
	message := "🗺️ You leave the cartographer to his fading trail."
	var route abyssCartographerRoute
	if action == "cartographer_buy" {
		current, exists, err := loadAbyssCartographerRouteInTx(ctx, tx, uid)
		if err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return true
		}
		if exists && buildAbyssCartographerRouteView(current, run.Depth).Active {
			writeJSON(w, map[string]any{"ok": false, "error": "your next floors are already charted"})
			return true
		}
		nextEventDepth, err := loadAbyssNextEventDepthInTx(ctx, tx, uid)
		if err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return true
		}
		route = planAbyssCartographerRoute(
			abyssCartographerPlanInput{
				Depth:          run.Depth,
				LastRestDepth:  run.LastRestDepth,
				NextEventDepth: nextEventDepth,
				Pacts:          pacts,
			},
			func() string { return rollFloorCandidates(1, false)[0].Type },
			func() int {
				// #nosec G404 -- non-cryptographic event-cadence roll
				return abyssEventGapMin + rand.IntN(abyssEventGapMax-abyssEventGapMin+1)
			},
		)
		cost := abyssCartographerMapCost(run.Depth)
		err = tx.QueryRowContext(
			ctx,
			"UPDATE users SET gold=gold-$1 WHERE client_uid=$2 AND gold >= $1 RETURNING gold",
			cost,
			uid,
		).Scan(&newGold)
		if errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, map[string]any{"ok": false, "error": "not enough gold"})
			return true
		}
		if err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return true
		}
		if err := saveAbyssCartographerRouteInTx(ctx, tx, uid, route); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return true
		}
		message = "🗺️ Route purchased: the next five floor types are inked and sealed."
	} else if err := tx.QueryRowContext(
		ctx,
		"SELECT gold FROM users WHERE client_uid=$1",
		uid,
	).Scan(&newGold); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return true
	}

	if err := clearAbyssSpecialRoomInTx(tx, uid); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return true
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return true
	}
	view := buildAbyssCartographerRouteView(route, run.Depth)
	if action == "cartographer_leave" {
		view = s.bot.abyssCartographerRouteView(uid, run.Depth)
	}
	writeJSON(w, map[string]any{
		"ok": true, "resolved": true, "msg": message,
		"gold": newGold, "map_route": view,
	})
	return true
}
