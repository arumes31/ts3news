// Package idely implements the "Idely" idle-music subsystem: a dedicated,
// always-on TeamSpeak client (driven over its own ClientQuery interface) watches
// every occupied channel, and when every real user in a channel has been idle
// past a threshold it dispatches a separate audio bot (TS3AudioBot) to join that
// channel and play lo-fi music. Playback stops the moment anyone in the channel
// becomes active again (talks or moves).
//
// This file holds the pure decision logic: given a snapshot of the clients on the
// server, decide which channels are "all idle" and whether an occupied channel
// has become active again. It has no I/O so it is fully unit-testable.
package idely

import "time"

// Client is a snapshot of one client on the server, as read via ClientQuery.
type Client struct {
	CLID     int           // client id (session-scoped)
	CID      int           // current channel id
	UID      string        // client_unique_identifier (stable identity)
	Nickname string        // display name
	Type     int           // 0 = voice client, 1 = query/ClientQuery client
	Idle     time.Duration // client_idle_time (since last input or >1s of voice)
	Talking  bool          // client_flag_talking, when available
}

// Config parameterises the idle detector.
type Config struct {
	// IdleThreshold is how long every user in a channel must have been idle
	// before Idely considers the channel dormant (e.g. 15m).
	IdleThreshold time.Duration
	// ActiveGrace is the idle time at or below which a user counts as "active
	// again" — used to decide when to stop playback. Kept small (a few seconds)
	// so a user talking or moving resets their idle time below it promptly.
	ActiveGrace time.Duration
	// ExcludeUIDs are identities never counted as real users (the Idely client
	// itself). A channel occupied only by excluded clients is not a candidate.
	ExcludeUIDs map[string]bool
	// ExcludeNicks are display names never counted as real users. Idely names
	// every audio bot from a fixed pool, so excluding that pool keeps the spawned
	// music bots — which each connect with their own identity — from being seen
	// as active users in the very channel they are serenading.
	ExcludeNicks map[string]bool
	// AllowCIDs, when non-empty, restricts Idely to only serenade these channel
	// ids. Empty means every channel is eligible. Useful to scope the subsystem
	// (or a live test) to specific channels.
	AllowCIDs map[int]bool
}

// channelAllowed reports whether cid is eligible given the AllowCIDs whitelist
// (an empty whitelist allows all channels).
func (cfg Config) channelAllowed(cid int) bool {
	return len(cfg.AllowCIDs) == 0 || cfg.AllowCIDs[cid]
}

// isRealUser reports whether c is a human voice client that should count toward
// idle/activity decisions (i.e. not a query client and not one of ours).
func (cfg Config) isRealUser(c Client) bool {
	if c.Type != 0 {
		return false // ServerQuery / ClientQuery client, not a voice user
	}
	if cfg.ExcludeUIDs[c.UID] {
		return false // the Idely client
	}
	if cfg.ExcludeNicks[c.Nickname] {
		return false // one of our audio bots
	}
	return true
}

// IdleChannels returns the channel ids in which at least one real user is present
// and every real user has been idle for at least IdleThreshold. Channels with no
// real users are omitted. The result is deterministic (ascending channel id).
func (cfg Config) IdleChannels(clients []Client) []int {
	type agg struct {
		users   int
		allIdle bool
	}
	byCID := map[int]*agg{}
	// Preserve first-seen channel order is unnecessary; we sort at the end.
	for _, c := range clients {
		if !cfg.isRealUser(c) || !cfg.channelAllowed(c.CID) {
			continue
		}
		a := byCID[c.CID]
		if a == nil {
			a = &agg{allIdle: true}
			byCID[c.CID] = a
		}
		a.users++
		if c.Talking || c.Idle < cfg.IdleThreshold {
			a.allIdle = false
		}
	}

	var out []int
	for cid, a := range byCID {
		if a.users > 0 && a.allIdle {
			out = append(out, cid)
		}
	}
	sortInts(out)
	return out
}

// ChannelActive reports whether channel cid currently has a real user who is
// active again — talking, or idle at/below ActiveGrace — meaning any playback
// there should stop. It also returns true when the channel has no real users
// left (everyone left), so the caller stops in that case too.
func (cfg Config) ChannelActive(clients []Client, cid int) bool {
	realUsers := 0
	for _, c := range clients {
		if c.CID != cid || !cfg.isRealUser(c) {
			continue
		}
		realUsers++
		if c.Talking || c.Idle <= cfg.ActiveGrace {
			return true
		}
	}
	return realUsers == 0
}

// PickIdleChannel returns the best channel to start playing in given the current
// snapshot, preferring the channel with the most idle users (ties broken by
// lowest channel id for determinism). It returns ok=false if none qualify.
func (cfg Config) PickIdleChannel(clients []Client) (cid int, ok bool) {
	idle := cfg.IdleChannels(clients)
	if len(idle) == 0 {
		return 0, false
	}
	count := map[int]int{}
	for _, c := range clients {
		if cfg.isRealUser(c) {
			count[c.CID]++
		}
	}
	best, bestN := idle[0], count[idle[0]]
	for _, c := range idle[1:] {
		if count[c] > bestN { // idle is ascending, so ties keep the lower id
			best, bestN = c, count[c]
		}
	}
	return best, true
}

// sortInts is a tiny insertion sort (slices are small: number of occupied
// channels) to avoid pulling in sort for a hot, tiny input.
func sortInts(a []int) {
	for i := 1; i < len(a); i++ {
		for j := i; j > 0 && a[j-1] > a[j]; j-- {
			a[j-1], a[j] = a[j], a[j-1]
		}
	}
}
