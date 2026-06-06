package games

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"
)

func httpGetJSON(url, userAgent string, v interface{}) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	if userAgent != "" {
		req.Header.Set("User-Agent", userAgent)
	}
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 12 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s: status %d", url, resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(v)
}

// ---- GamerPower ----

func fetchGamerPower(_ Options) ([]Game, error) {
	var all []Game
	if err := httpGetJSON(
		"https://www.gamerpower.com/api/giveaways?type=game&platform=pc&sort-by=date",
		"", &all,
	); err != nil {
		return nil, err
	}
	return all, nil
}

// ---- Epic Games Store (free games promotions API) ----

type epicResponse struct {
	Data struct {
		Catalog struct {
			SearchStore struct {
				Elements []epicElement `json:"elements"`
			} `json:"searchStore"`
		} `json:"Catalog"`
	} `json:"data"`
}

type epicElement struct {
	Title         string `json:"title"`
	ProductSlug   string `json:"productSlug"`
	URLSlug       string `json:"urlSlug"`
	OfferMappings []struct {
		PageSlug string `json:"pageSlug"`
	} `json:"offerMappings"`
	CatalogNs struct {
		Mappings []struct {
			PageSlug string `json:"pageSlug"`
		} `json:"mappings"`
	} `json:"catalogNs"`
	Price struct {
		TotalPrice struct {
			OriginalPrice int    `json:"originalPrice"`
			DiscountPrice int    `json:"discountPrice"`
			CurrencyCode  string `json:"currencyCode"`
		} `json:"totalPrice"`
	} `json:"price"`
	Promotions struct {
		PromotionalOffers []struct {
			PromotionalOffers []struct {
				EndDate string `json:"endDate"`
			} `json:"promotionalOffers"`
		} `json:"promotionalOffers"`
	} `json:"promotions"`
}

func fetchEpic(_ Options) ([]Game, error) {
	var r epicResponse
	if err := httpGetJSON(
		"https://store-site-backend-static.ak.epicgames.com/freeGamesPromotions?locale=en-US&country=DE",
		"Mozilla/5.0 (ts3news bot)", &r,
	); err != nil {
		return nil, err
	}

	var out []Game
	for _, e := range r.Data.Catalog.SearchStore.Elements {
		// Currently free = has an active promotional offer and discounted to 0.
		if len(e.Promotions.PromotionalOffers) == 0 {
			continue
		}
		if e.Price.TotalPrice.DiscountPrice != 0 || e.Price.TotalPrice.OriginalPrice <= 0 {
			continue
		}
		out = append(out, Game{
			Title:     e.Title,
			URL:       epicStoreURL(e),
			Worth:     fmtCents(e.Price.TotalPrice.OriginalPrice, e.Price.TotalPrice.CurrencyCode),
			Platforms: "PC, Epic Games Store",
			EndDate:   epicEndDate(e),
		})
	}
	return out, nil
}

func epicStoreURL(e epicElement) string {
	slug := e.ProductSlug
	if slug == "" && len(e.OfferMappings) > 0 {
		slug = e.OfferMappings[0].PageSlug
	}
	if slug == "" && len(e.CatalogNs.Mappings) > 0 {
		slug = e.CatalogNs.Mappings[0].PageSlug
	}
	if slug == "" {
		slug = e.URLSlug
	}
	slug = strings.TrimSuffix(slug, "/home")
	if slug == "" {
		return "https://store.epicgames.com/en-US/free-games"
	}
	return "https://store.epicgames.com/en-US/p/" + slug
}

func epicEndDate(e epicElement) string {
	for _, po := range e.Promotions.PromotionalOffers {
		for _, o := range po.PromotionalOffers {
			if t, err := time.Parse(time.RFC3339, o.EndDate); err == nil {
				return t.Format("2006-01-02 15:04:05")
			}
		}
	}
	return "N/A"
}

func fmtCents(cents int, currency string) string {
	sym := map[string]string{"EUR": "€", "USD": "$", "GBP": "£"}[strings.ToUpper(currency)]
	if sym == "" {
		sym = currency + " "
	}
	return fmt.Sprintf("%s%.2f", sym, float64(cents)/100.0)
}

// ---- Reddit /r/FreeGameFindings ----

type redditResponse struct {
	Data struct {
		Children []struct {
			Data struct {
				Title     string `json:"title"`
				URL       string `json:"url"`
				FlairText string `json:"link_flair_text"`
			} `json:"data"`
		} `json:"children"`
	} `json:"data"`
}

var bracketTag = regexp.MustCompile(`\[[^\]]*\]|\([^)]*\)`)

func fetchReddit(_ Options) ([]Game, error) {
	var r redditResponse
	if err := httpGetJSON(
		"https://www.reddit.com/r/FreeGameFindings/new.json?limit=75",
		"ts3news:free-game-bot:v2 (by /u/ts3news)", &r,
	); err != nil {
		return nil, err
	}

	var out []Game
	for _, c := range r.Data.Children {
		title := c.Data.Title
		lower := strings.ToLower(title)
		flair := strings.ToLower(c.Data.FlairText)

		if strings.Contains(lower, "expired") || strings.Contains(flair, "expired") {
			continue
		}
		// Only full games (FreeGameFindings tags type as "(Game)").
		if !strings.Contains(lower, "(game)") {
			continue
		}
		drm := redditDRM(lower, flair)
		if drm == "" {
			continue
		}

		out = append(out, Game{
			Title:      cleanRedditTitle(title),
			URL:        c.Data.URL,
			Worth:      "", // price unknown from Reddit
			Platforms:  "PC, " + drm,
			EndDate:    "N/A",
			AssumePaid: true, // curated as a normally-paid game by the subreddit
		})
	}
	return out, nil
}

func redditDRM(lowerTitle, flair string) string {
	for _, s := range []struct{ key, name string }{
		{"steam", "Steam"},
		{"epic", "Epic Games Store"},
		{"gog", "GOG"},
	} {
		if strings.Contains(flair, s.key) || strings.Contains(lowerTitle, "["+s.key) {
			return s.name
		}
	}
	return ""
}

func cleanRedditTitle(title string) string {
	t := bracketTag.ReplaceAllString(title, " ")
	t = strings.TrimSpace(regexp.MustCompile(`\s+`).ReplaceAllString(t, " "))
	for _, suffix := range []string{" is free", " free", " is now free", " giveaway", " (100% off)"} {
		t = strings.TrimSuffix(t, suffix)
		t = strings.TrimSuffix(strings.TrimSpace(t), suffix)
	}
	return strings.TrimSpace(t)
}

// ---- IsThereAnyDeal (optional; requires an API key) ----

type itadDeal struct {
	Title string `json:"title"`
	Slug  string `json:"slug"`
	Deal  struct {
		URL   string `json:"url"`
		Price struct {
			Amount float64 `json:"amount"`
		} `json:"price"`
		Regular struct {
			Amount float64 `json:"amount"`
		} `json:"regular"`
		Cut   int `json:"cut"`
		Shop  struct {
			Name string `json:"name"`
		} `json:"shop"`
	} `json:"deal"`
}

type itadResponse struct {
	List []itadDeal `json:"list"`
}

func fetchITAD(opts Options) ([]Game, error) {
	if opts.ITADKey == "" {
		return nil, nil // disabled without a key
	}
	url := "https://api.isthereanydeal.com/deals/v2?key=" + opts.ITADKey +
		"&country=DE&limit=200&sort=-cut"
	var r itadResponse
	if err := httpGetJSON(url, "ts3news bot", &r); err != nil {
		return nil, err
	}

	var out []Game
	for _, d := range r.List {
		// "reduced to zero": fully free deal of a normally-paid game.
		if d.Deal.Cut < 100 || d.Deal.Price.Amount > 0 || d.Deal.Regular.Amount <= 0 {
			continue
		}
		shop := strings.ToLower(d.Deal.Shop.Name)
		platform := "PC, " + d.Deal.Shop.Name
		_ = shop
		out = append(out, Game{
			Title:     d.Title,
			URL:       d.Deal.URL,
			Worth:     fmt.Sprintf("€%.2f", d.Deal.Regular.Amount),
			Platforms: platform,
			EndDate:   "N/A",
		})
	}
	return out, nil
}
