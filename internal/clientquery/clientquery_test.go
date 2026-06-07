package clientquery

import (
	"reflect"
	"testing"
)

func TestEscape(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"hello", "hello"},
		{"hello world", "hello\\sworld"},
		{"back\\slash", "back\\\\slash"},
		{"forward/slash", "forward\\/slash"},
		{"pipe|bar", "pipe\\pbar"},
		{"tabs\tnewline\nreturn\r", "tabs\\tnewline\\nreturn\\r"},
	}
	for _, tt := range tests {
		if got := Escape(tt.in); got != tt.want {
			t.Errorf("Escape(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestUnescape(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"hello", "hello"},
		{"hello\\sworld", "hello world"},
		{"back\\\\slash", "back\\slash"},
		{"forward\\/slash", "forward/slash"},
		{"pipe\\pbar", "pipe|bar"},
		{"tabs\\tnewline\\nreturn\\r", "tabs\tnewline\nreturn\r"},
	}
	for _, tt := range tests {
		if got := Unescape(tt.in); got != tt.want {
			t.Errorf("Unescape(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestFirstWord(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"command", "command"},
		{"command arg1 arg2", "command"},
		{"", ""},
		{" ", ""},
	}
	for _, tt := range tests {
		if got := firstWord(tt.in); got != tt.want {
			t.Errorf("firstWord(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestParseError(t *testing.T) {
	tests := []struct {
		in      string
		wantID  int
		wantMsg string
	}{
		{"error id=0 msg=ok", 0, "ok"},
		{"error id=256 msg=invalid\\sparameter", 256, "invalid parameter"},
		{"error id=123", 123, ""},
		{"not an error line", 0, ""},
	}
	for _, tt := range tests {
		id, msg := parseError(tt.in)
		if id != tt.wantID || msg != tt.wantMsg {
			t.Errorf("parseError(%q) = %d, %q; want %d, %q", tt.in, id, msg, tt.wantID, tt.wantMsg)
		}
	}
}

func TestFirstKV(t *testing.T) {
	lines := []string{"key1=val1 key2=val\\s2", "key3=val3"}
	want := map[string]string{
		"key1": "val1",
		"key2": "val 2",
	}
	got := firstKV(lines)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("firstKV() = %v, want %v", got, want)
	}
}
