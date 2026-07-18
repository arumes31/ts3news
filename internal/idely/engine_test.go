package idely

import (
	"errors"
	"math/rand/v2"
	"reflect"
	"testing"
	"time"
)

// fakePlayer records spawn/stop calls and tracks which handle serves which
// channel, so tests can assert the multi-session behaviour.
type fakePlayer struct {
	nextID     int
	started    []int       // channel ids started, in order
	stopped    []int       // handles stopped, in order
	active     map[int]int // cid -> handle
	lastName   string
	lastTracks []string
	startErr   error
	stopErr    error
}

func (f *fakePlayer) Start(cid int, name string, tracks []string) (int, error) {
	if f.startErr != nil {
		return 0, f.startErr
	}
	if f.active == nil {
		f.active = map[int]int{}
	}
	f.nextID++
	f.started = append(f.started, cid)
	f.active[cid] = f.nextID
	f.lastName, f.lastTracks = name, tracks
	return f.nextID, nil
}

func (f *fakePlayer) Stop(handle int) error {
	if f.stopErr != nil {
		return f.stopErr
	}
	f.stopped = append(f.stopped, handle)
	for cid, h := range f.active {
		if h == handle {
			delete(f.active, cid)
		}
	}
	return nil
}

func newEngine(p Player) *Engine {
	return NewEngine(EngineConfig{
		Detect: cfg(),
		Player: p,
		Tracks: []string{"a.wav", "b.wav", "c.wav"},
		Names:  []string{"Chill Bot", "Sleepy DJ"},
		Rand:   rand.New(rand.NewPCG(1, 2)),
	})
}

func TestEngine_StartsWhenChannelIdle(t *testing.T) {
	p := &fakePlayer{}
	e := newEngine(p)
	e.OnSnapshot([]Client{
		voice(1, 10, "a", 20*time.Minute),
		voice(2, 10, "b", 18*time.Minute),
	})
	if !reflect.DeepEqual(e.ActiveChannels(), []int{10}) {
		t.Fatalf("active = %v, want [10]", e.ActiveChannels())
	}
	if len(p.started) != 1 || p.started[0] != 10 {
		t.Fatalf("expected one start in channel 10, got %v", p.started)
	}
	if p.lastName == "" || len(p.lastTracks) != 3 {
		t.Fatalf("expected a name and 3 tracks, got name=%q tracks=%v", p.lastName, p.lastTracks)
	}
}

func TestEngine_SpawnsOneBotPerIdleChannel(t *testing.T) {
	p := &fakePlayer{}
	e := newEngine(p)
	// Two distinct all-idle channels => two concurrent bots.
	e.OnSnapshot([]Client{
		voice(1, 10, "a", 20*time.Minute),
		voice(2, 20, "b", 20*time.Minute),
		voice(3, 20, "c", 30*time.Minute),
	})
	if got := e.ActiveChannels(); !reflect.DeepEqual(got, []int{10, 20}) {
		t.Fatalf("active = %v, want [10 20] (one bot per idle channel)", got)
	}
	if len(p.started) != 2 {
		t.Fatalf("expected 2 spawns, got %v", p.started)
	}
}

func TestEngine_MaxBotsCap(t *testing.T) {
	p := &fakePlayer{}
	e := NewEngine(EngineConfig{
		Detect: cfg(), Player: p, Tracks: []string{"a.wav"}, Names: []string{"x"},
		MaxBots: 1, Rand: rand.New(rand.NewPCG(1, 2)),
	})
	e.OnSnapshot([]Client{
		voice(1, 10, "a", 20*time.Minute),
		voice(2, 20, "b", 20*time.Minute),
	})
	if got := len(e.ActiveChannels()); got != 1 {
		t.Fatalf("with MaxBots=1 expected 1 active channel, got %d (%v)", got, e.ActiveChannels())
	}
}

func TestEngine_StopsOnlyTheWokenChannel(t *testing.T) {
	p := &fakePlayer{}
	e := newEngine(p)
	e.OnSnapshot([]Client{
		voice(1, 10, "a", 20*time.Minute),
		voice(2, 20, "b", 20*time.Minute),
	})
	if len(e.ActiveChannels()) != 2 {
		t.Fatalf("precondition: 2 active, got %v", e.ActiveChannels())
	}
	// User in channel 10 returns; channel 20 stays idle.
	e.OnSnapshot([]Client{
		voice(1, 10, "a", 2*time.Second),
		voice(2, 20, "b", 20*time.Minute),
	})
	if got := e.ActiveChannels(); !reflect.DeepEqual(got, []int{20}) {
		t.Fatalf("active = %v, want [20] (only channel 10 stopped)", got)
	}
	if len(p.stopped) != 1 {
		t.Fatalf("expected exactly 1 stop, got %v", p.stopped)
	}
}

func TestEngine_StopsWhenEveryoneLeaves(t *testing.T) {
	p := &fakePlayer{}
	e := newEngine(p)
	e.OnSnapshot([]Client{voice(1, 10, "a", 20*time.Minute)})
	e.OnSnapshot([]Client{}) // channel emptied
	if e.Playing() {
		t.Fatalf("should stop when channel empties, active=%v", e.ActiveChannels())
	}
}

func TestEngine_OnTalkStartStopsThatChannel(t *testing.T) {
	p := &fakePlayer{}
	e := newEngine(p)
	e.OnSnapshot([]Client{
		voice(1, 10, "a", 20*time.Minute),
		voice(2, 20, "b", 20*time.Minute),
	})
	if acted := e.OnTalkStart(1); !acted {
		t.Fatal("talk from a user in an active channel should stop that channel")
	}
	if got := e.ActiveChannels(); !reflect.DeepEqual(got, []int{20}) {
		t.Fatalf("active = %v, want [20]", got)
	}
}

func TestEngine_OnTalkStartUnknownClidNoop(t *testing.T) {
	p := &fakePlayer{}
	e := newEngine(p)
	e.OnSnapshot([]Client{voice(1, 10, "a", 20*time.Minute)})
	if acted := e.OnTalkStart(999); acted {
		t.Fatal("unknown clid should not act")
	}
}

func TestEngine_AllowCIDsRestrictsChannels(t *testing.T) {
	c := cfg()
	c.AllowCIDs = map[int]bool{15276: true}
	p := &fakePlayer{}
	e := NewEngine(EngineConfig{Detect: c, Player: p, Tracks: []string{"a.wav"},
		Names: []string{"x"}, Rand: rand.New(rand.NewPCG(1, 2))})
	e.OnSnapshot([]Client{
		voice(1, 10, "a", 20*time.Minute),    // not allowed
		voice(2, 15276, "b", 20*time.Minute), // allowed
	})
	if got := e.ActiveChannels(); !reflect.DeepEqual(got, []int{15276}) {
		t.Fatalf("active = %v, want [15276] (allow-list honoured)", got)
	}
}

func TestEngine_StartFailureLeavesChannelInactive(t *testing.T) {
	p := &fakePlayer{startErr: errors.New("audiobot down")}
	e := newEngine(p)
	e.OnSnapshot([]Client{voice(1, 10, "a", 20*time.Minute)})
	if e.Playing() {
		t.Fatal("failed spawn must not register a session")
	}
	p.startErr = nil
	e.OnSnapshot([]Client{voice(1, 10, "a", 20*time.Minute)})
	if !e.Playing() {
		t.Fatal("should start once player recovers")
	}
}

func TestEngine_StopForTalkCooldownPreventsImmediateRespawn(t *testing.T) {
	clk := &fakeClock{t: time.Unix(1000, 0)}
	p := &fakePlayer{}
	e := NewEngine(EngineConfig{
		Detect: cfg(), Player: p, Tracks: []string{"a.wav"}, Names: []string{"x"},
		TalkCooldown: 60 * time.Second, Now: clk.now, Rand: rand.New(rand.NewPCG(1, 2)),
	})
	clients := []Client{voice(1, 10, "a", 20*time.Minute)}
	e.OnSnapshot(clients)
	if !e.Playing() {
		t.Fatal("precondition: playing")
	}
	// User talks -> stop with cooldown.
	e.StopForTalk(10)
	if e.Playing() {
		t.Fatal("StopForTalk should have stopped the session")
	}
	// The channel still looks idle to the tracker; within cooldown it must NOT respawn.
	e.OnSnapshot(clients)
	if e.Playing() {
		t.Fatalf("should not respawn during cooldown, active=%v", e.ActiveChannels())
	}
	// After the cooldown elapses, it may serenade again.
	clk.add(61 * time.Second)
	e.OnSnapshot(clients)
	if !e.Playing() {
		t.Fatal("should respawn after cooldown elapses")
	}
}

func TestEngine_ShutdownStopsAll(t *testing.T) {
	p := &fakePlayer{}
	e := newEngine(p)
	e.OnSnapshot([]Client{
		voice(1, 10, "a", 20*time.Minute),
		voice(2, 20, "b", 20*time.Minute),
	})
	e.Shutdown()
	if e.Playing() {
		t.Fatalf("shutdown should stop all, active=%v", e.ActiveChannels())
	}
	if len(p.stopped) != 2 {
		t.Fatalf("expected 2 stops on shutdown, got %v", p.stopped)
	}
}
