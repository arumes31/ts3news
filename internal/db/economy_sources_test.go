package db

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// Direct SQL writes must identify their static origin even when an older path
// does not yet provide an explicit transaction/request context.
func TestEconomyMutationSQLHasStaticSource(t *testing.T) {
	mutation := regexp.MustCompile(`(?i)\b(?:INSERT\s+INTO|UPDATE|DELETE\s+FROM)\s+(?:users|user_materials|user_consumables)\b`)
	tag := regexp.MustCompile(`^/\* economy:[A-Za-z][A-Za-z0-9_.]{0,127} \*/\s`)
	count := 0
	for _, dir := range []string{".", "../bot"} {
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatal(err)
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
				continue
			}
			path := filepath.Join(dir, entry.Name())
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, path, nil, 0)
			if err != nil {
				t.Fatal(err)
			}
			ast.Inspect(file, func(node ast.Node) bool {
				literal, ok := node.(*ast.BasicLit)
				if !ok || literal.Kind != token.STRING {
					return true
				}
				value, err := strconv.Unquote(literal.Value)
				if err != nil {
					t.Fatal(err)
				}
				if !mutation.MatchString(value) {
					return true
				}
				count++
				if !tag.MatchString(value) {
					t.Errorf("%s: economy mutation lacks static source", fset.Position(literal.Pos()))
				}
				return true
			})
		}
	}
	if count < 100 {
		t.Fatalf("unexpectedly few mutation statements checked: %d", count)
	}
}
