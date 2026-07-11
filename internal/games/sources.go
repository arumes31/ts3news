package games

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"ts3news/internal/i18n"
)

func httpGet(url, userAgent string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	if userAgent != "" {
		req.Header.Set("User-Agent", userAgent)
	}
	client := &http.Client{Timeout: 12 * time.Second}
	return client.Do(req)
}

func httpGetJSON(url, userAgent string, v interface{}) error {
	resp, err := httpGet(url, userAgent)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s: status %d", url, resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(v)
}

func httpGetXML(url, userAgent string, v interface{}) error {
	resp, err := httpGet(url, userAgent)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s: status %d", url, resp.StatusCode)
	}
	return xml.NewDecoder(resp.Body).Decode(v)
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
			Platforms: i18n.T("games.platform.epic"),
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
	return i18n.T("games.platform.na")
}

func fmtCents(cents int, currency string) string {
	sym := map[string]string{"EUR": "€", "USD": "$", "GBP": "£"}[strings.ToUpper(currency)]
	if sym == "" {
		sym = currency + " "
	}
	return fmt.Sprintf("%s%.2f", sym, float64(cents)/100.0)
}

// ---- Reddit /r/FreeGameFindings (RSS) ----

type atomFeed struct {
	Entries []atomEntry `xml:"entry"`
}

type atomEntry struct {
	Title   string `xml:"title"`
	Link    link   `xml:"link"`
	Content string `xml:"content"`
	Updated string `xml:"updated"`
}

type link struct {
	Href string `xml:"href,attr"`
}

var bracketTag = regexp.MustCompile(`\[[^\]]*\]|\([^)]*\)`)
var hrefRe = regexp.MustCompile(`(?i)href="(https?://[^"]+)"`)

func fetchReddit(_ Options) ([]Game, error) {
	var feed atomFeed
	if err := httpGetXML(
		"https://www.reddit.com/r/FreeGameFindings/new.rss?limit=75",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36", &feed,
	); err != nil {
		return nil, err
	}

	var out []Game
	for _, e := range feed.Entries {
		// Skip posts older than 48 hours to avoid expired giveaways
		if e.Updated != "" {
			if t, err := time.Parse(time.RFC3339, e.Updated); err == nil {
				if time.Since(t) > 48*time.Hour {
					continue
				}
			}
		}

		title := e.Title
		lower := strings.ToLower(title)

		if strings.Contains(lower, "expired") {
			continue
		}
		// Only full games (FreeGameFindings tags type as "(Game)").
		if !strings.Contains(lower, "(game)") {
			continue
		}
		drm := redditDRM(lower, "") // Flair is not easily available in RSS, rely on title
		if drm == "" {
			continue
		}

		// Try to extract the external link from the HTML content (often marked with >[link]<)
		// If not found, fallback to the Reddit post URL.
		gameURL := e.Link.Href
		if m := hrefRe.FindStringSubmatch(e.Content); len(m) > 1 {
			// Often the first link in content is the actual external link,
			// or we can look specifically for steam/epic links.
			if !strings.Contains(m[1], "reddit.com") {
				gameURL = m[1]
			}
		}

		out = append(out, Game{
			Title:      cleanRedditTitle(title),
			URL:        strings.ReplaceAll(gameURL, "&amp;", "&"),
			Worth:      "", // price unknown from Reddit
			Platforms:  redditPlatform(drm),
			EndDate:    i18n.T("games.platform.na"),
			AssumePaid: true, // curated as a normally-paid game by the subreddit
		})
	}
	return out, nil
}

func redditDRM(lowerTitle, flair string) string {
	for _, s := range []struct{ key, i18nKey string }{
		{"steam", "games.platform.steam"},
		{"epic", "games.platform.epic"},
		{"gog", "games.platform.gog"},
	} {
		if strings.Contains(flair, s.key) || strings.Contains(lowerTitle, "["+s.key) {
			return s.i18nKey
		}
	}
	return ""
}

func redditPlatform(drmI18nKey string) string {
	if drmI18nKey == "" {
		return ""
	}
	return i18n.T(drmI18nKey)
}

func itadPlatform(lowerShop, displayName string) string {
	switch {
	case strings.Contains(lowerShop, "steam"):
		return i18n.T("games.platform.steam")
	case strings.Contains(lowerShop, "epic"):
		return i18n.T("games.platform.epic")
	case strings.Contains(lowerShop, "gog"):
		return i18n.T("games.platform.gog")
	default:
		return "PC, " + displayName
	}
}

func cleanRedditTitle(title string) string {
	t := bracketTag.ReplaceAllString(title, " ")
	// ⚡ Bolt: Use strings.Fields+Join instead of regexp.MustCompile(\`\s+\`) inside a function for a ~10x speedup
	t = strings.Join(strings.Fields(t), " ")

	suffixes := []string{" is free", " free", " is now free", " giveaway", " giveaways", " (100% off)"}

	changed := true
	for changed {
		changed = false
		lower := strings.ToLower(t)
		for _, s := range suffixes {
			if strings.HasSuffix(lower, s) {
				t = t[:len(t)-len(s)]
				t = strings.TrimSpace(t)
				lower = strings.ToLower(t)
				changed = true
			}
		}
	}
	return t
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
		Cut  int `json:"cut"`
		Shop struct {
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
		platform := itadPlatform(shop, d.Deal.Shop.Name)
		out = append(out, Game{
			Title:     d.Title,
			URL:       d.Deal.URL,
			Worth:     fmt.Sprintf("€%.2f", d.Deal.Regular.Amount),
			Platforms: platform,
			EndDate:   i18n.T("games.platform.na"),
		})
	}
	return out, nil
}
