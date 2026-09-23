// Package rift implements the server-owned simulation for solo Rift Brawl.
package rift

import (
	"fmt"
	"math"
	"slices"
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
	Elevation     float64 `json:"elevation,omitempty"`
	LedgeRoute     string  `json:"ledge_route,omitempty"`
	LedgeRouteX    float64 `json:"ledge_route_x,omitempty"`
	LedgeRouteSide bool    `json:"ledge_route_side,omitempty"`
	WeakPoint float64 `json:"weak_point,omitempty"`
	Fleeing   bool    `json:"fleeing,omitempty"`
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
	RecoilX   float64 `json:"recoil_x,omitempty"`
	TargetX   float64 `json:"target_x"`
	TargetY   float64 `json:"target_y"`
	Phase     int     `json:"phase,omitempty"`
}

// HurtCue returns the creature-family hurt audio cue identifier.
func (a Actor) HurtCue() string {
	switch a.Kind {
	case "totem", "generator", "cage":
		return a.Kind + "_hurt"
	case "goblin", "knight", "archer", "treasure", "boss", "wolf", "spore":
		return a.Kind + "_hurt"
	default:
		return "hurt"
	}
}

// DeathCue returns the creature-family death audio cue identifier.
func (a Actor) DeathCue() string {
	switch a.Kind {
	case "goblin", "knight", "archer", "treasure", "boss", "wolf", "spore":
		return a.Kind + "_death"
	default:
		return "goblin_death"
	}
}

type Projectile struct {
	Elevation float64 `json:"elevation,omitempty"`
	OwnerID string  `json:"owner_id,omitempty"`
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
	Elevation float64 `json:"elevation,omitempty"`
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
	Elevation float64 `json:"elevation,omitempty"`
	ActorName string  `json:"actor_name,omitempty"`
	ID        int     `json:"id"`
	Kind      string  `json:"kind"`
	X         float64 `json:"x"`
	Y         float64 `json:"y"`
	Value     float64 `json:"value,omitempty"`
}

// EncounterBossState snapshots a boss at the end of an encounter.
type EncounterBossState struct {
	Name  string  `json:"name"`
	HP    float64 `json:"hp"`
	MaxHP float64 `json:"max_hp"`
	Phase int     `json:"phase"`
}

// EncounterSummary preserves an accessible structured summary of the last encounter.
type EncounterSummary struct {
	DefeatedByHazard *HazardDefeat `json:"defeated_by_hazard,omitempty"`
	HazardDamageTaken float64 `json:"hazard_damage_taken"`
	EnemyDamageTaken float64 `json:"enemy_damage_taken"`
	Bosses          []EncounterBossState `json:"bosses,omitempty"`
	DefeatedByBoss  string               `json:"defeated_by_boss,omitempty"`
	TreasureEscaped int                  `json:"treasure_escaped,omitempty"`
	Mission         int                  `json:"mission"`
	MissionName     string               `json:"mission_name"`
	Room            int                  `json:"room"`
	RoomName        string               `json:"room_name"`
	Outcome         string               `json:"outcome"` // "cleared", "defeated", "completed"
	Seconds         float64              `json:"seconds"`
	PlayerHP        float64              `json:"player_hp"`
	PlayerMaxHP     float64              `json:"player_max_hp"`
	Enemies         int                  `json:"enemies"`
	BossEncounter   bool                 `json:"boss_encounter"`
	BossName        string               `json:"boss_name,omitempty"`
	DamageDealt     float64              `json:"damage_dealt"`
	DamageTaken     float64              `json:"damage_taken"`
	HitsTaken       int                  `json:"hits_taken"`
	GuardBlocked    float64              `json:"guard_blocked"`
	BarrierBlocked  float64              `json:"barrier_blocked"`
	Healing         float64              `json:"healing"`
	GoldGained      int64                `json:"gold_gained"`
	LootItems       int                  `json:"loot_items"`
}

// RoomBaseline tracks starting metrics at the beginning of each room to calculate encounter deltas.
type RoomBaseline struct {
	HazardDamageTaken float64 `json:"hazard_damage_taken"`
	EnemyDamageTaken float64 `json:"enemy_damage_taken"`
	Seconds        float64 `json:"seconds"`
	DamageDealt    float64 `json:"damage_dealt"`
	DamageTaken    float64 `json:"damage_taken"`
	HitsTaken      int     `json:"hits_taken"`
	GuardBlocked   float64 `json:"guard_blocked"`
	BarrierBlocked float64 `json:"barrier_blocked"`
	Healing        float64 `json:"healing"`
	Kills          int     `json:"kills"`
	Bosses         int     `json:"bosses"`
	Gold           int64   `json:"gold"`
}

type Run struct {
	DefeatedByHazard *HazardDefeat `json:"defeated_by_hazard,omitempty"`
 RoomObjective *RoomObjective `json:"room_objective,omitempty"`
 ObjectiveHistory map[string]map[string]int `json:"objective_history,omitempty"`
 LastObjectives *MissionObjectives `json:"last_objectives,omitempty"`
 Objectives *MissionObjectives `json:"objectives,omitempty"`
	DefeatedByBoss      string                   `json:"defeated_by_boss,omitempty"`
	MonsterRecords      map[string]MonsterRecord `json:"monster_records,omitempty"`
	Practice            *PracticeState           `json:"practice,omitempty"`
	ClearStreak         int                      `json:"clear_streak,omitempty"`
	BestClearStreak     int                      `json:"best_clear_streak,omitempty"`
	RoomSplits          [3]*float64              `json:"room_splits"`
	PauseStartedMS      *int64                   `json:"pause_started_ms,omitempty"`
	RoomStartSeconds    *float64                 `json:"room_start_seconds,omitempty"`
	RoomStartHits       *int                     `json:"room_start_hits,omitempty"`
	RoomBaseline        *RoomBaseline            `json:"room_baseline,omitempty"`
	LastEncounter       *EncounterSummary        `json:"last_encounter,omitempty"`
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
	Catchup             bool                     `json:"catchup,omitempty"`
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
	SavedAtMS           int64                    `json:"saved_at_ms,omitempty"`
	AttemptHistory      []AttemptRecord          `json:"attempt_history,omitempty"`
	LastClear           *ClearResult             `json:"last_clear,omitempty"`
	PastExpeditions     CareerTotals             `json:"past_expeditions"`
	BankedObjectiveGold int64 `json:"banked_objective_gold,omitempty"`
	BankedGold          int64                    `json:"banked_gold"`
	BankedLoot          []BankedLoot             `json:"banked_loot,omitempty"`
	BankedItems         []string                 `json:"banked_items"`
	Clock               float64                  `json:"clock"`
	LastMS              int64                    `json:"last_ms"`
	Counter             int                      `json:"counter"`
	Combo               int                      `json:"combo"`
	ComboTime           float64                  `json:"combo_time,omitempty"`
	Floor               string                   `json:"floor,omitempty"`
	jumpAir             float64
	jumpDist            float64
	heavyRecovery       float64
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
	r.DefeatedByBoss = ""
	r.DefeatedByHazard = nil
	hits := r.Stats.HitsTaken
	r.RoomStartHits = &hits
	seconds := r.Stats.Seconds
	r.RoomStartSeconds = &seconds
	r.RoomBaseline = &RoomBaseline{
		Seconds:        r.Stats.Seconds,
		DamageDealt:    r.Stats.DamageDealt,
		DamageTaken:    r.Stats.DamageTaken,
		HazardDamageTaken: r.Stats.HazardDamageTaken,
		EnemyDamageTaken: r.Stats.EnemyDamageTaken,
		HitsTaken:      r.Stats.HitsTaken,
		GuardBlocked:   r.Stats.GuardBlocked,
		BarrierBlocked: r.Stats.BarrierBlocked,
		Healing:        r.Stats.Healing,
		Kills:          r.Stats.Kills,
		Bosses:         r.Stats.Bosses,
		Gold:           r.Gold,
	}
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
	r.Arena().settleEnemySpawns(r.Enemies)
	for i := range r.Enemies {
		// Give arrivals time to read the arena before ranged attacks begin.
		// Keep the stable archer offsets so the opening volley stays staggered.
		if r.Enemies[i].Kind == "archer" || r.Enemies[i].Kind == "boss" {
			r.Enemies[i].Cooldown = math.Max(r.Enemies[i].Cooldown, 1.5+rangedCooldownOffset(&r.Enemies[i]))
		}
	}
	r.beginRoomObjective()
	for i := range r.Enemies {
		r.Enemies[i].Elevation = r.Arena().Elevation(r.Enemies[i].X, r.Enemies[i].Y)
	}
	r.observeRoomMonsters()
	r.Projectiles = []Projectile{}
	r.Player.X = 160
	r.Player.Y = 410
	r.Player.Elevation = r.Arena().Elevation(r.Player.X, r.Player.Y)
	r.heavyRecovery = 0
	r.Floor = r.FloorMaterial()
	r.event("area", r.Player.X, r.Player.Y, float64(r.Room))
}

func (r *Run) RecordEncounterSummary(outcome string) {
	missionID := 1
	missionName := "Mossbound Ruins"
	if r.Level != nil {
		missionID = r.Level.ID
		if r.Level.Name != "" {
			missionName = r.Level.Name
		}
	}
	roomName := fmt.Sprintf("Tier %d", r.Room+1)
	if r.Level != nil && r.Room >= 0 && r.Room < len(r.Level.Rooms) {
		roomName = r.Level.Rooms[r.Room].Name
	} else if r.Room >= 0 && r.Room < len(Rooms) {
		roomName = Rooms[r.Room]
	}

	bossEncounter := false
	bossName := ""
	if r.Room >= 0 && r.Room < len(r.EncounterPlan) {
		for _, a := range r.EncounterPlan[r.Room] {
			if a.Kind == "boss" {
				bossEncounter = true
				bossName = a.Name
				break
			}
		}
	}

	seconds := 0.0
	if r.RoomStartSeconds != nil {
		seconds = max(0, r.Stats.Seconds-*r.RoomStartSeconds)
	}
	damageDealt := r.Stats.DamageDealt
	damageTaken := r.Stats.DamageTaken
	hazardDamageTaken, enemyDamageTaken := r.Stats.HazardDamageTaken, r.Stats.EnemyDamageTaken
	hitsTaken := r.Stats.HitsTaken
	guardBlocked := r.Stats.GuardBlocked
	barrierBlocked := r.Stats.BarrierBlocked
	healing := r.Stats.Healing
	kills := r.Stats.Kills
	goldGained := r.Gold

	if r.RoomBaseline != nil {
		damageDealt = max(0, damageDealt-r.RoomBaseline.DamageDealt)
		damageTaken = max(0, damageTaken-r.RoomBaseline.DamageTaken)
		hazardDamageTaken = max(0, hazardDamageTaken-r.RoomBaseline.HazardDamageTaken)
		enemyDamageTaken = max(0, enemyDamageTaken-r.RoomBaseline.EnemyDamageTaken)
		hitsTaken = max(0, hitsTaken-r.RoomBaseline.HitsTaken)
		guardBlocked = max(0, guardBlocked-r.RoomBaseline.GuardBlocked)
		barrierBlocked = max(0, barrierBlocked-r.RoomBaseline.BarrierBlocked)
		healing = max(0, healing-r.RoomBaseline.Healing)
		kills = max(0, kills-r.RoomBaseline.Kills)
		goldGained = max(0, goldGained-r.RoomBaseline.Gold)
	}

	lootItems := 0
	for _, d := range r.Drops {
		if d.Collected && d.Gear != nil {
			lootItems++
		}
	}

	enemiesCount := 0
	if r.Room >= 0 && r.Room < len(r.EncounterPlan) {
		enemiesCount = len(r.EncounterPlan[r.Room])
	}
	if kills > enemiesCount && enemiesCount > 0 {
		enemiesCount = kills
	}

	var bosses []EncounterBossState
	treasureEscaped := 0
	for _, e := range r.Enemies {
		if e.Kind == "boss" {
			bosses = append(bosses, EncounterBossState{Name: e.Name, HP: e.HP, MaxHP: e.MaxHP, Phase: max(1, e.Phase)})
		}
		if e.Kind == "treasure" && e.Pose == "escape" {
			treasureEscaped++
		}
	}

	r.LastEncounter = &EncounterSummary{
		Bosses:          bosses,
		DefeatedByBoss:  r.DefeatedByBoss,
		DefeatedByHazard: r.DefeatedByHazard,
		TreasureEscaped: treasureEscaped,
		Mission:         missionID,
		MissionName:     missionName,
		Room:            r.Room,
		RoomName:        roomName,
		Outcome:         outcome,
		Seconds:         seconds,
		PlayerHP:        r.Player.HP,
		PlayerMaxHP:     r.Player.MaxHP,
		Enemies:         kills,
		BossEncounter:   bossEncounter,
		BossName:        bossName,
		DamageDealt:     damageDealt,
		DamageTaken:     damageTaken,
		HazardDamageTaken: hazardDamageTaken,
		EnemyDamageTaken: enemyDamageTaken,
		HitsTaken:       hitsTaken,
		GuardBlocked:    guardBlocked,
		BarrierBlocked:  barrierBlocked,
		Healing:         healing,
		GoldGained:      goldGained,
		LootItems:       lootItems,
	}
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
	r.eventAtHeight(kind, x, y, value, r.Arena().Elevation(x, y))
}

// eventAtHeight preserves source height when visual Y is offset from its feet.
func (r *Run) eventAtHeight(kind string, x, y, value, elevation float64) {
	r.Counter++
	r.Events = append(r.Events, Event{ID: r.Counter, Kind: kind, X: x, Y: y, Value: value, Elevation: elevation})
	if len(r.Events) > 40 {
		r.Events = r.Events[len(r.Events)-40:]
	}
}

// Land emits a landing event with the specified intensity.
func (r *Run) Land(intensity float64) {
	r.event("land", r.Player.X, r.Player.Y, intensity)
}

// Step uses elapsed server time, capped to avoid catch-up damage after a disconnect.
func (r *Run) Step(in Input, now time.Time) {
	if !in.ValidMovement() {
		return
	}
	in = r.practiceInput(in)
	dt := math.Max(0, math.Min(.2, float64(now.UnixMilli()-r.LastMS)/1000))
	r.Catchup = dt > 1.0/15
	r.LastMS = max(r.LastMS, now.UnixMilli())
	r.SavedAtMS = r.LastMS
	r.Floor = r.FloorMaterial()
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

const (
	comboWindow            = 1.2
	basicMeleeForwardReach = 95.0
	basicMeleeRearOverlap  = 10.0
	basicMeleeLaneReach    = 32.0
)

func inBasicMeleeRange(attacker, target *Actor) bool {
	forward := (target.X - attacker.X) * attacker.Facing
	// The small rear overlap covers touching sprites, not attacks behind the fighter.
	return forward >= -basicMeleeRearOverlap && forward < basicMeleeForwardReach &&
		math.Abs(target.Y-attacker.Y) < basicMeleeLaneReach
}

func (r *Run) tick(in Input, dt float64) {
 defer r.UpdateObjectives()
	if r.Combo > 0 {
		// Older saves have a partial combo but no remaining-time field.
		if r.ComboTime <= 0 {
			r.ComboTime = comboWindow
		}
		r.ComboTime = math.Max(0, r.ComboTime-dt)
		if r.ComboTime < .000001 {
			r.Combo = 0
			r.ComboTime = 0
		}
	}
	r.Clock += dt
	if r.Status == "fighting" {
		r.Stats.Seconds += dt
	}
	p := &r.Player
	p.Cooldown = math.Max(0, p.Cooldown-dt)
	wasJumping := p.Jump > 0
	p.Jump = math.Max(0, p.Jump-dt)
	p.PoseTime = math.Max(0, p.PoseTime-dt)
	if p.PoseTime < 0.0001 {
		p.PoseTime = 0
	}
	if p.PoseTime == 0 {
		p.RecoilX = 0
	} else if p.RecoilX != 0 {
		p.RecoilX = math.Copysign(math.Max(0, math.Abs(p.RecoilX)-50*dt), p.RecoilX)
	}
	p.Mana = math.Min(100, p.Mana+dt*6)
	wasSlowed := r.SkillTimers != nil && r.SkillTimers["slowed"] > 0
	for id, remaining := range r.SkillTimers {
		val := math.Max(0, remaining-dt)
		if val < 0.0001 {
			val = 0
		}
		r.SkillTimers[id] = val
	}
	if wasSlowed && r.SkillTimers["slowed"] == 0 {
		r.eventAtHeight("thaw", p.X, p.Y-25, 0, p.Elevation)
	}
	wasGuarding := p.Guard
	p.Guard = in.Guard && p.Jump == 0
	if p.Guard && !wasGuarding {
		if r.SkillTimers == nil {
			r.SkillTimers = map[string]float64{}
		}
		r.SkillTimers["perfect_guard"] = .22
	} else if !p.Guard && r.SkillTimers != nil {
		delete(r.SkillTimers, "perfect_guard")
	}
	speed := 235.0
	if r.SkillTimers["slowed"] > 0 {
		speed *= .6
	}
	if p.Guard {
		speed = 75
	}
	if r.RoomObjective != nil && r.RoomObjective.Kind == "carry_relic" && r.RoomObjective.Carrying {
		speed *= .7
	}
	x, y := in.X, in.Y
	length := math.Hypot(x, y)
	if length > 1 {
		x /= length
		y /= length
	}
	if wasJumping {
		r.jumpAir += dt
		r.jumpDist += length * dt
		if p.Jump == 0 {
			intensity := 0.35
			if p.Knockdown > 0 {
				intensity = 1.0
			} else if r.jumpDist >= 0.45 {
				intensity = 0.85
			} else if r.jumpDist >= 0.15 {
				intensity = 0.65
			}
			r.event("land", p.X, p.Y, intensity)
			if p.Knockdown == 0 && p.HP > 0 && p.Pose != "attack" && p.Pose != "cast" && p.Pose != "hit" && p.Pose != "recovery" && p.Pose != "ultimate_anticipation" {
				p.Pose = "land"
				p.PoseTime = 0.14
			}
		}
	}
	r.moveActor(p, x*speed*dt, y*speed*.6*dt, false)
	if x != 0 {
		p.Facing = math.Copysign(1, x)
	}
	if p.PoseTime == 0 {
		if r.heavyRecovery > 0 && p.HP > 0 && p.Knockdown == 0 && (p.Pose == "cast" || p.Pose == "ultimate_anticipation") {
			p.Pose = "recovery"
			p.PoseTime = r.heavyRecovery
			r.heavyRecovery = 0
			r.eventAtHeight("heavy_recovery", p.X, p.Y-35, 0, p.Elevation)
		} else {
			p.Pose = "idle"
			if length > 0 {
				p.Pose = "run"
			}
			if p.Guard {
				if length > 0 {
					p.Pose = "guard_walk"
				} else {
					p.Pose = "guard"
				}
			}
			if (r.Status == "cleared" || r.Status == "complete") && r.Room == len(Rooms)-1 && length == 0 && !p.Guard {
				p.Pose = "victory"
			}
		}
	} else if length > 0 && p.Pose == "victory" {
		p.Pose = "run"
		p.PoseTime = 0
	}
	if in.Jump && p.Jump == 0 && r.SkillTimers["jump"] == 0 && p.Pose != "recovery" && p.Pose != "ultimate_anticipation" {
		r.Stats.Jumps++
		p.Jump = .65
		p.Pose = "jump"
		p.PoseTime = .65
		r.SkillTimers["jump"] = 1.05
		r.jumpAir = 0
		r.jumpDist = 0
		r.event("jump", p.X, p.Y, 0)
	}
	if in.Attack && (in.Skill == "" || !r.canCast(in.Skill)) && p.Cooldown == 0 && !p.Guard && p.Pose != "recovery" {
		r.Stats.Attacks++
		p.Cooldown = .38
		p.Pose = "attack"
		p.PoseTime = .32
		r.Combo = r.Combo%3 + 1
		r.ComboTime = comboWindow
		r.Stats.HighestCombo = max(r.Stats.HighestCombo, r.Combo)
		r.eventAtHeight("slash", p.X+p.Facing*38, p.Y-25, float64(r.Combo), p.Elevation)
		for i := range r.Enemies {
			e := &r.Enemies[i]
			if e.HP > 0 && inBasicMeleeRange(p, e) && r.clearMeleePath(p, e) {
				r.hurtEnemy(i, r.Build.Damage*(1+float64(r.Combo-1)*.2), "hit_"+r.WeaponFamily())
				if e.HP == 0 && !e.isObjectiveProp() && p.Jump > .1 && r.Practice == nil {
					r.Stats.AerialFinishes++
				}
				if r.Combo == 3 {
					r.eventAtHeight("third_strike", e.X, e.Y-25, float64(r.Combo), e.Elevation)
					if e.HP > 0 {
						if !EnemyTraining(e.Kind).ResistsKnockdown {
							e.Knockdown = .55
							e.Windup = 0
							if r.Practice == nil || e.ID != "practice-target" {
								direction := p.Facing
								if e.X != p.X {
									direction = math.Copysign(1, e.X-p.X)
								}
								r.knockbackActor(e, direction*35, 0)
							}
							r.event("knockdown", e.X, e.Y, 0)
						} else if e.Kind == "boss" {
							e.Pose = "stagger"
							e.PoseTime = .45
							e.Windup = 0
							e.Cooldown = math.Max(e.Cooldown, 0.8)
							r.eventAtHeight("boss_stagger", e.X, e.Y-30, 0, e.Elevation)
						}
					}
				}
			}
		}
		r.attackTerrainCover(r.Build.Damage * (1 + float64(r.Combo-1)*.2))
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
		if r.deadBossProjectile(shot) {
			continue
		}
		fromX, fromY := shot.X, shot.Y
		shot.Life -= dt
		shot.X += shot.VX * dt
		shot.Y += shot.VY * dt
		if shot.Life <= 0 || shot.X < 0 || shot.X > Width {
			r.eventAtHeight("projectile_expire", shot.X, shot.Y, 0, shot.Elevation)
			continue
		}
		impact, coverIndex := r.projectileCoverImpact(fromX, fromY, shot.X, shot.Y)
		if impact <= 1 {
			r.damageTerrainCover(coverIndex, shot.Power)
			r.eventAtHeight("projectile_impact", fromX+(shot.X-fromX)*impact, fromY+(shot.Y-fromY)*impact, 0, shot.Elevation)
			continue
		}
		hit := false
		if shot.Enemy {
			if math.Abs(shot.X-p.X) < projectilePlayerRadiusX && math.Abs(shot.Y-p.Y) < projectilePlayerRadiusY {
				if p.Jump < .1 {
					r.hurtPlayerFromEnemy(shot.Power, shot.X, shot.Y, shot.OwnerID)
					hit = true
				} else {
					r.recordDodge()
					hit = true
				}
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
		if hit {
			r.eventAtHeight("projectile_impact", shot.X, shot.Y, 0, shot.Elevation)
		} else {
			shots = append(shots, shot)
		}
	}
	r.Projectiles = slices.DeleteFunc(shots, r.deadBossProjectile)
	for i := range r.Drops {
		d := &r.Drops[i]
		if !d.Collected && math.Hypot(d.X-p.X, d.Y-p.Y) < 65 {
			d.Collected = true
			r.Gold += d.Gold
			if d.Gear != nil && d.Gear.Rarity >= content.RarityRare {
				r.event("rare_item", d.X, d.Y, float64(d.Gear.Rarity))
			} else {
				r.event("pickup", d.X, d.Y, float64(d.Gold))
			}
		}
	}
	r.tickCollapseObjective(dt)
	r.tickLanternObjective(dt)
	r.tickDefenseObjective(dt)
	if p.HP <= 0 || r.lanternExtinguished() || r.defenseLost() {
		p.Pose = "defeat"
		p.Knockdown = 0
		r.Status = "defeated"
		r.finishMissionHistory("defeated")
		r.RecordEncounterSummary("defeated")
		r.Gold = 0
		r.Drops = []Drop{}
		r.Projectiles = []Projectile{}
		r.event("defeat", p.X, p.Y, 0)
		return
	}
	if r.Practice != nil {
		r.practiceTick()
	}
	r.tickRoomObjective(dt)
	alive := 0
	for _, e := range r.Enemies {
		if e.HP > 0 {
			alive++
		}
	}
	if r.Practice == nil && alive == 0 && r.Status == "fighting" && (r.RoomObjective == nil || r.RoomObjective.Complete) {
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
				if d.Gear != nil && d.Gear.Rarity >= content.RarityRare {
					r.event("rare_item", d.X, d.Y, float64(d.Gear.Rarity))
				} else {
					r.event("pickup", d.X, d.Y, float64(d.Gold))
				}
			}
		}
		r.Status = "cleared"
		r.recordBossClear()
		r.Projectiles = []Projectile{}
		r.RecordEncounterSummary("cleared")
		r.event("clear", p.X, p.Y, 0)
		if r.Room == len(Rooms)-1 {
			p.Pose = "victory"
			p.PoseTime = 4.0
			r.eventAtHeight("victory", p.X, p.Y-30, 0, p.Elevation)
		}
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
		if skill.Kind != "slash" && skill.Kind != "heal" && skill.Kind != "shield" {
			r.Stats.NonMeleeCasts++
		}
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
		isUlt := skill.Kind == "ultimate" || (r.Build.Ultimate != nil && skill.ID == r.Build.Ultimate.ID)
		if isUlt {
			r.Stats.UltimateCasts++
			p.Pose = "ultimate_anticipation"
			p.PoseTime = .55
			p.Cooldown = .55
			r.eventAtHeight("ultimate_anticipation", p.X, p.Y-35, 0, p.Elevation)
		}
		r.SkillTimers[id] = skill.Cooldown
		charges, marked := r.classCast(skill)
		if skill.Role == "finisher" {
			r.eventAtHeight("finisher_cast", p.X, p.Y-35, float64(charges), p.Elevation)
		}
		if isUlt || (skill.Role == "finisher" && charges > 0) || skill.Kind == "quake" || skill.Kind == "slam" {
			r.heavyRecovery = .22
			p.Cooldown += r.heavyRecovery
		} else {
			r.heavyRecovery = 0
		}
		r.eventAtHeight(skill.Kind, p.X+p.Facing*35, p.Y-35, 0, p.Elevation)
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
			r.Projectiles = append(r.Projectiles, Projectile{Elevation: p.Elevation, ID: r.Counter, X: p.X + p.Facing*35, Y: p.Y, VX: p.Facing * 530, Power: power, Life: 2.5, Kind: skill.Kind, Skill: skill, Charges: charges, Marked: marked})
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
	if e.Kind == "boss" && e.WeakPoint > 0 {
		damage *= 1.25
	}
	damage *= 1 - armor*(1-clamp(pierce, 0, 1))
	if r.guardianBondActive(e.ID) {
		damage *= .5
	}
	damage = math.Min(e.HP, math.Max(0, damage))
	prevHP := e.HP
	e.HP = math.Max(0, e.HP-damage)
	r.Stats.DamageDealt += damage
	r.Stats.LargestHit = math.Max(r.Stats.LargestHit, damage)
	if e.Pose != "stagger" {
		e.Pose = "hit"
		e.PoseTime = .2
	}
	hitDir := r.Player.Facing
	if e.X != r.Player.X {
		hitDir = math.Copysign(1, e.X-r.Player.X)
	}
	recoilDist := 10.0
	switch e.Kind {
	case "boss":
		recoilDist = 4.0
	case "knight":
		recoilDist = 6.5
	}
	e.RecoilX = hitDir * recoilDist
	r.eventAtHeight(effect, e.X, e.Y-30, damage, e.Elevation)
	if damage > 0 {
		r.interruptRitual(e.ID)
		r.eventAtHeight(e.HurtCue(), e.X, e.Y-30, damage, e.Elevation)
	}
	if e.Kind == "boss" && e.HP > 0 && e.MaxHP > 0 {
		if e.Phase < 1 {
			e.Phase = 1
		}
		previousPhase := e.Phase
		if e.Phase < 3 && e.HP <= e.MaxHP*float64(bossPhaseTraining[2].AtHealthPercent)/100 && prevHP > e.MaxHP*float64(bossPhaseTraining[2].AtHealthPercent)/100 {
			e.Phase = 3
			r.eventAtHeight("boss_phase", e.X, e.Y-30, 3, e.Elevation)
		} else if e.Phase < 2 && e.HP <= e.MaxHP*float64(bossPhaseTraining[1].AtHealthPercent)/100 && prevHP > e.MaxHP*float64(bossPhaseTraining[1].AtHealthPercent)/100 {
			e.Phase = 2
			r.eventAtHeight("boss_phase", e.X, e.Y-30, 2, e.Elevation)
		}
		if e.Phase != previousPhase {
			e.Windup = 0
			e.AttackName = ""
			e.Cooldown = math.Max(e.Cooldown, 1)
			e.Pose = "stagger"
			e.PoseTime = math.Max(e.PoseTime, .6)
		}
	}
	if r.Practice != nil {
		if damage > 0 && e.ID == "practice-target" && (effect == "hit" || strings.HasPrefix(effect, "hit_")) {
			r.Practice.Hits++
		}
		if r.Practice.Mode != "boss" {
			e.HP = e.MaxHP
		}
		return
	}
	if e.HP == 0 {
		e.RecoilX = 0
		if r.Marked == e.ID {
			r.Marked = ""
		}
		if e.isObjectiveProp() {
			r.event(e.Kind+"_break", e.X, e.Y, 0)
			if e.Kind == "generator" {
				r.disableGenerator(e.ID)
			} else if e.Kind == "cage" {
				r.updateRescueObjective()
			} else {
				r.updateTotemObjective()
			}
			return
		}
		r.recordPriorityDefeat(*e)
		r.recordMonsterDefeat(*e)
		r.Stats.Kills++
		if e.Kind == "treasure" {
			r.Stats.TreasureGoblins++
		}
		if e.Kind == "boss" {
			r.Stats.Bosses++
		}
		r.event(e.DeathCue(), e.X, e.Y, 0)
		if e.Kind == "boss" {
			r.Events[len(r.Events)-1].ActorName = e.Name
		}
		mission := 1
		if r.Level != nil {
			mission = r.Level.ID
		}
		r.Drops = append(r.Drops, Drop{Elevation: r.Arena().Elevation(e.X, e.Y), Mission: mission, Tier: r.Room + 1, ID: e.ID, X: e.X, Y: e.Y, Gold: int64(15 * (r.Room + 1)), NeedsGear: e.Kind == "boss" || e.Kind == "knight" || e.Kind == "treasure" || i == 0})
	}
}

// EscapeEnemy marks an enemy as having escaped rather than being defeated.
// It emits a Kind_escape event (e.g. treasure_escape) without granting kills, drops, or defeat credit.
func (r *Run) EscapeEnemy(i int) {
	r.escapeEnemy(i)
}

func (r *Run) escapeEnemy(i int) {
	if i < 0 || i >= len(r.Enemies) {
		return
	}
	e := &r.Enemies[i]
	if e.HP <= 0 {
		return
	}
	e.HP = 0
	e.Pose = "escape"
	if r.Marked == e.ID {
		r.Marked = ""
	}
	r.event(e.Kind+"_escape", e.X, e.Y, 0)
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
		if r.SkillTimers != nil && r.SkillTimers["perfect_guard"] > 0 {
			kind = "perfect_guard"
		} else {
			kind = "block"
		}
	}
	if r.Barrier > 0 {
		absorbed := r.absorbBarrier(damage)
		damage -= absorbed
		if absorbed > 0 {
			r.eventAtHeight("shield_absorb", p.X, p.Y-30, absorbed, p.Elevation)
		}
		if kind != "perfect_guard" {
			kind = "block"
		}
	}
	damage = math.Min(p.HP, damage)
	p.HP = math.Max(0, p.HP-damage)
	r.Stats.DamageTaken += damage
	if damage > 0 {
		r.Stats.HitsTaken++
	}
	if p.HP == 0 {
		p.Pose = "defeat"
		p.Knockdown = 0
		p.RecoilX = 0
	} else {
		p.Pose = "hit"
		p.PoseTime = .18
		hitDir := -p.Facing
		if p.X != x {
			hitDir = math.Copysign(1, p.X-x)
		}
		recoilDist := 9.0
		if p.Guard {
			recoilDist = 3.0
		}
		p.RecoilX = hitDir * recoilDist
	}
	r.heavyRecovery = 0
	r.eventAtHeight(kind, p.X, p.Y-30, damage, p.Elevation)
}

// canStartEnemyAttack counts telegraphs and strikes, not movement or recovery.
func (r *Run) canStartEnemyAttack(candidate *Actor) bool {
	limit := r.Arena().MaxAttackers
	if limit <= 0 {
		limit = 3
	}
	limit = min(limit, 8)
	active := 0
	for _, enemy := range r.Enemies {
		if candidate.Kind == "boss" && enemy.Kind == "boss" && enemy.HP > 0 && enemy.Windup > 0 {
			return false
		}
		if enemy.HP > 0 && (enemy.Windup > 0 || enemy.Pose == "attack" && enemy.PoseTime > 0) {
			active++
		}
	}
	return active < limit
}

const treasureEscapeMargin = 55.0

func (r *Run) enemyTick(i int, dt float64) {
	if r.tickRitualEnemy(i, dt) {
		return
	}
	if r.Enemies[i].isObjectiveProp() {
		e := &r.Enemies[i]
		e.PoseTime = math.Max(0, e.PoseTime-dt)
		if e.PoseTime == 0 {
			e.Pose = "idle"
			e.RecoilX = 0
		}
		return
	}
	if r.Practice != nil && r.Enemies[i].ID == "practice-target" {
		e := &r.Enemies[i]
		e.PoseTime = math.Max(0, e.PoseTime-dt)
		e.Knockdown = math.Max(0, e.Knockdown-dt)
		if e.PoseTime == 0 {
			e.RecoilX = 0
		} else if e.RecoilX != 0 {
			e.RecoilX = math.Copysign(math.Max(0, math.Abs(e.RecoilX)-50*dt), e.RecoilX)
		}
		if e.PoseTime == 0 && e.Knockdown == 0 {
			e.Pose = "idle"
		}
		return
	}
	e := &r.Enemies[i]
	if e.HP <= 0 {
		return
	}
	e.WeakPoint = math.Max(0, e.WeakPoint-dt)
	e.Cooldown = math.Max(0, e.Cooldown-dt)
	e.PoseTime = math.Max(0, e.PoseTime-dt)
	if e.PoseTime < 0.0001 {
		e.PoseTime = 0
	}
	if e.Jump > 0 {
		e.Jump = math.Max(0, e.Jump-dt)
		if e.Jump < 0.0001 {
			e.Jump = 0
		}
	}
	if e.PoseTime == 0 {
		e.RecoilX = 0
		if e.Pose == "stagger" || e.Pose == "hit" {
			e.Pose = "idle"
		}
	} else if e.RecoilX != 0 {
		e.RecoilX = math.Copysign(math.Max(0, math.Abs(e.RecoilX)-50*dt), e.RecoilX)
	}
	if e.Pose == "stagger" && e.PoseTime > 0 {
		return
	}
	if e.Knockdown > 0 {
		e.Knockdown = math.Max(0, e.Knockdown-dt)
		e.Pose = "knockdown"
		return
	}
	if r.tickDefenseEnemy(e, dt) {
		return
	}
	p := &r.Player
	dx, dy := p.X-e.X, p.Y-e.Y
	if e.RouteY != 0 && r.clearPursuitPath(e, p) {
		e.RouteX, e.RouteY = 0, 0
	}
	if e.Kind == "treasure" && (e.X <= treasureEscapeMargin || e.X >= Width-treasureEscapeMargin) {
		r.escapeEnemy(i)
		return
	}
	if e.Kind == "treasure" && (e.Fleeing || math.Abs(dx) < 240) {
		e.Fleeing = true
		e.Facing = -math.Copysign(1, dx)
		r.moveActor(e, -math.Copysign(e.Speed*dt, dx), 0, true)
		e.Pose = "run"
		if e.X <= treasureEscapeMargin || e.X >= Width-treasureEscapeMargin {
			r.escapeEnemy(i)
		}
		return
	}
	if dx != 0 {
		e.Facing = math.Copysign(1, dx)
	}
	if e.Kind == "knight" && e.Pose == "attack" && e.PoseTime > 0 {
		return
	}
	if e.Windup > 0 {
		e.Windup = math.Max(0, e.Windup-dt)
		if e.Windup == 0 {
			plan := r.NextBossAttack(*e)
			e.AttackName = ""
			e.Attacks++
			e.Pose = "attack"
			e.PoseTime = .4
			e.Cooldown = 1.6 + rangedCooldownOffset(e)
			if e.Kind == "boss" {
				e.Cooldown = plan.Recovery
			}
			if e.Kind == "archer" || plan.Kind == "projectile" {
				if !r.clearProjectilePath(e, e) || e.Kind == "archer" && !r.clearProjectilePath(e, p) {
					e.Pose = "idle"
					e.PoseTime = 0
					e.Cooldown = .3
					return
				}
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
				r.Projectiles = append(r.Projectiles, Projectile{Elevation: e.Elevation, OwnerID: e.ID, ID: r.Counter, X: e.X, Y: e.Y, VX: dx / distance * 300, VY: dy / distance * 300, Power: power, Enemy: true, Life: 4, Kind: shot})
				r.eventAtHeight(shot, e.X, e.Y-30, 0, e.Elevation)
			} else if e.Kind == "boss" {
				r.event("slam", e.TargetX, e.TargetY, 0)
				if math.Abs(p.X-e.TargetX) < 125 && math.Abs(p.Y-e.TargetY) < 62 {
					if p.Jump < .1 {
						r.hurtPlayerFromEnemy(max(32, e.Damage*1.4), e.X, e.Y, e.ID)
					} else {
						r.recordDodge()
					}
				}
				if math.Abs(p.X-e.TargetX) >= 125 || math.Abs(p.Y-e.TargetY) >= 62 || p.Jump >= .1 {
					e.WeakPoint = .8
				}
			} else {
				r.event(e.Kind+"_attack", e.X, e.Y, 0)
				inReach := math.Abs(dx) < 85 && math.Abs(dy) < 33 && r.clearMeleePath(e, p)
				if e.Kind == "knight" && (!inReach || p.Jump >= .25) {
					e.Cooldown = 2.2
					e.PoseTime = .8
				}
				if inReach {
					if p.Jump < .25 {
						power := e.Damage
						if power <= 0 {
							power = 19 + float64(r.Room)*4
						}
						r.hurtPlayerFromEnemy(power, e.X, e.Y, e.ID)
					} else {
						r.recordDodge()
					}
				}
			}
		}
		return
	}
	if e.Kind == "archer" && math.Abs(dx) < 150 && math.Abs(dy) <= 24 && r.clearProjectilePath(e, p) {
		speed := e.Speed
		if speed <= 0 {
			speed = 80
		}
		beforeX := e.X
		r.moveActor(e, -math.Copysign(speed*dt, dx), 0, false)
		if e.X != beforeX {
			if e.PoseTime == 0 {
				e.Pose = "run"
			}
			return
		}
	}
	rangeX := 65.0
	if e.Kind == "archer" {
		rangeX = 430
	}
	if e.Kind == "boss" {
		rangeX = 190
	}
	blocked := e.Kind == "archer" && !r.clearProjectilePath(e, p) || e.Kind != "archer" && e.Kind != "boss" && !r.clearMeleePath(e, p)
	if math.Abs(dx) > rangeX || math.Abs(dy) > 24 || blocked {
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
	} else if e.Cooldown == 0 && r.canStartEnemyAttack(e) {
		e.Windup = enemyWindup(e)
		e.Pose = "windup"
		e.TargetX = p.X
		e.TargetY = p.Y
		if e.Kind == "boss" {
			r.event("boss_roar", e.X, e.Y, 0)
			plan := r.NextBossAttack(*e)
			e.AttackName, e.Windup = plan.Name, plan.Windup
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
