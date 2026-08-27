package bot

import "time"

func (b *Bot) abyssCompetitionDepthRows(uid, tier, build string, start time.Time) ([]abyssCompetitionRow, int) {
	rows, err := b.DB.Query(`WITH best AS (
		SELECT DISTINCT ON (r.client_uid) r.client_uid,r.depth,r.gold_banked,r.duration_ms,
		 r.build_key,r.pact_multiplier,r.audit_hash,r.created_at
		FROM abyss_runs r WHERE r.tier=$1 AND r.created_at >= $2 AND ($3='' OR r.build_key=$3)
		ORDER BY r.client_uid,r.depth DESC,r.gold_banked DESC,r.created_at ASC
	), ranked AS (
		SELECT b.*,RANK() OVER (ORDER BY depth DESC,gold_banked DESC,created_at ASC,client_uid) AS rank
		FROM best b
	)
	SELECT rank,r.client_uid,COALESCE(NULLIF(u.nickname,''),'Adventurer'),r.build_key,
	 r.depth,r.gold_banked,r.duration_ms,r.pact_multiplier,'Verified descent',r.audit_hash
	FROM ranked r LEFT JOIN users u ON u.client_uid=r.client_uid
	WHERE rank<=$4 OR r.client_uid=$5 ORDER BY rank`, tier, start, build, abyssCompetitionMaxRows, uid)
	if err != nil {
		return nil, 0
	}
	result := scanAbyssCompetitionRows(rows, uid)
	var total int
	_ = b.DB.QueryRow(`SELECT COUNT(DISTINCT client_uid) FROM abyss_runs
		WHERE tier=$1 AND created_at >= $2 AND ($3='' OR build_key=$3)`, tier, start, build).Scan(&total)
	return result, total
}

func (b *Bot) abyssCompetitionSpeedRows(uid, tier, build string, start time.Time) []abyssCompetitionRow {
	rows, err := b.DB.Query(`WITH best AS (
		SELECT DISTINCT ON (r.client_uid) r.client_uid,r.depth,r.gold_banked,r.duration_ms,
		 r.build_key,r.pact_multiplier,r.audit_hash,r.created_at
		FROM abyss_runs r WHERE r.tier=$1 AND r.created_at >= $2 AND r.depth>=20
		 AND r.audit_hash<>'' AND ($3='' OR r.build_key=$3)
		ORDER BY r.client_uid,r.duration_ms ASC,r.depth DESC,r.created_at ASC
	), ranked AS (
		SELECT b.*,RANK() OVER (ORDER BY duration_ms ASC,depth DESC,created_at ASC,client_uid) AS rank FROM best b
	)
	SELECT rank,r.client_uid,COALESCE(NULLIF(u.nickname,''),'Adventurer'),r.build_key,
	 r.depth,r.gold_banked,r.duration_ms,r.pact_multiplier,'F20+ audited speedrun',r.audit_hash
	FROM ranked r LEFT JOIN users u ON u.client_uid=r.client_uid
	WHERE rank<=$4 OR r.client_uid=$5 ORDER BY rank`, tier, start, build, abyssCompetitionMaxRows, uid)
	if err != nil {
		return nil
	}
	return scanAbyssCompetitionRows(rows, uid)
}

func (b *Bot) abyssCompetitionEconomyRows(uid, tier, build string, at time.Time) []abyssCompetitionRow {
	_, start, end := abyssCompetitionWeekAt(at)
	rows, err := b.DB.Query(`WITH totals AS (
		SELECT r.client_uid,MAX(r.depth) AS depth,SUM(r.gold_banked) AS gold,
		 MIN(r.duration_ms) AS duration_ms,MAX(r.build_key) AS build_key,
		 MAX(r.pact_multiplier) AS pact_multiplier,MAX(r.audit_hash) AS audit_hash
		FROM abyss_runs r WHERE r.tier=$1 AND r.created_at >= $2 AND r.created_at < $3
		 AND ($4='' OR r.build_key=$4) GROUP BY r.client_uid
	), ranked AS (
		SELECT t.*,RANK() OVER (ORDER BY gold DESC,depth DESC,client_uid) AS rank FROM totals t
	)
	SELECT rank,r.client_uid,COALESCE(NULLIF(u.nickname,''),'Adventurer'),r.build_key,
	 r.depth,r.gold,r.duration_ms,r.pact_multiplier,'Weekly banked gold',r.audit_hash
	FROM ranked r LEFT JOIN users u ON u.client_uid=r.client_uid
	WHERE rank<=$5 OR r.client_uid=$6 ORDER BY rank`, tier, start, end, build, abyssCompetitionMaxRows, uid)
	if err != nil {
		return nil
	}
	return scanAbyssCompetitionRows(rows, uid)
}

func (b *Bot) abyssCompetitionPactRows(uid, tier, build string, start time.Time) []abyssCompetitionRow {
	rows, err := b.DB.Query(`WITH best AS (
		SELECT DISTINCT ON (r.client_uid) r.client_uid,r.depth,r.gold_banked,r.duration_ms,
		 r.build_key,r.pact_multiplier,r.audit_hash,r.created_at
		FROM abyss_runs r WHERE r.tier=$1 AND r.created_at >= $2 AND r.victory=TRUE
		 AND ($3='' OR r.build_key=$3)
		ORDER BY r.client_uid,r.pact_multiplier DESC,r.depth DESC,r.duration_ms ASC
	), ranked AS (
		SELECT b.*,RANK() OVER (ORDER BY pact_multiplier DESC,depth DESC,duration_ms ASC,client_uid) AS rank FROM best b
	)
	SELECT rank,r.client_uid,COALESCE(NULLIF(u.nickname,''),'Adventurer'),r.build_key,
	 r.depth,r.gold_banked,r.duration_ms,r.pact_multiplier,'Survived pact stack',r.audit_hash
	FROM ranked r LEFT JOIN users u ON u.client_uid=r.client_uid
	WHERE rank<=$4 OR r.client_uid=$5 ORDER BY rank`, tier, start, build, abyssCompetitionMaxRows, uid)
	if err != nil {
		return nil
	}
	return scanAbyssCompetitionRows(rows, uid)
}

func (b *Bot) abyssCompetitionBestiaryRows(uid string) []abyssCompetitionRow {
	rows, err := b.DB.Query(`WITH totals AS (
		SELECT b.client_uid,COALESCE(NULLIF(b.mob_family,''),'Unknown') AS family,SUM(b.kills)::BIGINT AS kills
		FROM abyss_bestiary b GROUP BY b.client_uid,COALESCE(NULLIF(b.mob_family,''),'Unknown')
	), ranked AS (
		SELECT t.*,RANK() OVER (ORDER BY kills DESC,family,client_uid) AS rank FROM totals t
	)
	SELECT rank,r.client_uid,COALESCE(NULLIF(u.nickname,''),'Adventurer'),'bestiary',
	 LEAST(r.kills,2147483647)::INTEGER,0,0,1,r.family||' family kills',''
	FROM ranked r LEFT JOIN users u ON u.client_uid=r.client_uid
	WHERE rank<=$1 OR r.client_uid=$2 ORDER BY rank`, abyssCompetitionMaxRows, uid)
	if err != nil {
		return nil
	}
	return scanAbyssCompetitionRows(rows, uid)
}

func (b *Bot) abyssCompetitionShameRows(uid string) []abyssCompetitionRow {
	rows, err := b.DB.Query(`WITH ranked AS (
		SELECT d.*,ROW_NUMBER() OVER (ORDER BY d.depth DESC,d.died_at DESC,d.id DESC) AS rank
		FROM abyss_deaths d JOIN abyss_social_profiles p ON p.client_uid=d.client_uid AND p.shame_opt_in=TRUE
	)
	SELECT rank,r.client_uid,COALESCE(NULLIF(u.nickname,''),'Adventurer'),'fallen',
	 r.depth,COALESCE(r.lost_cache,0),0,1,'Defeated by '||r.killer_name||' · '||r.killer_family,''
	FROM ranked r LEFT JOIN users u ON u.client_uid=r.client_uid
	WHERE rank<=$1 OR r.client_uid=$2 ORDER BY rank`, abyssCompetitionMaxRows, uid)
	if err != nil {
		return nil
	}
	return scanAbyssCompetitionRows(rows, uid)
}

func (b *Bot) abyssCompetitionStreakRows(uid string) []abyssCompetitionRow {
	rows, err := b.DB.Query(`WITH ranked AS (
		SELECT u.client_uid,u.nickname,u.abyss_bank_streak,u.abyss_lifetime_banked,
		 COALESCE(NULLIF(u.active_specialization,''),'initiate') AS build_key,
		 RANK() OVER (ORDER BY u.abyss_bank_streak DESC,u.abyss_lifetime_banked DESC,u.client_uid) AS rank
		FROM users u WHERE u.abyss_bank_streak>0
	)
	SELECT rank,client_uid,COALESCE(NULLIF(nickname,''),'Adventurer'),build_key,
	 abyss_bank_streak,abyss_lifetime_banked,0,1,'Consecutive banks',''
	FROM ranked WHERE rank<=$1 OR client_uid=$2 ORDER BY rank`, abyssCompetitionMaxRows, uid)
	if err != nil {
		return nil
	}
	return scanAbyssCompetitionRows(rows, uid)
}

func (b *Bot) abyssCompetitionPetRows(uid string) []abyssCompetitionRow {
	rows, err := b.DB.Query(`WITH ranked AS (
		SELECT p.client_uid,p.name,p.level,(p.max_hp+p.str+p.def+p.spd+p.level*10) AS power,
		 RANK() OVER (ORDER BY (p.max_hp+p.str+p.def+p.spd+p.level*10) DESC,p.level DESC,p.pet_id) AS rank
		FROM user_pets p
	)
	SELECT rank,r.client_uid,COALESCE(NULLIF(u.nickname,''),'Adventurer'),'companion',
	 r.power,0,0,1,r.name||' · level '||r.level,''
	FROM ranked r LEFT JOIN users u ON u.client_uid=r.client_uid
	WHERE rank<=$1 OR r.client_uid=$2 ORDER BY rank`, abyssCompetitionMaxRows, uid)
	if err != nil {
		return nil
	}
	return scanAbyssCompetitionRows(rows, uid)
}

func (b *Bot) abyssCompetitionChannelRows(uid, tier string, start time.Time) []abyssCompetitionRow {
	var channel int
	if b.DB.QueryRow(`SELECT channel_id FROM abyss_competition_presence
		WHERE client_uid=$1 AND seen_at>NOW()-INTERVAL '30 minutes'`, uid).Scan(&channel) != nil {
		return nil
	}
	rows, err := b.DB.Query(`WITH members AS (
		SELECT client_uid FROM abyss_competition_presence WHERE channel_id=$1 AND seen_at>NOW()-INTERVAL '30 minutes'
	), best AS (
		SELECT DISTINCT ON (r.client_uid) r.client_uid,r.depth,r.gold_banked,r.duration_ms,
		 r.build_key,r.pact_multiplier,r.audit_hash
		FROM abyss_runs r JOIN members m ON m.client_uid=r.client_uid
		WHERE r.tier=$2 AND r.created_at >= $3 ORDER BY r.client_uid,r.depth DESC,r.gold_banked DESC
	), ranked AS (
		SELECT b.*,RANK() OVER (ORDER BY depth DESC,gold_banked DESC,client_uid) AS rank FROM best b
	)
	SELECT rank,r.client_uid,COALESCE(NULLIF(u.nickname,''),'Adventurer'),r.build_key,
	 r.depth,r.gold_banked,r.duration_ms,r.pact_multiplier,'TS3 channel '||$1,r.audit_hash
	FROM ranked r LEFT JOIN users u ON u.client_uid=r.client_uid ORDER BY rank`, channel, tier, start)
	if err != nil {
		return nil
	}
	return scanAbyssCompetitionRows(rows, uid)
}
