package bot

import (
	"encoding/json"
	"strings"
	"time"
	"unicode"

	"ts3news/internal/content"
)

const (
	abyssPetBaseCap = 3
	abyssPetMaxCap  = 5
)

type abyssPetProfile struct {
	Heal            *bool    `json:"heal,omitempty"`
	XP              int      `json:"xp,omitempty"`
	Favorite        bool     `json:"favorite,omitempty"`
	FusionRank      int      `json:"fusion_rank,omitempty"`
	Shiny           bool     `json:"shiny,omitempty"`
	BossVariant     bool     `json:"boss_variant,omitempty"`
	Cosmetic        string   `json:"cosmetic,omitempty"`
	OwnedCosmetics  []string `json:"owned_cosmetics,omitempty"`
	BarkStyle       string   `json:"bark_style,omitempty"`
	DaycareSince    string   `json:"daycare_since,omitempty"`
	ExpeditionUntil string   `json:"expedition_until,omitempty"`
	ExpeditionKind  string   `json:"expedition_kind,omitempty"`
}

func decodeAbyssPetProfile(raw string) abyssPetProfile {
	var profile abyssPetProfile
	if raw != "" {
		if err := json.Unmarshal([]byte(raw), &profile); err != nil {
			return abyssPetProfile{}
		}
	}
	profile.XP = max(0, profile.XP)
	profile.FusionRank = max(0, profile.FusionRank)
	return profile
}

func encodeAbyssPetProfile(profile abyssPetProfile) (string, error) {
	encoded, err := json.Marshal(profile)
	return string(encoded), err
}

func (profile abyssPetProfile) healEnabled() bool {
	return profile.Heal == nil || *profile.Heal
}

func (profile abyssPetProfile) busy(now time.Time) bool {
	if profile.DaycareSince != "" {
		return true
	}
	until, err := time.Parse(time.RFC3339, profile.ExpeditionUntil)
	return err == nil && until.After(now)
}

func abyssPetClass(mobType content.MobType) string {
	switch mobType {
	case content.MobBoss, content.MobMiniboss:
		return "tank"
	case content.MobElite:
		return "damage"
	default:
		return "support"
	}
}

func applyAbyssPetClass(pet *content.Mob) {
	if pet == nil {
		return
	}
	pet.PetClass = abyssPetClass(pet.Type)
	switch pet.PetClass {
	case "tank":
		pet.Stats.DEF = pet.Stats.DEF * 115 / 100
		pet.MaxHP = pet.MaxHP * 110 / 100
		pet.Stats.HP = min(pet.Stats.HP, pet.MaxHP)
	case "damage":
		pet.Stats.STR = pet.Stats.STR * 115 / 100
	default:
		pet.Stats.SPD = pet.Stats.SPD * 110 / 100
	}
}

func abyssPetLoyaltyBonusPct(loyalty int) int {
	switch {
	case loyalty >= 90:
		return 5
	case loyalty >= 70:
		return 3
	case loyalty >= 50:
		return 1
	default:
		return 0
	}
}

func abyssPetBetrayalChance(loyalty int, talentReduction float64) float64 {
	loyalty = min(100, max(0, loyalty))
	chance := 0.03 * float64(100-loyalty) / 100
	return max(0.0, chance-talentReduction)
}

func abyssPetCaptureChance(mobType content.MobType) float64 {
	switch mobType {
	case content.MobBoss:
		return 0.01
	case content.MobMiniboss:
		return 0.12
	case content.MobElite:
		return 0.35
	default:
		return 0.50
	}
}

func abyssPetCaptureLimitWithBonus(mindControlLevel, talentBonus int) int {
	baseLimit := min(abyssPetBaseCap, max(0, mindControlLevel))
	return min(abyssPetMaxCap, baseLimit+max(0, talentBonus))
}

func abyssPetXPThreshold(level int) int {
	return max(100, max(1, level)*100)
}

func abyssPetNameValid(name string) bool {
	name = strings.TrimSpace(name)
	if name == "" || len([]rune(name)) > 24 {
		return false
	}
	for _, char := range name {
		if !unicode.IsLetter(char) && !unicode.IsNumber(char) && char != ' ' && char != '-' && char != '\'' {
			return false
		}
	}
	compact := strings.ToLower(strings.Map(func(char rune) rune {
		if unicode.IsLetter(char) || unicode.IsNumber(char) {
			return char
		}
		return -1
	}, name))
	for _, blocked := range []string{"fuck", "shit", "cunt", "nigger", "nazi"} {
		if strings.Contains(compact, blocked) {
			return false
		}
	}
	return true
}

func abyssPetBark(style, name, event string) string {
	if style == "quiet" {
		return ""
	}
	switch event {
	case "heal":
		if style == "bold" {
			return "🐾 " + name + " barks: Back on your feet!"
		}
		return "🐾 " + name + " gives an encouraging chirp."
	case "kill":
		if style == "bold" {
			return "🐾 " + name + " roars in triumph!"
		}
		return "🐾 " + name + " celebrates the takedown."
	default:
		return ""
	}
}
