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

// Run performs a single notification cycle: it waits for the TeamSpeak client to
// be connected (via ClientQuery), then pokes the recipients about each new
// limited-time paid Steam/Epic giveaway, and returns. The entrypoint invokes the
// bot once per cycle and disconnects the client afterwards.
func (b *Bot) Run() error {
	apiKey := b.apiKey()
	if apiKey == "" {
		return fmt.Errorf("no ClientQuery API key (set TS3_APIKEY or ensure %s exists)", b.Cfg.ClientQueryINI)
	}

	if b.Cfg.TargetNick != "" {
		log.Printf("Poke target restricted to nickname %q (testing mode)", b.Cfg.TargetNick)
	}

	cq, err := b.connectAndWait(apiKey, 90*time.Second)
	if err != nil {
		return err
	}
	defer cq.Close()

	return b.notify(cq)
}

// connectAndWait dials ClientQuery (retrying until the plugin's port is up) and
// waits until the client reports it is connected to the server.
func (b *Bot) connectAndWait(apiKey string, timeout time.Duration) (*clientquery.Client, error) {
	deadline := time.Now().Add(timeout)
	for {
		cq, err := clientquery.Dial(b.Cfg.ClientQueryAddr, 5*time.Second)
		if err == nil {
			if err = cq.Auth(apiKey); err == nil {
				_ = cq.Use(1)
				if cq.IsConnected() {
					return cq, nil
				}
			}
			cq.Close()
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("timed out waiting for the TeamSpeak client to connect: %v", err)
		}
		log.Printf("Waiting for the TeamSpeak client to connect...")
		time.Sleep(3 * time.Second)
	}
}

// notify finds the recipients and pokes them about each new qualifying giveaway.
func (b *Bot) notify(cq *clientquery.Client) error {
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

	pokeDelay := time.Duration(b.Cfg.PokeDelayMS) * time.Millisecond
	first := true
	pause := func() {
		if !first {
			time.Sleep(pokeDelay) // space out commands to avoid anti-flood
		}
		first = false
	}

	for _, g := range fresh {
		poke := formatPoke(g)    // short popup (pokes are limited to ~100 chars)
		msg := formatMessage(g)  // full details incl. link, via private message
		for _, t := range targets {
			pause()
			if err := cq.Poke(t.CLID, poke); err != nil {
				log.Printf("Failed to poke %s (clid %d): %v", t.Nickname, t.CLID, err)
			}
			pause()
			if err := cq.SendPrivateMessage(t.CLID, msg); err != nil {
				log.Printf("Failed to message %s (clid %d): %v", t.Nickname, t.CLID, err)
				continue
			}
			log.Printf("Notified %s (clid %d): %s", t.Nickname, t.CLID, msg)
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

// formatPoke builds a short poke text, limited to 100 characters.
func formatPoke(g games.Game) string {
	text := fmt.Sprintf("Free on %s: %s (was %s)", g.Store(), cleanTitle(g.Title), g.Worth)
	if len(text) > 100 {
		return text[:97] + "..."
	}
	return text
}

// formatMessage builds the private message text including link and full details.
func formatMessage(g games.Game) string {
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
