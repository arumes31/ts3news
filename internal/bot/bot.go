package bot

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"
	"ts3news/internal/clientquery"
	"ts3news/internal/config"
	"ts3news/internal/games"
)

// Where the IDs of already-announced giveaways are persisted, so the same deal
// is not poked on every cycle. Lives in the client profile dir (survives restarts).
const sentGamesPath = "/root/.ts3client/sent_games.json"

var (
	botInstance *Bot
	once        sync.Once
)

type Bot struct {
	Cfg  *config.Config
	mu   sync.Mutex
	sent map[int]bool
}

func NewBot(cfg *config.Config) *Bot {
	once.Do(func() {
		botInstance = &Bot{Cfg: cfg, sent: loadSent()}
	})
	return botInstance
}

// Run connects to the local TeamSpeak client via ClientQuery, sends an initial
// notification, and then keeps notifying on the configured interval. It returns
// once the initial connection and notification have been attempted; the periodic
// loop runs in the background.
func (b *Bot) Run() error {
	apiKey := b.apiKey()
	if apiKey == "" {
		return fmt.Errorf("no ClientQuery API key (set TS3_APIKEY or ensure %s exists)", b.Cfg.ClientQueryINI)
	}

	if b.Cfg.TargetNick != "" {
		log.Printf("Poke target restricted to nickname %q (testing mode)", b.Cfg.TargetNick)
	}

	if err := b.waitAndNotify(apiKey); err != nil {
		log.Printf("Initial notification failed: %v", err)
	}

	interval := time.Duration(b.Cfg.CheckIntervalHours) * time.Hour
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			if err := b.notifyOnce(apiKey); err != nil {
				log.Printf("Notification cycle failed: %v", err)
			}
		}
	}()
	return nil
}

// waitAndNotify retries until the client's ClientQuery interface is reachable and
// the client is connected to the server, then performs one notification.
func (b *Bot) waitAndNotify(apiKey string) error {
	deadline := time.Now().Add(90 * time.Second)
	for {
		err := b.notifyOnce(apiKey)
		if err == nil {
			return nil
		}
		if time.Now().After(deadline) {
			return err
		}
		log.Printf("Waiting for TeamSpeak client to be ready: %v", err)
		time.Sleep(3 * time.Second)
	}
}

// notifyOnce opens a fresh ClientQuery session, finds the recipients, and pokes
// them about each new limited-time paid Steam/Epic giveaway.
func (b *Bot) notifyOnce(apiKey string) error {
	cq, err := clientquery.Dial(b.Cfg.ClientQueryAddr, 5*time.Second)
	if err != nil {
		return fmt.Errorf("connect ClientQuery: %w", err)
	}
	defer cq.Close()

	if err := cq.Auth(apiKey); err != nil {
		return fmt.Errorf("auth: %w", err)
	}
	if err := cq.Use(1); err != nil {
		return fmt.Errorf("use 1: %w", err)
	}

	clients, err := cq.ClientList()
	if err != nil {
		return fmt.Errorf("clientlist: %w", err)
	}
	targets := b.selectTargets(clients)
	if len(targets) == 0 {
		if b.Cfg.TargetNick != "" {
			log.Printf("Target %q is not online; nothing to poke.", b.Cfg.TargetNick)
		} else {
			log.Println("No clients online to poke.")
		}
		return nil
	}

	all, err := games.FetchGiveaways()
	if err != nil {
		return fmt.Errorf("fetching giveaways: %w", err)
	}

	// Keep only limited-time, normally-paid Steam/Epic giveaways we haven't sent yet.
	var fresh []games.Game
	for _, g := range all {
		if g.IsLimitedTimePaidGiveaway() && !b.alreadySent(g.ID) {
			fresh = append(fresh, g)
		}
	}
	if len(fresh) == 0 {
		log.Println("No new limited-time paid Steam/Epic giveaways to announce.")
		return nil
	}

	for _, g := range fresh {
		msg := formatGame(g)
		for _, t := range targets {
			if err := cq.Poke(t.CLID, msg); err != nil {
				log.Printf("Failed to poke %s (clid %d): %v", t.Nickname, t.CLID, err)
				continue
			}
			log.Printf("Poked %s (clid %d): %s", t.Nickname, t.CLID, msg)
		}
		b.markSent(g.ID)
	}
	return nil
}

// selectTargets returns the clients to poke: only real voice clients (type 0),
// excluding the bot itself, and — when TargetNick is set — only that nickname.
func (b *Bot) selectTargets(clients []clientquery.ClientInfo) []clientquery.ClientInfo {
	var out []clientquery.ClientInfo
	for _, c := range clients {
		if c.Type != 0 {
			continue
		}
		if strings.EqualFold(c.Nickname, b.Cfg.TS3Nickname) {
			continue
		}
		if b.Cfg.TargetNick != "" && !strings.EqualFold(c.Nickname, b.Cfg.TargetNick) {
			continue
		}
		out = append(out, c)
	}
	return out
}

// formatGame builds the poke text for a giveaway, e.g.
// "Free on Steam (was $29.99): Tell Me Why - https://...".
func formatGame(g games.Game) string {
	return fmt.Sprintf("Free on %s (was %s): %s - %s", g.Store(), g.Worth, cleanTitle(g.Title), g.URL)
}

// cleanTitle strips GamerPower's "(Steam) Giveaway" / "(Epic Games) Giveaway" suffixes.
func cleanTitle(title string) string {
	if i := strings.Index(title, " ("); i > 0 {
		return strings.TrimSpace(title[:i])
	}
	return strings.TrimSpace(strings.TrimSuffix(title, " Giveaway"))
}

func (b *Bot) alreadySent(id int) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.sent[id]
}

func (b *Bot) markSent(id int) {
	b.mu.Lock()
	b.sent[id] = true
	snapshot := make([]int, 0, len(b.sent))
	for k := range b.sent {
		snapshot = append(snapshot, k)
	}
	b.mu.Unlock()
	saveSent(snapshot)
}

func loadSent() map[int]bool {
	m := make(map[int]bool)
	data, err := os.ReadFile(sentGamesPath)
	if err != nil {
		return m
	}
	var ids []int
	if json.Unmarshal(data, &ids) == nil {
		for _, id := range ids {
			m[id] = true
		}
	}
	return m
}

func saveSent(ids []int) {
	if data, err := json.Marshal(ids); err == nil {
		_ = os.WriteFile(sentGamesPath, data, 0644)
	}
}

// apiKey returns the ClientQuery API key, preferring the explicit config value
// and otherwise reading it from clientquery.ini.
func (b *Bot) apiKey() string {
	if b.Cfg.APIKey != "" {
		return b.Cfg.APIKey
	}
	f, err := os.Open(b.Cfg.ClientQueryINI)
	if err != nil {
		return ""
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if k, v, ok := strings.Cut(line, "="); ok && strings.TrimSpace(k) == "api_key" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
