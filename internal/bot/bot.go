package bot

import (
	"bufio"
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"

	_ "github.com/lib/pq"
	"ts3news/internal/clientquery"
	"ts3news/internal/config"
	"ts3news/internal/games"
)

type Bot struct {
	Cfg *config.Config
	DB  *sql.DB
}

func NewBot(cfg *config.Config) *Bot {
	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Simple health check and table initialization
	if err := db.Ping(); err != nil {
		log.Printf("Warning: Database ping failed: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS sent_notifications (
		client_nickname TEXT,
		game_id INTEGER,
		PRIMARY KEY (client_nickname, game_id)
	)`)
	if err != nil {
		log.Fatalf("Failed to initialize database table: %v", err)
	}

	return &Bot{
		Cfg: cfg,
		DB:  db,
	}
}

func (b *Bot) Run() error {
	addr := b.Cfg.ClientQueryAddr
	if addr == "" {
		addr = "127.0.0.1:25639"
	}

	var c *clientquery.Client
	var err error

	log.Printf("Connecting to ClientQuery at %s...", addr)
	for i := 0; i < 60; i++ {
		c, err = clientquery.Dial(addr, 2*time.Second)
		if err == nil {
			break
		}
		time.Sleep(1 * time.Second)
	}

	if err != nil {
		return fmt.Errorf("failed to dial ClientQuery after retries: %w", err)
	}
	defer c.Close()

	// Authenticate with ClientQuery API Key
	apiKey := b.getAPIKey()
	if apiKey != "" {
		log.Println("Authenticating with ClientQuery...")
		if _, err := c.Command("auth apikey=" + apiKey); err != nil {
			log.Printf("Warning: ClientQuery authentication failed: %v", err)
		}
	}

	// Wait for connection to TS3 server
	log.Println("Waiting for TS3 client to be connected to server (max 120s)...")
	connected := false
	for i := 0; i < 120; i++ {
		if c.IsConnected() {
			connected = true
			break
		}
		time.Sleep(1 * time.Second)
	}

	if !connected {
		return fmt.Errorf("TS3 client did not connect to server in time")
	}

	log.Println("TS3 client connected. Running notification cycle...")

	if err := c.Use(1); err != nil {
		log.Printf("Warning: 'use 1' failed: %v", err)
	}

	return b.RunCycle(c)
}

func (b *Bot) RunCycle(c *clientquery.Client) error {
	freeGames, err := games.FetchFreeGames()
	if err != nil {
		return fmt.Errorf("failed to fetch games: %w", err)
	}

	clients, err := c.ClientList()
	if err != nil {
		return fmt.Errorf("failed to list clients: %w", err)
	}

	targetNick := b.Cfg.TargetNick
	log.Printf("Online clients: %d. Target filter: '%s'. Found %d free games.", len(clients), targetNick, len(freeGames))

	pokedCount := 0
	for _, client := range clients {
		if client.Type != 0 {
			continue
		}

		if targetNick != "" && client.Nickname != targetNick {
			continue
		}

		alreadySent, err := b.getSentGames(client.Nickname)
		if err != nil {
			log.Printf("Failed to get sent games for %s: %v", client.Nickname, err)
			continue
		}

		var candidates []games.Game
		for _, g := range freeGames {
			sent := false
			for _, id := range alreadySent {
				if id == g.ID {
					sent = true
					break
				}
			}
			if !sent {
				candidates = append(candidates, g)
			}
		}

		if len(candidates) > 0 {
			game := candidates[rand.Intn(len(candidates))]

			shortURL, err := games.ShortenURL(game.URL)
			if err != nil {
				log.Printf("Shortening failed: %v", err)
				shortURL = game.URL
			}

			// POKE: max 100 chars, MUST include link.
			prefix := "Free: "
			availableLen := 100 - len(prefix) - 1 - len(shortURL)
			title := game.Title
			if len(title) > availableLen {
				title = title[:availableLen-3] + "..."
			}
			pokeMsg := fmt.Sprintf("%s%s %s", prefix, title, shortURL)
			pmMsg := fmt.Sprintf("Daily Free Game! Title: %s. Link: %s", game.Title, shortURL)

			log.Printf("Notifying %s about %s (%s)", client.Nickname, game.Title, shortURL)

			if err := c.Poke(client.CLID, pokeMsg); err != nil {
				log.Printf("Poke failed for %s: %v", client.Nickname, err)
			}
			if err := c.SendPrivateMessage(client.CLID, pmMsg); err != nil {
				log.Printf("PM failed for %s: %v", client.Nickname, err)
			}

			if err := b.markAsSent(client.Nickname, game.ID); err != nil {
				log.Printf("Failed to mark game as sent for %s: %v", client.Nickname, err)
			}
			pokedCount++

			time.Sleep(time.Duration(b.Cfg.PokeDelayMS) * time.Millisecond)
		}
	}

	log.Printf("Cycle finished. Poked %d users.", pokedCount)
	return nil
}

func (b *Bot) getSentGames(nickname string) ([]int, error) {
	rows, err := b.DB.Query("SELECT game_id FROM sent_notifications WHERE client_nickname = $1", nickname)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func (b *Bot) markAsSent(nickname string, gameID int) error {
	_, err := b.DB.Exec("INSERT INTO sent_notifications (client_nickname, game_id) VALUES ($1, $2) ON CONFLICT DO NOTHING", nickname, gameID)
	return err
}

func (b *Bot) getAPIKey() string {
	if b.Cfg.APIKey != "" {
		return b.Cfg.APIKey
	}
	path := b.Cfg.ClientQueryINI
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "api_key=") {
			return strings.TrimPrefix(line, "api_key=")
		}
	}
	return ""
}
