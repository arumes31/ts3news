package bot

import (
	"encoding/json"
	"fmt"
	"net/http"
)

const (
	abyssRunFlagEchoOriginal      = "echo_original_reward"
	abyssRunFlagEchoStreak        = "echo_floor_streak"
	abyssRunFlagEchoJustClaimed   = "echo_just_claimed"
	abyssRunFlagBountyActive      = "run_bounty_active"
	abyssRunFlagBountyReward      = "run_bounty_reward"
	abyssRunFlagBountiesCompleted = "run_bounties_completed"
)

type abyssContractRoomEvent struct {
	Type              string `json:"type"`
	PreviousReward    int64  `json:"previous_reward"`
	EchoReward        int64  `json:"echo_reward"`
	EchoPercent       int    `json:"echo_percent"`
	BountyReward      int64  `json:"bounty_reward"`
	BountiesCompleted int64  `json:"bounties_completed"`
	BountyActive      bool   `json:"bounty_active"`
	DoubleReward      bool   `json:"double_reward"`
}

func abyssEchoReward(original, streak int64) (reward int64, percent int) {
	if original <= 0 {
		return 0, 50
	}
	percent = 50
	if streak > 0 {
		percent = 75
	}
	return original * int64(percent) / 100, percent
}

func abyssRunBountyReward(depth int) int64 {
	return int64(400 + max(1, depth)*75)
}

func enrichAbyssContractRoom(state map[string]any, flags map[string]int64, depth int) {
	typ, _ := state["type"].(string)
	switch typ {
	case "echo_floor":
		reward, percent := abyssEchoReward(flags[abyssRunFlagEchoOriginal], flags[abyssRunFlagEchoStreak])
		state["previous_reward"] = flags[abyssRunFlagEchoOriginal]
		state["echo_reward"] = reward
		state["echo_percent"] = percent
	case "bounty_board":
		completed := flags[abyssRunFlagBountiesCompleted]
		reward := abyssRunBountyReward(depth)
		state["bounty_reward"] = reward
		state["bounties_completed"] = completed
		state["bounty_active"] = flags[abyssRunFlagBountyActive] == 1
		state["double_reward"] = completed == 3
	}
}

func settleAbyssRunBounty(flags map[string]int64) (reward int64, doubled bool) {
	if flags[abyssRunFlagBountyActive] != 1 {
		return 0, false
	}
	reward = max(flags[abyssRunFlagBountyReward], 0)
	doubled = flags[abyssRunFlagBountiesCompleted] == 3
	if doubled {
		reward *= 2
	}
	flags[abyssRunFlagBountyActive] = 0
	flags[abyssRunFlagBountyReward] = 0
	flags[abyssRunFlagBountiesCompleted]++
	return reward, doubled
}

func rememberAbyssFloorReward(flags map[string]int64, reward int64) {
	flags[abyssRunFlagEchoOriginal] = max(reward, 0)
	flags[abyssRunFlagEchoStreak] = 0
	flags[abyssRunFlagEchoJustClaimed] = 0
}

func rememberAbyssNonCombatReward(flags map[string]int64, reward int64) {
	if flags[abyssRunFlagEchoJustClaimed] == 1 {
		flags[abyssRunFlagEchoJustClaimed] = 0
		return
	}
	rememberAbyssFloorReward(flags, reward)
}

func appendAbyssSecondaryGoal(existing, addition string) string {
	if existing == "" {
		return addition
	}
	return existing + " · " + addition
}

func (b *Bot) enrichAbyssContractEvent(uid string, state map[string]any) {
	depth, _ := state["depth"].(float64)
	enrichAbyssContractRoom(state, b.loadRunFlags(uid), int(depth))
}

func (s *WebServer) handleAbyssContractRoom(w http.ResponseWriter, uid string, run abyssRun, action string) bool {
	var event abyssContractRoomEvent
	if json.Unmarshal([]byte(run.EventState), &event) != nil {
		return false
	}
	if event.Type != "echo_floor" && event.Type != "bounty_board" {
		return false
	}

	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return true
	}
	defer func() { _ = tx.Rollback() }()
	flags, err := loadAbyssRunFlagsInTx(tx, uid)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return true
	}

	var escrow int64
	msg := ""
	switch event.Type {
	case "echo_floor":
		if action != "echo_claim" && action != "echo_leave" {
			writeJSON(w, map[string]any{"ok": false, "error": "invalid echo choice"})
			return true
		}
		reward, percent := abyssEchoReward(flags[abyssRunFlagEchoOriginal], flags[abyssRunFlagEchoStreak])
		if action == "echo_claim" && reward > 0 {
			if err := tx.QueryRow(`UPDATE abyss_active SET escrow=escrow+$1, event_state=NULL,
				last_action_at=NOW() WHERE client_uid=$2 RETURNING escrow`, reward, uid).Scan(&escrow); err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "db"})
				return true
			}
			flags[abyssRunFlagEchoStreak]++
			flags[abyssRunFlagEchoJustClaimed] = 1
			msg = fmt.Sprintf("🔁 Echo recovered %d%% of the original floor reward: +%d cache.", percent, reward)
		} else {
			if err := tx.QueryRow(`UPDATE abyss_active SET event_state=NULL, last_action_at=NOW()
				WHERE client_uid=$1 RETURNING escrow`, uid).Scan(&escrow); err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "db"})
				return true
			}
			flags[abyssRunFlagEchoStreak] = 0
			flags[abyssRunFlagEchoJustClaimed] = 0
			msg = "The echo fades without repeating its reward."
		}

	case "bounty_board":
		if action != "bounty_accept" && action != "bounty_leave" {
			writeJSON(w, map[string]any{"ok": false, "error": "invalid bounty choice"})
			return true
		}
		if action == "bounty_accept" {
			if flags[abyssRunFlagBountyActive] == 1 {
				writeJSON(w, map[string]any{"ok": false, "error": "a run bounty is already active"})
				return true
			}
			reward := abyssRunBountyReward(run.Depth)
			flags[abyssRunFlagBountyActive] = 1
			flags[abyssRunFlagBountyReward] = reward
			msg = fmt.Sprintf("🎯 Bounty accepted: win your next combat for +%d cache.", reward)
			if flags[abyssRunFlagBountiesCompleted] == 3 {
				msg += " This is your fourth contract — its payout will double."
			}
		} else {
			msg = "You leave the bounty unsigned."
		}
		if err := tx.QueryRow(`UPDATE abyss_active SET event_state=NULL, last_action_at=NOW()
			WHERE client_uid=$1 RETURNING escrow`, uid).Scan(&escrow); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return true
		}
	}

	if err := saveAbyssRunFlagsInTx(tx, uid, flags); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return true
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return true
	}
	writeJSON(w, map[string]any{
		"ok": true, "resolved": true, "msg": msg, "escrow": escrow,
	})
	return true
}
