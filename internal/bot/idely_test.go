package bot

import "testing"

func TestParseClientActivity(t *testing.T) {
	cases := []struct {
		name          string
		line          string
		wantCLID      int
		wantTalkStart bool
		wantOK        bool
	}{
		{"talk start", "notifytalkstatuschange status=1 isreceivedwhisper=0 clid=42", 42, true, true},
		{"talk stop", "notifytalkstatuschange status=0 isreceivedwhisper=0 clid=42", 0, false, false},
		{"client updated", "notifyclientupdated clid=42 client_input_muted=1", 42, false, true},
		{"client moved", "notifyclientmoved ctid=5 clid=42", 42, false, true},
		{"other event", "notifymessagelist ...", 0, false, false},
		{"missing clid", "notifytalkstatuschange status=1 isreceivedwhisper=0", 0, false, false},
		{"zero clid", "notifytalkstatuschange status=1 clid=0", 0, false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			clid, talkStart, ok := parseClientActivity(tc.line)
			if ok != tc.wantOK || talkStart != tc.wantTalkStart || clid != tc.wantCLID {
				t.Fatalf("parseClientActivity(%q) = (%d,%v,%v), want (%d,%v,%v)", tc.line, clid, talkStart, ok, tc.wantCLID, tc.wantTalkStart, tc.wantOK)
			}
		})
	}
}
