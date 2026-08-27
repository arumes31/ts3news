package bot

import (
	"database/sql"
	"fmt"
	"time"
)

type abyssProgramFriendView struct {
	UID             string
	Nick            string
	State           string
	Online          bool
	InRun           bool
	IncomingRequest bool
	Accepted        bool
}

type abyssProgramMessageView struct {
	Nick    string
	Message string
	When    string
}

type abyssProgramTradeView struct {
	ID        int64
	OtherUID  string
	OtherNick string
	Offer     string
	Request   string
	Incoming  bool
}

type abyssProgramDuelView struct {
	ID        int64
	OtherUID  string
	OtherNick string
	Wager     int
	Incoming  bool
}

type abyssProgramRescueView struct {
	DeathID   int64
	Nick      string
	Depth     int
	LostCache int64
}

type abyssProgramGuildView struct {
	Exists     bool
	Name       string
	Tag        string
	Banner     string
	InviteCode string
	Role       string
	Members    int
	Floors     int
	Goal       int
}

type abyssProgramTeamView struct {
	Exists  bool
	Name    string
	Code    string
	Members int
}

type abyssProgramRaidView struct {
	Exists  bool
	Code    string
	Owner   bool
	Members int
}

type abyssProgramRankView struct {
	Name  string
	Meta  string
	Value int
}

type abyssSocialProgramView struct {
	Friends          []abyssProgramFriendView
	Messages         []abyssProgramMessageView
	Trades           []abyssProgramTradeView
	Duels            []abyssProgramDuelView
	Rescues          []abyssProgramRescueView
	Guild            abyssProgramGuildView
	Tournament       abyssProgramTeamView
	Raid             abyssProgramRaidView
	GuildLeaders     []abyssProgramRankView
	KudosLeaders     []abyssProgramRankView
	TournamentBoard  []abyssProgramRankView
	Activity         []abyssProgramMessageView
	ReferralCode     string
	MentorStatus     string
	DailyFloors      int64
	DailyGoal        int
	DailyGoalPercent int
	HappyHour        bool
	Kudos            int
	DuelWins         int
	HelperAssists    int
	PartyOwnerNick   string
}

func (b *Bot) abyssSocialProgram(uid string) abyssSocialProgramView {
	view := abyssSocialProgramView{DailyGoal: abyssDailyServerGoal}
	view.ReferralCode = abyssReferralCode(uid)
	_, _ = b.DB.Exec(`INSERT INTO abyss_referral_codes (client_uid,code) VALUES ($1,$2)
		ON CONFLICT (client_uid) DO NOTHING`, uid, view.ReferralCode)
	_ = b.DB.QueryRow(`SELECT COALESCE(SUM(floors_cleared),0) FROM abyss_runs
		WHERE created_at>=CURRENT_DATE AND created_at<CURRENT_DATE+INTERVAL '1 day'`).Scan(&view.DailyFloors)
	view.DailyGoalPercent = abyssSocialGoalPercent(view.DailyFloors)
	view.HappyHour = view.DailyFloors >= abyssDailyServerGoal

	rows, err := b.DB.Query(`SELECT CASE WHEN f.uid_low=$1 THEN f.uid_high ELSE f.uid_low END,
		COALESCE(NULLIF(u.nickname,''),'Adventurer'),u.last_seen,f.requested_by,
		EXISTS(SELECT 1 FROM abyss_active a WHERE a.client_uid=u.client_uid),
		f.accepted_at IS NOT NULL FROM abyss_friendships f JOIN users u ON u.client_uid=
		CASE WHEN f.uid_low=$1 THEN f.uid_high ELSE f.uid_low END
		WHERE f.uid_low=$1 OR f.uid_high=$1 ORDER BY f.accepted_at NULLS FIRST,u.nickname`, uid)
	if err == nil {
		defer func() { _ = rows.Close() }()
		for rows.Next() {
			var friend abyssProgramFriendView
			var lastSeen time.Time
			var requestedBy string
			if rows.Scan(&friend.UID, &friend.Nick, &lastSeen, &requestedBy, &friend.InRun, &friend.Accepted) == nil {
				friend.Online = abyssSocialOnline(lastSeen, timeNowUTC())
				friend.IncomingRequest = !friend.Accepted && requestedBy != uid
				friend.State = "pending"
				if friend.Accepted {
					friend.State = "friend"
				}
				if friend.InRun {
					friend.State = "in run"
				}
				view.Friends = append(view.Friends, friend)
			}
		}
	}

	_ = b.DB.QueryRow(`SELECT g.name,g.tag,g.banner,g.invite_code,m.role,
		(SELECT COUNT(*) FROM abyss_guild_members gm WHERE gm.guild_id=g.guild_id),
		COALESCE(p.floors,0),COALESCE(p.goal,$2) FROM abyss_guild_members m
		JOIN abyss_guilds g ON g.guild_id=m.guild_id LEFT JOIN abyss_guild_weekly_progress p
		ON p.guild_id=g.guild_id AND p.week_key=$3 WHERE m.client_uid=$1`, uid,
		abyssGuildWeeklyGoal, abyssCurrentWeek(timeNowUTC())).Scan(&view.Guild.Name, &view.Guild.Tag,
		&view.Guild.Banner, &view.Guild.InviteCode, &view.Guild.Role, &view.Guild.Members,
		&view.Guild.Floors, &view.Guild.Goal)
	view.Guild.Exists = view.Guild.Name != ""

	rows, err = b.DB.Query(`SELECT COALESCE(NULLIF(u.nickname,''),'Adventurer'),m.message,m.created_at
		FROM abyss_shoutbox_messages m JOIN users u ON u.client_uid=m.sender_uid
		WHERE m.created_at>NOW()-INTERVAL '24 hours' ORDER BY m.created_at DESC LIMIT 12`)
	if err == nil {
		defer func() { _ = rows.Close() }()
		for rows.Next() {
			var message abyssProgramMessageView
			var created time.Time
			if rows.Scan(&message.Nick, &message.Message, &created) == nil {
				message.When = created.Format("15:04")
				view.Messages = append(view.Messages, message)
			}
		}
	}

	rows, err = b.DB.Query(`SELECT t.trade_id,CASE WHEN t.sender_uid=$1 THEN t.recipient_uid ELSE t.sender_uid END,
		COALESCE(NULLIF(u.nickname,''),'Adventurer'),t.offer_quantity||'× '||t.offer_cons_id,
		t.request_quantity||'× '||t.request_cons_id,t.recipient_uid=$1 FROM abyss_consumable_trades t
		JOIN users u ON u.client_uid=CASE WHEN t.sender_uid=$1 THEN t.recipient_uid ELSE t.sender_uid END
		WHERE (t.sender_uid=$1 OR t.recipient_uid=$1) AND t.status='open'
		ORDER BY t.created_at DESC`, uid)
	if err == nil {
		defer func() { _ = rows.Close() }()
		for rows.Next() {
			var trade abyssProgramTradeView
			if rows.Scan(&trade.ID, &trade.OtherUID, &trade.OtherNick, &trade.Offer, &trade.Request, &trade.Incoming) == nil {
				view.Trades = append(view.Trades, trade)
			}
		}
	}

	rows, err = b.DB.Query(`SELECT d.duel_id,CASE WHEN d.challenger_uid=$1 THEN d.opponent_uid ELSE d.challenger_uid END,
		COALESCE(NULLIF(u.nickname,''),'Adventurer'),d.wager_tokens,d.opponent_uid=$1 FROM abyss_duels d
		JOIN users u ON u.client_uid=CASE WHEN d.challenger_uid=$1 THEN d.opponent_uid ELSE d.challenger_uid END
		WHERE (d.challenger_uid=$1 OR d.opponent_uid=$1) AND d.status='pending' ORDER BY d.created_at DESC`, uid)
	if err == nil {
		defer func() { _ = rows.Close() }()
		for rows.Next() {
			var duel abyssProgramDuelView
			if rows.Scan(&duel.ID, &duel.OtherUID, &duel.OtherNick, &duel.Wager, &duel.Incoming) == nil {
				view.Duels = append(view.Duels, duel)
			}
		}
	}

	rows, err = b.DB.Query(`SELECT d.id,COALESCE(NULLIF(u.nickname,''),'Adventurer'),d.depth,d.lost_cache
		FROM abyss_deaths d JOIN users u ON u.client_uid=d.client_uid
		JOIN abyss_friendships f ON f.uid_low=LEAST($1,d.client_uid) AND f.uid_high=GREATEST($1,d.client_uid)
		WHERE d.client_uid<>$1 AND d.rescued_at IS NULL AND d.lost_cache>0 AND f.accepted_at IS NOT NULL
		AND NOT EXISTS(SELECT 1 FROM abyss_rescue_missions r WHERE r.death_id=d.id)
		ORDER BY d.died_at DESC LIMIT 5`, uid)
	if err == nil {
		defer func() { _ = rows.Close() }()
		for rows.Next() {
			var rescue abyssProgramRescueView
			if rows.Scan(&rescue.DeathID, &rescue.Nick, &rescue.Depth, &rescue.LostCache) == nil {
				view.Rescues = append(view.Rescues, rescue)
			}
		}
	}

	week := abyssCurrentWeek(timeNowUTC())
	var raidOwner string
	if b.DB.QueryRow(`SELECT l.lobby_code,l.owner_uid,(SELECT COUNT(*) FROM abyss_raid_members rm
		WHERE rm.lobby_code=l.lobby_code) FROM abyss_raid_members m JOIN abyss_raid_lobbies l
		ON l.lobby_code=m.lobby_code WHERE m.week_key=$1 AND m.client_uid=$2 AND l.status='open'`,
		week, uid).Scan(&view.Raid.Code, &raidOwner, &view.Raid.Members) == nil {
		view.Raid.Exists = true
		view.Raid.Owner = raidOwner == uid
	}
	if b.DB.QueryRow(`SELECT t.name,t.team_code,(SELECT COUNT(*) FROM abyss_tournament_members tm
		WHERE tm.week_key=t.week_key AND tm.team_code=t.team_code) FROM abyss_tournament_members m
		JOIN abyss_tournament_teams t ON t.week_key=m.week_key AND t.team_code=m.team_code
		WHERE m.week_key=$1 AND m.client_uid=$2`, week, uid).Scan(&view.Tournament.Name,
		&view.Tournament.Code, &view.Tournament.Members) == nil {
		view.Tournament.Exists = true
	}

	rows, err = b.DB.Query(`SELECT g.name,g.tag,p.floors FROM abyss_guild_weekly_progress p
		JOIN abyss_guilds g ON g.guild_id=p.guild_id WHERE p.week_key=$1
		ORDER BY p.floors DESC,g.name LIMIT 8`, week)
	if err == nil {
		defer func() { _ = rows.Close() }()
		for rows.Next() {
			var rank abyssProgramRankView
			if rows.Scan(&rank.Name, &rank.Meta, &rank.Value) == nil {
				view.GuildLeaders = append(view.GuildLeaders, rank)
			}
		}
	}
	rows, err = b.DB.Query(`SELECT COALESCE(NULLIF(u.nickname,''),'Adventurer'),COUNT(*) FROM abyss_kudos k
		JOIN users u ON u.client_uid=k.recipient_uid WHERE k.week_key=$1
		GROUP BY u.client_uid,u.nickname ORDER BY COUNT(*) DESC,u.nickname LIMIT 8`, week)
	if err == nil {
		defer func() { _ = rows.Close() }()
		for rows.Next() {
			var rank abyssProgramRankView
			if rows.Scan(&rank.Name, &rank.Value) == nil {
				rank.Meta = "kudos"
				view.KudosLeaders = append(view.KudosLeaders, rank)
			}
		}
	}
	rows, err = b.DB.Query(`SELECT t.name,t.team_code,COUNT(m.client_uid) FROM abyss_tournament_teams t
		JOIN abyss_tournament_members m ON m.week_key=t.week_key AND m.team_code=t.team_code
		WHERE t.week_key=$1 GROUP BY t.name,t.team_code ORDER BY COUNT(m.client_uid) DESC,t.created_at LIMIT 12`, week)
	if err == nil {
		defer func() { _ = rows.Close() }()
		for rows.Next() {
			var rank abyssProgramRankView
			if rows.Scan(&rank.Name, &rank.Meta, &rank.Value) == nil {
				view.TournamentBoard = append(view.TournamentBoard, rank)
			}
		}
	}
	rows, err = b.DB.Query(`SELECT COALESCE(NULLIF(u.nickname,''),'Adventurer'),
		CASE WHEN r.victory THEN 'banked at depth '||r.depth ELSE 'fell at depth '||r.depth END,r.created_at
		FROM abyss_runs r JOIN users u ON u.client_uid=r.client_uid
		ORDER BY r.created_at DESC LIMIT 12`)
	if err == nil {
		defer func() { _ = rows.Close() }()
		for rows.Next() {
			var activity abyssProgramMessageView
			var created time.Time
			if rows.Scan(&activity.Nick, &activity.Message, &created) == nil {
				activity.When = created.Format("Jan 2 15:04")
				view.Activity = append(view.Activity, activity)
			}
		}
	}

	_ = b.DB.QueryRow(`SELECT COALESCE(SUM(assists),0) FROM abyss_helper_bonds
		WHERE uid_low=$1 OR uid_high=$1`, uid).Scan(&view.HelperAssists)
	_ = b.DB.QueryRow(`SELECT COUNT(*) FROM abyss_duels WHERE winner_uid=$1`, uid).Scan(&view.DuelWins)
	_ = b.DB.QueryRow(`SELECT COUNT(*) FROM abyss_kudos WHERE week_key=$1 AND recipient_uid=$2`, week, uid).Scan(&view.Kudos)
	_ = b.DB.QueryRow(`SELECT COALESCE(NULLIF(u.nickname,''),'Adventurer') FROM abyss_party_members p
		JOIN users u ON u.client_uid=p.owner_uid WHERE p.member_uid=$1`, uid).Scan(&view.PartyOwnerNick)
	var mentor, mentee sql.NullString
	_ = b.DB.QueryRow(`SELECT MAX(CASE WHEN mentor_uid=$1 THEN mentee_uid END),
		MAX(CASE WHEN mentee_uid=$1 THEN mentor_uid END) FROM abyss_mentor_pairs
		WHERE (mentor_uid=$1 OR mentee_uid=$1) AND completed_at IS NULL`, uid).Scan(&mentee, &mentor)
	if mentee.Valid {
		view.MentorStatus = fmt.Sprintf("Mentoring %s", mentee.String)
	} else if mentor.Valid {
		view.MentorStatus = fmt.Sprintf("Mentored by %s", mentor.String)
	}
	return view
}
