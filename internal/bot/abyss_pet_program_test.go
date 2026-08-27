package bot

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"ts3news/internal/content"
)

func TestAbyssPetProgramRulesAreBounded(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, time.August, 27, 12, 0, 0, 0, time.UTC)
	profile := abyssPetProfile{DaycareSince: now.Add(-3 * time.Hour).Format(time.RFC3339)}
	if got := abyssPetDaycareXP(profile, now); got != 15 {
		t.Fatalf("daycare XP = %d, want 15", got)
	}
	if !profile.busy(now) {
		t.Fatal("daycare companion is not busy")
	}
	profile = abyssPetProfile{
		ExpeditionKind: "dust", ExpeditionUntil: now.Add(-time.Minute).Format(time.RFC3339),
	}
	if !abyssPetExpeditionReady(profile, now) || !profile.busy(now) {
		t.Fatal("completed unclaimed expedition must be ready and remain busy")
	}
	profile = abyssPetProfile{GiftUntil: now.Add(time.Hour).Format(time.RFC3339)}
	if !profile.busy(now) || profile.busy(now.Add(2*time.Hour)) {
		t.Fatal("gift reservation must be busy only until its authoritative expiry")
	}
	if material, amount := abyssPetExpeditionReward("dust", 300); material != "dust" || amount != 50 {
		t.Fatalf("bounded expedition reward = %s %d, want dust 50", material, amount)
	}
	if gain := abyssPetFusionGain(1, 1_000_000); gain != 100_000 {
		t.Fatalf("fusion gain = %d, want 100000", gain)
	}
}

func TestAbyssBestiaryCapturePercentRejectsUnknownLegacyFamily(t *testing.T) {
	t.Parallel()

	if got := abyssBestiaryCapturePercent(abyssLegacyBestiaryFamily); got != 0 {
		t.Fatalf("legacy capture percent = %d, want 0", got)
	}
	if got := abyssBestiaryCapturePercent(string(content.MobElite)); got != 35 {
		t.Fatalf("elite capture percent = %d, want 35", got)
	}
}

func TestAbyssPetGiftCodesUseSecureBoundedAlphabet(t *testing.T) {
	t.Parallel()
	seen := make(map[string]struct{}, 64)
	valid := regexp.MustCompile(`^[A-Z2-7]{15}$`)
	for range 64 {
		code, err := abyssPetGiftCode()
		if err != nil {
			t.Fatal(err)
		}
		if !valid.MatchString(code) {
			t.Fatalf("gift code %q has invalid format", code)
		}
		if _, exists := seen[code]; exists {
			t.Fatalf("duplicate gift code %q", code)
		}
		seen[code] = struct{}{}
	}
}

func TestAbyssPetProgramUIAndMigrationAreComplete(t *testing.T) {
	t.Parallel()
	partial, err := webAssets.ReadFile("webassets/abyss_social.html")
	if err != nil {
		t.Fatal(err)
	}
	page, err := webAssets.ReadFile("webassets/abyss.html")
	if err != nil {
		t.Fatal(err)
	}
	styles, err := webAssets.ReadFile("webassets/abyss_social.css")
	if err != nil {
		t.Fatal(err)
	}
	for _, token := range []string{
		"Choose consumable", "daycare_start", "expedition_start", "pet/fusion", "pet/cosmetic",
		"pet/revive", "pet/gift/create", "pet/gift/claim", "Companion power board",
	} {
		if !strings.Contains(string(partial), token) {
			t.Errorf("companion stable UI is missing %q", token)
		}
	}
	if !strings.Contains(string(page), "Active companions") || !strings.Contains(string(page), "data-capture-pct") {
		t.Error("Abyss page is missing active-companion or Bestiary capture-rate data")
	}
	for _, token := range []string{".ab-pet-activity", ".ab-pet-power-board", ".ab-side-pets"} {
		if !strings.Contains(string(styles), token) {
			t.Errorf("companion stable CSS is missing %q", token)
		}
	}
	root := filepath.Clean(filepath.Join("..", ".."))
	up, err := os.ReadFile(filepath.Join(root, "internal", "db", "migrations", "0099_abyss_pet_gifts.up.sql"))
	if err != nil {
		t.Fatal(err)
	}
	down, err := os.ReadFile(filepath.Join(root, "internal", "db", "migrations", "0099_abyss_pet_gifts.down.sql"))
	if err != nil {
		t.Fatal(err)
	}
	for _, token := range []string{"pet_id BIGINT NOT NULL UNIQUE", "sender_uid", "recipient_uid", "expires_at"} {
		if !strings.Contains(string(up), token) {
			t.Errorf("pet gift migration is missing %q", token)
		}
	}
	if !strings.Contains(string(down), "DROP TABLE IF EXISTS abyss_pet_gifts") {
		t.Error("pet gift migration is not reversible")
	}
}
