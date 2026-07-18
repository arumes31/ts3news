package idely

import (
	"reflect"
	"testing"
	"time"
)

func cfg() Config {
	return Config{
		IdleThreshold: 15 * time.Minute,
		ActiveGrace:   5 * time.Second,
		ExcludeUIDs:   map[string]bool{"idely-uid": true, "audiobot-uid": true},
	}
}

func voice(clid, cid int, uid string, idle time.Duration) Client {
	return Client{CLID: clid, CID: cid, UID: uid, Type: 0, Idle: idle}
}

func TestIdleChannels_AllIdle(t *testing.T) {
	clients := []Client{
		voice(1, 10, "a", 20*time.Minute),
		voice(2, 10, "b", 16*time.Minute),
		voice(3, 20, "c", 2*time.Minute), // active channel
	}
	got := cfg().IdleChannels(clients)
	if want := []int{10}; !reflect.DeepEqual(got, want) {
		t.Fatalf("IdleChannels = %v, want %v", got, want)
	}
}

func TestIdleChannels_OneActiveBlocksChannel(t *testing.T) {
	clients := []Client{
		voice(1, 10, "a", 20*time.Minute),
		voice(2, 10, "b", 30*time.Second), // one user active => channel not idle
	}
	if got := cfg().IdleChannels(clients); len(got) != 0 {
		t.Fatalf("IdleChannels = %v, want none", got)
	}
}

func TestIdleChannels_TalkingBlocksChannel(t *testing.T) {
	c := voice(1, 10, "a", 40*time.Minute)
	c.Talking = true
	clients := []Client{c, voice(2, 10, "b", 40*time.Minute)}
	if got := cfg().IdleChannels(clients); len(got) != 0 {
		t.Fatalf("talking user should block idle detection, got %v", got)
	}
}

func TestIdleChannels_ExcludesOwnClientsAndQuery(t *testing.T) {
	clients := []Client{
		voice(1, 10, "a", 20*time.Minute),
		voice(2, 10, "idely-uid", 0),                   // our detector client, excluded
		voice(3, 10, "audiobot-uid", 0),                // the audio bot, excluded
		{CLID: 4, CID: 10, UID: "q", Type: 1, Idle: 0}, // query client, excluded
	}
	if got := cfg().IdleChannels(clients); !reflect.DeepEqual(got, []int{10}) {
		t.Fatalf("IdleChannels = %v, want [10] (excluded clients ignored)", got)
	}
}

func TestIdleChannels_EmptyOrOnlyExcludedChannelNotCandidate(t *testing.T) {
	clients := []Client{
		voice(1, 10, "idely-uid", 999*time.Minute), // only our client here
	}
	if got := cfg().IdleChannels(clients); len(got) != 0 {
		t.Fatalf("channel with only excluded clients must not qualify, got %v", got)
	}
}

func TestChannelActive_TalkingStops(t *testing.T) {
	talker := voice(1, 10, "a", 40*time.Minute)
	talker.Talking = true
	clients := []Client{talker, voice(2, 10, "b", 40*time.Minute)}
	if !cfg().ChannelActive(clients, 10) {
		t.Fatal("a talking user must make the channel active")
	}
}

func TestChannelActive_MovementResetsIdleStops(t *testing.T) {
	clients := []Client{
		voice(1, 10, "a", 2*time.Second), // idle reset below ActiveGrace
		voice(2, 10, "b", 40*time.Minute),
	}
	if !cfg().ChannelActive(clients, 10) {
		t.Fatal("a user with idle <= ActiveGrace must make the channel active")
	}
}

func TestChannelActive_StillAllIdleDoesNotStop(t *testing.T) {
	clients := []Client{
		voice(1, 10, "a", 40*time.Minute),
		voice(2, 10, "b", 20*time.Minute),
	}
	if cfg().ChannelActive(clients, 10) {
		t.Fatal("all users still idle => channel not active")
	}
}

func TestChannelActive_EveryoneLeftStops(t *testing.T) {
	clients := []Client{voice(1, 99, "a", time.Minute)} // nobody in cid 10
	if !cfg().ChannelActive(clients, 10) {
		t.Fatal("empty channel must be treated as active (stop)")
	}
}

func TestPickIdleChannel_PrefersMostUsers(t *testing.T) {
	clients := []Client{
		voice(1, 10, "a", 20*time.Minute),
		voice(2, 20, "b", 20*time.Minute),
		voice(3, 20, "c", 20*time.Minute), // channel 20 has 2 idle users
	}
	cid, ok := cfg().PickIdleChannel(clients)
	if !ok || cid != 20 {
		t.Fatalf("PickIdleChannel = (%d,%v), want (20,true)", cid, ok)
	}
}

func TestPickIdleChannel_NoneQualify(t *testing.T) {
	clients := []Client{voice(1, 10, "a", time.Minute)}
	if _, ok := cfg().PickIdleChannel(clients); ok {
		t.Fatal("no idle channels should yield ok=false")
	}
}
