package bot

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"
)

const (
	abyssRunProvenanceVersion    = 1
	abyssRunProvenanceMaxChoices = 200
	abyssRunProvenanceMaxFloors  = 100
	abyssRunProvenanceMaxLogs    = 10
)

type abyssRunChoice struct {
	Depth int    `json:"depth"`
	Kind  string `json:"kind"`
	Value string `json:"value"`
	At    string `json:"at"`
}

type abyssRunFloorRecord struct {
	Depth         int       `json:"depth"`
	Biome         string    `json:"biome,omitempty"`
	Victory       bool      `json:"victory"`
	HP            int       `json:"hp"`
	MaxHP         int       `json:"max_hp"`
	LegendaryDrop bool      `json:"legendary_drop,omitempty"`
	Seed          [2]uint64 `json:"seed"`
	Logs          []string  `json:"logs"`
}

type abyssRunProvenance struct {
	Version int                   `json:"version"`
	Seed    [2]uint64             `json:"seed"`
	Choices []abyssRunChoice      `json:"choices"`
	Floors  []abyssRunFloorRecord `json:"floors"`
}

func abyssRunProvenanceKey(uid string) string {
	return "abyss_run_provenance_" + uid
}

func newAbyssRunProvenance() (abyssRunProvenance, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return abyssRunProvenance{}, fmt.Errorf("generating run seed: %w", err)
	}
	return abyssRunProvenance{
		Version: abyssRunProvenanceVersion,
		Seed: [2]uint64{
			binary.BigEndian.Uint64(raw[:8]),
			binary.BigEndian.Uint64(raw[8:]),
		},
		Choices: []abyssRunChoice{},
		Floors:  []abyssRunFloorRecord{},
	}, nil
}

func saveAbyssRunProvenance(
	exec dbExecQuerier,
	uid string,
	provenance abyssRunProvenance,
) error {
	encoded, err := json.Marshal(provenance)
	if err != nil {
		return fmt.Errorf("encoding run provenance: %w", err)
	}
	if _, err := exec.Exec(
		`INSERT INTO app_meta (key,value) VALUES ($1,$2)
		 ON CONFLICT (key) DO UPDATE SET value=EXCLUDED.value`,
		abyssRunProvenanceKey(uid),
		string(encoded),
	); err != nil {
		return fmt.Errorf("saving run provenance: %w", err)
	}
	return nil
}

func initAbyssRunProvenance(exec dbExecQuerier, uid string) error {
	provenance, err := newAbyssRunProvenance()
	if err != nil {
		return err
	}
	return saveAbyssRunProvenance(exec, uid, provenance)
}

func (b *Bot) loadAbyssRunProvenance(uid string) (abyssRunProvenance, error) {
	var raw string
	err := b.DB.QueryRow(
		"SELECT value FROM app_meta WHERE key=$1",
		abyssRunProvenanceKey(uid),
	).Scan(&raw)
	if err != nil {
		return abyssRunProvenance{}, fmt.Errorf("loading run provenance: %w", err)
	}
	var provenance abyssRunProvenance
	if err := json.Unmarshal([]byte(raw), &provenance); err != nil {
		return abyssRunProvenance{}, fmt.Errorf("decoding run provenance: %w", err)
	}
	if provenance.Version != abyssRunProvenanceVersion || provenance.Seed == [2]uint64{} {
		return abyssRunProvenance{}, errors.New("run provenance is invalid")
	}
	if provenance.Choices == nil {
		provenance.Choices = []abyssRunChoice{}
	}
	if provenance.Floors == nil {
		provenance.Floors = []abyssRunFloorRecord{}
	}
	return provenance, nil
}

func (b *Bot) ensureAbyssRunProvenance(uid string) (abyssRunProvenance, error) {
	provenance, err := b.loadAbyssRunProvenance(uid)
	if err == nil {
		return provenance, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return abyssRunProvenance{}, err
	}
	provenance, err = newAbyssRunProvenance()
	if err != nil {
		return abyssRunProvenance{}, err
	}
	if err := saveAbyssRunProvenance(b.DB, uid, provenance); err != nil {
		return abyssRunProvenance{}, err
	}
	return provenance, nil
}

func abyssRunFloorSeed(seed [2]uint64, depth int) [2]uint64 {
	var material [24]byte
	binary.BigEndian.PutUint64(material[:8], seed[0])
	binary.BigEndian.PutUint64(material[8:16], seed[1])
	binary.BigEndian.PutUint64(material[16:], uint64(max(depth, 0)))
	digest := sha256.Sum256(material[:])
	return [2]uint64{
		binary.BigEndian.Uint64(digest[:8]),
		binary.BigEndian.Uint64(digest[8:16]),
	}
}

func (b *Bot) abyssRunSeedForFloor(uid string, depth int) ([2]uint64, error) {
	provenance, err := b.ensureAbyssRunProvenance(uid)
	if err != nil {
		return [2]uint64{}, err
	}
	return abyssRunFloorSeed(provenance.Seed, depth), nil
}

func (b *Bot) recordAbyssRunChoice(uid string, depth int, kind, value string) {
	provenance, err := b.ensureAbyssRunProvenance(uid)
	if err != nil {
		slog.Warn("load Abyss run provenance", "error", err)
		return
	}
	provenance.Choices = append(provenance.Choices, abyssRunChoice{
		Depth: depth,
		Kind:  boundedAbyssReplayText(kind, 80),
		Value: boundedAbyssReplayText(value, abyssReplayViewTextMaxRunes),
		At:    time.Now().UTC().Format(time.RFC3339Nano),
	})
	if len(provenance.Choices) > abyssRunProvenanceMaxChoices {
		provenance.Choices = append(
			[]abyssRunChoice{},
			provenance.Choices[len(provenance.Choices)-abyssRunProvenanceMaxChoices:]...,
		)
	}
	if err := saveAbyssRunProvenance(b.DB, uid, provenance); err != nil {
		slog.Warn("save Abyss run choice", "error", err)
	}
}

func (b *Bot) recordAbyssRunFloor(uid string, result abyssFloorResult) {
	if result.RandomSeed == [2]uint64{} {
		return
	}
	provenance, err := b.ensureAbyssRunProvenance(uid)
	if err != nil {
		slog.Warn("load Abyss run provenance", "error", err)
		return
	}
	logs := make([]string, 0, min(len(result.LogsHTML), abyssRunProvenanceMaxLogs))
	start := max(0, len(result.LogsHTML)-abyssRunProvenanceMaxLogs)
	for _, line := range result.LogsHTML[start:] {
		logs = append(logs, boundedAbyssReplayText(line, abyssReplayViewTextMaxRunes))
	}
	provenance.Floors = append(provenance.Floors, abyssRunFloorRecord{
		Depth:         result.Depth,
		Biome:         boundedAbyssReplayText(result.Biome, 120),
		Victory:       result.Victory,
		HP:            max(result.CurrentHP, 0),
		MaxHP:         max(result.MaxHP, 0),
		LegendaryDrop: result.LegendaryDrop,
		Seed:          result.RandomSeed,
		Logs:          logs,
	})
	if len(provenance.Floors) > abyssRunProvenanceMaxFloors {
		provenance.Floors = append(
			[]abyssRunFloorRecord{},
			provenance.Floors[len(provenance.Floors)-abyssRunProvenanceMaxFloors:]...,
		)
	}
	if err := saveAbyssRunProvenance(b.DB, uid, provenance); err != nil {
		slog.Warn("save Abyss run floor", "error", err)
	}
}

func deleteAbyssRunProvenance(exec dbExecQuerier, uid string) error {
	if _, err := exec.Exec("DELETE FROM app_meta WHERE key=$1", abyssRunProvenanceKey(uid)); err != nil {
		return fmt.Errorf("deleting run provenance: %w", err)
	}
	return nil
}
