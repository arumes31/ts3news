package bot

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"ts3news/internal/content"
)

func TestAbyssForgeQuoteTokenRoundTripAndTamperRejection(t *testing.T) {
	server := &WebServer{}
	copy(server.forgeQuoteKey[:], []byte("01234567890123456789012345678901"))
	claims := abyssForgeQuoteClaims{
		UID: "user", Operation: "temper", InvID: 7, Parameters: json.RawMessage(`{"target":5}`),
		Gear: "gear", Inventory: "inventory", ForgeFloor: true, ExpiresUnix: time.Now().Add(time.Minute).Unix(),
	}
	token, err := server.signForgeClaims(claims)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := server.verifyForgeClaims(token)
	if err != nil || decoded.UID != claims.UID || decoded.Operation != claims.Operation || !decoded.ForgeFloor {
		t.Fatalf("decoded claims = %+v, %v", decoded, err)
	}
	tampered := token[:len(token)-1] + map[bool]string{true: "A", false: "B"}[token[len(token)-1] != 'A']
	if _, err := server.verifyForgeClaims(tampered); err == nil {
		t.Fatal("tampered forge quote token was accepted")
	}
	claims.ExpiresUnix = time.Now().Add(-time.Second).Unix()
	expired, _ := server.signForgeClaims(claims)
	if _, err := server.verifyForgeClaims(expired); err == nil {
		t.Fatal("expired forge quote token was accepted")
	}
}

func TestCanonicalForgeParameters(t *testing.T) {
	first, err := canonicalForgeParameters(json.RawMessage(`{"b":2,"a":1}`))
	if err != nil {
		t.Fatal(err)
	}
	second, err := canonicalForgeParameters(json.RawMessage(" { \"a\" : 1, \"b\" : 2 } "))
	if err != nil || !bytes.Equal(first, second) {
		t.Fatalf("canonical parameters = %s / %s, %v", first, second, err)
	}
	if _, err := canonicalForgeParameters(json.RawMessage(`{} {}`)); err == nil {
		t.Fatal("multiple parameter documents were accepted")
	}
}

func TestTargetCraftKeepsSlotAsSignedParameterAndStripsMiddlewareConfirmation(t *testing.T) {
	payload := []byte(`{"slot":"Head","rarity":4,"confirmation":"FORGE TARGET_CRAFT"}`)
	invID, identitySlot, confirmation, parameters := forgeRequestIdentity(payload, "target_craft")
	if invID != 0 || identitySlot != "" || confirmation != "FORGE TARGET_CRAFT" ||
		!bytes.Equal(parameters, []byte(`{"rarity":4,"slot":"Head"}`)) {
		t.Fatalf("target craft identity = inv:%d slot:%q confirmation:%q parameters:%s", invID, identitySlot, confirmation, parameters)
	}
	clean := forgeHandlerPayload(payload)
	if bytes.Contains(clean, []byte("confirmation")) || !bytes.Contains(clean, []byte(`"slot":"Head"`)) {
		t.Fatalf("handler payload = %s", clean)
	}
}

func TestForgeMutationRequiresQuoteAndRejectsAlteredParameters(t *testing.T) {
	server := &WebServer{bot: &Bot{}, abyssFeatures: &abyssFeatureConfig{forge: true, forgeRollout: 100}}
	copy(server.forgeQuoteKey[:], []byte("01234567890123456789012345678901"))
	var calls atomic.Int64
	handler := server.forgeMutation("temper", func(w http.ResponseWriter, _ *http.Request, _ string) {
		calls.Add(1)
		writeJSON(w, map[string]any{"ok": true})
	})

	recorder := httptest.NewRecorder()
	handler(recorder, httptest.NewRequest(http.MethodPost, "/api/abyss/temper", strings.NewReader(`{"inv_id":7,"target":3}`)), "user")
	if calls.Load() != 0 || !strings.Contains(recorder.Body.String(), errForgeQuoteRequired.Error()) {
		t.Fatalf("missing quote calls=%d body=%q", calls.Load(), recorder.Body.String())
	}

	claims := abyssForgeQuoteClaims{
		UID: "user", Operation: "temper", InvID: 7, Parameters: json.RawMessage(`{"target":2}`),
		Gear: "none", Inventory: "unused", ExpiresUnix: time.Now().Add(time.Minute).Unix(),
	}
	token, err := server.signForgeClaims(claims)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/abyss/temper", strings.NewReader(`{"inv_id":7,"target":3}`))
	request.Header.Set(abyssForgeQuoteHeader, token)
	recorder = httptest.NewRecorder()
	handler(recorder, request, "user")
	if calls.Load() != 0 || !strings.Contains(recorder.Body.String(), errForgeQuoteAltered.Error()) {
		t.Fatalf("altered quote calls=%d body=%q", calls.Load(), recorder.Body.String())
	}
}

func expectForgeRevision(mock sqlmock.Sqlmock, uid string) {
	mock.ExpectQuery("SELECT id::text, gear_id, COALESCE\\(item_data::text,''\\), durability FROM user_inventory").WithArgs(uid).
		WillReturnRows(sqlmock.NewRows([]string{"id", "gear_id", "item_data", "durability"}).AddRow("7", "U_LEG_2", "", 100))
	mock.ExpectQuery("SELECT slot, gear_id, COALESCE\\(item_data::text,''\\), durability FROM user_gear").WithArgs(uid).
		WillReturnRows(sqlmock.NewRows([]string{"slot", "gear_id", "item_data", "durability"}))
	mock.ExpectQuery("SELECT gold, abyss_tokens FROM users").WithArgs(uid).
		WillReturnRows(sqlmock.NewRows([]string{"gold", "abyss_tokens"}).AddRow(1000, 50))
	mock.ExpectQuery("SELECT mat_id, count").WithArgs(uid).
		WillReturnRows(sqlmock.NewRows([]string{"mat_id", "count"}).AddRow("dust", 20))
}

func TestForgeCommitRejectsStaleInventoryAndGear(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	server := &WebServer{bot: &Bot{DB: database}, abyssFeatures: &abyssFeatureConfig{forge: true, forgeRollout: 100}}
	copy(server.forgeQuoteKey[:], []byte("01234567890123456789012345678901"))
	const uid = "user"

	expectForgeRevision(mock, uid)
	revision, err := server.forgeInventoryRevision(context.Background(), uid)
	if err != nil {
		t.Fatal(err)
	}
	validate := func(claims abyssForgeQuoteClaims) error {
		token, signErr := server.signForgeClaims(claims)
		if signErr != nil {
			t.Fatal(signErr)
		}
		request := httptest.NewRequest(http.MethodPost, "/api/abyss/temper", strings.NewReader(`{"inv_id":7}`))
		request.Header.Set(abyssForgeQuoteHeader, token)
		return server.validateForgeCommitQuote(request, uid, "temper", []byte(`{"inv_id":7}`))
	}

	expectForgeRevision(mock, uid)
	claims := abyssForgeQuoteClaims{
		UID: uid, Operation: "temper", InvID: 7, Parameters: json.RawMessage(`{}`), Gear: "none",
		Inventory: "stale", ExpiresUnix: time.Now().Add(time.Minute).Unix(),
	}
	if err := validate(claims); err != errForgeQuoteStaleInventory {
		t.Fatalf("stale inventory error = %v", err)
	}

	expectForgeRevision(mock, uid)
	mock.ExpectQuery("SELECT gear_id, COALESCE\\(item_data::text,''\\), durability FROM user_inventory").WithArgs(int64(7), uid).
		WillReturnRows(sqlmock.NewRows([]string{"gear_id", "item_data", "durability"}).AddRow("U_LEG_2", "", 100))
	claims.Inventory = revision
	claims.Gear = "outdated-fingerprint"
	if err := validate(claims); err != errForgeQuoteStaleGear {
		t.Fatalf("stale gear error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestForgeQuoteOutcomeBoundsAndBalanceConservation(t *testing.T) {
	server := &WebServer{}
	gear := content.Gear{
		Slot: content.SlotMainHand, Rarity: content.RarityLegendary,
		Stats: content.Stats{HP: 100, STR: 20, DEF: 10, SPD: 8, CRT: 5},
	}
	before := abyssForgeBalance{Gold: 1_000_000, Tokens: 100, Materials: map[string]int64{"dust": 100, "shard": 100, "core": 100, "prism": 100}}
	for _, operation := range abyssForgeCatalog {
		cost := server.forgeQuoteBaseCost("user", operation.ID, nil)
		after := subtractForgeBalance(before, cost)
		if before.Gold-after.Gold != cost.Gold || before.Tokens-after.Tokens != int64(cost.Tokens) {
			t.Fatalf("%s currency conservation failed: before=%+v cost=%+v after=%+v", operation.ID, before, cost, after)
		}
		for material, amount := range cost.Materials {
			if before.Materials[material]-after.Materials[material] != int64(amount) {
				t.Fatalf("%s material conservation failed for %s", operation.ID, material)
			}
		}
		chance, _, _ := forgeQuoteChance(operation, &gear)
		if chance < operation.Success.Min || chance > operation.Success.Max {
			t.Fatalf("%s chance %v outside [%v,%v]", operation.ID, chance, operation.Success.Min, operation.Success.Max)
		}
		outcome := forgeQuoteOutcome(operation.ID, &gear, chance)
		if outcome.ExpectedCR < outcome.MinimumCR-0.001 || outcome.ExpectedCR > outcome.MaximumCR+0.001 {
			t.Fatalf("%s expected CR outside outcome bounds: %+v", operation.ID, outcome)
		}
	}
}

func TestAbyssForgeWorkbenchGuidesCoverFeatureFamilies(t *testing.T) {
	guides := abyssForgeOperationGuides()
	want := []string{"temper", "reforge", "socket_gem", "etch_rune", "transfer_enchant", "awaken", "corrupt", "fuse", "masterwork", "special_reroll", "brand"}
	for _, operation := range want {
		found := false
		for _, guide := range guides {
			found = found || guide.Operation == operation
		}
		if !found {
			t.Fatalf("missing workbench guide for %s", operation)
		}
	}
	recipes := abyssForgeRecipeGuides()
	var starter, event, targeted bool
	for _, recipe := range recipes {
		starter = starter || recipe.Starter
		event = event || recipe.Event
		targeted = targeted || recipe.SlotTarget != "" || recipe.SetTarget != ""
		if len(recipe.Materials) > 0 && (len(recipe.Sources) == 0 || len(recipe.Substitutions) == 0) {
			t.Fatalf("recipe %s has incomplete sourcing metadata", recipe.ID)
		}
	}
	if !starter || !event || !targeted {
		t.Fatalf("recipe families starter=%v event=%v targeted=%v", starter, event, targeted)
	}
}

func TestDecodeBoundedForgeRequestRejectsUnknownAndOversizedInput(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/abyss/forge/quote", strings.NewReader(`{"operation":"temper","unknown":true}`))
	if err := decodeBoundedForgeRequest(request, &abyssForgeQuoteRequest{}); err == nil {
		t.Fatal("unknown forge quote field was accepted")
	}
	request = httptest.NewRequest(http.MethodPost, "/api/abyss/forge/quote", strings.NewReader(strings.Repeat("x", abyssForgeRequestMax+1)))
	if err := decodeBoundedForgeRequest(request, &abyssForgeQuoteRequest{}); err == nil {
		t.Fatal("oversized forge quote was accepted")
	}
}

func TestForgeMutationRejectsOversizedCommitBody(t *testing.T) {
	server := &WebServer{}
	var calls atomic.Int64
	handler := server.forgeMutation("temper", func(w http.ResponseWriter, _ *http.Request, _ string) {
		calls.Add(1)
		writeJSON(w, map[string]any{"ok": true})
	})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/abyss/temper", strings.NewReader(strings.Repeat("x", abyssForgeRequestMax+1)))
	handler(recorder, request, "user")
	if calls.Load() != 0 || !strings.Contains(recorder.Body.String(), "too large") {
		t.Fatalf("oversized commit calls=%d body=%q", calls.Load(), recorder.Body.String())
	}
}

func TestForgeMutationPropagatesRequestContextToTransactions(t *testing.T) {
	server := &WebServer{}
	type contextKey string
	requestContext := context.WithValue(context.Background(), contextKey("forge"), "request")
	var inherited bool
	handler := server.forgeMutation("temper", func(w http.ResponseWriter, r *http.Request, _ string) {
		inherited = forgeContextFromWriter(w) == r.Context() && forgeContextFromWriter(w).Value(contextKey("forge")) == "request"
		writeJSON(w, map[string]any{"ok": true})
	})
	request := httptest.NewRequest(http.MethodPost, "/api/abyss/temper", strings.NewReader(`{}`)).WithContext(requestContext)
	handler(httptest.NewRecorder(), request, "user")
	if !inherited {
		t.Fatal("forge handler did not inherit the request context")
	}
}

func TestForgeMutationLocksGearCurrencyAndMaterials(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	mock.ExpectBegin()
	tx, err := database.Begin()
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectQuery("SELECT gear_id, item_data FROM user_inventory.*FOR UPDATE").WithArgs(int64(7), "user").
		WillReturnRows(sqlmock.NewRows([]string{"gear_id", "item_data"}).AddRow("U_LEG_2", nil))
	mock.ExpectExec("UPDATE users SET gold = gold -").WithArgs(int64(100), "user").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE user_materials SET count = count -").WithArgs(2, "user", "dust").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO abyss_forge_material_flow").WithArgs("user", "dust", 2).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectRollback()
	bot := &Bot{DB: database}
	if _, _, ok := loadForgeItem(tx, bot, "user", 7, ""); !ok {
		t.Fatal("locked gear row was not loaded")
	}
	if !deductGold(httptest.NewRecorder(), tx, "user", 100) || !spendMaterials(tx, "user", map[string]int{"dust": 2}) {
		t.Fatal("guarded balance mutation failed")
	}
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestIdempotentForgeMutationReplaysCommit(t *testing.T) {
	server := &WebServer{}
	var calls atomic.Int64
	handler := server.forgeMutation("temper", func(w http.ResponseWriter, _ *http.Request, _ string) {
		calls.Add(1)
		writeJSON(w, map[string]any{"ok": true, "value": 7})
	})
	request := func(body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/api/abyss/temper", strings.NewReader(body))
		req.Header.Set(abyssForgeIdempotencyKey, "same")
		recorder := httptest.NewRecorder()
		handler(recorder, req, "user")
		return recorder
	}
	first, second := request(`{"inv_id":1}`), request(`{"inv_id":1}`)
	if calls.Load() != 1 || first.Body.String() != second.Body.String() || second.Header().Get("X-Idempotent-Replay") != "true" {
		t.Fatalf("forge replay calls=%d first=%q second=%q", calls.Load(), first.Body.String(), second.Body.String())
	}
	conflict := request(`{"inv_id":2}`)
	if calls.Load() != 1 || !strings.Contains(conflict.Body.String(), "another request") {
		t.Fatalf("forge idempotency conflict calls=%d body=%q", calls.Load(), conflict.Body.String())
	}
}

func TestIdempotentForgeReplayDoesNotRequireNowStaleQuote(t *testing.T) {
	server := &WebServer{bot: &Bot{}, abyssFeatures: &abyssFeatureConfig{forge: true, forgeRollout: 100}}
	payload := []byte(`{"inv_id":1}`)
	hash := sha256.Sum256(payload)
	key := "user\x00temper\x00replay"
	server.abyssForgeOps.idempotency.Store(key, abyssTreeCachedResponse{
		RequestHash: hash, Status: http.StatusOK, Header: http.Header{"Content-Type": {"application/json"}},
		Body: []byte(`{"ok":true,"replayed":true}`), ExpiresAt: time.Now().Add(time.Minute),
	})
	var calls atomic.Int64
	handler := server.forgeMutation("temper", func(w http.ResponseWriter, _ *http.Request, _ string) {
		calls.Add(1)
		writeJSON(w, map[string]any{"ok": true})
	})
	request := httptest.NewRequest(http.MethodPost, "/api/abyss/temper", bytes.NewReader(payload))
	request.Header.Set(abyssForgeIdempotencyKey, "replay")
	recorder := httptest.NewRecorder()
	handler(recorder, request, "user")
	if calls.Load() != 0 || recorder.Header().Get("X-Idempotent-Replay") != "true" ||
		!strings.Contains(recorder.Body.String(), `"replayed":true`) {
		t.Fatalf("replay calls=%d headers=%v body=%q", calls.Load(), recorder.Header(), recorder.Body.String())
	}
}

func TestConcurrentForgeMutationWithSameKeyCommitsOnce(t *testing.T) {
	server := &WebServer{}
	var calls atomic.Int64
	handler := server.forgeMutation("temper", func(w http.ResponseWriter, _ *http.Request, _ string) {
		calls.Add(1)
		writeJSON(w, map[string]any{"ok": true})
	})
	const workers = 12
	start := make(chan struct{})
	var wait sync.WaitGroup
	wait.Add(workers)
	for range workers {
		go func() {
			defer wait.Done()
			<-start
			req := httptest.NewRequest(http.MethodPost, "/api/abyss/temper", strings.NewReader(`{"inv_id":1}`))
			req.Header.Set(abyssForgeIdempotencyKey, "concurrent")
			handler(httptest.NewRecorder(), req, "user")
		}()
	}
	close(start)
	wait.Wait()
	if got := calls.Load(); got != 1 {
		t.Fatalf("concurrent mutation committed %d times, want 1", got)
	}
}

func TestForgeOperationMetricsTrackRatesLatencyAndAbandonment(t *testing.T) {
	server := &WebServer{}
	success := server.forgeMutation("temper", func(w http.ResponseWriter, _ *http.Request, _ string) {
		writeJSON(w, map[string]any{"ok": true})
	})
	failure := server.forgeMutation("temper", func(w http.ResponseWriter, _ *http.Request, _ string) {
		writeJSON(w, map[string]any{"ok": false, "error": "test"})
	})
	success(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{}`)), "user")
	failure(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{}`)), "user")
	cancelledContext, cancel := context.WithCancel(context.Background())
	cancel()
	cancelled := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{}`)).WithContext(cancelledContext)
	success(httptest.NewRecorder(), cancelled, "user")

	now := time.Now()
	server.abyssForgeOps.trackQuote("committed", "temper", now.Add(-50*time.Millisecond), now.Add(time.Minute))
	server.abyssForgeOps.consumeQuote("committed")
	server.abyssForgeOps.trackQuote("abandoned", "temper", now.Add(-time.Minute), now.Add(-time.Second))
	snapshot := server.abyssForgeOps.snapshot()
	operations := snapshot["operations"].(map[string]map[string]int64)
	metrics := operations["temper"]
	if metrics["commits"] != 2 || metrics["successes"] != 1 || metrics["failures"] != 1 || metrics["cancellations"] != 1 {
		t.Fatalf("operation metrics = %+v", metrics)
	}
	if snapshot["abandonments"].(int64) != 1 || snapshot["quote_to_commit_ms"].(int64) <= 0 {
		t.Fatalf("quote lifecycle metrics = %+v", snapshot)
	}
}

func TestForgeAnomalyDetection(t *testing.T) {
	before, _ := json.Marshal(map[string]any{
		"item":      content.Gear{Stats: content.Stats{HP: 100, STR: 10}},
		"materials": map[string]int64{"core": 10},
	})
	powerSpike, _ := json.Marshal(map[string]any{
		"item":      content.Gear{Stats: content.Stats{HP: 100000, STR: 10000}},
		"materials": map[string]int64{"core": 10},
	})
	materialSpike, _ := json.Marshal(map[string]any{
		"item":      content.Gear{Stats: content.Stats{HP: 100, STR: 10}},
		"materials": map[string]int64{"core": 20000},
	})
	if !forgeMutationAnomalous(before, powerSpike) || !forgeMutationAnomalous(before, materialSpike) || forgeMutationAnomalous(before, before) {
		t.Fatal("forge anomaly detector did not distinguish safe and impossible deltas")
	}
}

func expectForgeAuditBalance(mock sqlmock.Sqlmock, uid string) {
	mock.ExpectQuery("SELECT gold FROM users").WithArgs(uid).
		WillReturnRows(sqlmock.NewRows([]string{"gold"}).AddRow(1000))
	mock.ExpectQuery("SELECT abyss_tokens FROM users").WithArgs(uid).
		WillReturnRows(sqlmock.NewRows([]string{"abyss_tokens"}).AddRow(50))
	mock.ExpectQuery("SELECT mat_id, count").WithArgs(uid).
		WillReturnRows(sqlmock.NewRows([]string{"mat_id", "count"}).AddRow("dust", 20))
}

func TestForgeMutationPersistsBalanceSnapshotsForNonItemCommit(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	server := &WebServer{bot: &Bot{DB: database}}
	expectForgeAuditBalance(mock, "user")
	expectForgeAuditBalance(mock, "user")
	mock.ExpectExec("INSERT INTO abyss_forge_mutation_audit").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("DELETE FROM abyss_forge_mutation_audit").
		WillReturnResult(sqlmock.NewResult(0, 0))
	handler := server.forgeMutation("craft", func(w http.ResponseWriter, _ *http.Request, _ string) {
		writeJSON(w, map[string]any{"ok": true})
	})
	recorder := httptest.NewRecorder()
	handler(recorder, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{}`)), "user")
	if !strings.Contains(recorder.Body.String(), `"ok":true`) {
		t.Fatalf("response = %q", recorder.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestTargetCraftCostAndDuplicatePolicies(t *testing.T) {
	legendary := abyssTargetCraftCost(content.RarityLegendary)
	celestial := abyssTargetCraftCost(content.RarityCelestial)
	if legendary["dust"] <= 0 || celestial["dust"] <= legendary["dust"] || celestial["prism"] <= 0 {
		t.Fatalf("target craft costs legendary=%v celestial=%v", legendary, celestial)
	}
	candidates := targetCraftCandidates(abyssTargetCraftRequest{Rarity: int(content.RarityLegendary)})
	if len(candidates) == 0 {
		t.Fatal("production catalog has no Legendary target-craft candidates")
	}
	owned := map[string]bool{candidates[0].ID: true}
	selected, duplicate, ok := chooseTargetCraft(candidates, abyssTargetCraftRequest{DuplicatePolicy: "avoid"}, owned)
	if !ok && len(candidates) > 1 {
		t.Fatal("avoid policy failed despite an unowned candidate")
	}
	if ok && (duplicate || owned[selected.ID]) {
		t.Fatalf("avoid policy selected duplicate %+v", selected)
	}
	selected, duplicate, ok = chooseTargetCraft(candidates, abyssTargetCraftRequest{DuplicatePolicy: "upgrade"}, owned)
	if !ok || selected.ID == "" || !duplicate {
		t.Fatalf("upgrade policy selection = %+v duplicate=%v ok=%v", selected, duplicate, ok)
	}
}
