package bot

import (
	"strings"
	"testing"
)

func TestAbyssDeferredEventExpiresAfterThreeFloors(t *testing.T) {
	t.Parallel()

	event := abyssDeferredEvent{State: `{"type":"trap_chamber"}`, Label: "Trap Chamber", OriginDepth: 10, ExpiresDepth: 13}
	available := abyssDeferredEventStatus(event, 13)
	if !available.Available || available.Expired || available.FloorsLeft != 0 {
		t.Fatalf("deadline status = %+v", available)
	}
	expired := abyssDeferredEventStatus(event, 14)
	if expired.Available || !expired.Expired {
		t.Fatalf("expired status = %+v", expired)
	}
}

func TestAbyssDeferredEventCannotBeDeferredTwice(t *testing.T) {
	t.Parallel()

	state, label, ok := markAbyssEventDeferred(`{"type":"trap_chamber","depth":7}`)
	if !ok || label != "Trap Chamber" || !strings.Contains(state, `"deferred":true`) {
		t.Fatalf("deferred event = %q, %q, %t", state, label, ok)
	}
	if _, _, ok := markAbyssEventDeferred(state); ok {
		t.Fatal("already deferred event was accepted again")
	}
}
