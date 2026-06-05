package main

import (
	"log"
	"ts3news/internal/bot"
	"ts3news/internal/config"
)

func main() {
	cfg := config.LoadConfig()
	b := bot.NewBot(cfg)

	log.Println("Starting notification cycle...")
	if err := b.Run(); err != nil {
		log.Fatalf("Error during run: %v", err)
	}
	
	log.Println("Notification cycle finished successfully.")
}
