package bot

const abyssDuplicateLoreTokens = 2

// grantAbyssLoreFragment makes every lore source duplicate-aware. A first copy
// unlocks the fragment; a duplicate becomes a small token grant in the same
// database scope instead of disappearing behind ON CONFLICT DO NOTHING.
func grantAbyssLoreFragment(db dbOrTx, uid string, loreID int) (unlocked bool, tokens int, err error) {
	result, err := db.Exec(`INSERT INTO abyss_lore_unlocked (client_uid,lore_id) VALUES ($1,$2)
		ON CONFLICT (client_uid,lore_id) DO NOTHING`, uid, loreID)
	if err != nil {
		return false, 0, err
	}
	inserted, err := result.RowsAffected()
	if err != nil {
		return false, 0, err
	}
	if inserted > 0 {
		if err := grantAbyssLoreCompletion(db, uid); err != nil {
			return false, 0, err
		}
		return true, 0, nil
	}
	if _, err := db.Exec("UPDATE users SET abyss_tokens=abyss_tokens+$1 WHERE client_uid=$2", abyssDuplicateLoreTokens, uid); err != nil {
		return false, 0, err
	}
	return false, abyssDuplicateLoreTokens, nil
}
