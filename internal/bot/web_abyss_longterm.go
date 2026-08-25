package bot

import (
	"context"
	"math"
	"time"
)

type abyssPaceView struct {
	CurrentFloorsPerMinute float64 `json:"current_floors_per_minute"`
	LastFloorsPerMinute    float64 `json:"last_floors_per_minute"`
	BestFloorsPerMinute    float64 `json:"best_floors_per_minute"`
	CurrentFloors          int     `json:"current_floors"`
	StartedUnix            int64   `json:"started_unix"`
	LastDepth              int     `json:"last_depth"`
	BestDepth              int     `json:"best_depth"`
}

type abyssBountyDayView struct {
	Date    string `json:"date"`
	Label   string `json:"label"`
	Claimed bool   `json:"claimed"`
}

type abyssMilestoneView struct {
	Depth   int    `json:"depth"`
	Label   string `json:"label"`
	Kind    string `json:"kind"`
	Reached bool   `json:"reached"`
}

type abyssLongTermView struct {
	Pace          abyssPaceView        `json:"pace"`
	BountyDays    []abyssBountyDayView `json:"bounty_days"`
	MaterialFlow  map[string][]int64   `json:"material_flow_7d"`
	Milestones    []abyssMilestoneView `json:"milestones"`
	BestDepth     int                  `json:"best_depth"`
	MapDepth      int                  `json:"map_depth"`
	LegendaryPity int                  `json:"legendary_pity"`
	LegendaryCap  int                  `json:"legendary_cap"`
}

func (b *Bot) abyssLongTermStatus(ctx context.Context, uid string, run abyssRun, history []abyssHistoryRow, bestDepth, legendaryPity int) abyssLongTermView {
	view := abyssLongTermView{
		BountyDays:    b.abyssBountyHistory(uid, time.Now()),
		MaterialFlow:  b.loadAbyssForgeMaterialFlow(uid),
		BestDepth:     bestDepth,
		MapDepth:      max(50, int(math.Ceil(float64(bestDepth)/10))*10),
		LegendaryPity: min(max(legendaryPity, 0), abyssLegendaryPityCap),
		LegendaryCap:  abyssLegendaryPityCap,
	}
	view.Pace = abyssPaceStatus(run, history)
	view.Pace.BestFloorsPerMinute, view.Pace.BestDepth = b.abyssBestPace(ctx, uid, history)
	view.Milestones = abyssMilestones(bestDepth)
	return view
}

func (b *Bot) abyssBestPace(ctx context.Context, uid string, history []abyssHistoryRow) (float64, int) {
	bestPace, bestDepth := bestPaceFromHistory(history)
	var depth, floors int
	var durationMS int64
	err := b.DB.QueryRowContext(ctx, `SELECT depth,floors_cleared,duration_ms
		FROM abyss_runs
		WHERE client_uid=$1 AND floors_cleared>0 AND duration_ms>0
		ORDER BY floors_cleared::numeric/duration_ms DESC,depth DESC,id DESC LIMIT 1`, uid).
		Scan(&depth, &floors, &durationMS)
	if err == nil {
		return float64(floors) * 60_000 / float64(durationMS), depth
	}
	return bestPace, bestDepth
}

func bestPaceFromHistory(history []abyssHistoryRow) (float64, int) {
	var bestPace float64
	var bestDepth int
	for _, row := range history {
		if row.FloorsCleared <= 0 || row.DurationMS <= 0 {
			continue
		}
		pace := float64(row.FloorsCleared) * 60_000 / float64(row.DurationMS)
		if pace > bestPace || pace == bestPace && row.Depth > bestDepth {
			bestPace, bestDepth = pace, row.Depth
		}
	}
	return bestPace, bestDepth
}

func abyssPaceStatus(run abyssRun, history []abyssHistoryRow) abyssPaceView {
	var view abyssPaceView
	if run.Active {
		view.CurrentFloors = abyssRunFloorsCleared(run)
		if !run.StartedAt.IsZero() {
			view.StartedUnix = run.StartedAt.Unix()
		}
		if duration := abyssRunDurationMS(run); duration > 0 {
			view.CurrentFloorsPerMinute = float64(abyssRunFloorsCleared(run)) * 60_000 / float64(duration)
		}
	}
	for _, row := range history {
		if row.FloorsCleared <= 0 || row.DurationMS <= 0 {
			continue
		}
		view.LastFloorsPerMinute = float64(row.FloorsCleared) * 60_000 / float64(row.DurationMS)
		view.LastDepth = row.Depth
		break
	}
	return view
}

func (b *Bot) abyssBountyHistory(uid string, now time.Time) []abyssBountyDayView {
	today := abyssBountyDay(now)
	start := today.AddDate(0, 0, -6)
	claimed := make(map[string]bool, 7)
	rows, err := b.DB.Query(`SELECT bounty_day FROM abyss_bounty_claims
		WHERE client_uid=$1 AND bounty_day >= $2::date AND bounty_day <= $3::date
		ORDER BY bounty_day`, uid, start, today)
	if err == nil {
		defer func() { _ = rows.Close() }()
		for rows.Next() {
			var day time.Time
			if rows.Scan(&day) == nil {
				claimed[day.UTC().Format("2006-01-02")] = true
			}
		}
	}
	out := make([]abyssBountyDayView, 0, 7)
	for offset := -6; offset <= 0; offset++ {
		day := today.AddDate(0, 0, offset)
		key := day.Format("2006-01-02")
		out = append(out, abyssBountyDayView{Date: key, Label: day.Format("Mon"), Claimed: claimed[key]})
	}
	return out
}

func abyssMilestones(bestDepth int) []abyssMilestoneView {
	mapDepth := max(50, int(math.Ceil(float64(bestDepth)/10))*10)
	definitions := make([]abyssMilestoneView, 0, mapDepth/abyssBossEvery+4)
	for depth := abyssBossEvery; depth <= mapDepth; depth += abyssBossEvery {
		kind, label := "boss", "Boss floor"
		if depth%10 == 0 {
			kind, label = "checkpoint", "Boss checkpoint"
		}
		definitions = append(definitions, abyssMilestoneView{Depth: depth, Label: label, Kind: kind})
	}
	definitions = append(definitions,
		abyssMilestoneView{Depth: 15, Label: "Nightmare tier", Kind: "tier"},
		abyssMilestoneView{Depth: abyssJackpotDepth, Label: "Deep-cache jackpot", Kind: "jackpot"},
		abyssMilestoneView{Depth: 30, Label: "Hell tier", Kind: "tier"},
		abyssMilestoneView{Depth: 50, Label: "Insanity tier", Kind: "tier"},
	)
	for index := range definitions {
		definitions[index].Reached = bestDepth >= definitions[index].Depth
	}
	return definitions
}
