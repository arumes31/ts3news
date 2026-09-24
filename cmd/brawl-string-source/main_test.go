package main

import (
	"strings"
	"testing"
)

func TestExtractGoStrings(t *testing.T) {
	source := []byte("package sample\n// \"not copy\"\nvar a = \"Hello\\nworld\"\nvar b = `Raw text`\nvar c = 'x'\n")
	got, err := extract("sample.go", source)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Strings) != 2 {
		t.Fatalf("strings: %+v", got.Strings)
	}
	if got.Strings[0].Text != "Hello\nworld" || got.Strings[0].Line != 3 || got.Strings[0].Column != 9 {
		t.Fatalf("first: %+v", got.Strings[0])
	}
	if got.Strings[1].Text != "Raw text" || got.SHA256 == "" {
		t.Fatalf("result: %+v", got)
	}
}

func TestMalformedGoFailsWithSource(t *testing.T) {
	_, err := extract("broken.go", []byte("package broken; var ="))
	if err == nil || !strings.Contains(err.Error(), "broken.go") {
		t.Fatalf("error: %v", err)
	}
}
