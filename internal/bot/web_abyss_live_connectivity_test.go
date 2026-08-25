package bot

import (
	"testing"
	"time"
)

func TestAbyssLiveConnectivityGraceIsBoundedPerParticipantAndRound(t *testing.T) {
	now := time.Now()
	combat := &abyssLiveCombat{
		participants:   map[string]bool{"player": true},
		phase:          "planning",
		round:          4,
		deadline:       now.Add(time.Second),
		deadlineSignal: make(chan struct{}, 1),
	}
	firstConnectionOpened := combat.openMemberConnection("player", now)
	secondConnectionOpened := combat.openMemberConnection("player", now)
	if !firstConnectionOpened || !secondConnectionOpened {
		t.Fatal("participant connections were not registered")
	}
	if combat.closeMemberConnection("player", now) {
		t.Fatal("closing one of multiple streams granted grace")
	}
	if !combat.closeMemberConnection("player", now) {
		t.Fatal("closing the final stream did not grant grace")
	}
	if got := combat.deadline.Sub(now); got != time.Second+abyssLiveConnectivityGrace {
		t.Fatalf("extended deadline = %v", got)
	}
	select {
	case <-combat.deadlineSignal:
	default:
		t.Fatal("deadline waiter was not notified")
	}
	if !combat.openMemberConnection("player", now) {
		t.Fatal("reconnection was not registered")
	}
	if combat.closeMemberConnection("player", now) {
		t.Fatal("same-round reconnection stacked grace")
	}
	combat.mu.Lock()
	combat.round++
	combat.deadline = now.Add(time.Second)
	combat.mu.Unlock()
	if !combat.openMemberConnection("player", now) || !combat.closeMemberConnection("player", now) {
		t.Fatal("a new round did not make one grace lease available")
	}
}

func TestAbyssLiveConnectivityGraceRejectsInvalidOrClosedPlanning(t *testing.T) {
	now := time.Now()
	combat := &abyssLiveCombat{
		participants: map[string]bool{"player": true}, phase: "planning", round: 1,
		deadline: now.Add(-time.Second),
	}
	if combat.openMemberConnection("stranger", now) {
		t.Fatal("non-participant connection was accepted")
	}
	if !combat.openMemberConnection("player", now) {
		t.Fatal("participant connection was rejected")
	}
	if combat.closeMemberConnection("player", now) {
		t.Fatal("expired planning deadline received grace")
	}
}
