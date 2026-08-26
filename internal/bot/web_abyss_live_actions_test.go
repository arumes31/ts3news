package bot

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"ts3news/internal/content"
)

func TestAbyssLiveCombatBestAction(t *testing.T) {
	tests := []struct {
		name         string
		tactic       string
		allyHP       int
		options      []abyssLiveOption
		expectedKind string
		expectedID   string
	}{
		{
			name:   "balanced heals critical ally with skill",
			tactic: "balanced", allyHP: 30,
			options: []abyssLiveOption{
				{Kind: "skill", ID: "heal", Target: "ally", Power: 1},
				{Kind: "skill", ID: "blast", Target: "enemy", Power: 4},
			},
			expectedKind: "skill", expectedID: "heal",
		},
		{
			name:   "defensive spends potion earlier",
			tactic: "defensive", allyHP: 40,
			options: []abyssLiveOption{
				{Kind: "item", ID: "potion", Target: "ally", Count: 1, Power: 50},
				{Kind: "skill", ID: "blast", Target: "enemy", Power: 4},
			},
			expectedKind: "item", expectedID: "potion",
		},
		{
			name:   "conserve items attacks outside emergency",
			tactic: "conserve_items", allyHP: 20,
			options: []abyssLiveOption{
				{Kind: "item", ID: "potion", Target: "ally", Count: 1, Power: 50},
				{Kind: "ultimate", ID: "meteor", Target: "enemy", Power: 8},
			},
			expectedKind: "ultimate", expectedID: "meteor",
		},
		{
			name:   "chooses strongest affordable offense",
			tactic: "balanced", allyHP: 100,
			options: []abyssLiveOption{
				{Kind: "skill", ID: "expensive", Target: "enemy", Mana: 120, Power: 20},
				{Kind: "skill", ID: "bolt", Target: "enemy", Mana: 20, Power: 3},
			},
			expectedKind: "skill", expectedID: "bolt",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			combat := &abyssLiveCombat{
				round:   4,
				tactics: map[string]string{"user": test.tactic},
				allies: []abyssLiveCombatantView{
					{ID: "ally:user", HP: test.allyHP, MaxHP: 100, Mana: 100, MaxMana: 100},
				},
				enemies: []abyssLiveCombatantView{
					{ID: "enemy:0", HP: 90, MaxHP: 100},
					{ID: "enemy:1", HP: 25, MaxHP: 100},
				},
				options: map[string][]abyssLiveOption{"user": test.options},
			}
			action := combat.bestActionLocked("user")
			if action.Kind != test.expectedKind || action.AbilityID != test.expectedID {
				t.Fatalf("best action = %s/%s, want %s/%s", action.Kind, action.AbilityID, test.expectedKind, test.expectedID)
			}
			if action.TargetID != "enemy:1" && action.Kind != "item" && action.AbilityID != "heal" {
				t.Fatalf("offensive action target = %q, want lowest-health enemy", action.TargetID)
			}
			if !action.Automatic {
				t.Fatal("best action must be marked automatic")
			}
		})
	}
}

func TestAbyssLiveCombatRecommendation(t *testing.T) {
	combat := &abyssLiveCombat{
		participants: map[string]bool{"user": true},
		phase:        "planning",
		round:        2,
		tactics:      map[string]string{"user": "balanced"},
		allies: []abyssLiveCombatantView{
			{ID: "ally:user", Name: "Delver", HP: 100, MaxHP: 100, Mana: 100},
		},
		enemies: []abyssLiveCombatantView{
			{ID: "enemy:0", Name: "Brute", HP: 75, MaxHP: 100},
			{ID: "enemy:1", Name: "Imp", HP: 20, MaxHP: 100},
		},
		options: map[string][]abyssLiveOption{
			"user": {{Kind: "skill", ID: "blast", Name: "Blast", Target: "enemy", Power: 3}},
		},
		queued: map[string]abyssLiveAction{},
	}

	snapshot := combat.snapshotForLocked("user")
	if snapshot.Recommended == nil {
		t.Fatal("planning snapshot has no recommendation")
	}
	if got := snapshot.Recommended.Action; got.Kind != "skill" || got.AbilityID != "blast" || got.TargetID != "enemy:1" {
		t.Fatalf("recommended action = %+v, want blast against lowest-health enemy", got)
	}
	if snapshot.Recommended.Reason == "" {
		t.Fatal("recommendation has no explanation")
	}
}

func TestLiveElementWeakness(t *testing.T) {
	tests := []struct {
		element content.Element
		want    content.Element
	}{
		{content.ElementFire, content.ElementWater},
		{content.ElementWater, content.ElementEarth},
		{content.ElementEarth, content.ElementAir},
		{content.ElementAir, content.ElementFire},
		{content.ElementPhysical, ""},
	}
	for _, test := range tests {
		if got := liveElementWeakness(test.element); got != test.want {
			t.Errorf("liveElementWeakness(%s) = %s, want %s", test.element, got, test.want)
		}
	}
}

func TestPlanLiveEnemyIntents(t *testing.T) {
	users := []activeUser{
		{u: &UserInCombat{UID: "front", Nickname: "Front", CurrentHP: 100, Position: content.PositionFrontline}},
		{u: &UserInCombat{UID: "back", Nickname: "Back", CurrentHP: 100, Position: content.PositionBackline}},
	}
	mobs := []*content.Mob{
		{Name: "Striker", Stats: content.Stats{HP: 100, SPD: 20}},
		{Name: "Stunned", Stats: content.Stats{HP: 100, SPD: 0}},
	}

	plans := planLiveEnemyIntents(3, users, mobs, nil)
	if got := plans[0]; got.TargetUID != "front" || got.Intent.TargetID != "ally:front" {
		t.Fatalf("frontline intent = %+v, want authoritative frontline target", got)
	}
	if got := plans[1]; got.Intent.Kind != "stunned" || got.Intent.Ability != "Skips this turn" {
		t.Fatalf("stunned intent = %+v", got.Intent)
	}
}

func TestAbyssLiveInitiativeMatchesExecutionOrder(t *testing.T) {
	users := []activeUser{
		{u: &UserInCombat{UID: "first", Nickname: "First", CurrentHP: 100, Stats: content.Stats{SPD: 10}}},
		{u: &UserInCombat{UID: "second", Nickname: "Second", CurrentHP: 100, Stats: content.Stats{SPD: 30}}},
	}
	mobs := []*content.Mob{
		{Name: "Enemy A", Stats: content.Stats{HP: 100, SPD: 40}},
		{Name: "Enemy B", Stats: content.Stats{HP: 100, SPD: 20}},
	}

	playerFirst := liveInitiative(users, mobs, true)
	wantPlayerFirst := []string{"ally:first", "ally:second", "enemy:0", "enemy:1"}
	for i, want := range wantPlayerFirst {
		if playerFirst[i].ID != want {
			t.Fatalf("player-first initiative[%d] = %q, want %q", i, playerFirst[i].ID, want)
		}
	}
	enemyFirst := liveInitiative(users, mobs, false)
	wantEnemyFirst := []string{"enemy:0", "enemy:1", "ally:first", "ally:second"}
	for i, want := range wantEnemyFirst {
		if enemyFirst[i].ID != want {
			t.Fatalf("enemy-first initiative[%d] = %q, want %q", i, enemyFirst[i].ID, want)
		}
	}
}

func TestAbyssLiveInterruptComboCleaveThreatAndRecap(t *testing.T) {
	users := []activeUser{{u: &UserInCombat{UID: "tank", Nickname: "Tank", CurrentHP: 100, Position: content.PositionFrontline}}}
	boss := &content.Mob{Name: "Channeler", Type: content.MobBoss, Stats: content.Stats{HP: 100, SPD: 0}}
	plans := planLiveEnemyIntents(4, users, []*content.Mob{boss}, []string{"Channeler begins a summoning ritual!"})
	if got := plans[0].Intent; got.Kind != "interruptible" || !strings.Contains(got.Ability, "Ultimate") {
		t.Fatalf("channel intent = %+v", got)
	}
	if !liveSkillComboFollowup("enemy:1", abyssLiveAction{Kind: "skill", TargetID: "enemy:1"}) {
		t.Fatal("ordered same-target skill did not form a combo")
	}
	if liveSkillComboFollowup("enemy:1", abyssLiveAction{Kind: "attack", TargetID: "enemy:1"}) {
		t.Fatal("basic attack incorrectly formed a skill combo")
	}
	mobs := []*content.Mob{
		{Name: "excluded", Stats: content.Stats{HP: -10}},
		{Name: "large", Stats: content.Stats{HP: 80}},
		{Name: "small", Stats: content.Stats{HP: 20}},
	}
	if got := lowestHealthMobExcept(mobs, mobs[0]); got != mobs[2] {
		t.Fatalf("cleave target = %v, want lowest living alternate", got)
	}
	if got := liveThreat(users[0].u); got != 100 {
		t.Fatalf("frontline threat = %d, want 100", got)
	}
	recap := liveRoundRecap(2,
		[]abyssLiveCombatantView{{ID: "ally:tank", HP: 100}},
		[]abyssLiveCombatantView{{ID: "ally:tank", HP: 70}},
		[]abyssLiveCombatantView{{ID: "enemy:0", HP: 50}, {ID: "enemy:1", HP: 20}},
		[]abyssLiveCombatantView{{ID: "enemy:0", HP: 10}},
	)
	if !strings.Contains(recap, "party lost 30 HP") || !strings.Contains(recap, "enemies lost 60 HP") || !strings.Contains(recap, "1 enemies defeated") {
		t.Fatalf("round recap = %q", recap)
	}
}

func TestMobTurnConsumesAbyssLiveEnemyPlan(t *testing.T) {
	combat := &abyssLiveCombat{
		enemyPlans: map[int]abyssLiveEnemyPlan{
			0: {
				Round: 4, TargetUID: "target", SpellIndex: 0,
				Intent: abyssLiveEnemyIntent{Kind: "cast"},
			},
		},
	}
	users := []activeUser{
		{u: &UserInCombat{
			UID: "other", Nickname: "Other", CurrentHP: 500, Position: content.PositionFrontline,
			Stats: content.Stats{HP: 500}, Equipped: map[content.GearSlot]content.Gear{}, live: combat,
		}},
		{u: &UserInCombat{
			UID: "target", Nickname: "Target", CurrentHP: 500, Position: content.PositionFrontline,
			Stats: content.Stats{HP: 500}, Equipped: map[content.GearSlot]content.Gear{}, live: combat,
		}},
	}
	mobs := []*content.Mob{{
		Name: "Caster", Element: content.ElementPhysical,
		Stats: content.Stats{HP: 100, STR: 40, SPD: 10}, STRMod: 1,
		Spells: []content.Skill{{ID: "planned", Name: "Planned Spell", Power: 2}},
	}}
	logs := []string{}
	totalMobDamage, totalUserDamage := 0, 0

	(&Bot{}).mobTurn(
		users,
		mobs,
		content.Zone{},
		1,
		&logs,
		&totalMobDamage,
		&totalUserDamage,
		4,
		false,
		nil,
		defaultCombatRandomSource{},
	)
	if users[0].u.CurrentHP != 500 {
		t.Fatalf("unplanned target HP = %d, want 500", users[0].u.CurrentHP)
	}
	if users[1].u.CurrentHP >= 500 {
		t.Fatalf("planned target HP = %d, want damage", users[1].u.CurrentHP)
	}
	if !strings.Contains(strings.Join(logs, "\n"), "Planned Spell") {
		t.Fatalf("combat logs = %q, want planned spell", logs)
	}
}

func TestLiveEffectViews(t *testing.T) {
	ally := &activeUser{
		effects: []content.ItemEffect{content.EffectBerserk, content.EffectBerserk},
		Stunned: true,
	}
	allyEffects := liveAllyEffects(ally)
	if len(allyEffects) != 2 {
		t.Fatalf("ally effects = %+v, want one encounter effect and one stun", allyEffects)
	}
	if allyEffects[0].Duration != abyssLiveEncounterDuration {
		t.Fatalf("encounter effect duration = %q", allyEffects[0].Duration)
	}
	if allyEffects[1].Name != "Stunned" || allyEffects[1].RemainingRounds != 1 {
		t.Fatalf("stun effect = %+v, want one remaining round", allyEffects[1])
	}

	mob := &content.Mob{
		Stats:   content.Stats{SPD: 0},
		Effects: []content.MobEffect{content.EffectEnraged, content.EffectEnraged},
	}
	mobEffects := liveMobEffects(mob)
	if len(mobEffects) != 2 || mobEffects[0].Duration != abyssLiveEncounterDuration || mobEffects[1].RemainingRounds != 1 {
		t.Fatalf("mob effects = %+v, want deduplicated encounter effect and one-round stun", mobEffects)
	}
	if !mobEffects[0].Affix || mobEffects[0].Key != "enraged" || mobEffects[0].Description == "" || mobEffects[0].Icon == "" {
		t.Fatalf("mob affix metadata = %+v, want authoritative enriched view", mobEffects[0])
	}

	vampiric := liveMobEffectsForModifier(mob, "storm_floor vampiric_mobs")
	if len(vampiric) != 3 {
		t.Fatalf("vampiric effects = %+v, want base affix, Vampiric, and stun", vampiric)
	}
	if got := vampiric[1]; got.Key != abyssLiveEffectVampiric || got.Name != "Vampiric" || !got.Affix || got.Description == "" {
		t.Fatalf("vampiric affix = %+v", got)
	}
}

func TestAbyssLiveDefendShowsReactiveGuard(t *testing.T) {
	ally := &activeUser{defendingRound: 3}
	effects := liveAllyEffects(ally)
	if len(effects) != 1 || effects[0].Name != "Guarded" || effects[0].Duration != "next enemy phase" {
		t.Fatalf("active guard effects = %+v", effects)
	}
	combat := &abyssLiveCombat{
		participants: map[string]bool{"user": true}, phase: "planning", round: 3,
		allies: []abyssLiveCombatantView{{ID: "ally:user", HP: 100, MaxHP: 100}},
		queued: map[string]abyssLiveAction{"user": {Kind: "defend", Round: 3}},
		ready:  map[string]bool{}, options: map[string][]abyssLiveOption{}, tactics: map[string]string{},
	}
	snapshot := combat.snapshotForLocked("user")
	if got := snapshot.Allies[0].Effects; len(got) != 1 || got[0].Name != "Guard queued" {
		t.Fatalf("queued guard effects = %+v", got)
	}
}

func TestEstimateLiveDamageRange(t *testing.T) {
	user := &UserInCombat{
		Stats:    content.Stats{STR: 100},
		Equipped: map[content.GearSlot]content.Gear{},
	}
	mob := &content.Mob{
		Element: content.ElementFire,
		Stats:   content.Stats{HP: 100, DEF: 10},
		MaxHP:   100,
		DEFMod:  1,
	}

	neutralMin, neutralMax := estimateLiveDamageRange(user, 1, 0, content.ElementPhysical, []*content.Mob{mob})
	if neutralMin <= 0 || neutralMax < neutralMin {
		t.Fatalf("neutral damage range = %d-%d", neutralMin, neutralMax)
	}
	user.Equipped[content.SlotMainHand] = content.Gear{Element: content.ElementWater}
	strongMin, strongMax := estimateLiveDamageRange(user, 1, 0, content.ElementWater, []*content.Mob{mob})
	if strongMin <= neutralMin || strongMax <= neutralMax {
		t.Fatalf("advantaged range = %d-%d, want greater than neutral %d-%d", strongMin, strongMax, neutralMin, neutralMax)
	}
}

func TestEstimateLiveDamageRangeIncludesRuneResonance(t *testing.T) {
	user := &UserInCombat{
		Stats: content.Stats{STR: 100},
		Equipped: map[content.GearSlot]content.Gear{
			content.SlotMainHand: {Element: content.ElementFire},
		},
	}
	mob := &content.Mob{Element: content.ElementPhysical, Stats: content.Stats{HP: 100}, MaxHP: 100, DEFMod: 1}
	plainMin, plainMax := estimateLiveDamageRange(user, 1, 0, content.ElementFire, []*content.Mob{mob})
	user.Equipped[content.SlotMainHand] = content.Gear{Element: content.ElementFire, Rune: string(content.ElementFire)}
	resonantMin, resonantMax := estimateLiveDamageRange(user, 1, 0, content.ElementFire, []*content.Mob{mob})
	if resonantMin <= plainMin || resonantMax <= plainMax {
		t.Fatalf("resonant range = %d-%d, want greater than plain %d-%d", resonantMin, resonantMax, plainMin, plainMax)
	}
}

func TestEstimateLiveHealing(t *testing.T) {
	users := []activeUser{
		{u: &UserInCombat{CurrentHP: 50, Stats: content.Stats{HP: 100}}},
		{u: &UserInCombat{CurrentHP: 100, Stats: content.Stats{HP: 200}}},
	}
	if minHeal, maxHeal := estimateLiveSkillHeal(0.25, users); minHeal != 25 || maxHeal != 50 {
		t.Fatalf("skill heal range = %d-%d, want 25-50", minHeal, maxHeal)
	}
	if minHeal, maxHeal := estimateLiveItemHeal(40, users); minHeal != 40 || maxHeal != 40 {
		t.Fatalf("flat item heal range = %d-%d, want 40-40", minHeal, maxHeal)
	}
}

func TestAbyssLiveCombatSubmit(t *testing.T) {
	combat := &abyssLiveCombat{
		id:           "session",
		participants: map[string]bool{"user": true},
		phase:        "planning",
		round:        3,
		deadline:     time.Now().Add(time.Minute),
		options: map[string][]abyssLiveOption{
			"user": {
				{Kind: "attack", Name: "Basic Attack", Target: "enemy"},
				{Kind: "skill", ID: "heal", Name: "Heal", Target: "ally", Mana: 20},
			},
		},
		allies: []abyssLiveCombatantView{
			{ID: "ally:user", HP: 50, MaxHP: 100, Mana: 100, MaxMana: 100},
		},
		enemies: []abyssLiveCombatantView{
			{ID: "enemy:0", HP: 50, MaxHP: 100},
		},
		queued:      map[string]abyssLiveAction{},
		idempotency: map[string]abyssLiveIdempotency{},
	}

	action := abyssLiveAction{
		SessionID: "session", Kind: "attack", TargetID: "enemy:0", Round: 3, IdempotencyKey: "same", Automatic: true,
	}
	if err := combat.submit("user", action); err != nil {
		t.Fatalf("valid action rejected: %v", err)
	}
	if err := combat.submit("user", action); err != nil {
		t.Fatalf("idempotent retry rejected: %v", err)
	}
	conflictingRetry := action
	conflictingRetry.Kind = "skill"
	conflictingRetry.AbilityID = "heal"
	conflictingRetry.TargetID = "ally:user"
	if err := combat.submit("user", conflictingRetry); !errors.Is(err, errAbyssLiveIdempotencyConflict) {
		t.Fatalf("conflicting idempotency retry error = %v, want errAbyssLiveIdempotencyConflict", err)
	}
	if got := combat.queued["user"]; got.Automatic {
		t.Fatal("manual action marked automatic")
	}
	replacement := conflictingRetry
	replacement.IdempotencyKey = "replacement"
	if err := combat.submit("user", replacement); err != nil {
		t.Fatalf("replacement action rejected: %v", err)
	}
	if got := combat.queued["user"]; got.Kind != "skill" || got.AbilityID != "heal" {
		t.Fatalf("queued replacement = %+v, want heal skill", got)
	}

	wrongSession := action
	wrongSession.SessionID = "another-session"
	wrongSession.IdempotencyKey = "wrong-session"
	if err := combat.submit("user", wrongSession); !errors.Is(err, errAbyssLiveStale) {
		t.Fatalf("wrong-session action error = %v, want errAbyssLiveStale", err)
	}

	stale := action
	stale.Round = 2
	stale.IdempotencyKey = "stale"
	if err := combat.submit("user", stale); !errors.Is(err, errAbyssLiveStale) {
		t.Fatalf("stale action error = %v, want errAbyssLiveStale", err)
	}

	invalidTarget := abyssLiveAction{
		SessionID: "session", Kind: "skill", AbilityID: "heal", TargetID: "enemy:0", Round: 3,
	}
	if err := combat.submit("user", invalidTarget); err == nil {
		t.Fatal("ally-targeted skill accepted an enemy target")
	}
}

func TestAbyssLiveCombatReadyConfirmation(t *testing.T) {
	combat := &abyssLiveCombat{
		id:           "session",
		participants: map[string]bool{"one": true, "two": true},
		phase:        "planning",
		round:        3,
		deadline:     time.Now().Add(time.Minute),
		options: map[string][]abyssLiveOption{
			"one": {{Kind: "attack", Target: "enemy"}},
			"two": {{Kind: "attack", Target: "enemy"}},
		},
		allies: []abyssLiveCombatantView{
			{ID: "ally:one", HP: 100, MaxHP: 100},
			{ID: "ally:two", HP: 100, MaxHP: 100},
		},
		enemies:     []abyssLiveCombatantView{{ID: "enemy:0", HP: 100, MaxHP: 100}},
		queued:      map[string]abyssLiveAction{},
		ready:       map[string]bool{},
		readySignal: make(chan struct{}, 1),
		idempotency: map[string]abyssLiveIdempotency{},
	}
	if err := combat.setReady("one", "session", 3); err == nil {
		t.Fatal("ready accepted without a queued action")
	}
	for _, uid := range []string{"one", "two"} {
		action := abyssLiveAction{SessionID: "session", Kind: "attack", TargetID: "enemy:0", Round: 3}
		if err := combat.submit(uid, action); err != nil {
			t.Fatalf("submit %s: %v", uid, err)
		}
	}
	if err := combat.setReady("one", "session", 3); err != nil {
		t.Fatalf("ready one: %v", err)
	}
	select {
	case <-combat.readySignal:
		t.Fatal("round released before every living participant was ready")
	default:
	}
	if err := combat.setReady("two", "session", 3); err != nil {
		t.Fatalf("ready two: %v", err)
	}
	combat.releaseReadyRound()
	select {
	case <-combat.readySignal:
	default:
		t.Fatal("all-ready confirmation did not release the round")
	}
	replacement := abyssLiveAction{SessionID: "session", Kind: "attack", TargetID: "enemy:0", Round: 3}
	if err := combat.submit("one", replacement); err != nil {
		t.Fatalf("replace one: %v", err)
	}
	if combat.ready["one"] {
		t.Fatal("replacing a queued action retained stale ready state")
	}
}

func TestAbyssLiveCombatTimeBank(t *testing.T) {
	deadline := time.Now().Add(time.Second)
	combat := &abyssLiveCombat{
		id:             "session",
		participants:   map[string]bool{"user": true},
		phase:          "planning",
		round:          5,
		deadline:       deadline,
		timeBank:       map[string]time.Duration{"user": 3 * time.Second},
		deadlineSignal: make(chan struct{}, 1),
	}
	if err := combat.spendTimeBank("user", "session", 5); err != nil {
		t.Fatalf("spend time bank: %v", err)
	}
	if got := combat.timeBank["user"]; got != time.Second {
		t.Fatalf("remaining time bank = %s, want 1s", got)
	}
	if got := combat.deadline.Sub(deadline); got != 2*time.Second {
		t.Fatalf("deadline extension = %s, want 2s", got)
	}
	select {
	case <-combat.deadlineSignal:
	default:
		t.Fatal("time-bank spend did not notify the active deadline waiter")
	}
	if got := combat.snapshotFor("user").TimeBankMS; got != 1000 {
		t.Fatalf("snapshot time bank = %dms, want 1000ms", got)
	}
	if err := combat.spendTimeBank("user", "wrong", 5); !errors.Is(err, errAbyssLiveStale) {
		t.Fatalf("wrong-session spend error = %v, want stale", err)
	}
}

func TestAbyssLivePauseConfiguration(t *testing.T) {
	tests := []struct {
		mode     string
		boss     bool
		phase    bool
		critical bool
	}{
		{mode: "adaptive", boss: true, phase: true, critical: true},
		{mode: "bosses", boss: true, phase: true, critical: false},
		{mode: "danger", boss: false, phase: true, critical: true},
		{mode: "fast", boss: false, phase: false, critical: false},
	}
	for _, test := range tests {
		if got := livePauseEnabled(test.mode, "boss"); got != test.boss {
			t.Errorf("%s boss trigger = %t, want %t", test.mode, got, test.boss)
		}
		if got := livePauseEnabled(test.mode, "phase"); got != test.phase {
			t.Errorf("%s phase trigger = %t, want %t", test.mode, got, test.phase)
		}
		if got := livePauseEnabled(test.mode, "critical"); got != test.critical {
			t.Errorf("%s critical trigger = %t, want %t", test.mode, got, test.critical)
		}
	}
	combat := &abyssLiveCombat{ownerUID: "owner"}
	if err := combat.setPauseMode("member", "fast"); err == nil {
		t.Fatal("non-owner changed shared pause triggers")
	}
	if err := combat.setPauseMode("owner", "bosses"); err != nil {
		t.Fatalf("owner set pause mode: %v", err)
	}
	if combat.pauseMode != "bosses" {
		t.Fatalf("pause mode = %q, want bosses", combat.pauseMode)
	}
}

func TestAbyssLiveConditionalPolicyAndTargetPriorities(t *testing.T) {
	combat := &abyssLiveCombat{
		round:        2,
		participants: map[string]bool{"user": true},
		tactics:      map[string]string{"user": "aggressive"},
		policies: map[string]abyssLivePolicy{"user": {
			CriticalTactic: "defensive", AttackPriority: "highest_hp", SkillPriority: "weakness",
		}},
		allies: []abyssLiveCombatantView{{ID: "ally:user", Name: "User", HP: 25, MaxHP: 100, Mana: 100, Element: "Water"}},
		enemies: []abyssLiveCombatantView{
			{ID: "enemy:0", Name: "Healthy", HP: 90, MaxHP: 100},
			{ID: "enemy:1", Name: "Weak", HP: 20, MaxHP: 100, WeakTo: "Water"},
		},
		options: map[string][]abyssLiveOption{"user": {
			{Kind: "item", ID: "potion", Target: "ally", Count: 1},
			{Kind: "skill", ID: "blast", Name: "Blast", Target: "enemy", Power: 3},
		}},
	}
	if got := combat.selectEnemyLocked("user", "attack").ID; got != "enemy:0" {
		t.Fatalf("attack priority target = %q, want highest-HP enemy", got)
	}
	if got := combat.selectEnemyLocked("user", "skill").ID; got != "enemy:1" {
		t.Fatalf("skill priority target = %q, want elemental weakness", got)
	}
	action := combat.bestActionLocked("user")
	if action.Kind != "item" {
		t.Fatalf("critical conditional action = %+v, want defensive potion fallback", action)
	}
}

func TestAbyssLiveCombatIdempotencyExpiresByRound(t *testing.T) {
	combat := &abyssLiveCombat{
		id:           "session",
		participants: map[string]bool{"user": true},
		phase:        "planning",
		round:        4,
		deadline:     time.Now().Add(time.Minute),
		options: map[string][]abyssLiveOption{
			"user": {{Kind: "attack", Name: "Basic Attack", Target: "enemy"}},
		},
		enemies: []abyssLiveCombatantView{{ID: "enemy:0", HP: 50, MaxHP: 100}},
		queued:  map[string]abyssLiveAction{},
		idempotency: map[string]abyssLiveIdempotency{
			"user:old": {Round: 3},
		},
	}

	combat.pruneIdempotencyLocked()
	if len(combat.idempotency) != 0 {
		t.Fatalf("expired idempotency records = %d, want 0", len(combat.idempotency))
	}

	for i := range abyssLiveMaxIdempotencyKeysPerRound {
		action := abyssLiveAction{
			SessionID:      "session",
			Kind:           "attack",
			TargetID:       "enemy:0",
			Round:          4,
			IdempotencyKey: fmt.Sprintf("key-%d", i),
		}
		if err := combat.submit("user", action); err != nil {
			t.Fatalf("submit key %d: %v", i, err)
		}
	}
	snapshot := combat.snapshotFor("user")
	if snapshot.ActionBudget.Remaining != 0 ||
		snapshot.ActionBudget.Limit != abyssLiveMaxIdempotencyKeysPerRound {
		t.Fatalf(
			"action budget = %d/%d, want 0/%d",
			snapshot.ActionBudget.Remaining,
			snapshot.ActionBudget.Limit,
			abyssLiveMaxIdempotencyKeysPerRound,
		)
	}
	overflow := abyssLiveAction{
		SessionID:      "session",
		Kind:           "attack",
		TargetID:       "enemy:0",
		Round:          4,
		IdempotencyKey: "overflow",
	}
	if err := combat.submit("user", overflow); !errors.Is(err, errAbyssLiveIdempotencyLimit) {
		t.Fatalf("overflow idempotency error = %v, want errAbyssLiveIdempotencyLimit", err)
	}
}

func TestLiveTargetLookups(t *testing.T) {
	mobs := []*content.Mob{
		{Name: "alive", Stats: content.Stats{HP: 10}},
		{Name: "dead", Stats: content.Stats{HP: 0}},
	}
	if got := liveMobFromTarget("enemy:0", mobs); got != mobs[0] {
		t.Fatalf("liveMobFromTarget returned %p, want %p", got, mobs[0])
	}
	for _, target := range []string{"enemy:1", "enemy:9", "ally:user", "enemy:nope"} {
		if got := liveMobFromTarget(target, mobs); got != nil {
			t.Fatalf("liveMobFromTarget(%q) = %v, want nil", target, got)
		}
	}
}
