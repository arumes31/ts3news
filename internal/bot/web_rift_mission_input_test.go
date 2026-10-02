package bot

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRiftMalformedMissionIDsRejectedBeforeBanking(t *testing.T) {
	for _, kind := range []string{"start", "bank", "exit", "next", "advance"} {
		for _, id := range []string{`-1`, `101`, `2147483648`, `9223372036854775808`, `1.5`, `"1"`, `"../100"`, `true`, `[]`, `{}`} {
			t.Run(kind+"/"+id, func(t *testing.T) {
				body := fmt.Sprintf(`{"kind":%q,"run_id":"existing-run","request_id":"invalid-mission-request","revision":1,"level_id":%s}`, kind, id)
				req := httptest.NewRequest(http.MethodPost, "https://game.test/api/abyss/rift", strings.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				w := httptest.NewRecorder()
				// No Bot or DB is supplied: passing the input gate would fail this test.
				(&WebServer{}).handleRiftAPI(w, req, "owner")
				if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "invalid expedition controls") {
					t.Fatalf("malformed mission accepted: %d %s", w.Code, w.Body.String())
				}
			})
		}
	}
	// Zero remains the existing default-mission sentinel for omitted level_id.
	for _, id := range []int{0, 1, 100} {
		if !validRiftRequest(riftRequest{Kind: "start", LevelID: id, RequestID: "valid-mission-request"}) {
			t.Fatalf("valid mission boundary %d rejected", id)
		}
	}
}
