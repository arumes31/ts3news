package bot

import "testing"

func TestParseTalkStart(t *testing.T) {
	cases := []struct {
		name     string
		line     string
		wantCLID int
		wantOK   bool
	}{
		{"talk start", "notifytalkstatuschange status=1 isreceivedwhisper=0 clid=42", 42, true},
		{"talk stop", "notifytalkstatuschange status=0 isreceivedwhisper=0 clid=42", 0, false},
		{"other event", "notifyclientmoved ctid=5 clid=42", 0, false},
		{"missing clid", "notifytalkstatuschange status=1 isreceivedwhisper=0", 0, false},
		{"zero clid", "notifytalkstatuschange status=1 clid=0", 0, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			clid, ok := parseTalkStart(tc.line)
			if ok != tc.wantOK || clid != tc.wantCLID {
				t.Fatalf("parseTalkStart(%q) = (%d,%v), want (%d,%v)", tc.line, clid, ok, tc.wantCLID, tc.wantOK)
			}
		})
	}
}
