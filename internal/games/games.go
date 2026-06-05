package games

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

type Game struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
	URL   string `json:"open_giveaway_url"`
}

func FetchFreeGames() ([]Game, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get("https://www.gamerpower.com/api/giveaways?type=game&platform=pc")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var games []Game
	if err := json.NewDecoder(resp.Body).Decode(&games); err != nil {
		return nil, err
	}
	return games, nil
}

type ShortenRequest struct {
	LongURL     string `json:"long_url"`
	PreviewMode bool   `json:"preview_mode"`
}

type ShortenResponse struct {
	ShortURL string `json:"short_url"`
}

func ShortenURL(longURL string) (string, error) {
	apiKey := os.Getenv("REDRX_API_KEY")
	if apiKey == "" {
		return longURL, nil
	}

	reqBody, _ := json.Marshal(ShortenRequest{
		LongURL:     longURL,
		PreviewMode: false,
	})
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
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return longURL, fmt.Errorf("redrx returned status %d", resp.StatusCode)
	}

	var res ShortenResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return longURL, err
	}

	return res.ShortURL, nil
}
