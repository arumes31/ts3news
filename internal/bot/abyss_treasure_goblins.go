package bot

import (
	"strings"
)

type abyssTreasureGoblinReward struct {
	Label  string
	Grant  abyssLootGrant
	RunKey bool
}

func abyssTreasureGoblinSignature(name string) (abyssTreasureGoblinReward, bool) {
	switch {
	case strings.Contains(name, "Gem Goblin"):
		return abyssTreasureGoblinReward{
			Label: "💎 Gem Goblin cache: Eldritch Prism ×1",
			Grant: abyssLootGrant{Type: "mat", MatID: "prism", MatN: 1},
		}, true
	case strings.Contains(name, "Token Goblin"):
		return abyssTreasureGoblinReward{
			Label: "🜲 Token Goblin purse: 5 Abyss Tokens",
			Grant: abyssLootGrant{Type: "tokens", Tokens: 5},
		}, true
	case strings.Contains(name, "Key Goblin"):
		return abyssTreasureGoblinReward{
			Label:  "🔑 Key Goblin ring: +1 run vault key",
			RunKey: true,
		}, true
	default:
		return abyssTreasureGoblinReward{}, false
	}
}

func (b *Bot) grantAbyssRunVaultKey(uid string) error {
	tx, err := b.DB.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	flags, err := loadAbyssRunFlagsInTx(tx, uid)
	if err != nil {
		return err
	}
	flags[abyssRunFlagVaultKeys]++
	if err := saveAbyssRunFlagsInTx(tx, uid, flags); err != nil {
		return err
	}
	return tx.Commit()
}
