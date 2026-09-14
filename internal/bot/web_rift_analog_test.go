package bot

import (
	"encoding/json"
	"math"
	"testing"

	"ts3news/internal/rift"
)

func TestRiftAnalogRequestValidation(t *testing.T) {
	for _, tc := range []struct {
		name  string
		input rift.Input
		valid bool
	}{
		{"digital", rift.Input{X: 1, Y: -1}, true},
		{"analog", rift.Input{X: .25, Y: -.75}, true},
		{"nan", rift.Input{X: math.NaN()}, false},
		{"infinity", rift.Input{Y: math.Inf(1)}, false},
		{"out of range", rift.Input{X: 1.01}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := riftRequest{Kind: "step", RequestID: "analog-request-12345", Input: tc.input}
			if got := validRiftRequest(req); got != tc.valid {
				t.Fatalf("valid = %v, want %v", got, tc.valid)
			}
			if tc.valid {
				body, err := json.Marshal(req)
				if err != nil {
					t.Fatal(err)
				}
				var decoded riftRequest
				if err := json.Unmarshal(body, &decoded); err != nil {
					t.Fatal(err)
				}
				if decoded.Input != tc.input || !validRiftRequest(decoded) {
					t.Fatal("JSON roundtrip changed movement")
				}
			}
		})
	}
}
