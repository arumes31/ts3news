package bot

import (
	"crypto/rand"
	"database/sql"
	"encoding/base32"
	"time"

	"ts3news/internal/content"
)

const (
	abyssPetExpeditionDuration = 8 * time.Hour
	abyssPetGiftLifetime       = 7 * 24 * time.Hour
	abyssPetCosmeticCost       = 20
)

type abyssPetCosmetic struct {
	Key  string
	Name string
	Icon string
}

var abyssPetCosmetics = []abyssPetCosmetic{
	{Key: "moonlit", Name: "Moonlit Harness", Icon: "🌙"},
	{Key: "ember", Name: "Ember Pennant", Icon: "🔥"},
	{Key: "verdant", Name: "Verdant Wreath", Icon: "🌿"},
}

func abyssPetCosmeticByKey(key string) (abyssPetCosmetic, bool) {
	for _, cosmetic := range abyssPetCosmetics {
		if cosmetic.Key == key {
			return cosmetic, true
		}
	}
	return abyssPetCosmetic{}, false
}

func abyssPetDaycareXP(profile abyssPetProfile, now time.Time) int {
	started, err := time.Parse(time.RFC3339, profile.DaycareSince)
	if err != nil || !now.After(started) {
		return 0
	}
	return min(abyssPetXPThreshold(100), int(now.Sub(started)/time.Hour)*5)
}

func abyssPetExpeditionReady(profile abyssPetProfile, now time.Time) bool {
	until, err := time.Parse(time.RFC3339, profile.ExpeditionUntil)
	return err == nil && !until.After(now)
}

func abyssPetExpeditionReward(kind string, level int) (material string, amount int) {
	amount = max(1, min(10, level/10+1))
	switch kind {
	case "crystal":
		return "crystal", amount
	case "prism":
		return "prism", max(1, amount/3)
	default:
		return "dust", amount * 5
	}
}

func abyssPetFusionGain(value, donor int) int {
	return max(1, max(value, donor)/10)
}

func abyssPetRevivedStats(level int) (hp, strength, defense, speed int) {
	level = max(1, level)
	return 50 + level*10, 10 + level*3, 10 + level*2, 10 + level*2
}

func abyssPetGiftCode() (string, error) {
	random := make([]byte, 9)
	if _, err := rand.Read(random); err != nil {
		return "", err
	}
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(random), nil
}

func abyssPetStableLimitTx(tx *sql.Tx, uid string) (int, error) {
	rows, err := tx.Query("SELECT node_id FROM user_abyss_tree WHERE client_uid=$1", uid)
	if err != nil {
		return 0, err
	}
	defer func() { _ = rows.Close() }()
	allocated := make([]int, 0)
	for rows.Next() {
		var nodeID int
		if err := rows.Scan(&nodeID); err != nil {
			return 0, err
		}
		allocated = append(allocated, nodeID)
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	bonus := int(content.AbyssTree().BonusFor(allocated).Pct["pet_cap"])
	return min(abyssPetMaxCap, abyssPetBaseCap+max(0, bonus)), nil
}
