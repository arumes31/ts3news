package bot

import (
	"bytes"
	"encoding/json"
	"regexp"
	"strings"
	"testing"
)

func TestAbyssBoonDialogBootstrapEscapesStringsOnce(t *testing.T) {
	server, err := NewWebServer(nil)
	if err != nil {
		t.Fatal(err)
	}
	name := "Deep \"Well\" </script><script>alert(1)</script>"
	view := abyssRunIdentityView{
		BiomeChoices: []abyssBiomeContract{{ID: 1, Name: name}},
		Draft: abyssBoonDraftView{
			Pending: true, Depth: 5,
			Options: []abyssRunBoon{{ID: 1, Name: name, Effect: "+10% HP"}},
		},
	}
	var rendered bytes.Buffer
	fixture := abyssGoldenFixture(true)
	fixture["RunIdentity"] = view
	if err := server.tmpl.ExecuteTemplate(&rendered, "abyss", fixture); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(rendered.String(), "</script><script>alert(1)</script>") {
		t.Fatal("catalog text can terminate its script element")
	}
	start := strings.Index(rendered.String(), "var abyssBiomeContracts=")
	end := strings.Index(rendered.String()[start:], "var abyssRunIdentityState=")
	names := regexp.MustCompile(`name:\s*("(?:\\.|[^"\\])*")`).FindAllStringSubmatch(rendered.String()[start:start+end], -1)
	if len(names) != 2 {
		t.Fatalf("got %d catalog names, want biome and boon", len(names))
	}
	for _, match := range names {
		var got string
		if err := json.Unmarshal([]byte(match[1]), &got); err != nil {
			t.Fatal(err)
		}
		if got != name {
			t.Fatalf("decoded name = %q, want %q", got, name)
		}
	}
}
