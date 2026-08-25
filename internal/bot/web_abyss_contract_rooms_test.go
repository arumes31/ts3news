package bot

import (
	"strings"
	"testing"
)

func TestAbyssEchoRewardKeepsOriginalAcrossConsecutiveRooms(t *testing.T) {
	if reward, percent := abyssEchoReward(1_000, 0); reward != 500 || percent != 50 {
		t.Fatalf("first echo = %d at %d%%, want 500 at 50%%", reward, percent)
	}
	if reward, percent := abyssEchoReward(1_000, 1); reward != 750 || percent != 75 {
		t.Fatalf("consecutive echo = %d at %d%%, want 750 at 75%%", reward, percent)
	}
	if reward, _ := abyssEchoReward(0, 2); reward != 0 {
		t.Fatalf("empty echo reward = %d, want 0", reward)
	}
}

func TestAbyssRunFourthBountyDoublesOnce(t *testing.T) {
	flags := map[string]int64{
		abyssRunFlagBountyActive:      1,
		abyssRunFlagBountyReward:      900,
		abyssRunFlagBountiesCompleted: 3,
	}
	reward, doubled := settleAbyssRunBounty(flags)
	if reward != 1_800 || !doubled {
		t.Fatalf("fourth bounty = %d, doubled %t; want 1800, true", reward, doubled)
	}
	if flags[abyssRunFlagBountiesCompleted] != 4 || flags[abyssRunFlagBountyActive] != 0 || flags[abyssRunFlagBountyReward] != 0 {
		t.Fatalf("settled bounty flags = %#v", flags)
	}
	if reward, doubled = settleAbyssRunBounty(flags); reward != 0 || doubled {
		t.Fatalf("inactive bounty settled again = %d, %t", reward, doubled)
	}
}

func TestAbyssContractRoomEnvelopeUsesRunState(t *testing.T) {
	flags := map[string]int64{
		abyssRunFlagEchoOriginal:      2_000,
		abyssRunFlagEchoStreak:        1,
		abyssRunFlagBountiesCompleted: 3,
	}
	echo := map[string]any{"type": "echo_floor"}
	enrichAbyssContractRoom(echo, flags, 12)
	if echo["echo_reward"] != int64(1_500) || echo["echo_percent"] != 75 {
		t.Fatalf("echo event = %#v", echo)
	}
	bounty := map[string]any{"type": "bounty_board"}
	enrichAbyssContractRoom(bounty, flags, 12)
	if bounty["bounty_reward"] != int64(1_300) || bounty["double_reward"] != true || bounty["bounties_completed"] != int64(3) {
		t.Fatalf("bounty event = %#v", bounty)
	}
}

func TestAbyssEchoChainBreaksOnlyAfterAnotherFloor(t *testing.T) {
	flags := map[string]int64{
		abyssRunFlagEchoOriginal:    1_000,
		abyssRunFlagEchoStreak:      1,
		abyssRunFlagEchoJustClaimed: 1,
	}
	rememberAbyssNonCombatReward(flags, 100)
	if flags[abyssRunFlagEchoOriginal] != 1_000 || flags[abyssRunFlagEchoStreak] != 1 || flags[abyssRunFlagEchoJustClaimed] != 0 {
		t.Fatalf("claimed echo did not preserve original chain: %#v", flags)
	}
	rememberAbyssNonCombatReward(flags, 200)
	if flags[abyssRunFlagEchoOriginal] != 200 || flags[abyssRunFlagEchoStreak] != 0 {
		t.Fatalf("intervening floor did not reset echo chain: %#v", flags)
	}
}

func TestAbyssContractRoomUIContracts(t *testing.T) {
	page, err := webAssets.ReadFile("webassets/abyss.html")
	if err != nil {
		t.Fatal(err)
	}
	joined := string(page)
	for _, contract := range []string{
		"echo_floor", "echo_claim", "echo_percent", "bounty_board", "bounty_accept",
		"bounties_completed", "double_reward", "next combat pays",
	} {
		if !strings.Contains(joined, contract) {
			t.Errorf("contract-room UI is missing %q", contract)
		}
	}
}
