package rift

import (
	"encoding/json"
	"math"
	"testing"
	"time"
)

func TestAnalogMovementPreservesMagnitude(t *testing.T) {
	for _, tc := range []struct {
		name, payload string
		dx, dy        float64
	}{
		{"keyboard", `{"x":1,"y":0}`, 47, 0},
		{"half horizontal", `{"x":0.5,"y":0}`, 23.5, 0},
		{"half vertical", `{"x":0,"y":-0.5}`, 0, -14.1},
		{"neutral", `{"x":0,"y":0}`, 0, 0},
		{"diagonal capped", `{"x":1,"y":1}`, 47 / math.Sqrt2, 28.2 / math.Sqrt2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var input Input
			if err := json.Unmarshal([]byte(tc.payload), &input); err != nil {
				t.Fatal(err)
			}
			r := testRun()
			r.Status = "cleared"
			x, y := r.Player.X, r.Player.Y
			r.Step(input, time.Unix(100, 200_000_000))
			if math.Abs(r.Player.X-x-tc.dx) > 1e-8 || math.Abs(r.Player.Y-y-tc.dy) > 1e-8 {
				t.Fatalf("movement = (%g, %g), want (%g, %g)", r.Player.X-x, r.Player.Y-y, tc.dx, tc.dy)
			}
		})
	}
}

func TestInvalidAnalogMovementCannotChangeRun(t *testing.T) {
	for _, tc := range []struct {
		name  string
		input Input
	}{
		{"nan x", Input{X: math.NaN()}},
		{"nan y", Input{Y: math.NaN()}},
		{"positive infinity", Input{X: math.Inf(1)}},
		{"negative infinity", Input{Y: math.Inf(-1)}},
		{"x above bound", Input{X: 1.001}},
		{"y below bound", Input{Y: -1.001}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := testRun()
			before, err := json.Marshal(r)
			if err != nil {
				t.Fatal(err)
			}
			r.Step(tc.input, time.Unix(100, 200_000_000))
			after, err := json.Marshal(r)
			if err != nil {
				t.Fatal(err)
			}
			if string(before) != string(after) {
				t.Fatal("invalid movement changed run")
			}
		})
	}
}
