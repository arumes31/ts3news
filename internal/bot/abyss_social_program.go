package bot

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"fmt"
	"strings"
	"time"
	"unicode"
)

const (
	abyssDailyServerGoal       = 10_000
	abyssGuildWeeklyGoal       = 500
	abyssFriendCheerFights     = 3
	abyssFriendCheerBonusPct   = 5
	abyssMentorRewardTokens    = 3
	abyssReferralRewardTokens  = 20
	abyssRaidMaxMembers        = 5
	abyssPartyMaxMembers       = 3
	abyssSocialMessageMaxRunes = 240
)

func abyssSocialCode(prefix string) (string, error) {
	random := make([]byte, 7)
	if _, err := rand.Read(random); err != nil {
		return "", err
	}
	code := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(random)
	return prefix + code, nil
}

func abyssReferralCode(uid string) string {
	sum := sha256.Sum256([]byte("abyss-referral:" + uid))
	return "ABYSS-" + strings.ToUpper(fmt.Sprintf("%x", sum[:5]))
}

func abyssSocialTextValid(value string, maxRunes int) bool {
	value = strings.TrimSpace(value)
	if value == "" || len([]rune(value)) > maxRunes {
		return false
	}
	for _, char := range value {
		if char < 32 || char == 127 {
			return false
		}
	}
	return true
}

func abyssSocialNameValid(value string, minRunes, maxRunes int) bool {
	value = strings.TrimSpace(value)
	count := len([]rune(value))
	if count < minRunes || count > maxRunes {
		return false
	}
	for _, char := range value {
		if !unicode.IsLetter(char) && !unicode.IsNumber(char) && char != ' ' && char != '-' && char != '_' {
			return false
		}
	}
	return true
}

func abyssHelperDepthReward(depth int) int {
	return min(20, 5+max(0, depth)/10)
}

func abyssSocialGoalPercent(floors int64) int {
	return min(100, int(max(0, floors)*100/abyssDailyServerGoal))
}

func abyssSocialOnline(lastSeen time.Time, now time.Time) bool {
	return !lastSeen.IsZero() && now.Sub(lastSeen) <= 10*time.Minute
}
