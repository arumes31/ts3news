package idely

import (
	"math/rand/v2"
	"time"
)

// Player is the "voice" side of Idely. Start puts an audio bot into a channel
// playing tracks and returns an opaque handle; Stop tears that instance down.
// The concrete implementation spawns a TS3AudioBot instance per channel; tests
// use a fake.
type Player interface {
	Start(cid int, name string, tracks []string) (handle int, err error)
	Stop(handle int) error
}

// session is one active serenade: an audio-bot instance playing in a channel.
type session struct {
	handle int
	name   string
}

// Engine is the pure decision core of Idely. Fed snapshots of the server's
// clients (and optional real-time talk events), it keeps one audio-bot session
// per idle channel: it spawns a bot for each newly-dormant channel and stops the
// bot for any channel that wakes up (someone talks/moves) or empties. It does no
// I/O itself — all side effects go through Player — so it is fully unit-testable.
//
// Engine is not safe for concurrent use; the runner calls it from one goroutine.
type Engine struct {
	cfg          Config
	player       Player
	tracks       []string
	names        []string
	maxBots      int
	talkCooldown time.Duration
	now          func() time.Time
	rng          *rand.Rand
	logf         func(format string, args ...any)

	sessions      map[int]*session  // channel id -> active session
	cooldownUntil map[int]time.Time // channel id -> earliest time it may be serenaded again
	last          []Client          // most recent snapshot, for clid -> channel resolution
}

// EngineConfig configures an Engine.
type EngineConfig struct {
	Detect  Config
	Player  Player
	Tracks  []string
	Names   []string
	MaxBots int // cap on concurrent audio bots (<=0 => unlimited)
	// TalkCooldown is how long a channel stays off-limits after being stopped
	// because a user talked. Because the headless detector can't hear talk, its
	// idle tracker doesn't know the user is active, so without this the channel
	// would be re-detected as idle and re-serenaded immediately (flapping).
	TalkCooldown time.Duration
	Now          func() time.Time // injectable clock for tests
	Rand         *rand.Rand
	Logf         func(format string, args ...any)
}

// NewEngine builds an Engine.
func NewEngine(c EngineConfig) *Engine {
	r := c.Rand
	if r == nil {
		r = rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64()))
	}
	logf := c.Logf
	if logf == nil {
		logf = func(string, ...any) {}
	}
	names := c.Names
	if len(names) == 0 {
		names = []string{"Idely"}
	}
	now := c.Now
	if now == nil {
		now = time.Now
	}
	return &Engine{
		cfg:           c.Detect,
		player:        c.Player,
		tracks:        c.Tracks,
		names:         names,
		maxBots:       c.MaxBots,
		talkCooldown:  c.TalkCooldown,
		now:           now,
		rng:           r,
		logf:          logf,
		sessions:      map[int]*session{},
		cooldownUntil: map[int]time.Time{},
	}
}

// ActiveChannels returns the channel ids currently being serenaded.
func (e *Engine) ActiveChannels() []int {
	out := make([]int, 0, len(e.sessions))
	for cid := range e.sessions {
		out = append(out, cid)
	}
	sortInts(out)
	return out
}

// Playing reports whether any channel is currently being serenaded.
func (e *Engine) Playing() bool { return len(e.sessions) > 0 }

// Sessions returns a copy of the active channel->bot-handle map, so the caller
// can poll each bot (e.g. for talk activity) without holding engine internals.
func (e *Engine) Sessions() map[int]int {
	out := make(map[int]int, len(e.sessions))
	for cid, s := range e.sessions {
		out[cid] = s.handle
	}
	return out
}

// StopChannel stops and tears down the session serenading cid, if any.
func (e *Engine) StopChannel(cid int, reason string) {
	e.stop(cid, reason)
}

// StopForTalk stops the session serenading cid because a real user talked, and
// puts the channel on cooldown so it is not immediately re-serenaded (the
// headless detector can't hear the talking, so its idle tracker still thinks the
// channel is dormant).
func (e *Engine) StopForTalk(cid int) {
	if _, ok := e.sessions[cid]; !ok {
		return
	}
	e.stop(cid, "user talking")
	if e.talkCooldown > 0 {
		e.cooldownUntil[cid] = e.now().Add(e.talkCooldown)
	}
}

// OnSnapshot reconciles the active sessions with the current set of idle
// channels: it stops sessions whose channel is no longer all-idle, and starts a
// new session (spawning a bot) for each newly-idle channel, up to MaxBots.
func (e *Engine) OnSnapshot(clients []Client) {
	e.last = clients

	idle := e.cfg.IdleChannels(clients)
	idleSet := make(map[int]bool, len(idle))
	for _, cid := range idle {
		idleSet[cid] = true
	}

	// Stop sessions for channels that are no longer all-idle (someone talked,
	// moved, or everyone left — all of which drop the channel out of idleSet).
	for cid := range e.sessions {
		if !idleSet[cid] {
			e.stop(cid, "channel active or empty")
		}
	}

	// Start a session for each newly-idle channel, respecting the bot cap and any
	// post-talk cooldown.
	now := e.now()
	for _, cid := range idle {
		if _, ok := e.sessions[cid]; ok {
			continue
		}
		if until, ok := e.cooldownUntil[cid]; ok {
			if now.Before(until) {
				continue // recently stopped for talk; leave the channel alone
			}
			delete(e.cooldownUntil, cid)
		}
		if e.maxBots > 0 && len(e.sessions) >= e.maxBots {
			e.logf("idely: bot cap (%d) reached; not serenading channel %d yet", e.maxBots, cid)
			break
		}
		e.start(cid)
	}
}

// OnTalkStart is an optional fast path: given a real-time talk-start event for
// clid, it stops the session for that client's channel if one is active. Returns
// true if it acted; false (unknown clid / no session there) means the caller
// should fall back to a snapshot poll.
func (e *Engine) OnTalkStart(clid int) bool {
	for _, c := range e.last {
		if c.CLID != clid || !e.cfg.isRealUser(c) {
			continue
		}
		if _, ok := e.sessions[c.CID]; ok {
			e.stop(c.CID, "talk started")
			return true
		}
	}
	return false
}

// Shutdown stops every active session. Called when the service is exiting.
func (e *Engine) Shutdown() {
	for cid := range e.sessions {
		e.stop(cid, "shutting down")
	}
}

func (e *Engine) start(cid int) {
	name := e.names[e.rng.IntN(len(e.names))]
	tracks := e.shuffledTracks()
	handle, err := e.player.Start(cid, name, tracks)
	if err != nil {
		e.logf("idely: failed to start playback in channel %d: %v", cid, err)
		return
	}
	e.sessions[cid] = &session{handle: handle, name: name}
	e.logf("idely: %q now playing lo-fi in channel %d (%d tracks, %d active)", name, cid, len(tracks), len(e.sessions))
}

func (e *Engine) stop(cid int, reason string) {
	s := e.sessions[cid]
	if s == nil {
		return
	}
	if err := e.player.Stop(s.handle); err != nil {
		e.logf("idely: failed to stop bot in channel %d: %v", cid, err)
		// Still drop the session; leaving it would wedge this channel.
	}
	delete(e.sessions, cid)
	e.logf("idely: stopped playback in channel %d (%s; %d still active)", cid, reason, len(e.sessions))
}

// shuffledTracks returns a copy of the track list in random order.
func (e *Engine) shuffledTracks() []string {
	out := make([]string, len(e.tracks))
	copy(out, e.tracks)
	e.rng.Shuffle(len(out), func(i, j int) { out[i], out[j] = out[j], out[i] })
	return out
}
