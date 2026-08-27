package bot

import (
	"database/sql"
	"fmt"
	"math"
	"strings"
	"time"
)

type abyssCompetitionRow struct {
	Rank           int
	UID            string
	Nickname       string
	Build          string
	Depth          int
	Gold           int64
	DurationMS     int64
	PactMultiplier float64
	Extra          string
	AuditHash      string
	IsCurrent      bool
}

type abyssCompetitionBoard struct {
	Key      string
	Title    string
	Subtitle string
	TieBreak string
	Rows     []abyssCompetitionRow
}

type abyssCompetitionArchive struct {
	Season string
	Tier   string
	Rank   int
	Nick   string
	Depth  int
	Reward string
}

type abyssCompetitionDepthPoint struct {
	Date  string
	Depth int
	Pct   int
}

type abyssCompetitionAuditView struct {
	Depth int
	When  string
	Hash  string
	Valid bool
}

type abyssCompetitionWagerView struct {
	Bracket  int
	Fee      int64
	Entrants int
	Pool     int64
	Joined   bool
}

type abyssCompetitionView struct {
	Tier          string
	Build         string
	Period        string
	PeriodLabel   string
	Builds        []string
	Boards        []abyssCompetitionBoard
	CurrentRank   int
	TotalPlayers  int
	TopPercent    int
	CloseGap      int
	CloseToRecord bool
	BestDepth     int
	LastWeekDepth int
	DepthDelta    int
	Archives      []abyssCompetitionArchive
	DepthHistory  []abyssCompetitionDepthPoint
	Audits        []abyssCompetitionAuditView
	Trophies      []string
	Wagers        []abyssCompetitionWagerView
	ShameOptIn    bool
}

func canonicalAbyssCompetitionPeriod(value string) string {
	switch value {
	case "weekly", "all_time":
		return value
	default:
		return "season"
	}
}

func canonicalAbyssCompetitionBuild(value string) string {
	switch value {
	case "initiate", "delver", "plunderer", "warden":
		return value
	default:
		return ""
	}
}

func abyssCompetitionPeriodStart(period string, at time.Time) (time.Time, string) {
	switch period {
	case "weekly":
		key, start, _ := abyssCompetitionWeekAt(at)
		return start, key
	case "all_time":
		return time.Unix(0, 0).UTC(), "All time"
	default:
		key, start, _ := abyssCompetitionSeasonAt(at)
		return start, key
	}
}

func (b *Bot) abyssCompetition(uid, tier, build, period string, at time.Time) abyssCompetitionView {
	period = canonicalAbyssCompetitionPeriod(period)
	build = canonicalAbyssCompetitionBuild(build)
	start, label := abyssCompetitionPeriodStart(period, at)
	view := abyssCompetitionView{
		Tier: tier, Build: build, Period: period, PeriodLabel: label,
		Builds: []string{"initiate", "delver", "plunderer", "warden"},
	}
	depthRows, total := b.abyssCompetitionDepthRows(uid, tier, build, start)
	view.TotalPlayers = total
	for _, row := range depthRows {
		if row.IsCurrent {
			view.CurrentRank = row.Rank
			view.BestDepth = row.Depth
			break
		}
	}
	if view.CurrentRank > 0 && total > 0 {
		view.TopPercent = max(1, int(math.Ceil(float64(view.CurrentRank)*100/float64(total))))
	}
	if len(depthRows) >= 10 {
		target := depthRows[9].Depth
		view.CloseGap = max(0, target-view.BestDepth)
		view.CloseToRecord = view.CloseGap > 0 && view.CloseGap <= 2
	}
	view.Boards = []abyssCompetitionBoard{
		{Key: "depth", Title: "Deepest descents", Subtitle: label, TieBreak: "Depth, then banked gold, then earliest verified run.", Rows: depthRows},
		{Key: "speed", Title: "Depth 20 speedruns", Subtitle: "Fastest verified real time", TieBreak: "Duration, then deeper finish, then earliest run.", Rows: b.abyssCompetitionSpeedRows(uid, tier, build, start)},
		{Key: "economy", Title: "Weekly vault", Subtitle: "Most gold banked this ISO week", TieBreak: "Gold, then deepest run, then player ID.", Rows: b.abyssCompetitionEconomyRows(uid, tier, build, at)},
		{Key: "pact", Title: "Pact survival", Subtitle: "Highest total pact multiplier banked", TieBreak: "Multiplier, then depth, then duration.", Rows: b.abyssCompetitionPactRows(uid, tier, build, start)},
		{Key: "bestiary", Title: "Bestiary families", Subtitle: "Lifetime kills by monster family", TieBreak: "Kills, then family, then player ID.", Rows: b.abyssCompetitionBestiaryRows(uid)},
		{Key: "shame", Title: "Hall of shame", Subtitle: "Opt-in-worthy deaths the Abyss remembers", TieBreak: "Depth, then most recent spectacular defeat.", Rows: b.abyssCompetitionShameRows(uid)},
		{Key: "streak", Title: "Bank streaks", Subtitle: "Current consecutive successful banks", TieBreak: "Streak, then lifetime banked gold.", Rows: b.abyssCompetitionStreakRows(uid)},
		{Key: "pets", Title: "Companion power", Subtitle: "Strongest active and stable companions", TieBreak: "Power, then pet level, then oldest companion.", Rows: b.abyssCompetitionPetRows(uid)},
	}
	if channel := b.abyssCompetitionChannelRows(uid, tier, start); channel != nil {
		view.Boards = append(view.Boards, abyssCompetitionBoard{Key: "channel", Title: "TS3 channel rivals", Subtitle: "Local bragging rights", TieBreak: "Depth, then gold among recently seen channel members.", Rows: channel})
	}
	view.Archives = b.abyssCompetitionArchives()
	view.LastWeekDepth, view.DepthDelta = b.abyssCompetitionPersonalDelta(uid, view.BestDepth, at)
	view.DepthHistory = b.abyssCompetitionDepthHistory(uid)
	view.Audits = b.abyssCompetitionAudits(uid)
	view.Trophies = b.abyssCompetitionTrophies(uid)
	view.Wagers = b.abyssCompetitionWagers(uid, at)
	_ = b.DB.QueryRow(`SELECT shame_opt_in FROM abyss_social_profiles WHERE client_uid=$1`, uid).Scan(&view.ShameOptIn)
	return view
}

func scanAbyssCompetitionRows(rows *sql.Rows, uid string) []abyssCompetitionRow {
	defer func() { _ = rows.Close() }()
	result := make([]abyssCompetitionRow, 0, abyssCompetitionMaxRows)
	for rows.Next() {
		var row abyssCompetitionRow
		if rows.Scan(&row.Rank, &row.UID, &row.Nickname, &row.Build, &row.Depth, &row.Gold,
			&row.DurationMS, &row.PactMultiplier, &row.Extra, &row.AuditHash) != nil {
			return nil
		}
		if row.Build != "" {
			row.Build = strings.ToUpper(row.Build[:1]) + row.Build[1:]
		}
		if len(row.AuditHash) > 12 {
			row.AuditHash = row.AuditHash[:12]
		}
		row.IsCurrent = row.UID == uid
		result = append(result, row)
	}
	if rows.Err() != nil {
		return nil
	}
	return result
}

func abyssCompetitionExtra(format string, values ...any) string {
	return fmt.Sprintf(format, values...)
}
