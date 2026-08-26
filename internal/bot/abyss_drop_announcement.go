package bot

import (
	"fmt"
	"log"
	"strings"
	"time"

	"ts3news/internal/clientquery"
	"ts3news/internal/content"
)

const abyssDropAnnouncementConcurrency = 4

var abyssDropAnnouncementSlots = make(chan struct{}, abyssDropAnnouncementConcurrency)

func abyssHighRarityEscrowDrop(grant abyssLootGrant) (string, content.Rarity, bool) {
	if grant.Type != "gear" || grant.Gear == nil || grant.Gear.Rarity < content.RarityMythic {
		return "", 0, false
	}
	return grant.Gear.Name, grant.Gear.Rarity, true
}

func abyssHighRarityDropFanfare(nickname, itemName string, rarity content.Rarity) (string, string, bool) {
	if rarity < content.RarityMythic {
		return "", "", false
	}
	nickname = sanitizeBBCode(nickname)
	itemName = sanitizeBBCode(itemName)
	rank := strings.ToUpper(rarity.String())
	return rarity.String() + " Drop!",
		fmt.Sprintf("🌟 %s! %s has obtained %s — a Mythic+ treasure of the Abyss!", rank, nickname, itemName),
		true
}

// broadcastAbyssHighRarityDrop sends presentation-only fanfare after a
// Mythic+ gear award has persisted. Failures never roll back or delay loot.
func (b *Bot) broadcastAbyssHighRarityDrop(uid, itemName string, rarity content.Rarity) {
	if rarity < content.RarityMythic {
		return
	}
	var nickname string
	if err := b.DB.QueryRow("SELECT nickname FROM users WHERE client_uid=$1", uid).Scan(&nickname); err != nil || nickname == "" {
		return
	}
	announcementNick, message, ok := abyssHighRarityDropFanfare(nickname, itemName, rarity)
	if !ok {
		return
	}

	addr := b.Cfg.ClientQueryAddr
	if addr == "" {
		addr = "127.0.0.1:25639"
	}
	client, err := clientquery.Dial(addr, 2*time.Second)
	if err != nil {
		return
	}
	defer func() { _ = client.Close() }()
	if apiKey := b.getAPIKey(); apiKey != "" {
		_ = client.Auth(apiKey)
	}
	_ = client.Use(1)

	oldNickname := b.Cfg.TS3Nickname
	_ = client.SetNickname(announcementNick)
	clients, err := client.ClientList()
	if err == nil {
		for _, listed := range clients {
			if listed.Type != 0 {
				continue
			}
			_ = client.Poke(listed.CLID, message)
			time.Sleep(time.Duration(b.Cfg.PokeDelayMS) * time.Millisecond)
		}
	}
	time.Sleep(3 * time.Second)
	_ = client.SetNickname(oldNickname)
}

func (b *Bot) queueAbyssHighRarityDrop(uid, itemName string, rarity content.Rarity) {
	if rarity < content.RarityMythic {
		return
	}
	select {
	case abyssDropAnnouncementSlots <- struct{}{}:
		go func() {
			defer func() { <-abyssDropAnnouncementSlots }()
			b.broadcastAbyssHighRarityDrop(uid, itemName, rarity)
		}()
	default:
		log.Printf("abyss Mythic+ announcement capacity reached for %s", uid)
	}
}
