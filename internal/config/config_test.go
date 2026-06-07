package config

import (
	"os"
	"reflect"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Setup env vars
	os.Setenv("TS3_HOST", "localhost")
	os.Setenv("TS3_PORT", "9987")
	os.Setenv("TS3_SERVER_ID", "1")
	os.Setenv("ENABLE_GAMERPOWER", "false")
	os.Setenv("DRM_FILTER", "steam,gog")
	
	defer func() {
		os.Unsetenv("TS3_HOST")
		os.Unsetenv("TS3_PORT")
		os.Unsetenv("TS3_SERVER_ID")
		os.Unsetenv("ENABLE_GAMERPOWER")
		os.Unsetenv("DRM_FILTER")
	}()

	cfg := LoadConfig()

	if cfg.TS3Host != "localhost" {
		t.Errorf("TS3Host = %q, want %q", cfg.TS3Host, "localhost")
	}
	if cfg.TS3Port != 9987 {
		t.Errorf("TS3Port = %d, want 9987", cfg.TS3Port)
	}
	if cfg.EnableGamerPower != false {
		t.Errorf("EnableGamerPower = %v, want false", cfg.EnableGamerPower)
	}
	if !reflect.DeepEqual(cfg.DRMFilter, []string{"steam", "gog"}) {
		t.Errorf("DRMFilter = %v, want [steam gog]", cfg.DRMFilter)
	}
}

func TestEnvBool(t *testing.T) {
	tests := []struct {
		key, val string
		def      bool
		want     bool
	}{
		{"TEST_BOOL", "1", false, true},
		{"TEST_BOOL", "true", false, true},
		{"TEST_BOOL", "yes", false, true},
		{"TEST_BOOL", "on", false, true},
		{"TEST_BOOL", "0", true, false},
		{"TEST_BOOL", "", true, true},
		{"TEST_BOOL", "invalid", true, false},
	}
	for _, tt := range tests {
		os.Setenv(tt.key, tt.val)
		if got := envBool(tt.key, tt.def); got != tt.want {
			t.Errorf("envBool(%q, %v) with val %q = %v, want %v", tt.key, tt.def, tt.val, got, tt.want)
		}
		os.Unsetenv(tt.key)
	}
}

func TestEnvInt(t *testing.T) {
	tests := []struct {
		key, val string
		def      int
		want     int
	}{
		{"TEST_INT", "123", 0, 123},
		{"TEST_INT", "  456  ", 0, 456},
		{"TEST_INT", "invalid", 10, 10},
		{"TEST_INT", "", 20, 20},
	}
	for _, tt := range tests {
		os.Setenv(tt.key, tt.val)
		if got := envInt(tt.key, tt.def); got != tt.want {
			t.Errorf("envInt(%q, %v) with val %q = %v, want %v", tt.key, tt.def, tt.val, got, tt.want)
		}
		os.Unsetenv(tt.key)
	}
}

func TestEnvList(t *testing.T) {
	tests := []struct {
		key, val string
		def      []string
		want     []string
	}{
		{"TEST_LIST", "a,b,c", nil, []string{"a", "b", "c"}},
		{"TEST_LIST", " A , B ", nil, []string{"a", "b"}},
		{"TEST_LIST", "", []string{"def"}, []string{"def"}},
		{"TEST_LIST", " , ", []string{"def"}, []string{"def"}},
	}
	for _, tt := range tests {
		os.Setenv(tt.key, tt.val)
		got := envList(tt.key, tt.def)
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("envList(%q, %v) with val %q = %v, want %v", tt.key, tt.def, tt.val, got, tt.want)
		}
		os.Unsetenv(tt.key)
	}
}

func TestLoadDotEnv(t *testing.T) {
	const filename = "test_config.env"
	content := `
# Comment
KEY1=VAL1
  KEY2  =  "VAL2"  
KEY3='VAL3'
INVALID_LINE
=VALUE
ONLYKEY
`
	os.WriteFile(filename, []byte(content), 0644)
	defer os.Remove(filename)

	// Set one existing to test precedence
	os.Setenv("KEY1", "ORIGINAL")
	defer os.Unsetenv("KEY1")
	defer os.Unsetenv("KEY2")
	defer os.Unsetenv("KEY3")

	loadDotEnv(filename)

	if v := os.Getenv("KEY1"); v != "ORIGINAL" {
		t.Errorf("KEY1 = %q, want ORIGINAL (precedence)", v)
	}
	if v := os.Getenv("KEY2"); v != "VAL2" {
		t.Errorf("KEY2 = %q, want VAL2", v)
	}
	if v := os.Getenv("KEY3"); v != "VAL3" {
		t.Errorf("KEY3 = %q, want VAL3", v)
	}
}
