package bot

func (b *Bot) addAbyssLegendaryPity(out map[string]any, uid string) {
	var pity int
	if err := b.DB.QueryRow("SELECT legendary_pity FROM users WHERE client_uid=$1", uid).Scan(&pity); err != nil {
		return
	}
	if pity < 0 {
		pity = 0
	}
	if pity > abyssLegendaryPityCap {
		pity = abyssLegendaryPityCap
	}
	out["legendary_pity"] = pity
}
