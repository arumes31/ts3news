package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"ts3news/internal/bot"
	"ts3news/internal/config"
)

func main() {
	log.Println("Starting headless TS3 news client...")

	cfg := config.LoadConfig()
	if cfg.TS3Host == "" {
		log.Fatal("TS3_HOST is not set; check config.env")
	}
	if cfg.TS3Port == 0 {
		log.Fatal("TS3_PORT is not set or invalid; check config.env")
	}

	b := bot.NewBot(cfg)
	if err := b.Run(); err != nil {
		log.Fatalf("Bot failed to start: %v", err)
	}

	// Run starts the periodic notification loop in the background, so block here
	// until the process is asked to terminate.
	log.Println("Bot running. Waiting for the next notification cycle (Ctrl+C to exit)...")
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	log.Println("Shutting down.")
}
