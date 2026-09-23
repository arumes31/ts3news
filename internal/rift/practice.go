package rift

import (
	"errors"
	"math"
	"time"
	"ts3news/internal/content"
)

// PracticeState describes an isolated drill; it cannot bank or advance a campaign.
type PracticeState struct {
	HazardIntensity string `json:"hazard_intensity,omitempty"`
	SlowTelegraphs bool    `json:"slow_telegraphs,omitempty"`
	BossStart      *Actor  `json:"boss_start,omitempty"`
	Dodges         int     `json:"dodges,omitempty"`
	PulseCycle     int     `json:"pulse_cycle,omitempty"`
	PulseHits      int     `json:"pulse_hits,omitempty"`
	PulseResolved  bool    `json:"pulse_resolved,omitempty"`
	Mode           string  `json:"mode"`
	GoalX          float64 `json:"goal_x"`
	Completed      bool    `json:"completed"`
	Hits           int     `json:"hits"`
	Arena          Arena   `json:"arena"`
}

// ValidPracticeMode reports whether mode names a supported isolated drill.
func ValidPracticeMode(mode string) bool {
	return mode == "boss" || mode == "movement" || mode == "jump" || mode == "combo" || mode == "guard" || mode == "hazard"
}

func NewPracticeRun(id string, build Build, mode string, now time.Time) (*Run, error) {
	if !ValidPracticeMode(mode) {
		return nil, errors.New("unknown practice drill")
	}
	if mode == "boss" {
		for _, mob := range content.AbyssMobCatalog() {
			if AdaptMonster(mob).Kind == "boss" {
				return NewBossPracticeRun(id, build, mob.Name, 1, now)
			}
		}
		return nil, errors.New("no practice bosses available")
	}
	return newPracticeRun(id, build, mode, now)
}

func newPracticeRun(id string, build Build, mode string, now time.Time) (*Run, error) {
	r := NewRunWithCatalog(id, build, now, nil)
	r.Practice = &PracticeState{Mode: mode, GoalX: 900, Arena: Arena{Name: "Practice lane", Obstacles: []Obstacle{}, Hazards: []Hazard{}, Floor: "stone"}}
	r.Floor = "stone"
	r.Enemies = []Actor{}
	for i := range r.EncounterPlan {
		r.EncounterPlan[i] = []Actor{}
	}
	if mode == "jump" {
		r.Practice.Arena.Obstacles = []Obstacle{{X: 450, Y: 250, W: 70, H: 300}}
	}
	if mode == "combo" {
		r.Enemies = []Actor{{ID: "practice-target", Name: "Training target", Kind: "knight", X: 220, Y: r.Player.Y, HP: 1000000, MaxHP: 1000000, Facing: -1}}
	}
	if mode == "guard" {
		r.Enemies = []Actor{{ID: "practice-guard", Name: "Guard trainer", Kind: "knight", X: 220, Y: r.Player.Y, HP: 1000000, MaxHP: 1000000, Facing: -1, Damage: 8, Cooldown: 1}}
	}
	if mode == "hazard" {
		r.Practice.Arena.Name = "Warning zone"
		r.Practice.Arena.Hazards = []Hazard{{Obstacle: Obstacle{X: 110, Y: 365, W: 100, H: 90}, Kind: "fire", Jumpable: true, Period: 3.5, Duration: .45}}
	}
	return r, nil
}

func (r *Run) ResetPractice(now time.Time) error {
	if r.Practice == nil {
		return errors.New("not a practice run")
	}
	var fresh *Run
	var err error
	if r.Practice.Mode == "boss" {
		if r.Practice.BossStart == nil {
			return errors.New("missing boss practice snapshot")
		}
		fresh, err = newBossPracticeSnapshot(r.ID, r.Build, *r.Practice.BossStart, now)
	} else {
		fresh, err = NewPracticeRun(r.ID, r.Build, r.Practice.Mode, now)
	}
	if err != nil {
		return err
	}
	if r.Practice.Mode == "hazard" {
		if err := fresh.ConfigureHazardPractice(r.Practice.HazardIntensity); err != nil {
			return err
		}
	}
	fresh.Practice.SlowTelegraphs = r.Practice.SlowTelegraphs
	fresh.Revision = r.Revision
	fresh.StartKey = r.StartKey
	fresh.Epoch = r.Epoch
	*r = *fresh
	return nil
}

func (r *Run) practiceInput(in Input) Input {
	if r.Practice == nil {
		return in
	}
	switch r.Practice.Mode {
	case "boss":
		return in
	case "movement":
		return Input{X: in.X, Y: in.Y}
	case "jump", "hazard":
		return Input{X: in.X, Y: in.Y, Jump: in.Jump}
	case "guard":
		return Input{X: in.X, Y: in.Y, Guard: in.Guard}
	case "combo":
		return Input{X: in.X, Y: in.Y, Attack: in.Attack, Guard: in.Guard}
	}
	return Input{}
}

func (r *Run) practiceTick() {
	if r.Status != "fighting" {
		return
	}
	complete := r.Player.X >= r.Practice.GoalX
	if r.Practice.Mode == "boss" {
		complete = len(r.Enemies) == 1 && r.Enemies[0].HP <= 0
	}
	if r.Practice.Mode == "jump" {
		complete = complete && r.Stats.Jumps > 0
	}
	if r.Practice.Mode == "combo" {
		complete = r.Practice.Hits >= 3 && r.Combo == 3
	}
	if r.Practice.Mode == "guard" {
		complete = r.Stats.Guards >= 3
	}
	if r.Practice.Mode == "hazard" {
		practice := r.Practice
		h := &practice.Arena.Hazards[0]
		cycle := int(math.Floor(r.Clock / h.Period))
		if cycle > practice.PulseCycle {
			practice.PulseCycle = cycle
			practice.PulseHits = r.Stats.HitsTaken
			practice.PulseResolved = false
			h.X = clamp(r.Player.X-h.W/2, 0, Width-h.W)
			h.Y = r.Player.Y - h.H/2
		}
		if h.Phase(r.Clock) >= 1.2+h.Duration && !practice.PulseResolved {
			practice.PulseResolved = true
			if r.Stats.HitsTaken == practice.PulseHits {
				practice.Dodges++
			} else {
				practice.Dodges = 0
			}
		}
		complete = practice.Dodges >= 3
	}
	if complete {
		r.Practice.Completed = true
		r.Status = "complete"
		r.event("clear", r.Player.X, r.Player.Y, 0)
	}
}

// PracticeTool changes practice resources without changing drill or campaign records.
func (r *Run) PracticeTool(kind string) error {
	if r.Practice == nil || r.Status != "fighting" {
		return errors.New("practice recovery requires an active drill")
	}
	switch kind {
	case "practice_health":
		r.Player.HP = r.Player.MaxHP
	case "practice_mana":
		r.Player.Mana = 100
	case "practice_cooldowns":
		for _, skill := range r.Build.Skills {
			r.SkillTimers[skill.ID] = 0
		}
		for _, skill := range r.Build.Signatures {
			r.SkillTimers[skill.ID] = 0
		}
		if r.Build.Ultimate != nil {
			r.SkillTimers[r.Build.Ultimate.ID] = 0
		}
	default:
		return errors.New("unknown practice recovery control")
	}
	return nil
}
