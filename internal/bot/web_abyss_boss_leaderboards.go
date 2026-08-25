package bot

import "time"

type abyssBossSpeedBoard struct {
	Name string
	Rows []bossKillRow
}

func (b *Bot) topBossSpeedBoards(tier string, limit int) []abyssBossSpeedBoard {
	rows, err := b.DB.Query(
		`WITH personal_best AS (
		   SELECT DISTINCT ON (k.boss_name,k.client_uid)
		          k.client_uid,COALESCE(NULLIF(u.nickname,''),'Adventurer') AS nick,
		          k.boss_name,k.depth,k.kill_time_ms,k.killed_at
		     FROM abyss_boss_kills k
		     LEFT JOIN users u ON u.client_uid=k.client_uid
		    WHERE k.tier=$1
		    ORDER BY k.boss_name,k.client_uid,k.kill_time_ms,k.depth DESC,k.killed_at
		 ), ranked AS (
		   SELECT *,ROW_NUMBER() OVER (
		          PARTITION BY boss_name
		          ORDER BY kill_time_ms,depth DESC,killed_at,client_uid
		        ) AS boss_rank
		     FROM personal_best
		 )
		 SELECT boss_rank,client_uid,nick,boss_name,depth,kill_time_ms,killed_at
		   FROM ranked WHERE boss_rank<=$2 ORDER BY boss_name,boss_rank`, tier, limit)
	if err != nil {
		return nil
	}
	defer func() { _ = rows.Close() }()

	boards := make([]abyssBossSpeedBoard, 0)
	boardIndex := make(map[string]int)
	for rows.Next() {
		var row bossKillRow
		var killedAt time.Time
		if err := rows.Scan(&row.Rank, &row.UID, &row.Nickname, &row.BossName,
			&row.Depth, &row.KillTimeMs, &killedAt); err != nil {
			continue
		}
		row.KilledAt = killedAt.Format("2006-01-02 15:04")
		index, ok := boardIndex[row.BossName]
		if !ok {
			index = len(boards)
			boardIndex[row.BossName] = index
			boards = append(boards, abyssBossSpeedBoard{Name: row.BossName})
		}
		boards[index].Rows = append(boards[index].Rows, row)
	}
	return boards
}
