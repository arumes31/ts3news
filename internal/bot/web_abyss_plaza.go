package bot

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"time"
)

var (
	errAbyssPlazaOwned = errors.New("monument already owned")
	errAbyssPlazaFunds = errors.New("not enough gold")
)

type abyssPlazaMonument struct {
	Key   string
	Name  string
	Tier  string
	Icon  string
	Desc  string
	Cost  int64
	Order int
}

var abyssPlazaCatalog = []abyssPlazaMonument{
	{Key: "bronze_bust", Name: "Bronze Delver Bust", Tier: "Founder's Walk", Icon: "♟", Desc: "A hand-cast likeness set among the names of first patrons.", Cost: 250_000, Order: 1},
	{Key: "obsidian_runestone", Name: "Obsidian Runestone", Tier: "Echo Court", Icon: "◆", Desc: "Your name cut into black glass that catches the Abyss glow.", Cost: 2_500_000, Order: 2},
	{Key: "gilded_depthspire", Name: "Gilded Depthspire", Tier: "Laureate Rise", Icon: "♜", Desc: "A brass-crowned spire honoring wealth willingly left above ground.", Cost: 25_000_000, Order: 3},
	{Key: "eternal_gate", Name: "Eternal Gate", Tier: "Crown of the Plaza", Icon: "♛", Desc: "A monumental arch bearing one delver's mark in permanent fire.", Cost: 250_000_000, Order: 4},
}

type abyssPlazaCatalogView struct {
	abyssPlazaMonument
	Owned bool
}

type abyssPlazaExhibit struct {
	Key        string
	Name       string
	Tier       string
	Icon       string
	Nickname   string
	GoldSpent  int64
	AcquiredAt string
	Order      int
}

type abyssPlazaView struct {
	Catalog     []abyssPlazaCatalogView
	Exhibits    []abyssPlazaExhibit
	Patrons     int
	Monuments   int
	GoldRetired int64
}

func abyssPlazaMonumentByKey(key string) (abyssPlazaMonument, bool) {
	for _, monument := range abyssPlazaCatalog {
		if monument.Key == key {
			return monument, true
		}
	}
	return abyssPlazaMonument{}, false
}

func (b *Bot) abyssPlazaPage(ctx context.Context, uid string) (abyssPlazaView, error) {
	view := abyssPlazaView{
		Catalog:  make([]abyssPlazaCatalogView, 0, len(abyssPlazaCatalog)),
		Exhibits: []abyssPlazaExhibit{},
	}
	owned := map[string]bool{}
	rows, err := b.DB.QueryContext(ctx, "SELECT monument_key FROM abyss_plaza_monuments WHERE client_uid=$1", uid)
	if err != nil {
		return view, err
	}
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			_ = rows.Close()
			return view, err
		}
		owned[key] = true
	}
	if err := rows.Close(); err != nil {
		return view, err
	}
	if err := rows.Err(); err != nil {
		return view, err
	}
	for _, monument := range abyssPlazaCatalog {
		view.Catalog = append(view.Catalog, abyssPlazaCatalogView{abyssPlazaMonument: monument, Owned: owned[monument.Key]})
	}

	rows, err = b.DB.QueryContext(ctx, `SELECT m.monument_key, COALESCE(NULLIF(u.nickname,''),'Adventurer'),
		m.gold_spent, m.acquired_at
		FROM abyss_plaza_monuments m
		JOIN users u ON u.client_uid=m.client_uid
		ORDER BY m.acquired_at DESC
		LIMIT 96`)
	if err != nil {
		return view, err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var key, nickname string
		var goldSpent int64
		var acquiredAt time.Time
		if err := rows.Scan(&key, &nickname, &goldSpent, &acquiredAt); err != nil {
			return view, err
		}
		monument, ok := abyssPlazaMonumentByKey(key)
		if !ok {
			continue
		}
		view.Exhibits = append(view.Exhibits, abyssPlazaExhibit{
			Key: key, Name: monument.Name, Tier: monument.Tier, Icon: monument.Icon,
			Nickname: nickname, GoldSpent: goldSpent, Order: monument.Order,
			AcquiredAt: acquiredAt.UTC().Format("02 Jan 2006"),
		})
	}
	if err := rows.Err(); err != nil {
		return view, err
	}
	if err := b.DB.QueryRowContext(ctx, `SELECT COUNT(DISTINCT client_uid), COUNT(*), COALESCE(SUM(gold_spent),0)
		FROM abyss_plaza_monuments`).Scan(&view.Patrons, &view.Monuments, &view.GoldRetired); err != nil {
		return view, err
	}
	return view, nil
}

func buyAbyssPlazaMonument(ctx context.Context, tx *sql.Tx, uid string, monument abyssPlazaMonument) (int64, error) {
	var gold int64
	if err := tx.QueryRowContext(ctx, "SELECT gold FROM users WHERE client_uid=$1 FOR UPDATE", uid).Scan(&gold); err != nil {
		return 0, err
	}
	var owned bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM abyss_plaza_monuments
		WHERE client_uid=$1 AND monument_key=$2)`, uid, monument.Key).Scan(&owned); err != nil {
		return 0, err
	}
	if owned {
		return 0, errAbyssPlazaOwned
	}
	if gold < monument.Cost {
		return 0, errAbyssPlazaFunds
	}
	if _, err := tx.ExecContext(ctx, "UPDATE users SET gold=gold-$1 WHERE client_uid=$2", monument.Cost, uid); err != nil {
		return 0, err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO abyss_plaza_monuments (client_uid,monument_key,gold_spent)
		VALUES ($1,$2,$3)`, uid, monument.Key, monument.Cost); err != nil {
		return 0, err
	}
	return gold - monument.Cost, nil
}

func (s *WebServer) handleAbyssPlazaPage(w http.ResponseWriter, r *http.Request, uid string) {
	if r.URL.Path != "/abyss/plaza" {
		http.NotFound(w, r)
		return
	}
	u, err := s.loadWebUser(uid)
	if err != nil {
		http.Redirect(w, r, "/denied", http.StatusSeeOther)
		return
	}
	plaza, err := s.bot.abyssPlazaPage(r.Context(), uid)
	if err != nil {
		http.Error(w, "plaza unavailable", http.StatusInternalServerError)
		return
	}
	s.render(w, "abyss-plaza", map[string]any{
		"Title": "Hall of Delvers", "Nav": "plaza", "U": u, "Plaza": plaza,
	})
}

func (s *WebServer) handleAbyssPlazaBuy(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Key string `json:"key"`
	}
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}
	monument, ok := abyssPlazaMonumentByKey(req.Key)
	if !ok {
		writeJSON(w, map[string]any{"ok": false, "error": "unknown monument"})
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()
	tx, err := s.bot.DB.BeginTx(r.Context(), nil)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()
	gold, err := buyAbyssPlazaMonument(r.Context(), tx, uid, monument)
	if errors.Is(err, errAbyssPlazaOwned) || errors.Is(err, errAbyssPlazaFunds) {
		writeJSON(w, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	writeJSON(w, map[string]any{
		"ok": true, "key": monument.Key, "gold": gold,
		"msg": monument.Name + " now stands in the Hall of Delvers.",
	})
}
