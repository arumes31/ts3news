package bot

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"ts3news/internal/content"
)

const (
	abyssForgeQuoteTTL       = 2 * time.Minute
	abyssForgeRequestMax     = 256 * 1024
	abyssForgeQuoteHeader    = "X-Abyss-Forge-Quote"
	abyssForgeIdempotencyKey = "Idempotency-Key"
)

type abyssForgeBalance struct {
	Gold      int64            `json:"gold"`
	Tokens    int64            `json:"tokens"`
	Materials map[string]int64 `json:"materials"`
}

type abyssForgeQuoteCost struct {
	Gold      int64          `json:"gold"`
	Tokens    int            `json:"tokens"`
	Materials map[string]int `json:"materials"`
}

type abyssForgeOutcome struct {
	MinimumStats  content.Stats `json:"minimum_stats"`
	ExpectedStats content.Stats `json:"expected_stats"`
	MaximumStats  content.Stats `json:"maximum_stats"`
	MinimumCR     float64       `json:"minimum_cr"`
	ExpectedCR    float64       `json:"expected_cr"`
	MaximumCR     float64       `json:"maximum_cr"`
	Gained        []string      `json:"gained_effects"`
	Lost          []string      `json:"lost_effects"`
	Consequences  []string      `json:"consequences"`
}

type abyssForgeQuote struct {
	SchemaVersion      int                 `json:"schema_version"`
	CatalogHash        string              `json:"catalog_hash"`
	Operation          string              `json:"operation"`
	Parameters         json.RawMessage     `json:"parameters"`
	GearFingerprint    string              `json:"gear_fingerprint"`
	InventoryRevision  string              `json:"inventory_revision"`
	ExpiresAt          time.Time           `json:"expires_at"`
	Token              string              `json:"token"`
	Irreversible       bool                `json:"irreversible"`
	Confirmation       string              `json:"confirmation_phrase,omitempty"`
	SuccessChance      float64             `json:"success_chance"`
	ChanceExplanation  string              `json:"chance_explanation"`
	FailureExplanation string              `json:"failure_explanation"`
	PityExplanation    string              `json:"pity_explanation"`
	UndoAvailable      bool                `json:"undo_available"`
	UndoWindowSeconds  int64               `json:"undo_window_seconds"`
	Cost               abyssForgeQuoteCost `json:"cost"`
	CostMinimum        abyssForgeQuoteCost `json:"cost_minimum"`
	CostMaximum        abyssForgeQuoteCost `json:"cost_maximum"`
	BalanceBefore      abyssForgeBalance   `json:"balance_before"`
	BalanceAfter       abyssForgeBalance   `json:"balance_after"`
	Current            *content.Gear       `json:"current_item,omitempty"`
	Outcome            abyssForgeOutcome   `json:"outcome"`
	Warnings           []string            `json:"warnings"`
	DurabilityBefore   int                 `json:"durability_before"`
	DurabilityAfter    int                 `json:"durability_after"`
	SocketsBefore      int                 `json:"sockets_before"`
	SocketsAfter       int                 `json:"sockets_after"`
	SetBefore          string              `json:"set_before"`
	SetAfter           string              `json:"set_after"`
	BoundAfter         bool                `json:"bound_after"`
	TradeableAfter     bool                `json:"tradeable_after"`
	Recovery           map[string]int      `json:"recovery"`
	CostExplanation    string              `json:"cost_explanation"`
}

type abyssForgeQuoteRequest struct {
	Operation    string          `json:"operation"`
	InvID        int64           `json:"inv_id"`
	Slot         string          `json:"slot"`
	Parameters   json.RawMessage `json:"parameters"`
	Confirmation string          `json:"confirmation"`
}

type abyssForgeQuoteClaims struct {
	UID         string          `json:"uid"`
	Operation   string          `json:"operation"`
	InvID       int64           `json:"inv_id"`
	Slot        string          `json:"slot"`
	Parameters  json.RawMessage `json:"parameters"`
	Gear        string          `json:"gear"`
	Inventory   string          `json:"inventory"`
	ForgeFloor  bool            `json:"forge_floor,omitempty"`
	ExpiresUnix int64           `json:"expires_unix"`
}

func canonicalForgeParameters(raw json.RawMessage) (json.RawMessage, error) {
	if len(bytes.TrimSpace(raw)) == 0 {
		return json.RawMessage(`{}`), nil
	}
	var value any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	if decoder.Decode(&struct{}{}) != io.EOF {
		return nil, errors.New("multiple parameter values")
	}
	encoded, err := json.Marshal(value)
	return encoded, err
}

func abyssForgeOperationByID(id string) (abyssForgeOperation, bool) {
	index := sort.Search(len(abyssForgeCatalog), func(i int) bool { return abyssForgeCatalog[i].ID >= id })
	if index >= len(abyssForgeCatalog) || abyssForgeCatalog[index].ID != id {
		return abyssForgeOperation{}, false
	}
	return abyssForgeCatalog[index], true
}

func (s *WebServer) signForgeClaims(claims abyssForgeQuoteClaims) (string, error) {
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	mac := hmac.New(sha256.New, s.forgeQuoteKey[:])
	_, _ = mac.Write(payload)
	return base64.RawURLEncoding.EncodeToString(payload) + "." +
		base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}

func (s *WebServer) verifyForgeClaims(token string) (abyssForgeQuoteClaims, error) {
	var claims abyssForgeQuoteClaims
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return claims, errors.New("invalid quote token")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return claims, errors.New("invalid quote token")
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return claims, errors.New("invalid quote token")
	}
	if base64.RawURLEncoding.EncodeToString(payload) != parts[0] ||
		base64.RawURLEncoding.EncodeToString(signature) != parts[1] {
		return claims, errors.New("invalid quote token")
	}
	mac := hmac.New(sha256.New, s.forgeQuoteKey[:])
	_, _ = mac.Write(payload)
	if !hmac.Equal(signature, mac.Sum(nil)) || json.Unmarshal(payload, &claims) != nil {
		return claims, errors.New("invalid quote token")
	}
	if time.Now().Unix() > claims.ExpiresUnix {
		return claims, errors.New("forge quote expired")
	}
	return claims, nil
}

func (s *WebServer) forgeQuoteRequiresFloor(r *http.Request, operation string) bool {
	claims, err := s.verifyForgeClaims(r.Header.Get(abyssForgeQuoteHeader))
	return err == nil && claims.Operation == operation && claims.ForgeFloor
}

func forgeGearFingerprint(g content.Gear, raw string, invID int64, slot string) string {
	hash := sha256.Sum256([]byte(fmt.Sprintf("%d\x00%s\x00%s\x00%s\x00%d", invID, slot, g.ID, raw, g.MaxDurability)))
	return hex.EncodeToString(hash[:16])
}

func (s *WebServer) forgeInventoryRevision(ctx context.Context, uid string) (string, error) {
	hash := sha256.New()
	queries := []string{
		"SELECT id::text, gear_id, COALESCE(item_data::text,''), durability FROM user_inventory WHERE client_uid=$1 ORDER BY id",
		"SELECT slot, gear_id, COALESCE(item_data::text,''), durability FROM user_gear WHERE client_uid=$1 ORDER BY slot",
	}
	for _, query := range queries {
		rows, err := s.bot.DB.QueryContext(ctx, query, uid)
		if err != nil {
			return "", err
		}
		for rows.Next() {
			var key, gearID, itemData string
			var durability int
			if err := rows.Scan(&key, &gearID, &itemData, &durability); err != nil {
				_ = rows.Close()
				return "", err
			}
			_, _ = fmt.Fprintf(hash, "%s\x00%s\x00%s\x00%d\n", key, gearID, itemData, durability)
		}
		if err := rows.Err(); err != nil {
			_ = rows.Close()
			return "", err
		}
		_ = rows.Close()
	}
	balance, err := s.loadForgeBalance(ctx, uid)
	if err != nil {
		return "", err
	}
	materials := balance.Materials
	keys := make([]string, 0, len(materials))
	for id := range materials {
		keys = append(keys, id)
	}
	sort.Strings(keys)
	for _, id := range keys {
		_, _ = fmt.Fprintf(hash, "%s=%d\n", id, materials[id])
	}
	_, _ = fmt.Fprintf(hash, "gold=%d tokens=%d", balance.Gold, balance.Tokens)
	return hex.EncodeToString(hash.Sum(nil)[:16]), nil
}

func (s *WebServer) loadForgeBalance(ctx context.Context, uid string) (abyssForgeBalance, error) {
	balance := abyssForgeBalance{Materials: map[string]int64{}}
	if err := s.bot.DB.QueryRowContext(ctx,
		"SELECT gold, abyss_tokens FROM users WHERE client_uid=$1", uid,
	).Scan(&balance.Gold, &balance.Tokens); err != nil {
		return balance, err
	}
	rows, err := s.bot.DB.QueryContext(ctx, "SELECT mat_id, count FROM user_materials WHERE client_uid=$1", uid)
	if err != nil {
		return balance, err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var material string
		var amount int64
		if err := rows.Scan(&material, &amount); err != nil {
			return balance, err
		}
		balance.Materials[material] = amount
	}
	return balance, rows.Err()
}

func (s *WebServer) loadForgeQuoteItem(ctx context.Context, uid string, invID int64, slot string) (content.Gear, string, bool) {
	var gearID string
	var raw string
	var durability int
	var err error
	if invID > 0 {
		err = s.bot.DB.QueryRowContext(ctx,
			"SELECT gear_id, COALESCE(item_data::text,''), durability FROM user_inventory WHERE id=$1 AND client_uid=$2", invID, uid,
		).Scan(&gearID, &raw, &durability)
	} else if slot != "" {
		err = s.bot.DB.QueryRowContext(ctx,
			"SELECT gear_id, COALESCE(item_data::text,''), durability FROM user_gear WHERE slot=$1 AND client_uid=$2", slot, uid,
		).Scan(&gearID, &raw, &durability)
	} else {
		return content.Gear{}, "", false
	}
	if err != nil {
		return content.Gear{}, "", false
	}
	gear, ok := s.bot.makeGear(gearID, sql.NullString{String: raw, Valid: raw != ""})
	return gear, raw + "\x00durability=" + strconv.Itoa(durability), ok
}

func (s *WebServer) forgeQuoteBaseCost(uid, operation string, gear *content.Gear) abyssForgeQuoteCost {
	cost := abyssForgeQuoteCost{Materials: map[string]int{}}
	baseGold := map[string]int64{
		"identify": 100, "polish": 150, "reinforce": 100, "sharpen": 100,
		"reforge": 300, "reforge_lock": 600, "rebalance": 200, "rebalance_all": 500,
		"transmute_gem": 150, "socket_gem": 50,
		"extract_gem": 100, "etch_rune": 150, "cleanse": 800, "insure_item": 200,
		"infuse_curse": 500, "temper_surge": 2000, "transmute": 100,
	}[operation]
	if operation == "temper" && gear != nil {
		baseGold = int64(400 * (gear.Temper + 1))
	}
	if baseGold > 0 && gear != nil {
		cost.Gold = s.bot.forgeGoldCost(uid, baseGold, gear.Rarity)
	} else {
		cost.Gold = baseGold
	}
	materials := map[string]map[string]int{
		"reinforce": {"dust": 2}, "sharpen": {"dust": 2}, "awaken": {"core": 3},
		"imbue": {"prism": 2}, "punch_socket": {"shard": 10}, "corrupt": {"core": 5},
		"embrace": {"prism": 1}, "infuse_curse": {"core": 3}, "infuse_eldritch": {"prism": 3},
		"prismatic_rune": {"prism": 2}, "brand": {"core": 10}, "special_reroll": {"core": 6},
		"temper_guard": {"core": 2}, "craft_repair_kit2": {"dust": 8}, "unbrand": {"core": 2},
		"awaken_guided": {"core": 6}, "imbue_remove": {"prism": 1}, "swap_special": {"core": 8},
	}
	if value := materials[operation]; value != nil {
		cost.Materials = value
	}
	cost.Tokens = map[string]int{"attune": 15, "recalibrate": 5, "unattune": 50, "transfer_enchant": 5}[operation]
	return cost
}

func forgeQuoteOutcome(operation string, gear *content.Gear, chance float64) abyssForgeOutcome {
	if gear == nil {
		return abyssForgeOutcome{Consequences: []string{"Result is delivered to inventory with a server receipt."}}
	}
	minimum := *gear
	expected := *gear
	maximum := *gear
	scale := 1.0
	switch operation {
	case "temper", "reinforce", "sharpen":
		scale = 1.02
	case "masterwork":
		scale = 1.03
	case "attune", "prismatic_rune":
		scale = 1.05
	case "infuse_curse", "infuse_eldritch":
		scale = 1.25
	case "corrupt":
		scale = 1.50
	}
	maximum.Stats = gear.Stats.Scaled(scale)
	expected.Stats = gear.Stats.Scaled(1 + (scale-1)*chance)
	result := abyssForgeOutcome{
		MinimumStats: minimum.Stats, ExpectedStats: expected.Stats, MaximumStats: maximum.Stats,
		MinimumCR: minimum.CombatRating(), ExpectedCR: expected.CombatRating(), MaximumCR: maximum.CombatRating(),
	}
	if scale > 1 {
		result.Gained = append(result.Gained, fmt.Sprintf("up to %.0f%% combat-stat improvement", (scale-1)*100))
	}
	switch operation {
	case "corrupt":
		result.Consequences = append(result.Consequences, "Adds corruption and an HP penalty; trade and later cleanse rules change.")
	case "attune":
		result.Consequences = append(result.Consequences, "Binds the item; auction, fusion, salvage, and dismantle become unavailable.")
	case "extract_gem":
		result.Lost = append(result.Lost, "socketed gem")
		result.Consequences = append(result.Consequences, "The socket remains, but the extracted gem is converted to recovery materials.")
	case "scrape_rune":
		result.Lost = append(result.Lost, "etched rune")
	}
	if setID := gear.EffectiveSetID(); setID != "" {
		result.Consequences = append(result.Consequences, "Set membership remains "+setID+"; set resonance is recalculated after commit.")
	}
	if gear.FoundAt != "" || gear.Lore != "" {
		result.Consequences = append(result.Consequences, "The item's discovery date and lore provenance remain attached to its receipt.")
	}
	return result
}

func forgeQuoteChance(operation abyssForgeOperation, gear *content.Gear) (float64, string, string) {
	chance := operation.Success.Max
	pity := "No hidden mastery modifier changes these published odds."
	if operation.ID == "temper" && gear != nil {
		chance = temperChance(gear.Temper, 0)
		pity = "Displayed base odds exclude only the visible +5% per failed-attempt pity stack, capped by the operation rules."
	} else if operation.Success.Kind == "bounded_chance" {
		chance = (operation.Success.Min + operation.Success.Max) / 2
	}
	explanation := fmt.Sprintf("%.1f%% chance from the catalog's %s model (range %.1f–%.1f%%).",
		chance*100, operation.Success.Kind, operation.Success.Min*100, operation.Success.Max*100)
	return chance, explanation, pity
}

func subtractForgeBalance(before abyssForgeBalance, cost abyssForgeQuoteCost) abyssForgeBalance {
	after := abyssForgeBalance{Gold: before.Gold - cost.Gold, Tokens: before.Tokens - int64(cost.Tokens), Materials: map[string]int64{}}
	for id, value := range before.Materials {
		after.Materials[id] = value - int64(cost.Materials[id])
	}
	for id, amount := range cost.Materials {
		if _, found := after.Materials[id]; !found {
			after.Materials[id] = -int64(amount)
		}
	}
	for _, material := range abyssMaterials {
		if _, found := after.Materials[material.ID]; !found {
			after.Materials[material.ID] = 0
		}
	}
	return after
}

func (s *WebServer) buildAbyssForgeQuote(ctx context.Context, uid string, request abyssForgeQuoteRequest) (abyssForgeQuote, error) {
	operation, ok := abyssForgeOperationByID(request.Operation)
	if !ok {
		return abyssForgeQuote{}, errors.New("unknown forge operation")
	}
	parameters, err := canonicalForgeParameters(request.Parameters)
	if err != nil || len(parameters) > 32*1024 {
		return abyssForgeQuote{}, errors.New("invalid operation parameters")
	}
	request.Parameters = parameters
	revision, err := s.forgeInventoryRevision(ctx, uid)
	if err != nil {
		return abyssForgeQuote{}, fmt.Errorf("inventory revision: %w", err)
	}
	var gear *content.Gear
	fingerprint := "none"
	if operation.RequiresGear {
		loaded, raw, found := s.loadForgeQuoteItem(ctx, uid, request.InvID, request.Slot)
		if !found {
			return abyssForgeQuote{}, errors.New("item not found")
		}
		if loaded.Rarity < operation.MinimumRarity {
			return abyssForgeQuote{}, errors.New("item rarity is too low")
		}
		compatible := false
		for _, slot := range operation.CompatibleSlots {
			if loaded.Slot == slot {
				compatible = true
				break
			}
		}
		if !compatible {
			return abyssForgeQuote{}, errors.New("operation is incompatible with this gear slot")
		}
		gear = &loaded
		fingerprint = forgeGearFingerprint(loaded, raw, request.InvID, request.Slot)
	}
	var parameterValues map[string]any
	_ = json.Unmarshal(parameters, &parameterValues)
	var targetCraft *abyssTargetCraftRequest
	targetCraftDuplicate := false
	if operation.ID == "target_craft" {
		craftRequest := abyssTargetCraftRequest{
			Slot: stringParameter(parameterValues, "slot"), SetID: stringParameter(parameterValues, "set_id"),
			Event: stringParameter(parameterValues, "event"), DuplicatePolicy: stringParameter(parameterValues, "duplicate_policy"),
			GearID: stringParameter(parameterValues, "gear_id"), Rarity: int(numberParameter(parameterValues, "rarity")),
		}
		candidates := targetCraftCandidates(craftRequest)
		if len(candidates) == 0 {
			return abyssForgeQuote{}, errors.New("no catalog item matches the targeted craft")
		}
		preview, duplicate, selected := chooseTargetCraft(candidates, craftRequest, s.ownedGearIDs(uid))
		if !selected {
			return abyssForgeQuote{}, errors.New("no eligible non-duplicate item matches those targets")
		}
		gear = &preview
		targetCraft = &craftRequest
		targetCraftDuplicate = duplicate
	}
	chance, chanceText, pityText := forgeQuoteChance(operation, gear)
	cost, minimumCost, maximumCost, err := s.resolveAbyssForgeQuoteCost(ctx, uid, operation.ID, gear, parameterValues)
	if err != nil {
		return abyssForgeQuote{}, fmt.Errorf("forge cost: %w", err)
	}
	if targetCraft != nil {
		cost = abyssForgeQuoteCost{Materials: abyssTargetCraftCost(content.Rarity(targetCraft.Rarity))}
		minimumCost, maximumCost = cost, cost
	}
	if operation.ID == "batch_temper" && gear != nil {
		target := min(temperMax, int(numberParameter(parameterValues, "batch_target")))
		if target == 0 {
			target = min(temperMax, int(numberParameter(parameterValues, "target")))
		}
		if target <= gear.Temper {
			return abyssForgeQuote{}, errors.New("batch temper target must exceed the current level")
		}
		minimumCost = abyssForgeQuoteCost{Gold: s.forge4GoldCost(uid, int64(400*(gear.Temper+1)), gear.Rarity), Materials: map[string]int{}}
		maximumCost = abyssForgeQuoteCost{Materials: map[string]int{}}
		for level := gear.Temper; level < target; level++ {
			maximumCost.Gold += s.forge4GoldCost(uid, int64(400*(level+1)), gear.Rarity)
		}
		if boolParameter(parameterValues, "use_protection") || boolParameter(parameterValues, "auto_protection") {
			maximumCost.Materials["core"] = 2
		}
		cost = maximumCost
	}
	if operation.ID == "gem_upgrade_all" && gear != nil {
		stopAtTier := int(numberParameter(parameterValues, "stop_at_tier"))
		if stopAtTier == 0 {
			stopAtTier = 2
		}
		cost = abyssForgeQuoteCost{Materials: map[string]int{}}
		for _, gem := range gear.Gemstones {
			_, tier := parseGem(gem)
			for tier < min(3, stopAtTier) {
				if tier == 1 {
					cost.Gold += s.forge4GoldCost(uid, 200, gear.Rarity)
					cost.Materials["shard"] += 5
				} else {
					cost.Gold += s.forge4GoldCost(uid, 500, gear.Rarity)
					cost.Materials["core"] += 2
				}
				tier++
			}
		}
		minimumCost, maximumCost = cost, cost
	}
	if operation.ID == "etch_rune" && gear != nil {
		runeName := stringParameter(parameterValues, "rune")
		if strings.EqualFold(stringParameter(parameterValues, "rune_family"), "defensive") {
			runeName = content.DefensiveRuneName(content.Element(runeName))
		}
		if runeName != "" {
			var known bool
			if err := s.bot.DB.QueryRowContext(ctx,
				"SELECT EXISTS(SELECT 1 FROM user_runes WHERE client_uid=$1 AND rune=$2)", uid, runeName,
			).Scan(&known); err != nil {
				return abyssForgeQuote{}, fmt.Errorf("rune library: %w", err)
			}
			baseGold := int64(150)
			if known {
				baseGold = 50
			}
			cost.Gold = s.bot.forgeGoldCost(uid, baseGold, gear.Rarity)
			minimumCost, maximumCost = cost, cost
		}
	}
	if operation.ID == "corrupt" && boolParameter(parameterValues, "use_protection") {
		cost.Materials["prism"]++
		minimumCost, maximumCost = cost, cost
	}
	var transferTarget *content.Gear
	if operation.ID == "masterwork_transfer" && gear != nil {
		targetInvID := int64(numberParameter(parameterValues, "target_inv_id"))
		targetSlot := stringParameter(parameterValues, "target_slot")
		if targetInvID == 0 {
			targetInvID = int64(numberParameter(parameterValues, "inv_id2"))
		}
		if targetSlot == "" {
			targetSlot = stringParameter(parameterValues, "slot2")
		}
		if targetInvID == request.InvID && targetInvID > 0 || targetSlot == request.Slot && targetSlot != "" {
			return abyssForgeQuote{}, errors.New("source and target must be different items")
		}
		target, _, found := s.loadForgeQuoteItem(ctx, uid, targetInvID, targetSlot)
		if !found {
			return abyssForgeQuote{}, errors.New("select a masterwork transfer target")
		}
		if target.Slot != gear.Slot {
			return abyssForgeQuote{}, errors.New("masterwork source and target must share a slot")
		}
		if gear.Quality <= 0 {
			return abyssForgeQuote{}, errors.New("source item has no masterwork quality")
		}
		transferTarget = &target
		cost = abyssForgeQuoteCost{Gold: s.forge4GoldCost(uid, 300, target.Rarity), Materials: map[string]int{"core": 2}}
		minimumCost, maximumCost = cost, cost
	}
	forgeFloorFree := s.bot.abyssForgeFloorAvailable(ctx, uid, operation.ID)
	cost, minimumCost, maximumCost = applyAbyssForgeFloorQuoteCost(
		forgeFloorFree,
		cost,
		minimumCost,
		maximumCost,
	)
	before, err := s.loadForgeBalance(ctx, uid)
	if err != nil {
		return abyssForgeQuote{}, fmt.Errorf("forge balance: %w", err)
	}
	expires := time.Now().Add(abyssForgeQuoteTTL).UTC()
	claims := abyssForgeQuoteClaims{
		UID: uid, Operation: operation.ID, InvID: request.InvID, Slot: request.Slot, Parameters: parameters,
		Gear: fingerprint, Inventory: revision, ForgeFloor: forgeFloorFree, ExpiresUnix: expires.Unix(),
	}
	token, err := s.signForgeClaims(claims)
	if err != nil {
		return abyssForgeQuote{}, err
	}
	quote := abyssForgeQuote{
		SchemaVersion: abyssForgeCatalogSchemaVersion, CatalogHash: currentAbyssForgeCatalogSummary().CatalogHash,
		Operation: operation.ID, Parameters: parameters, GearFingerprint: fingerprint, InventoryRevision: revision,
		ExpiresAt: expires, Token: token, Irreversible: !operation.Reversible, SuccessChance: chance,
		ChanceExplanation: chanceText, FailureExplanation: operation.Failure, PityExplanation: pityText,
		UndoAvailable: operation.Reversible, UndoWindowSeconds: operation.UndoWindowSeconds,
		Cost: cost, CostMinimum: minimumCost, CostMaximum: maximumCost,
		BalanceBefore: before, BalanceAfter: subtractForgeBalance(before, cost), Current: gear,
		Outcome:        forgeQuoteOutcome(operation.ID, gear, chance),
		TradeableAfter: true, Recovery: map[string]int{},
	}
	if operation.ID == "gem_upgrade_all" && gear != nil {
		quote.Outcome = forgeBulkGemOutcome(*gear, int(numberParameter(parameterValues, "stop_at_tier")))
	}
	if operation.ID == "corrupt" && gear != nil {
		quote.SuccessChance = 0.05
		quote.ChanceExplanation = "Corruption always applies; 5.0% of outcomes are perfect and avoid the HP malus."
		quote.FailureExplanation = "The remaining 95.0% apply the displayed HP malus; protection halves that malus."
		quote.Outcome = forgeCorruptionOutcome(*gear, boolParameter(parameterValues, "use_protection"))
	}
	if operation.ID == "awaken" || operation.ID == "awaken_guided" {
		quote.PityExplanation = "Pity: not applicable — awakening is guaranteed; guided awakening locks three choices until one is committed."
	}
	if operation.ID == "special_reroll" && gear != nil {
		excluded := map[string]bool{}
		for _, effect := range stringSliceParameter(parameterValues, "excluded_effects") {
			excluded[strings.ToLower(effect)] = true
		}
		pool := make([]string, 0, len(awakenPool))
		for _, effect := range awakenPool {
			if effect != gear.Special && !excluded[strings.ToLower(string(effect))] {
				pool = append(pool, string(effect))
			}
		}
		if len(pool) == 0 {
			return abyssForgeQuote{}, errors.New("exclusions leave no alternative Special")
		}
		sort.Strings(pool)
		quote.Outcome.Gained = []string{"one of: " + strings.Join(pool, ", ")}
		quote.Outcome.Lost = []string{"current Special: " + string(gear.Special)}
		quote.Outcome.Consequences = append(quote.Outcome.Consequences,
			fmt.Sprintf("%d of %d alternative Specials remain after exclusions.", len(pool), max(0, len(awakenPool)-1)))
	}
	if transferTarget != nil && gear != nil {
		moved := min(gear.Quality*4/5, masterworkMax-transferTarget.Quality)
		if moved < 1 {
			return abyssForgeQuote{}, errors.New("nothing would transfer at 80% efficiency")
		}
		after := *transferTarget
		for range moved {
			after.Stats = after.Stats.Scaled(1.03)
		}
		after.Quality += moved
		quote.Outcome = abyssForgeOutcome{
			MinimumStats: transferTarget.Stats, ExpectedStats: after.Stats, MaximumStats: after.Stats,
			MinimumCR: transferTarget.CombatRating(), ExpectedCR: after.CombatRating(), MaximumCR: after.CombatRating(),
			Gained: []string{fmt.Sprintf("target quality +%d to %s", moved, qualityNames[after.Quality])},
			Lost:   []string{fmt.Sprintf("source quality %d→0; source's baked stats remain", gear.Quality)},
			Consequences: []string{fmt.Sprintf("80%% efficiency transfers %d of %d source levels; target CR %.1f→%.1f",
				moved, gear.Quality, transferTarget.CombatRating(), after.CombatRating())},
		}
	}
	if gear != nil {
		quote.DurabilityBefore, quote.DurabilityAfter = gear.MaxDurability, gear.MaxDurability
		quote.SocketsBefore, quote.SocketsAfter = gear.Sockets, gear.Sockets
		quote.SetBefore, quote.SetAfter = gear.EffectiveSetID(), gear.EffectiveSetID()
		quote.BoundAfter = gear.Attuned || operation.ID == "attune"
		quote.TradeableAfter = !quote.BoundAfter
		switch operation.ID {
		case "polish":
			quote.DurabilityAfter = min(gear.MaxDurability+10, gear.MaxDurability+100)
		case "punch_socket":
			quote.SocketsAfter = min(4, gear.Sockets+1)
		case "scrape_rune":
			var known bool
			if gear.Rune != "" {
				if err := s.bot.DB.QueryRowContext(ctx,
					"SELECT EXISTS(SELECT 1 FROM user_runes WHERE client_uid=$1 AND rune=$2)", uid, gear.Rune,
				).Scan(&known); err != nil {
					return abyssForgeQuote{}, fmt.Errorf("rune recovery: %w", err)
				}
			}
			quote.Recovery["dust"] = 75
			if known {
				quote.Recovery["dust"] = 25
			}
		case "extract_gem":
			index := int(numberParameter(parameterValues, "gem_index"))
			if index < 0 || index >= len(gear.Gemstones) {
				return abyssForgeQuote{}, errors.New("no gem in the selected socket")
			}
			base, tier := parseGem(gear.Gemstones[index])
			baseStats, valid := gemBaseStats(base)
			if !valid || tier < 1 || tier > 3 {
				return abyssForgeQuote{}, errors.New("unknown gem in the selected socket")
			}
			quote.Recovery["shard"] = 2 * tier
			after := *gear
			after.Stats = after.Stats.Add(baseStats.Scaled(-[]float64{0, 1, 2, 4}[tier]))
			quote.Outcome.MinimumStats, quote.Outcome.ExpectedStats, quote.Outcome.MaximumStats = after.Stats, after.Stats, after.Stats
			quote.Outcome.MinimumCR, quote.Outcome.ExpectedCR, quote.Outcome.MaximumCR = after.CombatRating(), after.CombatRating(), after.CombatRating()
			quote.Outcome.Lost = []string{gemName(base, tier)}
		case "cleanse":
			if !gear.Corrupted || gear.Embraced {
				return abyssForgeQuote{}, errors.New("this item has no cleanseable corruption")
			}
			after := *gear
			after.Stats.HP += gear.CorruptHP
			after.CorruptHP = 0
			after.Corrupted = false
			quote.Recovery["max_hp"] = gear.CorruptHP
			quote.Outcome.MinimumStats, quote.Outcome.ExpectedStats, quote.Outcome.MaximumStats = after.Stats, after.Stats, after.Stats
			quote.Outcome.MinimumCR, quote.Outcome.ExpectedCR, quote.Outcome.MaximumCR = after.CombatRating(), after.CombatRating(), after.CombatRating()
			quote.Outcome.Gained = append(quote.Outcome.Gained, fmt.Sprintf("%d max HP restored", gear.CorruptHP))
		}
		equipped := s.bot.getEquippedItems(uid)
		if operation.ID == "socket_gem" {
			if preview := forgeGemResonancePreview(equipped, *gear, stringParameter(parameterValues, "gem")); preview != "" {
				quote.Outcome.Consequences = append(quote.Outcome.Consequences, preview)
			}
		}
		if preview := forgeSetResonancePreview(equipped, *gear); preview != "" {
			quote.Outcome.Consequences = append(quote.Outcome.Consequences, preview)
		}
		if operation.ID == "etch_rune" {
			runeName := stringParameter(parameterValues, "rune")
			if runeName != "" {
				quote.Outcome.Consequences = append(quote.Outcome.Consequences,
					forgeRuneImpact(stringParameter(parameterValues, "rune_family"), runeName))
			}
		}
	}
	switch operation.ID {
	case "craft":
		var week string
		var done int
		if err := s.bot.DB.QueryRowContext(ctx,
			"SELECT COALESCE(craft_quest_week,''), craft_quest_done FROM users WHERE client_uid=$1", uid,
		).Scan(&week, &done); err != nil {
			return abyssForgeQuote{}, fmt.Errorf("craft quest balance: %w", err)
		}
		if week == craftQuestWeek() && done+1 == craftQuestTarget {
			quote.Recovery["tokens"] = 15
			quote.BalanceAfter.Tokens += 15
		}
	case "convert_mats":
		conversion, ok := forge4ConvertRates[strings.ToLower(stringParameter(parameterValues, "from"))+":"+strings.ToLower(stringParameter(parameterValues, "to"))]
		if ok {
			count := int(numberParameter(parameterValues, "count"))
			if count < 1 {
				count = 1
			}
			quote.Recovery[conversion.to] = min(1000, count)
		}
	case "dismantle":
		tokens, materials, err := s.forgeDismantleQuoteRecovery(ctx, uid, int(numberParameter(parameterValues, "max_rarity")))
		if err != nil {
			return abyssForgeQuote{}, fmt.Errorf("dismantle recovery: %w", err)
		}
		quote.Recovery["tokens"] = int(tokens)
		quote.BalanceAfter.Tokens += tokens
		for material, amount := range materials {
			quote.Recovery[material] += amount
		}
	case "target_craft":
		if targetCraft != nil && targetCraftDuplicate && targetCraft.DuplicatePolicy == "materials" {
			material, amount := materialYieldForRarity(content.Rarity(targetCraft.Rarity))
			quote.Recovery[material] += amount * 2
		}
	}
	for material, amount := range quote.Recovery {
		if _, tracked := quote.BalanceAfter.Materials[material]; tracked {
			quote.BalanceAfter.Materials[material] += int64(amount)
		}
	}
	switch operation.ID {
	case "reforge":
		quote.CostExplanation = "Plain reforge uses the base rarity-scaled price."
	case "reforge_lock":
		quote.CostExplanation = "Locking one stat doubles the 300g base to 600g before rarity, reputation, happy-hour, and mastery modifiers."
	default:
		quote.CostExplanation = operation.Cost.Formula
	}
	if forgeFloorFree {
		quote.CostExplanation = "The active Silent Anvil makes one temper, socket punch, or full repair free and is consumed only when that mutation commits."
	}
	if quote.Irreversible {
		quote.Confirmation = "FORGE " + strings.ToUpper(operation.ID)
		quote.Warnings = append(quote.Warnings, "This operation is irreversible. Type the confirmation phrase exactly before committing.")
	}
	if quote.BalanceAfter.Gold < 0 || quote.BalanceAfter.Tokens < 0 {
		quote.Warnings = append(quote.Warnings, "Current currency balance is insufficient.")
	}
	for id, value := range quote.BalanceAfter.Materials {
		if value < 0 {
			quote.Warnings = append(quote.Warnings, "Insufficient "+abyssMaterialName(id)+".")
		}
	}
	return quote, nil
}

func forgeCorruptionOutcome(gear content.Gear, protected bool) abyssForgeOutcome {
	worst, best := gear, gear
	for _, candidate := range []*content.Gear{&worst, &best} {
		candidate.Stats.STR = candidate.Stats.STR * 3 / 2
		candidate.Stats.DEF = candidate.Stats.DEF * 3 / 2
		candidate.Stats.SPD = candidate.Stats.SPD * 3 / 2
	}
	malus := worst.Stats.Score()
	if protected {
		malus = max(1, malus/2)
	}
	worst.Stats.HP -= malus
	expected := worst
	expected.Stats.HP = (worst.Stats.HP*95 + best.Stats.HP*5) / 100
	protection := "No protection; the normal outcome applies the full HP malus."
	if protected {
		protection = "One Eldritch Prism halves the normal HP malus; perfect corruption still removes it entirely."
	}
	return abyssForgeOutcome{
		MinimumStats: worst.Stats, ExpectedStats: expected.Stats, MaximumStats: best.Stats,
		MinimumCR: worst.CombatRating(), ExpectedCR: expected.CombatRating(), MaximumCR: best.CombatRating(),
		Gained:       []string{"+50% STR, DEF, and SPD", "5% perfect-corruption outcome"},
		Lost:         []string{fmt.Sprintf("up to %d max HP on a normal outcome", malus)},
		Consequences: []string{protection, "Cleanse restores the quoted CorruptHP malus while retaining offensive gains."},
	}
}

func forgeBulkGemOutcome(gear content.Gear, stopAtTier int) abyssForgeOutcome {
	if stopAtTier == 0 {
		stopAtTier = 2
	}
	before := gear
	steps := 0
	for index, gem := range gear.Gemstones {
		base, tier := parseGem(gem)
		baseStats, valid := gemBaseStats(base)
		if !valid {
			continue
		}
		for tier < min(3, stopAtTier) {
			gear.Stats = gear.Stats.Add(baseStats.Scaled(float64(tier)))
			tier++
			steps++
		}
		gear.Gemstones[index] = gemName(base, tier)
	}
	return abyssForgeOutcome{
		MinimumStats: before.Stats, ExpectedStats: gear.Stats, MaximumStats: gear.Stats,
		MinimumCR: before.CombatRating(), ExpectedCR: gear.CombatRating(), MaximumCR: gear.CombatRating(),
		Gained:       []string{fmt.Sprintf("%d deterministic gem tier upgrades", steps)},
		Consequences: []string{fmt.Sprintf("Stops every eligible gem at tier %d.", stopAtTier)},
	}
}

func stringParameter(values map[string]any, key string) string {
	value, _ := values[key].(string)
	return value
}

func numberParameter(values map[string]any, key string) float64 {
	value, _ := values[key].(float64)
	return value
}

func boolParameter(values map[string]any, key string) bool {
	value, _ := values[key].(bool)
	return value
}

func stringSliceParameter(values map[string]any, key string) []string {
	raw, _ := values[key].([]any)
	result := make([]string, 0, len(raw))
	for _, value := range raw {
		if text, ok := value.(string); ok {
			text = strings.TrimSpace(text)
			if text != "" {
				result = append(result, text)
			}
		}
	}
	return result
}

func decodeBoundedForgeRequest(r *http.Request, target any) error {
	decoder := json.NewDecoder(io.LimitReader(r.Body, abyssForgeRequestMax+1))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if decoder.Decode(&struct{}{}) != io.EOF {
		return errors.New("multiple JSON values")
	}
	return nil
}

func (s *WebServer) handleAbyssForgeQuote(w http.ResponseWriter, r *http.Request, uid string) {
	started := time.Now()
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	if !s.abyssFeatures.enabled("forge", uid) {
		writeJSON(w, map[string]any{"ok": false, "error": "forge workbench is temporarily disabled"})
		return
	}
	var request abyssForgeQuoteRequest
	if err := decodeBoundedForgeRequest(r, &request); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "invalid forge quote request"})
		return
	}
	quote, err := s.buildAbyssForgeQuote(r.Context(), uid, request)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	s.abyssForgeOps.quotes.Add(1)
	s.abyssForgeOps.quoteNanos.Add(time.Since(started).Nanoseconds())
	s.abyssForgeOps.trackQuote(quote.Token, quote.Operation, started, quote.ExpiresAt)
	writeJSON(w, map[string]any{"ok": true, "quote": quote})
}
