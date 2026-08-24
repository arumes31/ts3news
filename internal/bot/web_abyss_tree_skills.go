package bot

import "ts3news/internal/content"

func (b *Bot) abyssTreeSkillDetails(uid string) []content.SkillDetail {
	rows, err := b.DB.Query("SELECT skill_id FROM user_skills WHERE client_uid=$1 ORDER BY slot LIMIT 5", uid)
	if err != nil {
		return nil
	}
	defer func() { _ = rows.Close() }()
	ids := make([]string, 0, 5)
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil {
			ids = append(ids, id)
		}
	}
	return content.SkillDetailsByID(ids)
}
