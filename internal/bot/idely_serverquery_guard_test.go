package bot

import (
	"errors"
	"testing"

	"ts3news/internal/config"
)

func TestServerQueryHostIsPrivate(t *testing.T) {
	for host, want := range map[string]bool{
		"127.0.0.1": true, "10.1.2.3": true, "192.168.0.5": true, "::1": true, "fd00::1": true,
		"8.8.8.8": false, "2001:4860:4860::8888": false, "": false,
	} {
		if got := serverQueryHostIsPrivate(host); got != want {
			t.Errorf("serverQueryHostIsPrivate(%q) = %v, want %v", host, got, want)
		}
	}
}

func TestSnapshotServerQueryRefusesPlaintextToPublicHost(t *testing.T) {
	_, err := (&Supervisor{}).snapshotServerQuery(&config.Config{TS3Host: "8.8.8.8", IdelyQueryPort: 10011, IdelyQueryUser: "u", IdelyQueryPass: "p"})
	if !errors.Is(err, errServerQueryPlaintextRefused) {
		t.Fatalf("snapshotServerQuery error = %v, want plaintext refusal", err)
	}
}
