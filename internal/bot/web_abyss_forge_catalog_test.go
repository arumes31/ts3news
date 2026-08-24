package bot

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"ts3news/internal/content"
)

func TestAbyssForgeCatalogValid(t *testing.T) {
	t.Parallel()

	operations := abyssForgeOperations()
	if err := validateAbyssForgeCatalog(operations); err != nil {
		t.Fatalf("validate production forge catalog: %v", err)
	}
	if len(operations) < 50 {
		t.Fatalf("operation count = %d, want at least 50", len(operations))
	}
	summary := currentAbyssForgeCatalogSummary()
	if summary.SchemaVersion != abyssForgeCatalogSchemaVersion || summary.OperationCount != len(operations) {
		t.Fatalf("catalog summary = %+v", summary)
	}
	if len(summary.CatalogHash) != 64 {
		t.Fatalf("catalog hash length = %d, want 64", len(summary.CatalogHash))
	}
}

func TestAbyssForgeCatalogDeterministicAndDefensivelyCopied(t *testing.T) {
	t.Parallel()

	first := abyssForgeOperations()
	second := abyssForgeOperations()
	if got, want := abyssForgeCatalogHash(first), abyssForgeCatalogHash(second); got != want {
		t.Fatalf("catalog hashes differ: %q != %q", got, want)
	}
	first[0].CompatibleSlots[0] = "invalid"
	first[0].Cost.MaterialIDs[0] = "invalid"
	if err := validateAbyssForgeCatalog(second); err != nil {
		t.Fatalf("mutation leaked into second catalog copy: %v", err)
	}
}

func TestAbyssForgeQuoteCostPoliciesCoverEveryOperation(t *testing.T) {
	t.Parallel()

	catalog := make(map[string]bool, len(abyssForgeCatalog))
	for _, operation := range abyssForgeCatalog {
		catalog[operation.ID] = true
		if policy := forgeQuoteCostCoverage[operation.ID]; policy == "" {
			t.Errorf("operation %q has no quote/commit cost policy", operation.ID)
		}
	}
	for operation := range forgeQuoteCostCoverage {
		if !catalog[operation] {
			t.Errorf("quote/commit cost policy references unknown operation %q", operation)
		}
	}
}

func TestAbyssForgeQuoteCommitParityForEveryOperation(t *testing.T) {
	t.Parallel()

	routes, err := os.ReadFile("web.go")
	if err != nil {
		t.Fatal(err)
	}
	client, err := os.ReadFile("webassets/abyss.html")
	if err != nil {
		t.Fatal(err)
	}
	for _, operation := range abyssForgeCatalog {
		if operation.ID == "fuse_preview" {
			continue
		}
		if !strings.Contains(string(routes), `forgeMutation("`+operation.ID+`"`) {
			t.Errorf("commit route for %q does not enforce a server quote", operation.ID)
		}
		if !strings.Contains(string(client), `:'`+operation.ID+`'`) &&
			!strings.Contains(string(client), `":"`+operation.ID+`"`) {
			t.Errorf("client commit map has no quote operation %q", operation.ID)
		}
	}
}

func TestAbyssForgeRollbackCoverageForEveryMultiWriteOperation(t *testing.T) {
	t.Parallel()

	routeSource, err := os.ReadFile("web.go")
	if err != nil {
		t.Fatal(err)
	}
	files, err := filepath.Glob("web_abyss*.go")
	if err != nil {
		t.Fatal(err)
	}
	var implementation strings.Builder
	for _, path := range files {
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			t.Fatal(readErr)
		}
		implementation.Write(data)
		implementation.WriteByte('\n')
	}
	allCode := implementation.String()
	for _, operation := range abyssForgeCatalog {
		if operation.ID == "fuse_preview" || operation.ID == "auto_repair" {
			continue
		}
		pattern := regexp.MustCompile(`forgeMutation\("` + regexp.QuoteMeta(operation.ID) + `", s\.(\w+)\)`)
		match := pattern.FindSubmatch(routeSource)
		if len(match) != 2 {
			t.Errorf("operation %q has no mutation handler", operation.ID)
			continue
		}
		marker := "func (s *WebServer) " + string(match[1]) + "("
		start := strings.Index(allCode, marker)
		if start < 0 {
			t.Errorf("operation %q handler %s was not found", operation.ID, match[1])
			continue
		}
		body := allCode[start:]
		if end := strings.Index(body[len(marker):], "\nfunc "); end >= 0 {
			body = body[:len(marker)+end]
		}
		if !strings.Contains(body, "beginForge") && !strings.Contains(body, "BeginTx") && !strings.Contains(body, "fuseCommon") {
			t.Errorf("multi-write operation %q does not enter a rollback-capable transaction", operation.ID)
		}
	}
}

func TestValidateAbyssForgeCatalogRejectsInvalidData(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mutate func([]abyssForgeOperation) []abyssForgeOperation
		want   string
	}{
		{name: "duplicate id", mutate: func(ops []abyssForgeOperation) []abyssForgeOperation {
			ops[1].ID = ops[0].ID
			return ops
		}, want: "duplicate"},
		{name: "unknown material", mutate: func(ops []abyssForgeOperation) []abyssForgeOperation {
			ops[0].Cost.MaterialIDs = []string{"counterfeit"}
			return ops
		}, want: "unknown material"},
		{name: "unknown slot", mutate: func(ops []abyssForgeOperation) []abyssForgeOperation {
			ops[0].CompatibleSlots = []content.GearSlot{"Pocket"}
			return ops
		}, want: "slot"},
		{name: "invalid probability", mutate: func(ops []abyssForgeOperation) []abyssForgeOperation {
			ops[0].Success.Max = 1.1
			return ops
		}, want: "success range"},
		{name: "unknown effect", mutate: func(ops []abyssForgeOperation) []abyssForgeOperation {
			ops[0].ItemEffects = []content.ItemEffect{"Counterfeit"}
			return ops
		}, want: "unknown item effect"},
		{name: "unknown prerequisite", mutate: func(ops []abyssForgeOperation) []abyssForgeOperation {
			ops[0].Prerequisites = []string{"missing"}
			return ops
		}, want: "unknown operation"},
		{name: "prerequisite cycle", mutate: func(ops []abyssForgeOperation) []abyssForgeOperation {
			ops[0].Prerequisites = []string{ops[1].ID}
			ops[1].Prerequisites = []string{ops[0].ID}
			return ops
		}, want: "cycle"},
		{name: "reversible without undo policy", mutate: func(ops []abyssForgeOperation) []abyssForgeOperation {
			ops[0].Reversible = true
			ops[0].UndoPolicy = "none"
			return ops
		}, want: "undo policy"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validateAbyssForgeCatalog(tt.mutate(abyssForgeOperations()))
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want substring %q", err, tt.want)
			}
		})
	}
}
