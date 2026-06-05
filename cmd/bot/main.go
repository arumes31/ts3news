package main

import (
	"log"
	"ts3news/internal/bot"
	"ts3news/internal/config"
)

// The bot runs a single poke cycle and exits; the container entrypoint connects
// the TeamSpeak client, invokes this once, disconnects, and sleeps until the next
// cycle.
func main() {
	log.Println("Starting poke cycle...")

	cfg := config.LoadConfig()
	if cfg.TS3Host == "" {
		log.Fatal("TS3_HOST is not set; check config.env")
	}

	b := bot.NewBot(cfg)
	if err := b.Run(); err != nil {
		log.Fatalf("Poke cycle failed: %v", err)
	}
	log.Println("Poke cycle complete.")
}
