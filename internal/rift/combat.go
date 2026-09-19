// Package rift implements the server-owned simulation for solo Rift Brawl.
package rift

import (
	"math"
	"strings"
	"time"

	"ts3news/internal/content"
)

const Width = 1600.0

var Rooms = []string{"Mossbound Approach", "The Lantern Court", "Heart of the Ruins"}

type Skill struct {
	Role     string  `json:"role,omitempty"`
	Damage   float64 `json:"damage,omitempty"`
	Heal     float64 `json:"heal,omitempty"`
	Pierce   float64 `json:"pierce,omitempty"`
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Kind     string  `json:"kind"`
	Power    float64 `json:"power"`
	Cost     float64 `json:"cost"`
	Cooldown float64 `json:"cooldown"`
}

type Build struct {
	BaseClass      string    `json:"base_class"`
	ClassName      string    `json:"class_name"`
	Resource       string    `json:"resource"`
	Sequence       string    `json:"sequence"`
	Signatures     []Skill   `json:"signatures"`
	Ultimate       *Skill    `json:"ultimate,omitempty"`
	OwnedUltimates *[]string `json:"owned_ultimates,omitempty"`
	Pets           int       `json:"pets"`
	Relic          bool      `json:"relic"`
	Name           string    `json:"name"`
	Class          string    `json:"class"`
	Level          int       `json:"level"`
	HP             float64   `json:"hp"`
	Damage         float64   `json:"damage"`
	Armor          float64   `json:"armor"`
	Weapon         string    `json:"weapon"`
	Skills         []Skill   `json:"skills"`
	Gear           []string  `json:"gear"`
}

type Actor struct {
	RouteX    float64 `json:"route_x,omitempty"`
	RouteY    float64 `json:"route_y,omitempty"`
	Attacks   int     `json:"attacks,omitempty"`
	ArtKey    string  `json:"art_key,omitempty"`
	Tier      string  `json:"tier,omitempty"`
	Element   string  `json:"element,omitempty"`
	Damage    float64 `json:"damage,omitempty"`
	Armor     float64 `json:"armor,omitempty"`
	Speed     float64 `json:"speed,omitempty"`
	Shot      string  `json:"shot,omitempty"`
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Kind       string  `json:"kind"`
	AttackName string  `json:"attack_name,omitempty"`
	X          float64 `json:"x"`
	Y         float64 `json:"y"`
	HP        float64 `json:"hp"`
	MaxHP     float64 `json:"max_hp"`
	Mana      float64 `json:"mana"`
	Facing    float64 `json:"facing"`
	Cooldown  float64 `json:"cooldown"`
	Windup    float64 `json:"windup"`
	Jump      float64 `json:"jump"`
	Guard     bool    `json:"guard"`
	Pose      string  `json:"pose"`
	PoseTime  float64 `json:"pose_time"`
	Knockdown float64 `json:"knockdown"`
	TargetX   float64 `json:"target_x"`
	TargetY   float64 `json:"target_y"`
}

type Projectile struct {
	Skill   Skill   `json:"skill"`
	Charges int     `json:"charges"`
	Marked  string  `json:"marked"`
	ID      int     `json:"id"`
	X       float64 `json:"x"`
	Y       float64 `json:"y"`
	VX      float64 `json:"vx"`
	VY      float64 `json:"vy"`
	Power   float64 `json:"power"`
	Life    float64 `json:"life"`
	Enemy   bool    `json:"enemy"`
	Kind    string  `json:"kind"`
}

type Drop struct {
	Mission   int           `json:"mission,omitempty"`
	Tier      int           `json:"tier,omitempty"`
	ID        string        `json:"id"`
	X         float64       `json:"x"`
	Y         float64       `json:"y"`
	Gold      int64         `json:"gold"`
	NeedsGear bool          `json:"needs_gear"`
	Gear      *content.Gear `json:"gear,omitempty"`
	Collected bool          `json:"collected"`
	Banked    bool          `json:"banked"`
}

type BankedLoot struct {
	Name   string `json:"name"`
	Rarity int    `json:"rarity"`
}

type Event struct {
	ID    int     `json:"id"`
	Kind  string  `json:"kind"`
	X     float64 `json:"x"`
	Y     float64 `json:"y"`
	Value float64 `json:"value,omitempty"`
}

type Run struct {
	MonsterRecords      map[string]MonsterRecord `json:"monster_records,omitempty"`
	Practice            *PracticeState           `json:"practice,omitempty"`
	ClearStreak         int                      `json:"clear_streak,omitempty"`
	BestClearStreak     int                      `json:"best_clear_streak,omitempty"`
	RoomSplits          [3]*float64              `json:"room_splits"`
	PauseStartedMS      *int64                   `json:"pause_started_ms,omitempty"`
	RoomStartSeconds    *float64                 `json:"room_start_seconds,omitempty"`
	RoomStartHits       *int                     `json:"room_start_hits,omitempty"`
	MissionStartHits    *int                     `json:"mission_start_hits,omitempty"`
	History             map[int]MissionHistory   `json:"mission_history,omitempty"`
	MissionStartSeconds float64                  `json:"mission_start_seconds,omitempty"`
	HistoryActive       bool                     `json:"history_active,omitempty"`
	Stats               CombatStats              `json:"stats"`
	Level               *Level                   `json:"level,omitempty"`
	CompletedLevels     []int                    `json:"completed_levels,omitempty"`
	EncounterPlan       [][]Actor                `json:"encounter_plan,omitempty"`
	Resource            int                      `json:"resource"`
	Marked              string                   `json:"marked"`
	Barrier             float64                  `json:"barrier"`
	BarrierSources      map[string]float64       `json:"barrier_sources,omitempty"`
	Schema              int                      `json:"schema"`
	ID                  string                   `json:"id"`
	StartKey            string                   `json:"start_key"`
	Epoch               string                   `json:"epoch"`
	Revision            int                      `json:"revision"`
	Room                int                      `json:"room"`
	Status              string                   `json:"status"`
	Paused              bool                     `json:"paused"`
	Build               Build                    `json:"build"`
	Player              Actor                    `json:"player"`
	Enemies             []Actor                  `json:"enemies"`
	Projectiles         []Projectile             `json:"projectiles"`
	Drops               []Drop                   `json:"drops"`
	Events              []Event                  `json:"events"`
	SkillTimers         map[string]float64       `json:"skill_timers"`
	Gold                int64                    `json:"gold"`
	BankedAtMS          int64                    `json:"banked_at_ms,omitempty"`
	AttemptHistory      []AttemptRecord          `json:"attempt_history,omitempty"`
	LastClear           *ClearResult             `json:"last_clear,omitempty"`
	PastExpeditions     CareerTotals             `json:"past_expeditions"`
	BankedGold          int64                    `json:"banked_gold"`
	BankedLoot          []BankedLoot             `json:"banked_loot,omitempty"`
	BankedItems         []string                 `json:"banked_items"`
	Clock               float64                  `json:"clock"`
	LastMS              int64                    `json:"last_ms"`
	Counter             int                      `json:"counter"`
	Combo               int                      `json:"combo"`
}

type Input struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Attack bool    `json:"attack"`
	Guard  bool    `json:"guard"`
	Jump   bool    `json:"jump"`
	Skill  string  `json:"skill"`
}

// ValidMovement accepts finite stick axes and legacy digital directions.
func (in Input) ValidMovement() bool {
	return in.X >= -1 && in.X <= 1 && in.Y >= -1 && in.Y <= 1
}

func NewRun(id string, build Build, now time.Time) *Run {
	return NewRunWithCatalog(id, build, now, content.AbyssMobCatalog())
}

func (r *Run) spawnRoom() {
	hits := r.Stats.HitsTaken
	r.RoomStartHits = &hits
	seconds := r.Stats.Seconds
	r.RoomStartSeconds = &seconds
	r.Marked = ""
	for key := range r.SkillTimers {
		if strings.HasPrefix(key, "hazard-") || key == "slowed" {
			delete(r.SkillTimers, key)
		}
	}
	// Old saved runs retain the current room, and use the live catalog on
	// their next room transition when they have no frozen encounter plan.
	if len(r.EncounterPlan) != len(Rooms) {
		r.EncounterPlan = planEncounters(r.ID, content.AbyssMobCatalog())
	}
	r.Enemies = append([]Actor{}, r.EncounterPlan[r.Room]...)
	r.observeRoomMonsters()
	r.Projectiles = []Projectile{}
	r.Player.X = 160
	r.Player.Y = 410
	r.event("area", r.Player.X, r.Player.Y, float64(r.Room))
}

func (r *Run) NextRoom() bool {
	if r.Practice != nil || r.Status != "cleared" || r.Room >= len(Rooms)-1 {
		return false
	}
	r.Room++
	r.Status = "fighting"
	r.Player.HP = math.Min(r.Player.MaxHP, r.Player.HP+r.Player.MaxHP*.25)
	r.Player.Mana = 100
	r.SetPaused(false, time.UnixMilli(r.LastMS))
	r.spawnRoom()
	return true
}

func (r *Run) event(kind string, x, y, value float64) {
	r.Counter++
	r.Events = append(r.Events, Event{r.Counter, kind, x, y, value})
	if len(r.Events) > 40 {
		r.Events = r.Events[len(r.Events)-40:]
	}
}

// Step uses elapsed server time, capped to avoid catch-up damage after a disconnect.
func (r *Run) Step(in Input, now time.Time) {
	if !in.ValidMovement() {
		return
	}
	in = r.practiceInput(in)
	dt := math.Max(0, math.Min(.2, float64(now.UnixMilli()-r.LastMS)/1000))
	r.LastMS = max(r.LastMS, now.UnixMilli())
	if r.Paused || (r.Status != "fighting" && r.Status != "cleared") {
		return
	}
	r.observeRoomMonsters()
	for dt > .000001 {
		slice := math.Min(dt, 1.0/30)
		r.tick(in, slice)
		dt -= slice
		if r.Status == "defeated" {
			break
		}
	}
}

func (r *Run) tick(in Input, dt float64) {
	r.Clock += dt
	if r.Status == "fighting" {
		r.Stats.Seconds += dt
	}
	p := &r.Player
	p.Cooldown = math.Max(0, p.Cooldown-dt)
	p.Jump = math.Max(0, p.Jump-dt)
	p.PoseTime = math.Max(0, p.PoseTime-dt)
	p.Mana = math.Min(100, p.Mana+dt*6)
	for id, remaining := range r.SkillTimers {
		r.SkillTimers[id] = math.Max(0, remaining-dt)
	}
	p.Guard = in.Guard && p.Jump == 0
	speed := 235.0
	if r.SkillTimers["slowed"] > 0 {
		speed *= .6
	}
	if p.Guard {
		speed = 75
	}
	x, y := in.X, in.Y
	length := math.Hypot(x, y)
	if length > 1 {
		x /= length
		y /= length
	}
	r.moveActor(p, x*speed*dt, y*speed*.6*dt, false)
	if x != 0 {
		p.Facing = math.Copysign(1, x)
	}
	if p.PoseTime == 0 {
		p.Pose = "idle"
		if length > 0 {
			p.Pose = "run"
		}
		if p.Guard {
			p.Pose = "guard"
		}
	}
	if in.Jump && p.Jump == 0 && r.SkillTimers["jump"] == 0 {
		r.Stats.Jumps++
		p.Jump = .65
		r.SkillTimers["jump"] = 1.05
		r.event("jump", p.X, p.Y, 0)
	}
	if in.Attack && (in.Skill == "" || !r.canCast(in.Skill)) && p.Cooldown == 0 && !p.Guard {
		r.Stats.Attacks++
		p.Cooldown = .38
		p.Pose = "attack"
		p.PoseTime = .32
		r.Combo = r.Combo%3 + 1
		r.Stats.HighestCombo = max(r.Stats.HighestCombo, r.Combo)
		r.event("slash", p.X+p.Facing*38, p.Y-25, float64(r.Combo))
		for i := range r.Enemies {
			e := &r.Enemies[i]
			if e.HP > 0 && math.Abs(e.Y-p.Y) < 32 && (e.X-p.X)*p.Facing >= -10 && (e.X-p.X)*p.Facing < 95 {
				r.hurtEnemy(i, r.Build.Damage*(1+float64(r.Combo-1)*.2), "hit")
				if r.Combo == 3 && e.HP > 0 && !EnemyTraining(e.Kind).ResistsKnockdown {
					e.Knockdown = .55
					e.Windup = 0
					if r.Practice == nil || e.ID != "practice-target" {
						e.X = clamp(e.X+p.Facing*35, 35, Width-35)
					}
					r.event("knockdown", e.X, e.Y, 0)
				}
			}
		}
	}
	if in.Skill != "" && !p.Guard {
		r.cast(in.Skill)
	}
	if r.Status == "fighting" {
		r.hazardTick()
		for i := range r.Enemies {
			r.enemyTick(i, dt)
		}
	}
	shots := r.Projectiles[:0]
	for _, shot := range r.Projectiles {
		shot.Life -= dt
		shot.X += shot.VX * dt
		shot.Y += shot.VY * dt
		if shot.Life <= 0 || shot.X < 0 || shot.X > Width {
			continue
		}
		hit := false
		if shot.Enemy {
			if math.Abs(shot.X-p.X) < 25 && math.Abs(shot.Y-p.Y) < 23 && p.Jump < .1 {
				r.hurtPlayer(shot.Power, shot.X, shot.Y)
				hit = true
			}
		} else {
			for i, e := range r.Enemies {
				if ref := shot.Skill.Reference(); e.HP > 0 && math.Abs(shot.X-e.X) < ref.Horizontal && math.Abs(shot.Y-e.Y) < ref.Depth {
					r.skillHit(i, shot.Power, shot.Skill, shot.Charges, shot.Marked)
					hit = true
					break
				}
			}
		}
		if !hit {
			shots = append(shots, shot)
		}
	}
	r.Projectiles = shots
	for i := range r.Drops {
		d := &r.Drops[i]
		if !d.Collected && math.Hypot(d.X-p.X, d.Y-p.Y) < 65 {
			d.Collected = true
			r.Gold += d.Gold
			r.event("pickup", d.X, d.Y, float64(d.Gold))
		}
	}
	if p.HP <= 0 {
		r.Status = "defeated"
		r.finishMissionHistory("defeated")
		r.Gold = 0
		r.Drops = []Drop{}
		r.Projectiles = []Projectile{}
		r.event("defeat", p.X, p.Y, 0)
		return
	}
	alive := 0
	for _, e := range r.Enemies {
		if e.HP > 0 {
			alive++
		}
	}
	if r.Practice != nil {
		r.practiceTick()
	}
	if r.Practice == nil && alive == 0 && r.Status == "fighting" {
		r.Stats.RoomsCleared++
		r.recordFlawlessRoom()
		if r.RoomStartSeconds != nil && r.RoomSplits[r.Room] == nil {
			seconds := r.Stats.Seconds - *r.RoomStartSeconds
			if seconds >= 0 {
				r.RoomSplits[r.Room] = &seconds
			}
		}
		// A secured room sweeps remaining drops into the bag before presenting
		// its checkpoint, so displayed rewards agree with the banking receipt.
		for i := range r.Drops {
			d := &r.Drops[i]
			if !d.Collected {
				d.Collected = true
				r.Gold += d.Gold
				r.event("pickup", d.X, d.Y, float64(d.Gold))
			}
		}
		r.Status = "cleared"
		r.Projectiles = []Projectile{}
		r.event("clear", p.X, p.Y, 0)
	}
}

func (r *Run) canCast(id string) bool {
	if r.SkillTimers[id] > 0 || r.Player.Cooldown > 0 {
		return false
	}
	for _, skill := range r.abilities() {
		if skill.ID == id {
			return r.Player.Mana >= skill.Cost
		}
	}
	return false
}

func (r *Run) cast(id string) {
	if r.SkillTimers[id] > 0 || r.Player.Cooldown > 0 {
		return
	}
	for _, skill := range r.abilities() {
		if skill.ID != id || r.Player.Mana < skill.Cost {
			continue
		}
		p := &r.Player
		p.Mana -= skill.Cost
		r.Stats.ManaSpent += skill.Cost
		r.Stats.SkillsCast++
		if r.Stats.SkillUses == nil {
			r.Stats.SkillUses = map[string]int{}
		}
		r.Stats.SkillUses[id]++
		if r.Stats.SkillMana == nil {
			r.Stats.SkillMana = map[string]float64{}
		}
		r.Stats.SkillMana[id] += skill.Cost
		p.Cooldown = .35
		p.Pose = "cast"
		p.PoseTime = .4
		r.SkillTimers[id] = skill.Cooldown
		charges, marked := r.classCast(skill)
		r.event(skill.Kind, p.X+p.Facing*35, p.Y-35, 0)
		base := skill.Damage
		if base <= 0 {
			base = r.Build.Damage
		}
		power := base * skill.Power * (1 + float64(charges)*.2)
		if skill.Heal > 0 {
			r.healPlayerBySkill(p.MaxHP*skill.Heal, skill.ID)
		}
		switch skill.Kind {
		case "shield":
			r.addBarrier(25+r.Build.Armor*4, skill.ID)
		case "heal":
			if skill.Heal == 0 {
				r.healPlayerBySkill(p.MaxHP*.15, skill.ID)
			}
		case "slash", "quake", "ultimate":
			ref := skill.Reference()
			for i, e := range r.Enemies {
				if e.HP > 0 && math.Abs(e.X-p.X) < ref.Horizontal && math.Abs(e.Y-p.Y) < ref.Depth {
					r.skillHit(i, power, skill, charges, marked)
				}
			}
		default:
			r.Projectiles = append(r.Projectiles, Projectile{ID: r.Counter, X: p.X + p.Facing*35, Y: p.Y, VX: p.Facing * 530, Power: power, Life: 2.5, Kind: skill.Kind, Skill: skill, Charges: charges, Marked: marked})
		}
		return
	}
}

func (r *Run) hurtEnemy(i int, damage float64, effect string) {
	r.hurtEnemyPiercing(i, damage, effect, 0)
}

func (r *Run) hurtEnemyPiercing(i int, damage float64, effect string, pierce float64) {
	e := &r.Enemies[i]
	if e.HP <= 0 {
		return
	}
	armor := e.Armor
	if e.ArtKey == "" && e.Kind == "knight" {
		armor = .3
	}
	if e.ArtKey == "" && e.Kind == "boss" {
		armor = .15
	}
	damage *= 1 - armor*(1-clamp(pierce, 0, 1))
	damage = math.Min(e.HP, math.Max(0, damage))
	e.HP = math.Max(0, e.HP-damage)
	r.Stats.DamageDealt += damage
	r.Stats.LargestHit = math.Max(r.Stats.LargestHit, damage)
	e.Pose = "hit"
	e.PoseTime = .2
	r.event(effect, e.X, e.Y-30, damage)
	if r.Practice != nil {
		if damage > 0 && e.ID == "practice-target" && effect == "hit" {
			r.Practice.Hits++
		}
		e.HP = e.MaxHP
		return
	}
	if e.HP == 0 {
		r.recordMonsterDefeat(*e)
		r.Stats.Kills++
		if e.Kind == "treasure" {
			r.Stats.TreasureGoblins++
		}
		if e.Kind == "boss" {
			r.Stats.Bosses++
		}
		r.event(e.Kind+"_death", e.X, e.Y, 0)
		mission := 1
		if r.Level != nil {
			mission = r.Level.ID
		}
		r.Drops = append(r.Drops, Drop{Mission: mission, Tier: r.Room + 1, ID: e.ID, X: e.X, Y: e.Y, Gold: int64(15 * (r.Room + 1)), NeedsGear: e.Kind == "boss" || e.Kind == "knight" || e.Kind == "treasure" || i == 0})
	}
}

func (r *Run) hurtPlayer(damage, x, y float64) {
	p := &r.Player
	if p.HP <= 0 {
		return
	}
	incoming := damage
	damage = math.Max(2, damage-r.Build.Armor*.4)
	r.Stats.ArmorBlocked += math.Max(0, incoming-damage)
	kind := "hurt"
	if p.Guard && (x-p.X)*p.Facing >= 0 {
		r.Stats.Guards++
		r.Stats.GuardBlocked += damage * .82
		damage *= .18
		kind = "block"
	}
	if r.Barrier > 0 {
		absorbed := r.absorbBarrier(damage)
		damage -= absorbed
		kind = "block"
	}
	damage = math.Min(p.HP, damage)
	p.HP = math.Max(0, p.HP-damage)
	r.Stats.DamageTaken += damage
	if damage > 0 {
		r.Stats.HitsTaken++
	}
	p.Pose = "hit"
	p.PoseTime = .18
	r.event(kind, p.X, p.Y-30, damage)
}

func (r *Run) enemyTick(i int, dt float64) {
	if r.Practice != nil && r.Enemies[i].ID == "practice-target" {
		e := &r.Enemies[i]
		e.PoseTime = math.Max(0, e.PoseTime-dt)
		e.Knockdown = math.Max(0, e.Knockdown-dt)
		if e.PoseTime == 0 && e.Knockdown == 0 {
			e.Pose = "idle"
		}
		return
	}
	e := &r.Enemies[i]
	if e.HP <= 0 {
		return
	}
	e.Cooldown = math.Max(0, e.Cooldown-dt)
	e.PoseTime = math.Max(0, e.PoseTime-dt)
	if e.Knockdown > 0 {
		e.Knockdown = math.Max(0, e.Knockdown-dt)
		e.Pose = "knockdown"
		return
	}
	p := &r.Player
	dx, dy := p.X-e.X, p.Y-e.Y
	if e.Kind == "treasure" && math.Abs(dx) < 240 && e.X > 55 && e.X < Width-55 {
		e.Facing = -math.Copysign(1, dx)
		r.moveActor(e, -math.Copysign(e.Speed*dt, dx), 0, true)
		e.Pose = "run"
		return
	}
	if dx != 0 {
		e.Facing = math.Copysign(1, dx)
	}
	if e.Windup > 0 {
		e.Windup = math.Max(0, e.Windup-dt)
		if e.Windup == 0 {
			e.AttackName = ""
			e.Attacks++
			e.Pose = "attack"
			e.PoseTime = .4
			e.Cooldown = 1.6
			if e.Kind == "archer" || e.Kind == "boss" && e.ArtKey != "" && e.Attacks%2 == 0 {
				distance := math.Max(1, math.Hypot(dx, dy))
				r.Counter++
				shot := e.Shot
				if shot == "" {
					shot = "arrow"
				}
				power := e.Damage
				if power <= 0 {
					power = 18
				}
				r.Projectiles = append(r.Projectiles, Projectile{ID: r.Counter, X: e.X, Y: e.Y, VX: dx / distance * 300, VY: dy / distance * 300, Power: power, Enemy: true, Life: 4, Kind: shot})
				r.event(shot, e.X, e.Y-30, 0)
			} else if e.Kind == "boss" {
				r.event("slam", e.TargetX, e.TargetY, 0)
				if math.Abs(p.X-e.TargetX) < 125 && math.Abs(p.Y-e.TargetY) < 62 && p.Jump < .1 {
					r.hurtPlayer(max(32, e.Damage*1.4), e.X, e.Y)
				}
				e.Cooldown = 2.3
			} else {
				r.event(e.Kind+"_attack", e.X, e.Y, 0)
				if math.Abs(dx) < 85 && math.Abs(dy) < 33 && p.Jump < .25 {
					power := e.Damage
					if power <= 0 {
						power = 19 + float64(r.Room)*4
					}
					r.hurtPlayer(power, e.X, e.Y)
				}
			}
		}
		return
	}
	rangeX := 65.0
	if e.Kind == "archer" {
		rangeX = 430
	}
	if e.Kind == "boss" {
		rangeX = 190
	}
	if math.Abs(dx) > rangeX || math.Abs(dy) > 24 {
		speed := 80.0
		if e.Speed > 0 {
			speed = e.Speed
		}
		if e.Kind == "goblin" && e.Speed == 0 {
			speed = 115
		}
		r.moveActor(e, math.Copysign(math.Min(math.Abs(dx), speed*dt), dx), math.Copysign(math.Min(math.Abs(dy), speed*.6*dt), dy), true)
		if e.PoseTime == 0 {
			e.Pose = "run"
		}
	} else if e.Cooldown == 0 {
		e.Windup = EnemyTraining(e.Kind).WindupSeconds
		e.Pose = "windup"
		e.TargetX = p.X
		e.TargetY = p.Y
		if e.Kind == "boss" {
			r.event("boss_roar", e.X, e.Y, 0)
			e.AttackName = r.bossAttackName(e)
		}
	} else if e.PoseTime == 0 {
		e.Pose = "idle"
	}
}

func (r *Run) bossAttackName(e *Actor) string {
	if e.Kind != "boss" {
		return ""
	}
	if e.ArtKey != "" && (e.Attacks+1)%2 == 0 {
		switch e.Shot {
		case "fire":
			return "Cinder Volley"
		case "ice":
			return "Glacial Shards"
		case "void":
			return "Void Rift"
		case "radiant":
			return "Solar Flare"
		case "poison":
			return "Toxic Spores"
		default:
			return "Aimed Volley"
		}
	}
	region := 0
	if r.Level != nil {
		region = r.Level.Region
	}
	switch region {
	case 0:
		return "Mossbound Slam"
	case 1:
		return "Molten Slam"
	case 2:
		return "Glacial Slam"
	case 3:
		return "Thunder Slam"
	case 4:
		return "Venom Slam"
	case 5:
		return "Abyssal Wave"
	case 6:
		return "Brutal Cleave"
	case 7:
		return "Spectral Slam"
	case 8:
		return "Void Cataclysm"
	case 9:
		return "Obsidian Slam"
	default:
		return "Ground Slam"
	}
}

func clamp(value, low, high float64) float64 { return math.Max(low, math.Min(high, value)) }
