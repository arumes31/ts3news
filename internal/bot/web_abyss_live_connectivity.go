package bot

import "time"

func (c *abyssLiveCombat) openMemberConnection(uid string, now time.Time) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.participants[uid] {
		return false
	}
	if c.connections == nil {
		c.connections = make(map[string]int, len(c.participants))
	}
	c.connections[uid]++
	c.ensureSocialLocked()
	c.social.lastSeen[uid] = now
	return true
}

// closeMemberConnection grants one bounded, server-owned planning extension
// when the last event stream for a participant disappears. Multiple browser
// streams cannot stack grace, and reconnecting cannot farm more time in the
// same round.
func (c *abyssLiveCombat) closeMemberConnection(uid string, now time.Time) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.connections == nil || c.connections[uid] <= 0 {
		return false
	}
	c.connections[uid]--
	if c.connections[uid] > 0 {
		return false
	}
	delete(c.connections, uid)
	if c.phase != "planning" || c.round <= 0 || !now.Before(c.deadline) {
		return false
	}
	if c.reconnectRound == nil {
		c.reconnectRound = make(map[string]int, len(c.participants))
	}
	if c.reconnectRound[uid] == c.round {
		return false
	}
	c.reconnectRound[uid] = c.round
	c.deadline = c.deadline.Add(abyssLiveConnectivityGrace)
	c.version++
	if c.pauseReason == "" {
		c.pauseReason = "Connection interrupted — action timer protected"
	}
	if c.deadlineSignal != nil {
		select {
		case c.deadlineSignal <- struct{}{}:
		default:
		}
	}
	return true
}
