package bot

import (
	"bytes"
	"strings"
	"testing"
)

// TestTemplatesParse ensures every embedded web template parses (catches HTML
// template syntax errors before they reach runtime). NewWebServer only parses
// templates and does not dereference the bot during construction.
func TestTemplatesParse(t *testing.T) {
	if _, err := NewWebServer(nil); err != nil {
		t.Fatalf("web templates failed to parse: %v", err)
	}
}

func TestAbyssLivePartials(t *testing.T) {
	server, err := NewWebServer(nil)
	if err != nil {
		t.Fatalf("NewWebServer: %v", err)
	}

	abyss := server.tmpl.Lookup("abyss")
	if abyss == nil {
		t.Fatal("abyss template is missing")
	}
	abyssSource := abyss.Tree.Root.String()
	for _, partialName := range []string{"abyssLiveControls", "abyssLiveActionBarJS"} {
		if !strings.Contains(abyssSource, `{{template "`+partialName+`" .}}`) {
			t.Errorf("abyss template does not invoke %q at its extraction point", partialName)
		}
	}

	tests := []struct {
		name         string
		templateName string
		contains     string
	}{
		{
			name:         "controls markup",
			templateName: "abyssLiveControls",
			contains:     `id="liveCombat"`,
		},
		{
			name:         "action bar script",
			templateName: "abyssLiveActionBarJS",
			contains:     "function startLiveCombat",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var rendered bytes.Buffer
			if err := server.tmpl.ExecuteTemplate(&rendered, tt.templateName, nil); err != nil {
				t.Fatalf("ExecuteTemplate(%q): %v", tt.templateName, err)
			}
			if !strings.Contains(rendered.String(), tt.contains) {
				t.Errorf("rendered %q does not contain %q", tt.templateName, tt.contains)
			}
		})
	}

	scriptServer, err := NewWebServer(nil)
	if err != nil {
		t.Fatalf("NewWebServer for script context: %v", err)
	}
	scriptTemplates := scriptServer.tmpl
	if _, err := scriptTemplates.Parse(
		`{{define "abyssLiveScriptHost"}}<script>{{template "abyssLiveActionBarJS" .}}</script>{{end}}`,
	); err != nil {
		t.Fatalf("parse live script host: %v", err)
	}
	var renderedScript bytes.Buffer
	if err := scriptTemplates.ExecuteTemplate(&renderedScript, "abyssLiveScriptHost", nil); err != nil {
		t.Fatalf("execute live script host: %v", err)
	}
	if strings.Contains(renderedScript.String(), "ZgotmplZ") {
		t.Fatal("live action bar script is unsafe in its script context")
	}
}
