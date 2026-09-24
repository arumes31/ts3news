package bot

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

func TestRiftSkillEffectNamesHaveRendererSupport(t *testing.T) {
	renderer, err := webAssets.ReadFile("webassets/rift_renderer.js")
	if err != nil {
		t.Fatal(err)
	}
	mappings := regexp.MustCompile(`effectRows = \{([^}]+)\}`).FindAllSubmatch(renderer, -1)
	if len(mappings) != 1 {
		t.Fatal("expected one renderer effect mapping")
	}
	supported := map[string]bool{}
	for _, entry := range strings.Split(string(mappings[0][1]), ",") {
		parts := strings.Split(strings.TrimSpace(entry), ":")
		if len(parts) != 2 {
			t.Fatalf("unrecognized effect mapping %q", entry)
		}
		row, err := strconv.Atoi(parts[1])
		if err != nil || row < 0 || row >= 6 {
			t.Fatalf("effect %q has invalid atlas row", parts[0])
		}
		supported[parts[0]] = true
	}
	// Arrow is intentionally procedural in both preview and world rendering.
	for _, branch := range []string{"if(skill.kind==='arrow')", "if(p.kind==='arrow')"} {
		if !strings.Contains(string(renderer), branch) {
			t.Fatalf("missing procedural arrow path %s", branch)
		}
	}
	supported["arrow"] = true
	source, err := os.ReadFile("web_rift.go")
	if err != nil {
		t.Fatal(err)
	}
	file, err := parser.ParseFile(token.NewFileSet(), "web_rift.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	effects := map[string]bool{}
	add := func(expr ast.Expr) {
		literal, ok := expr.(*ast.BasicLit)
		if !ok || literal.Kind != token.STRING {
			t.Fatal("effect contract must be updated for a nonliteral kind")
		}
		value, err := strconv.Unquote(literal.Value)
		if err != nil {
			t.Fatal(err)
		}
		effects[value] = true
	}
	found, signatureMappings := 0, 0
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || (fn.Name.Name != "riftBuildFromUser" && fn.Name.Name != "riftSignatureKind") {
			continue
		}
		found++
		ast.Inspect(fn.Body, func(node ast.Node) bool {
			if assignment, ok := node.(*ast.AssignStmt); ok {
				for i, left := range assignment.Lhs {
					ident, ok := left.(*ast.Ident)
					if !ok || i >= len(assignment.Rhs) {
						continue
					}
					right := assignment.Rhs[i]
					if ident.Name == "kind" {
						if call, ok := right.(*ast.CallExpr); ok {
							name, ok := call.Fun.(*ast.Ident)
							if !ok || name.Name != "riftSignatureKind" {
								t.Fatal("unrecognized dynamic skill kind")
							}
						} else {
							add(right)
						}
					}
					if ident.Name == "forms" {
						signatureMappings++
						forms, ok := right.(*ast.CompositeLit)
						if !ok {
							t.Fatal("unrecognized signature mapping")
						}
						for _, entry := range forms.Elts {
							pair, ok := entry.(*ast.KeyValueExpr)
							if !ok {
								t.Fatal("unrecognized signature entry")
							}
							values, ok := pair.Value.(*ast.CompositeLit)
							if !ok {
								t.Fatal("unrecognized signature effects")
							}
							for _, value := range values.Elts {
								add(value)
							}
						}
					}
				}
			}
			if pair, ok := node.(*ast.KeyValueExpr); ok {
				if key, ok := pair.Key.(*ast.Ident); ok && key.Name == "Kind" {
					if _, ok := pair.Value.(*ast.BasicLit); ok {
						add(pair.Value)
					}
				}
			}
			return true
		})
	}
	if found != 2 || signatureMappings != 1 || len(effects) == 0 {
		t.Fatal("skill producer contract not found")
	}
	for effect := range effects {
		if !supported[effect] {
			t.Errorf("skill effect %q has no renderer support", effect)
		}
	}
	t.Logf("validated %d emitted skill effect kinds", len(effects))
}
