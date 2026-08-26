package bot

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"ts3news/internal/content"
)

// forgeQuoteCostCoverage is deliberately exhaustive. A catalog operation may
// not ship without choosing an authoritative quote-cost policy, including the
// explicit no-cost policy used by state-only operations.
var forgeQuoteCostCoverage = map[string]string{
	"attune": "fixed", "auto_repair": "none", "awaken": "fixed", "awaken_guided": "fixed",
	"batch_temper": "range", "brand": "fixed", "celestial_fuse": "fusion", "celestial_fuse_boosted": "fusion",
	"cleanse": "fixed", "convert_mats": "parameters", "corrupt": "parameters", "craft": "recipe",
	"craft_legendary": "fixed", "craft_repair_kit2": "fixed", "dismantle": "recovery", "embrace": "fixed",
	"etch_rune": "library", "extract_gem": "fixed", "forge_queue": "range", "fuse": "fusion",
	"fuse_preview": "none", "gem_upgrade_all": "item", "identify": "fixed", "identify_all": "inventory",
	"imbue": "fixed", "imbue_remove": "fixed", "infuse_curse": "fixed", "infuse_eldritch": "fixed",
	"infuse_xp": "none", "insure_item": "fixed", "masterwork": "item", "masterwork_transfer": "target",
	"mythic_fuse": "fusion", "polish": "fixed", "polish_all": "inventory", "prismatic_rune": "fixed",
	"punch_socket": "fixed", "rebalance": "fixed", "rebalance_all": "fixed", "recalibrate": "fixed",
	"reforge": "fixed", "reforge_lock": "fixed", "reinforce": "fixed", "repair_all": "inventory", "reroll_ring_sockets": "fixed",
	"scrape_rune": "recovery", "sharpen": "fixed", "socket_gem": "fixed", "socket_relocate": "fixed",
	"special_reroll": "fixed", "swap_special": "fixed", "target_craft": "parameters", "temper": "item",
	"temper_guard": "fixed", "temper_surge": "fixed", "transfer_enchant": "fixed", "transmute": "fixed",
	"transmute_gem": "fixed", "unattune": "fixed", "unbrand": "fixed", "upgrade_gear": "item",
	"upgrade_gem": "item",
}

func cloneForgeQuoteCost(cost abyssForgeQuoteCost) abyssForgeQuoteCost {
	result := cost
	result.Materials = make(map[string]int, len(cost.Materials))
	for id, amount := range cost.Materials {
		result.Materials[id] = amount
	}
	return result
}

func forgeInt64SliceParameter(values map[string]any, key string) []int64 {
	raw, ok := values[key].([]any)
	if !ok {
		return nil
	}
	result := make([]int64, 0, len(raw))
	for _, value := range raw {
		if number, ok := value.(float64); ok {
			result = append(result, int64(number))
		}
	}
	return result
}

func (s *WebServer) resolveAbyssForgeQuoteCost(
	ctx context.Context,
	uid string,
	operation string,
	gear *content.Gear,
	parameters map[string]any,
) (abyssForgeQuoteCost, abyssForgeQuoteCost, abyssForgeQuoteCost, error) {
	if _, covered := forgeQuoteCostCoverage[operation]; !covered {
		return abyssForgeQuoteCost{}, abyssForgeQuoteCost{}, abyssForgeQuoteCost{},
			fmt.Errorf("forge operation %q has no quote cost policy", operation)
	}
	cost := s.forgeQuoteBaseCost(uid, operation, gear)
	minimum, maximum := cloneForgeQuoteCost(cost), cloneForgeQuoteCost(cost)
	setExact := func(value abyssForgeQuoteCost) {
		cost = cloneForgeQuoteCost(value)
		minimum = cloneForgeQuoteCost(value)
		maximum = cloneForgeQuoteCost(value)
	}

	switch operation {
	case "temper":
		if gear != nil {
			setExact(abyssForgeQuoteCost{Gold: s.forge4GoldCost(uid, int64(400*(gear.Temper+1)), gear.Rarity), Materials: map[string]int{}})
		}
	case "reforge_lock":
		if gear != nil {
			setExact(abyssForgeQuoteCost{Gold: s.forge4GoldCost(uid, 600, gear.Rarity), Materials: map[string]int{}})
		}
	case "rebalance_all":
		if gear != nil {
			setExact(abyssForgeQuoteCost{Gold: s.forge4GoldCost(uid, 500, gear.Rarity), Materials: map[string]int{}})
		}
	case "socket_relocate":
		if gear != nil {
			setExact(abyssForgeQuoteCost{Gold: s.forge4GoldCost(uid, 50, gear.Rarity), Materials: map[string]int{}})
		}
	case "reroll_ring_sockets":
		if gear == nil || !isAbyssRingSlot(gear.Slot) {
			return cost, minimum, maximum, errors.New("socket rerolling is limited to rings")
		}
		if gear.Unidentified {
			return cost, minimum, maximum, errors.New("identify this ring before rerolling its sockets")
		}
		if len(gear.Gemstones) > abyssRingSocketMaximum {
			return cost, minimum, maximum, fmt.Errorf("extract gems until at most %d remain before rerolling", abyssRingSocketMaximum)
		}
	case "masterwork":
		if gear == nil || gear.Quality >= masterworkMax {
			return cost, minimum, maximum, errors.New("item cannot be masterworked further")
		}
		quality := gear.Quality
		materials := map[string]int{"dust": (quality + 1) * 5, "shard": (quality + 1) * 2}
		if quality >= 2 {
			materials["core"] = quality - 1
		}
		setExact(abyssForgeQuoteCost{Materials: materials})
	case "upgrade_gear":
		if gear == nil || gear.Rarity >= content.RarityEternal {
			return cost, minimum, maximum, errors.New("item is already at maximum rarity")
		}
		setExact(abyssForgeQuoteCost{Tokens: int(abyssUpgradeGearCost(gear.Rarity + 1)), Materials: map[string]int{}})
	case "upgrade_gem":
		if gear == nil {
			break
		}
		index := int(numberParameter(parameters, "gem_index"))
		if index < 0 || index >= len(gear.Gemstones) {
			return cost, minimum, maximum, errors.New("no gem in the selected socket")
		}
		_, tier := parseGem(gear.Gemstones[index])
		switch tier {
		case 1:
			setExact(abyssForgeQuoteCost{Gold: s.bot.forgeGoldCost(uid, 200, gear.Rarity), Materials: map[string]int{"shard": 5}})
		case 2:
			setExact(abyssForgeQuoteCost{Gold: s.bot.forgeGoldCost(uid, 500, gear.Rarity), Materials: map[string]int{"core": 2}})
		default:
			return cost, minimum, maximum, errors.New("gem cannot be upgraded further")
		}
	case "craft":
		recipe, ok := craftRecipeByID(stringParameter(parameters, "recipe_id"))
		if !ok {
			return cost, minimum, maximum, errors.New("unknown recipe")
		}
		setExact(abyssForgeQuoteCost{Materials: recipe.Cost})
	case "craft_legendary":
		setExact(abyssForgeQuoteCost{Materials: craftLegendaryCost})
	case "convert_mats":
		conversion, ok := forge4ConvertRates[strings.ToLower(stringParameter(parameters, "from"))+":"+strings.ToLower(stringParameter(parameters, "to"))]
		if !ok {
			return cost, minimum, maximum, errors.New("unsupported material conversion")
		}
		count := int(numberParameter(parameters, "count"))
		if count < 1 {
			count = 1
		}
		count = min(1000, count)
		setExact(abyssForgeQuoteCost{Materials: map[string]int{conversion.from: conversion.rate * count}})
	case "fuse", "mythic_fuse", "celestial_fuse", "celestial_fuse_boosted":
		ids := forgeInt64SliceParameter(parameters, "inv_ids")
		if len(ids) == 0 {
			return cost, minimum, maximum, errors.New("select fusion items")
		}
		var gearID string
		var itemData sql.NullString
		if err := s.bot.DB.QueryRowContext(ctx,
			"SELECT gear_id, item_data FROM user_inventory WHERE id=$1 AND client_uid=$2 AND locked=FALSE", ids[0], uid,
		).Scan(&gearID, &itemData); err != nil {
			return cost, minimum, maximum, errors.New("fusion item not found")
		}
		item, ok := s.bot.makeGear(gearID, itemData)
		if !ok {
			return cost, minimum, maximum, errors.New("fusion item is invalid")
		}
		gold := s.bot.forgeGoldCost(uid, 2000, item.Rarity)
		materials := map[string]int{}
		if operation == "celestial_fuse_boosted" {
			gold = s.forge4GoldCost(uid, 2000, content.RarityCelestial)
			materials["prism"] = 10
		}
		setExact(abyssForgeQuoteCost{Gold: gold, Materials: materials})
	case "polish_all":
		rows, err := s.bot.DB.QueryContext(ctx, "SELECT gear_id, item_data FROM user_gear WHERE client_uid=$1", uid)
		if err != nil {
			return cost, minimum, maximum, err
		}
		defer func() { _ = rows.Close() }()
		var total int64
		for rows.Next() {
			var gearID string
			var itemData sql.NullString
			if err := rows.Scan(&gearID, &itemData); err != nil {
				return cost, minimum, maximum, err
			}
			item, ok := s.bot.makeGear(gearID, itemData)
			if !ok {
				continue
			}
			baseMaximum := item.MaxDurability
			if catalogItem, found := content.GetGearByID(item.ID); found {
				baseMaximum = catalogItem.MaxDurability
			}
			if item.MaxDurability < baseMaximum+100 {
				total += s.forge4GoldCost(uid, 150, item.Rarity)
			}
		}
		if err := rows.Err(); err != nil {
			return cost, minimum, maximum, err
		}
		setExact(abyssForgeQuoteCost{Gold: total, Materials: map[string]int{}})
	case "repair_all":
		cost := s.bot.abyssRepairAllCost(uid)
		setExact(abyssForgeQuoteCost{Gold: abyssRepairSubscriptionCharge(cost, s.bot.abyssRepairSubscriptionActive(uid, time.Now())), Materials: map[string]int{}})
	case "identify_all":
		count, err := s.countUnidentifiedForgeItems(ctx, uid)
		if err != nil {
			return cost, minimum, maximum, err
		}
		setExact(abyssForgeQuoteCost{Gold: int64(abyssIdentifyCost * count), Materials: map[string]int{}})
	case "forge_queue":
		if gear != nil {
			resolvedCost, resolvedMinimum, resolvedMaximum, resolveErr := s.forgeQueueQuoteCost(uid, *gear, parameters)
			if resolveErr != nil {
				return cost, minimum, maximum, resolveErr
			}
			cost, minimum, maximum = resolvedCost, resolvedMinimum, resolvedMaximum
		}
	}

	return cost, minimum, maximum, nil
}

func (s *WebServer) countUnidentifiedForgeItems(ctx context.Context, uid string) (int, error) {
	queries := []string{
		"SELECT gear_id, item_data FROM user_inventory WHERE client_uid=$1",
		"SELECT gear_id, item_data FROM user_gear WHERE client_uid=$1",
	}
	count := 0
	for _, query := range queries {
		rows, err := s.bot.DB.QueryContext(ctx, query, uid)
		if err != nil {
			return 0, err
		}
		for rows.Next() {
			var gearID string
			var itemData sql.NullString
			if err := rows.Scan(&gearID, &itemData); err != nil {
				_ = rows.Close()
				return 0, err
			}
			if item, ok := s.bot.makeGear(gearID, itemData); ok && item.Unidentified {
				count++
			}
		}
		if err := rows.Err(); err != nil {
			_ = rows.Close()
			return 0, err
		}
		_ = rows.Close()
	}
	return count, nil
}

func (s *WebServer) forgeDismantleQuoteRecovery(
	ctx context.Context,
	uid string,
	maximumRarity int,
) (int64, map[string]int, error) {
	reserved := s.bot.loadAbyssReservedLoot(uid)
	rows, err := s.bot.DB.QueryContext(ctx, "SELECT id, gear_id, item_data FROM user_inventory WHERE client_uid=$1 AND locked=FALSE", uid)
	if err != nil {
		return 0, nil, err
	}
	defer func() { _ = rows.Close() }()
	var tokens int64
	materials := map[string]int{}
	for rows.Next() {
		var inventoryID int64
		var gearID string
		var itemData sql.NullString
		if err := rows.Scan(&inventoryID, &gearID, &itemData); err != nil {
			return 0, nil, err
		}
		if reserved[inventoryID] {
			continue
		}
		item, ok := s.bot.makeGear(gearID, itemData)
		if !ok || item.Attuned || maximumRarity > 0 && int(item.Rarity) > maximumRarity {
			continue
		}
		yield := abyssDismantleTokens(item.Rarity)
		if yield <= 0 {
			continue
		}
		tokens += yield
		material, amount := materialYieldForRarity(item.Rarity)
		materials[material] += amount
	}
	if err := rows.Err(); err != nil {
		return 0, nil, err
	}
	return tokens, s.bot.boostedMaterials(uid, materials), nil
}

func (s *WebServer) forgeQueueQuoteCost(uid string, gear content.Gear, parameters map[string]any) (
	abyssForgeQuoteCost,
	abyssForgeQuoteCost,
	abyssForgeQuoteCost,
	error,
) {
	actions := stringSliceParameter(parameters, "actions")
	if len(actions) == 0 || len(actions) > 3 {
		return abyssForgeQuoteCost{}, abyssForgeQuoteCost{}, abyssForgeQuoteCost{}, errors.New("forge queue requires one to three actions")
	}
	minimum := abyssForgeQuoteCost{Materials: map[string]int{}}
	maximum := abyssForgeQuoteCost{Materials: map[string]int{}}
	minimumTemper, maximumTemper := gear.Temper, gear.Temper
	for _, action := range actions {
		switch action {
		case "polish":
			price := s.forge4GoldCost(uid, 150, gear.Rarity)
			minimum.Gold += price
			maximum.Gold += price
		case "reinforce", "sharpen":
			price := s.forge4GoldCost(uid, 100, gear.Rarity)
			minimum.Gold += price
			maximum.Gold += price
			minimum.Materials["dust"] += 2
			maximum.Materials["dust"] += 2
		case "masterwork":
			minimum.Materials["dust"] += (gear.Quality + 1) * 5
			maximum.Materials["dust"] += (gear.Quality + 1) * 5
			minimum.Materials["shard"] += (gear.Quality + 1) * 2
			maximum.Materials["shard"] += (gear.Quality + 1) * 2
			if gear.Quality >= 2 {
				minimum.Materials["core"] += gear.Quality - 1
				maximum.Materials["core"] += gear.Quality - 1
			}
			gear.Quality++
		case "temper":
			minimum.Gold += s.forge4GoldCost(uid, int64(400*(minimumTemper+1)), gear.Rarity)
			maximum.Gold += s.forge4GoldCost(uid, int64(400*(maximumTemper+1)), gear.Rarity)
			maximumTemper++
		default:
			return abyssForgeQuoteCost{}, minimum, maximum, fmt.Errorf("unknown forge queue action %q", action)
		}
	}
	return cloneForgeQuoteCost(maximum), minimum, maximum, nil
}
