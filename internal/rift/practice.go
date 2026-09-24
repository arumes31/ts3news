package rift

import (
	"errors"
	"math"
	"time"
	"ts3news/internal/content"
)

// PracticeState describes an isolated drill; it cannot bank or advance a campaign.
type PracticeState struct {
	ResourceCycles int `json:"resource_cycles,omitempty"`
	UltimateTimed bool `json:"ultimate_timed,omitempty"`
	UltimateWindow bool `json:"ultimate_window,omitempty"`
	FreezeUsed bool `json:"freeze_used,omitempty"`
	FreezeMovement bool `json:"freeze_movement,omitempty"`
	RangedHits int `json:"ranged_hits,omitempty"`
	PerfectGuards int `json:"perfect_guards,omitempty"`
	PickupRadius float64 `json:"pickup_radius,omitempty"`
	TargetHint     string  `json:"target_hint,omitempty"`
	ClassHits      int     `json:"class_hits,omitempty"`
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
	return mode == "pickup" || mode == "resource" || mode == "ultimate" || mode == "ranged" || mode == "perfect_guard" || mode == "skills" || mode == "class" || mode == "boss" || mode == "movement" || mode == "jump" || mode == "combo" || mode == "guard" || mode == "hazard"
}

func NewPracticeRun(id string, build Build, mode string, now time.Time) (*Run, error) {
	if !ValidPracticeMode(mode) {
		return nil, errors.New("unknown practice drill")
	}
	if mode == "ultimate" && (build.Ultimate == nil || build.Ultimate.ID == "") { return nil, errors.New("ultimate practice requires an equipped ultimate") }
	if mode == "ranged" {
        hasProjectile := false
        for _, skill := range append(append([]Skill{}, build.Skills...), build.Signatures...) { hasProjectile = hasProjectile || skill.Reference().Target == "projectile" }
        if build.Ultimate != nil { hasProjectile = hasProjectile || build.Ultimate.Reference().Target == "projectile" }
        if !hasProjectile { return nil, errors.New("ranged practice requires an equipped projectile ability") }
    }
    if mode == "class" || mode == "resource" {
		builder, finisher := false, false
		for _, skill := range build.Signatures {
			builder = builder || skill.Role == "builder"
			finisher = finisher || skill.Role == "finisher"
		}
		if !builder || !finisher {
			return nil, errors.New("class practice requires an equipped builder and finisher")
		}
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
	r.Practice = &PracticeState{PickupRadius: pickupRadius, Mode: mode, GoalX: 900, Arena: Arena{Name: "Practice lane", Obstacles: []Obstacle{}, Hazards: []Hazard{}, Floor: "stone"}}
	r.Floor = "stone"
	r.Enemies = []Actor{}
	for i := range r.EncounterPlan {
		r.EncounterPlan[i] = []Actor{}
	}
	if mode == "pickup" {
		r.Practice.Arena.Name = "Loot pickup demonstration"
		r.Drops = []Drop{{ID: "practice-token-1", X: 360, Y: 410}, {ID: "practice-token-2", X: 580, Y: 350}, {ID: "practice-token-3", X: 800, Y: 480}}
	}
	if mode == "jump" {
		r.Practice.Arena.Obstacles = []Obstacle{{X: 450, Y: 250, W: 70, H: 300}}
	}
	if mode == "resource" || mode == "ultimate" || mode == "combo" || mode == "class" || mode == "skills" || mode == "ranged" {
		r.Enemies = []Actor{{ID: "practice-target", Name: "Training target", Kind: "knight", X: 220, Y: r.Player.Y, HP: 1000000, MaxHP: 1000000, Facing: -1}}
	}
	if mode == "resource" {
		r.Practice.Arena.Name = "Resource management lane"
		r.Enemies[0].Name = "Resource training target"
	}
	if mode == "ultimate" {
		r.Practice.Arena.Name = "Ultimate timing lane"
		r.Enemies[0].Name = "Ultimate timing target"
		r.Player.HP = r.Player.MaxHP * .6
	}
	if mode == "class" {
		r.configureClassTarget()
	}
	if mode == "ranged" {
        r.Enemies[0].X = 650
        r.Enemies[0].Name = "Ranged aim target"
        r.Practice.Arena.Name = "Ranged aiming lane"
    }
    if mode == "skills" {
		r.Practice.Arena.Name = "Skill testing lane"
		r.Player.HP = r.Player.MaxHP * .6
	}
	if mode == "guard" || mode == "perfect_guard" {
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
	fresh.Practice.FreezeMovement = r.Practice.FreezeMovement
	fresh.Practice.FreezeUsed = fresh.Practice.FreezeMovement
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
	case "boss", "class", "skills", "ranged", "ultimate", "resource":
		return in
	case "movement", "pickup":
		return Input{X: in.X, Y: in.Y}
	case "jump", "hazard":
		return Input{X: in.X, Y: in.Y, Jump: in.Jump}
	case "guard", "perfect_guard":
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
	if r.Practice.Mode == "skills" {
		return // Free practice ends only when the player leaves or resets.
	}
	complete := r.Player.X >= r.Practice.GoalX
	if r.Practice.Mode == "pickup" {
		complete = len(r.Drops) == 3
		for _, drop := range r.Drops { complete = complete && drop.Collected }
	}
	if r.Practice.Mode == "resource" {
		complete = r.Practice.ResourceCycles >= 2
	}
	if r.Practice.Mode == "ultimate" {
		r.Practice.UltimateWindow = r.ultimatePracticeWindow()
		complete = r.Practice.UltimateTimed
	}
	if r.Practice.Mode == "boss" {
		complete = len(r.Enemies) == 1 && r.Enemies[0].HP <= 0
	}
	if r.Practice.Mode == "ranged" {
        complete = r.Practice.RangedHits >= 3
    }
    if r.Practice.Mode == "class" {
		complete = r.Practice.ClassHits > 0
	}
	if r.Practice.Mode == "jump" {
		complete = complete && r.Stats.Jumps > 0
	}
	if r.Practice.Mode == "combo" {
		complete = r.Practice.Hits >= 3 && r.Combo == 3
	}
	if r.Practice.Mode == "perfect_guard" {
		complete = r.Practice.PerfectGuards >= 3
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
	case "practice_freeze":
		r.Practice.FreezeMovement = !r.Practice.FreezeMovement
		r.Practice.FreezeUsed = r.Practice.FreezeUsed || r.Practice.FreezeMovement
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
