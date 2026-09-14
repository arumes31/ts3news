package rift

import (
	"errors"
	"time"
)

// PracticeState describes an isolated drill; it cannot bank or advance a campaign.
type PracticeState struct {
	Mode      string  `json:"mode"`
	GoalX     float64 `json:"goal_x"`
	Completed bool    `json:"completed"`
	Hits      int     `json:"hits"`
	Arena     Arena   `json:"arena"`
}

// ValidPracticeMode reports whether mode names a supported isolated drill.
func ValidPracticeMode(mode string) bool {
	return mode == "movement" || mode == "jump" || mode == "combo"
}

func NewPracticeRun(id string, build Build, mode string, now time.Time) (*Run, error) {
	if !ValidPracticeMode(mode) {
		return nil, errors.New("unknown practice drill")
	}
	r := NewRunWithCatalog(id, build, now, nil)
	r.Practice = &PracticeState{Mode: mode, GoalX: 900, Arena: Arena{Name: "Practice lane", Obstacles: []Obstacle{}, Hazards: []Hazard{}}}
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
	return r, nil
}

func (r *Run) ResetPractice(now time.Time) error {
	if r.Practice == nil {
		return errors.New("not a practice run")
	}
	fresh, err := NewPracticeRun(r.ID, r.Build, r.Practice.Mode, now)
	if err != nil {
		return err
	}
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
	case "movement":
		return Input{X: in.X, Y: in.Y}
	case "jump":
		return Input{X: in.X, Y: in.Y, Jump: in.Jump}
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
	if r.Practice.Mode == "jump" {
		complete = complete && r.Stats.Jumps > 0
	}
	if r.Practice.Mode == "combo" {
		complete = r.Practice.Hits >= 3 && r.Combo == 3
	}
	if complete {
		r.Practice.Completed = true
		r.Status = "complete"
		r.event("clear", r.Player.X, r.Player.Y, 0)
	}
}
