package bot

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
	"ts3news/internal/rift"
)

func TestRiftSoundEventNamesHaveAudioSupport(t *testing.T) {
	audio, err := webAssets.ReadFile("webassets/rift_audio.js")
	if err != nil {
		t.Fatal(err)
	}
	supported := map[string]bool{}
	for _, match := range regexp.MustCompile(`case '([^']+)'`).FindAllSubmatch(audio, -1) {
		supported[string(match[1])] = true
	}
	if len(supported) == 0 {
		t.Fatal("audio cue switch not found")
	}
	required := map[string]bool{}
	paths, err := filepath.Glob("../rift/*.go")
	if err != nil {
		t.Fatal(err)
	}
	events := regexp.MustCompile(`\.(?:event|eventAtHeight)\("([^"]+)"`)
	for _, path := range paths {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		source, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		for _, match := range events.FindAllSubmatch(source, -1) {
			required[string(match[1])] = true
		}
	}
	for _, name := range []string{"rift.js", "rift_renderer.js", "rift_feedback.js"} {
		source, err := webAssets.ReadFile("webassets/" + name)
		if err != nil {
			t.Fatal(err)
		}
		for _, pattern := range []string{`(?:RiftAudio|audio)\.play\('([^']+)'`, `sound:'([^']+)'`, `cue\('([^']+)',(?:'[^']*'|[^,\n]+),true\)`} {
			for _, match := range regexp.MustCompile(pattern).FindAllSubmatch(source, -1) {
				required[string(match[1])] = true
			}
		}
	}
	for _, mob := range riftMobCatalog(time.Unix(100, 0)) {
		actor := rift.AdaptMonster(mob)
		required[actor.HurtCue()] = true
		required[actor.DeathCue()] = true
		required[actor.Shot] = true
	}
	// Expiring projectiles are deliberately silent, including in the caller.
	renderer, _ := webAssets.ReadFile("webassets/rift_renderer.js")
	if !strings.Contains(string(renderer), "event.kind!=='projectile_expire'") {
		t.Fatal("silent projectile expiry is no longer filtered")
	}
	delete(required, "projectile_expire")
	if len(required) < 70 {
		t.Fatal("event producer scan unexpectedly incomplete")
	}
	for name := range required {
		if !supported[name] {
			t.Errorf("sound event %q has no audio cue", name)
		}
	}
	t.Logf("validated %d required sound event names", len(required))
}
