package main

import (
	"log"
	"os"

	"ts3news/internal/bot"
	"ts3news/internal/config"
	"ts3news/internal/icon"
)

func main() {
	cfg := config.LoadConfig()

	// Generate TS3 Leveling Icons if they don't exist yet
	if _, err := os.Stat("data/icons/tier_25.png"); os.IsNotExist(err) {
		log.Println("Generating TS3 rank tier icons in data/icons/...")
		if err := icon.GenerateTierIcons("data/icons", 25); err != nil {
			log.Printf("Warning: Failed to generate icons: %v", err)
		}
	}

	b := bot.NewBot(cfg)
	defer b.Close()

	log.Println("Starting TS3 free-games bot supervisor...")
	sup := bot.NewSupervisor(b)
	if err := sup.Run(); err != nil {
		log.Fatalf("Supervisor error: %v", err)
	}
	log.Println("Bot stopped cleanly.")
}
