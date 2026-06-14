package main

import (
	"context"
	"log"
	"os"
	"sync"

	"ts3news/internal/bot"
	"ts3news/internal/config"
	"ts3news/internal/content"
	"ts3news/internal/i18n"
	"ts3news/internal/icon"
)

func main() {
	cfg := config.LoadConfig()

	// Initialize i18n before any content/bot code runs
	localeID, err := i18n.ParseLocaleID(cfg.Lang)
	if err != nil {
		log.Printf("Warning: invalid LANG %q, falling back to en_US: %v", cfg.Lang, err)
		localeID = i18n.LocaleEnUS
	}
	if err := i18n.InitWithLocale(localeID); err != nil {
		log.Fatalf("i18n initialization failed: %v", err)
	}
	log.Printf("i18n: active locale %s", i18n.CurrentLocale())

	// Rebuild content names now that i18n is loaded; they were baked at package
	// init time when i18n was not yet available and would otherwise show raw
	// translation keys (e.g. "content.gear.novice").
	content.InitLocalized()

	// Generate TS3 Leveling Icons if they don't exist yet
	if _, err := os.Stat("data/icons/tier_25.png"); os.IsNotExist(err) {
		log.Println("Generating TS3 rank tier icons in data/icons/...")
		if err := icon.GenerateTierIcons("data/icons", 25); err != nil {
			log.Printf("Warning: Failed to generate icons: %v", err)
		}
	}

	b := bot.NewBot(cfg)
	defer b.Close()

	// Player web portal (armoury, inventory, auto-battler, arcade, shop, auction).
	if cfg.WebEnable {
		ws, err := bot.NewWebServer(b)
		if err != nil {
			log.Printf("Warning: web portal disabled (init failed): %v", err)
		} else {
			ctx, cancel := context.WithCancel(context.Background())
			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				defer wg.Done()
				if err := ws.Start(ctx, cfg.WebListenAddr); err != nil {
					log.Printf("Web portal stopped: %v", err)
				}
			}()
			// Stop the web server before b.Close() runs (defers are LIFO, so this
			// registers after the b.Close() defer and therefore runs first).
			defer func() {
				cancel()
				if err := ws.Shutdown(context.Background()); err != nil {
					log.Printf("Web portal shutdown error: %v", err)
				}
				wg.Wait()
			}()
		}
	}

	log.Println("Starting TS3 free-games bot supervisor...")
	sup := bot.NewSupervisor(b)
	if err := sup.Run(); err != nil {
		log.Fatalf("Supervisor error: %v", err)
	}
	log.Println("Bot stopped cleanly.")
}
