package bot

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math/rand/v2"
	"strconv"

	"ts3news/internal/content"
)

const (
	shopRarityPositions = 20
	shopStockPageSize   = 240
)

// Independent deterministic streams keep reloads from rerolling stock.
// These are game rolls, not security tokens.
func shopBuffRandom(seed int64, uid, purpose string) *rand.Rand {
	hash := sha256.Sum256([]byte(fmt.Sprintf("shop-buff-v1:%d:%s:%s", seed, uid, purpose)))
	return rand.New(rand.NewPCG(binary.LittleEndian.Uint64(hash[:8]), binary.LittleEndian.Uint64(hash[8:16]))) // #nosec G404 -- deterministic shop stock
}

func shopStockRevision(seed int64, uid string, buffs shopBuffState) string {
	// Price-policy changes invalidate already-open reviews before any gold debit.
	hash := sha256.Sum256([]byte(fmt.Sprintf("shop-v2:%d:%s:%d:%d", seed, uid, buffs.Rarity, buffs.Quantity)))
	return fmt.Sprintf("%x", hash[:16])
}

func shopQuantityExtra(seed int64, uid string, owned int64) int64 {
	// Divide first so even a large persisted counter cannot overflow.
	extra := owned / 1000 * shopStockSize
	remainder := owned % 1000 * shopStockSize
	extra += remainder / 1000
	if shopBuffRandom(seed, uid, "quantity").Int64N(1000) < remainder%1000 {
		extra++
	}
	return extra
}

func personalizedShopStock(seed int64, uid string, buffs shopBuffState, equipped map[string]content.Gear) []shopItemView {
	return personalizedShopStockPage(seed, uid, buffs, equipped, 0)
}

type shopStockPageView struct {
	Page, Pages, Total, Start, End int64
	PreviousURL, NextURL           string
}

func shopStockPagination(seed int64, uid string, buffs shopBuffState, page int64) shopStockPageView {
	normal := int64(shopStockSize) + shopQuantityExtra(seed, uid, buffs.Quantity)
	pages := (normal + shopStockPageSize - 1) / shopStockPageSize
	page = min(max(page, 0), pages-1)
	info := shopStockPageView{Page: page, Pages: pages, Total: normal + 1, Start: page*shopStockPageSize + 1, End: min((page+1)*shopStockPageSize, normal)}
	if page > 0 {
		info.PreviousURL = "/shop?stock_page=" + strconv.FormatInt(page-1, 10)
	}
	if page+1 < pages {
		info.NextURL = "/shop?stock_page=" + strconv.FormatInt(page+1, 10)
	}
	return info
}

func shopRarityRolls(seed int64, uid string) map[int64]int64 {
	r := shopBuffRandom(seed, uid, "rarity")
	positions := r.Perm(shopStockSize)
	selected := make(map[int64]int64, shopRarityPositions)
	// Select from the fixed base slots so buying quantity never takes an existing
	// rarity upgrade away. Extra offers do not reroll this rotation's 20 positions.
	for _, position := range positions[:min(shopRarityPositions, len(positions))] {
		selected[int64(position)] = r.Int64N(1000)
	}
	return selected
}

func applyShopRarityBuff(g content.Gear, owned, roll int64) content.Gear {
	steps := owned / 1000
	if roll < owned%1000 {
		steps++
	}
	toEternal := int64(content.RarityEternal - g.Rarity)
	g.Rarity += content.Rarity(min(steps, toEternal))
	// One upgrade = 1,000 tokens. Each excess upgrade grants 0.1% of the
	// original stats, with fractional progress. Integer math avoids rounding
	// 1,000 * 1.001 down to 1,000 because of floating-point representation.
	extra := max(owned-toEternal*1000, 0)
	if extra > 0 {
		scale := func(base int) int {
			return base + int(int64(base)*(extra/1_000_000)+int64(base)*(extra%1_000_000)/1_000_000)
		}
		s := &g.Stats
		s.HP, s.STR, s.DEF, s.SPD, s.LCK = scale(s.HP), scale(s.STR), scale(s.DEF), scale(s.SPD), scale(s.LCK)
		s.INT, s.STA, s.CRT, s.DGE, s.MNA = scale(s.INT), scale(s.STA), scale(s.CRT), scale(s.DGE), scale(s.MNA)
	}
	return g
}

func personalizedShopStockPage(seed int64, uid string, buffs shopBuffState, equipped map[string]content.Gear, page int64) []shopItemView {
	info := shopStockPagination(seed, uid, buffs, page)
	selected := shopRarityRolls(seed, uid)
	out := make([]shopItemView, 0, shopStockPageSize+1)
	if info.Page == 0 {
		out = append(out, featuredShopView(seed, equipped))
	}
	var stock []content.Gear
	lastBatch := int64(-2)
	for position := info.Start - 1; position < info.End; position++ {
		batch, index := int64(-1), position
		if position >= shopStockSize {
			batch, index = (position-shopStockSize)/shopStockSize, (position-shopStockSize)%shopStockSize
		}
		if batch != lastBatch {
			batchSeed := seed
			if batch >= 0 {
				batchSeed = shopBuffRandom(seed, uid, "extra-"+strconv.FormatInt(batch, 10)).Int64()
			}
			stock = content.ShopStock(batchSeed, shopStockSize)
			lastBatch = batch
		}
		if index >= int64(len(stock)) {
			break
		}
		g := stock[index]
		base := g
		roll, eligible := selected[position]
		if eligible && buffs.Rarity > 0 {
			g = applyShopRarityBuff(g, buffs.Rarity, roll)
		}
		boosted := g.Rarity != base.Rarity || g.Stats != base.Stats
		item := regularShopView(g, equipped)
		item.RarityBoosted = boosted
		if boosted {
			// The offer ID is separate from the catalog ID retained in item_data.
			item.ID = "SHOP_BOOST_" + strconv.FormatInt(position, 10) + "_" + g.ID
		}
		if eligible && g.Rarity == content.RarityEternal {
			extra := max(buffs.Rarity-int64(content.RarityEternal-base.Rarity)*1000, 0)
			if extra > 0 {
				item.EternalBonusPct = fmt.Sprintf("%.4f", float64(extra)/10000)
			}
		}
		out = append(out, item)
	}
	return out
}

type shopBuffView struct {
	Kind, Name, Bonus string
	Owned, Price      int64
}

func shopBuffViews(state shopBuffState) []shopBuffView {
	return []shopBuffView{
		{Kind: "rarity", Name: "Rarity", Bonus: fmt.Sprintf("%.1f", float64(state.Rarity)/10), Owned: state.Rarity, Price: shopBuffPrice(state.Rarity)},
		{Kind: "quantity", Name: "Quantity", Bonus: fmt.Sprintf("%.1f", float64(state.Quantity)/10), Owned: state.Quantity, Price: shopBuffPrice(state.Quantity)},
	}
}
