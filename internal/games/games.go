package games

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// Game describes one free game offer from any source.
type Game struct {
	ID         int    `json:"id"`                // source-local id (GamerPower); 0 for others
	Title      string `json:"title"`             // display title
	URL        string `json:"open_giveaway_url"` // claim/store URL
	Worth      string `json:"worth"`             // e.g. "$29.99"; "N/A" for always-free
	Platforms  string `json:"platforms"`         // e.g. "PC, Steam"
	EndDate    string `json:"end_date"`          // "2006-01-02 15:04:05" or "N/A"
	Source     string `json:"-"`                 // which source produced this entry
	AssumePaid bool   `json:"-"`                  // source vouches it is a normally-paid game even without a price
}

// keyTagRe strips parenthetical/bracket tags such as "(Epic Games)" or "[Steam]".
var keyTagRe = regexp.MustCompile(`\([^)]*\)|\[[^\]]*\]`)

// displayGiveawayRe strips the trailing "Giveaway"/"Giveaways" boilerplate word.
var displayGiveawayRe = regexp.MustCompile(`(?i)\bgiveaways?\b`)

// DisplayTitle returns a clean, human-facing title: platform tags like
// "(Epic Games)" / "[Steam]" and the word "Giveaway" are removed. Used in pokes,
// PMs and the trailer link so users see just the game name.
func (g Game) DisplayTitle() string {
	t := keyTagRe.ReplaceAllString(g.Title, " ")
	t = displayGiveawayRe.ReplaceAllString(t, " ")
	t = strings.Join(strings.Fields(t), " ")
	if t == "" {
		t = strings.TrimSpace(g.Title)
	}
	return t
}

// keyBoilerplateRe strips source boilerplate words so the same game from
// different sources normalises identically (GamerPower appends "Giveaway", etc.).
var keyBoilerplateRe = regexp.MustCompile(`(?i)\b(giveaway|free|key|drm-?free)\b`)

// Key returns a stable, cross-source identifier for a game derived from its
// normalised title. The same game from different sources yields the same key, so
// per-user dedup and cross-source dedup work even though source ids differ.
func (g Game) Key() string {
	t := strings.ToLower(g.Title)
	t = keyTagRe.ReplaceAllString(t, " ")
	t = keyBoilerplateRe.ReplaceAllString(t, " ")

	var b strings.Builder
	for _, r := range t {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	k := b.String()
	if k == "" {
		// Fallback: alnum of the raw title (title was only boilerplate/punctuation).
		for _, r := range strings.ToLower(g.Title) {
			if unicode.IsLetter(r) || unicode.IsDigit(r) {
				b.WriteRune(r)
			}
		}
		k = b.String()
	}
	return k
}

// Options controls which sources are queried and how results are filtered.
type Options struct {
	DRMFilter        []string // lowercased platforms to keep: "steam","epic","gog". Empty => steam+epic.
	EnableGamerPower bool
	EnableEpic       bool
	EnableReddit     bool
	EnableITAD       bool
	ITADKey          string
}

// DefaultOptions returns the production defaults: GamerPower + Epic + Reddit on,
// ITAD on only when a key is supplied, filtered to Steam & Epic.
func DefaultOptions() Options {
	return Options{
		DRMFilter:        []string{"steam", "epic"},
		EnableGamerPower: true,
		EnableEpic:       true,
		EnableReddit:     true,
		EnableITAD:       false,
	}
}

// source is one fetcher; best-effort — a failing source logs and returns nil.
type source struct {
	name    string
	enabled bool
	fetch   func(Options) ([]Game, error)
}

// FetchFreeGames queries all enabled sources, merges and de-duplicates the
// results by game key (first/highest-quality source wins), and keeps only
// currently-active, normally-paid games on the configured DRM platforms.
func FetchFreeGames(opts Options) ([]Game, error) {
	if len(opts.DRMFilter) == 0 {
		opts.DRMFilter = []string{"steam", "epic"}
	}

	sources := []source{
		// Order matters: priced sources first so they win dedup over price-less ones.
		{"GamerPower", opts.EnableGamerPower, fetchGamerPower},
		{"Epic", opts.EnableEpic, fetchEpic},
		{"ITAD", opts.EnableITAD, fetchITAD},
		{"Reddit", opts.EnableReddit, fetchReddit},
	}

	seen := map[string]bool{}
	var out []Game
	for _, s := range sources {
		if !s.enabled {
			continue
		}
		got, err := s.fetch(opts)
		if err != nil {
			log.Printf("games: source %s failed (continuing): %v", s.name, err)
			continue
		}
		for _, g := range got {
			g.Source = s.name
			if !matchesDRM(g, opts.DRMFilter) {
				continue
			}
			if !g.isNormallyPaid() && !g.AssumePaid {
				continue
			}
			if !g.isActive() {
				continue
			}
			key := g.Key()
			if key == "" || seen[key] {
				continue
			}
			seen[key] = true
			out = append(out, g)
		}
	}
	log.Printf("games: %d unique free game(s) after merge/filter across sources", len(out))
	return out, nil
}

// matchesDRM reports whether the game's platforms include any of the wanted DRMs.
func matchesDRM(g Game, want []string) bool {
	p := strings.ToLower(g.Platforms)
	for _, w := range want {
		switch w {
		case "epic":
			if strings.Contains(p, "epic") {
				return true
			}
		case "steam":
			if strings.Contains(p, "steam") {
				return true
			}
		case "gog":
			if strings.Contains(p, "gog") {
				return true
			}
		default:
			if strings.Contains(p, w) {
				return true
			}
		}
	}
	return false
}

// isNormallyPaid reports whether the game normally costs money (worth is a real
// non-zero price) — excludes free-to-play / always-free titles.
func (g Game) isNormallyPaid() bool {
	w := strings.TrimSpace(g.Worth)
	if w == "" || strings.EqualFold(w, "N/A") {
		return false
	}
	digits := strings.NewReplacer("$", "", "€", "", "£", "", ".", "", ",", "", " ", "").Replace(w)
	return strings.Trim(digits, "0") != ""
}

// WorthShown reports whether a concrete original price is available to display.
func (g Game) WorthShown() bool { return g.isNormallyPaid() }

// priceRe extracts the first number (with optional decimals) from a worth string.
var priceRe = regexp.MustCompile(`[0-9]+(?:[.,][0-9]+)?`)

// PriceEUR parses the game's original price (e.g. "€19.99") into a float. The bool
// is false when no concrete price is available (free-to-play / unknown).
func (g Game) PriceEUR() (float64, bool) {
	if !g.isNormallyPaid() {
		return 0, false
	}
	m := priceRe.FindString(g.Worth)
	if m == "" {
		return 0, false
	}
	m = strings.Replace(m, ",", ".", 1)
	f, err := strconv.ParseFloat(m, 64)
	if err != nil {
		return 0, false
	}
	return f, true
}

// isActive reports whether the giveaway is still running (end date in the future
// or open-ended).
func (g Game) isActive() bool {
	e := strings.TrimSpace(g.EndDate)
	if e == "" || strings.EqualFold(e, "N/A") {
		return true
	}
	if t, err := time.Parse("2006-01-02 15:04:05", e); err == nil {
		return t.After(time.Now())
	}
	return true
}

// TrailerSearchURL returns a YouTube search URL pointing at just the game name.
func TrailerSearchURL(name string) string {
	return "https://www.youtube.com/results?search_query=" + url.QueryEscape(strings.TrimSpace(name))
}

type shortenRequest struct {
	LongURL     string `json:"long_url"`
	PreviewMode bool   `json:"preview_mode"`
}

type shortenResponse struct {
	ShortURL string `json:"short_url"`
}

// ShortenURL shortens a URL via RedRx if REDRX_API_KEY is set; otherwise returns
// the original URL.
func ShortenURL(longURL string) (string, error) {
	apiKey := os.Getenv("REDRX_API_KEY")
	if apiKey == "" {
		return longURL, nil
	}

	reqBody, _ := json.Marshal(shortenRequest{LongURL: longURL, PreviewMode: false})
	req, err := http.NewRequest("POST", "https://redrx.eu/api/v1/shorten", bytes.NewBuffer(reqBody))
	if err != nil {
		return longURL, err
	}
	req.Header.Set("X-API-KEY", apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return longURL, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return longURL, fmt.Errorf("redrx returned status %d", resp.StatusCode)
	}

	var res shortenResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return longURL, err
	}
	return res.ShortURL, nil
}
