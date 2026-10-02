// Command brawl-string-source inventories Go string literals without loading game code.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type candidate struct {
	File       string   `json:"file"`
	Line       int      `json:"line"`
	Column     int      `json:"column"`
	Kind       string   `json:"kind"`
	Text       string   `json:"text"`
	Parameters []string `json:"parameters"`
	Context    string   `json:"context"`
}
type sourceFile struct {
	File    string      `json:"file"`
	SHA256  string      `json:"sha256"`
	Strings []candidate `json:"strings"`
}

func extract(name string, source []byte) (sourceFile, error) {
	sum := sha256.Sum256(source)
	result := sourceFile{File: name, SHA256: hex.EncodeToString(sum[:]), Strings: []candidate{}}
	positions := token.NewFileSet()
	tree, err := parser.ParseFile(positions, name, source, 0)
	if err != nil {
		return result, err
	}
	ast.Inspect(tree, func(node ast.Node) bool {
		literal, ok := node.(*ast.BasicLit)
		if !ok || literal.Kind != token.STRING {
			return true
		}
		text, err := strconv.Unquote(literal.Value)
		if err != nil || strings.TrimSpace(text) == "" {
			return true
		}
		position := positions.Position(literal.Pos())
		result.Strings = append(result.Strings, candidate{File: name, Line: position.Line, Column: position.Column, Kind: "go_literal", Text: text, Parameters: []string{}, Context: "GoStringLiteral"})
		return true
	})
	return result, nil
}
func run() error {
	if len(os.Args) != 1 {
		return fmt.Errorf("usage: go run ./cmd/brawl-string-source (from repository root)")
	}
	paths := []string{}
	for _, pattern := range []string{"internal/rift/*.go", "internal/content/*.go", "internal/bot/web_rift*.go"} {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return err
		}
		if len(matches) == 0 {
			return fmt.Errorf("no files match %s; run from repository root", pattern)
		}
		for _, name := range matches {
			if !strings.HasSuffix(name, "_test.go") {
				paths = append(paths, name)
			}
		}
	}
	sort.Strings(paths)
	files := make([]sourceFile, 0, len(paths))
	for _, name := range paths {
		source, err := os.ReadFile(name)
		if err != nil {
			return err
		}
		file, err := extract(filepath.ToSlash(name), source)
		if err != nil {
			return err
		}
		files = append(files, file)
	}
	return json.NewEncoder(os.Stdout).Encode(files)
}
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
