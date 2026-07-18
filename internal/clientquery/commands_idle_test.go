package clientquery

import "testing"

// TestClientListParsesIdleAndTalking verifies the Idely-facing fields
// (client_idle_time, client_flag_talking) are read from the per-client clientinfo
// that ClientList issues for voice clients.
func TestClientListParsesIdleAndTalking(t *testing.T) {
	c := startMockTS3(t, map[string]string{
		"clientlist -uid":   `clid=5 cid=10 client_nickname=Alice client_type=0 client_unique_identifier=aaa|clid=6 cid=10 client_nickname=Bob client_type=0 client_unique_identifier=bbb`,
		"clientinfo clid=5": `client_connected_time=1000 client_idle_time=900000 client_flag_talking=1`,
		"clientinfo clid=6": `client_connected_time=2000 client_idle_time=1200000 client_flag_talking=0`,
	}, nil)

	clients, err := c.ClientList()
	if err != nil {
		t.Fatalf("ClientList: %v", err)
	}
	if len(clients) != 2 {
		t.Fatalf("expected 2 clients, got %d", len(clients))
	}
	byCLID := map[int]ClientInfo{}
	for _, ci := range clients {
		byCLID[ci.CLID] = ci
	}

	alice := byCLID[5]
	if alice.IdleTimeMS != 900000 {
		t.Errorf("Alice IdleTimeMS = %d, want 900000", alice.IdleTimeMS)
	}
	if !alice.Talking {
		t.Error("Alice should be flagged talking")
	}

	bob := byCLID[6]
	if bob.IdleTimeMS != 1200000 {
		t.Errorf("Bob IdleTimeMS = %d, want 1200000", bob.IdleTimeMS)
	}
	if bob.Talking {
		t.Error("Bob should not be flagged talking")
	}
}
