package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"
)

func TestParseOptions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		args    []string
		want    commandOptions
		wantErr string
	}{
		{name: "check defaults", want: commandOptions{mode: "check", timeout: 45 * time.Second, maxBytes: defaultSnapshotLimit}},
		{name: "verify", args: []string{"-mode", "VERIFY", "-file", "snapshot.json", "-timeout", "2m"}, want: commandOptions{mode: "verify", path: "snapshot.json", timeout: 2 * time.Minute, maxBytes: defaultSnapshotLimit}},
		{name: "missing file", args: []string{"-mode", "drill"}, wantErr: "-file is required"},
		{name: "unsupported mode", args: []string{"-mode", "restore"}, wantErr: "unsupported mode"},
		{name: "invalid timeout", args: []string{"-timeout", "0s"}, wantErr: "timeout must be positive"},
		{name: "positional", args: []string{"extra"}, wantErr: "positional arguments"},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got, err := parseOptions(test.args, &bytes.Buffer{})
			if test.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantErr) {
					t.Fatalf("parseOptions error = %v, want containing %q", err, test.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseOptions: %v", err)
			}
			if got != test.want {
				t.Fatalf("parseOptions = %+v, want %+v", got, test.want)
			}
		})
	}
}

func TestRunRejectsUnsupportedModeWithoutDatabase(t *testing.T) {
	t.Parallel()

	err := run(context.Background(), []string{"-mode", "restore"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "unsupported mode") {
		t.Fatalf("run error = %v, want unsupported mode", err)
	}
}
