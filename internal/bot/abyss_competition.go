package bot

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"ts3news/internal/clientquery"
)

const (
	abyssCompetitionPageSize = 10
	abyssCompetitionMaxRows  = 200
)

type abyssCompetitionRunRecord struct {
	Build          string
	PactMultiplier float64
	ChannelID      sql.NullInt64
	AuditHash      string
	AuditJSON      string
}

type abyssCompetitionAudit struct {
	Version        int                   `json:"version"`
	UID            string                `json:"uid"`
	Depth          int                   `json:"depth"`
	Gold           int64                 `json:"gold"`
	Victory        bool                  `json:"victory"`
	Tier           string                `json:"tier"`
	Hardcore       bool                  `json:"hardcore"`
	Build          string                `json:"build"`
	PactMultiplier float64               `json:"pact_multiplier"`
	StartedAt      string                `json:"started_at"`
	EndedAt        string                `json:"ended_at"`
	EndReason      string                `json:"end_reason"`
	PreviousHash   string                `json:"previous_hash,omitempty"`
	RunSeed        *[2]uint64            `json:"run_seed,omitempty"`
	Choices        []abyssRunChoice      `json:"choices,omitempty"`
	Floors         []abyssRunFloorRecord `json:"floors,omitempty"`
}

func abyssCompetitionWeekAt(at time.Time) (string, time.Time, time.Time) {
	at = at.UTC()
	year, week := at.ISOWeek()
	weekday := (int(at.Weekday()) + 6) % 7
	start := time.Date(at.Year(), at.Month(), at.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, -weekday)
	return fmt.Sprintf("%04d-W%02d", year, week), start, start.AddDate(0, 0, 7)
}

func abyssCompetitionSeasonAt(at time.Time) (string, time.Time, time.Time) {
	campaign := abyssSeasonCampaignAt(at.UTC())
	return campaign.ID, campaign.Start, campaign.End
}

func (b *Bot) updateAbyssCompetitionPresence(clients []clientquery.ClientInfo) {
	for _, member := range clients {
		if member.Type != 0 || member.UID == "" || member.CID < 0 {
			continue
		}
		_, _ = b.DB.Exec(`INSERT INTO abyss_competition_presence (client_uid,channel_id,seen_at)
			VALUES ($1,$2,NOW()) ON CONFLICT (client_uid) DO UPDATE
			SET channel_id=EXCLUDED.channel_id,seen_at=EXCLUDED.seen_at`, member.UID, member.CID)
	}
}

func (b *Bot) newAbyssCompetitionRunRecord(
	uid string,
	run abyssRun,
	gold int64,
	victory, hardcore bool,
	endReason string,
	pactMultiplier float64,
) (abyssCompetitionRunRecord, error) {
	record := abyssCompetitionRunRecord{Build: "initiate", PactMultiplier: max(1, pactMultiplier)}
	_ = b.DB.QueryRow(`SELECT COALESCE(NULLIF(active_specialization,''),'initiate')
		FROM users WHERE client_uid=$1`, uid).Scan(&record.Build)
	_ = b.DB.QueryRow(`SELECT channel_id FROM abyss_competition_presence
		WHERE client_uid=$1 AND seen_at>NOW()-INTERVAL '30 minutes'`, uid).Scan(&record.ChannelID)
	var previousHash string
	_ = b.DB.QueryRow(`SELECT audit_hash FROM abyss_runs WHERE client_uid=$1 AND audit_hash<>''
		ORDER BY id DESC LIMIT 1`, uid).Scan(&previousHash)
	audit := abyssCompetitionAudit{
		Version: 1, UID: uid, Depth: run.Depth, Gold: gold, Victory: victory,
		Tier: run.Tier, Hardcore: hardcore, Build: record.Build,
		PactMultiplier: record.PactMultiplier, StartedAt: run.StartedAt.UTC().Format(time.RFC3339Nano),
		EndedAt: time.Now().UTC().Format(time.RFC3339Nano), EndReason: endReason, PreviousHash: previousHash,
	}
	provenance, provenanceErr := b.loadAbyssRunProvenance(uid)
	if provenanceErr == nil {
		seed := provenance.Seed
		audit.RunSeed = &seed
		audit.Choices = append([]abyssRunChoice{}, provenance.Choices...)
		audit.Floors = append([]abyssRunFloorRecord{}, provenance.Floors...)
	} else if !errors.Is(provenanceErr, sql.ErrNoRows) {
		return abyssCompetitionRunRecord{}, provenanceErr
	}
	data, err := json.Marshal(audit)
	if err != nil {
		return abyssCompetitionRunRecord{}, fmt.Errorf("marshal abyss run audit: %w", err)
	}
	digest := sha256.Sum256(data)
	record.AuditJSON = string(data)
	record.AuditHash = hex.EncodeToString(digest[:])
	return record, nil
}

func (b *Bot) abyssCompetitionPactMultiplier(uid string, pacts []string, flags map[string]int64) float64 {
	mastery, err := b.loadAbyssPactMastery(uid)
	if err != nil {
		return abyssPactRewardMult(pacts)
	}
	_, dailyAffix := b.abyssRunDailyChallenge(uid)
	return abyssPactRewardBreakdownForRunAt(
		pacts, mastery, dailyAffix, time.Now().UTC(), flags[abyssRunFlagMysteryPact] > 0,
	).Multiplier
}
