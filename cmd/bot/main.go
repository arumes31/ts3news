package main

import (
	"log"

	"ts3news/internal/bot"
	"ts3news/internal/config"
)

func main() {
	cfg := config.LoadConfig()
	b := bot.NewBot(cfg)
	defer b.Close()

	log.Println("Starting TS3 free-games bot supervisor...")
	sup := bot.NewSupervisor(b)
	if err := sup.Run(); err != nil {
		log.Fatalf("Supervisor error: %v", err)
	}
	log.Println("Bot stopped cleanly.")
}
