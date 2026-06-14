package bot

import "testing"

// TestTemplatesParse ensures every embedded web template parses (catches HTML
// template syntax errors before they reach runtime). NewWebServer only parses
// templates and does not dereference the bot during construction.
func TestTemplatesParse(t *testing.T) {
	if _, err := NewWebServer(nil); err != nil {
		t.Fatalf("web templates failed to parse: %v", err)
	}
}
