package bot

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"ts3news/internal/config"
	"ts3news/internal/content"
)

type abyssDiscordRoundTripFunc func(*http.Request) (*http.Response, error)

func (fn abyssDiscordRoundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}

func TestValidateAbyssDiscordWebhook(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		url  string
		want bool
	}{
		{name: "authorized endpoint", url: "https://discord.com/api/webhooks/123456/abc_DEF-9.xyz", want: true},
		{name: "plaintext", url: "http://discord.com/api/webhooks/123456/token", want: false},
		{name: "lookalike host", url: "https://discord.com.evil.test/api/webhooks/123456/token", want: false},
		{name: "explicit port", url: "https://discord.com:443/api/webhooks/123456/token", want: false},
		{name: "userinfo", url: "https://user@discord.com/api/webhooks/123456/token", want: false},
		{name: "query", url: "https://discord.com/api/webhooks/123456/token?wait=true", want: false},
		{name: "encoded path", url: "https://discord.com/api/webhooks/123456/token%2Fmore", want: false},
		{name: "wrong path", url: "https://discord.com/api/v10/webhooks/123456/token", want: false},
		{name: "nonnumeric id", url: "https://discord.com/api/webhooks/not-an-id/token", want: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, err := validateAbyssDiscordWebhook(test.url)
			if got := err == nil; got != test.want {
				t.Errorf("validateAbyssDiscordWebhook() success = %v, want %v (err=%v)", got, test.want, err)
			}
		})
	}
}

func TestBotSendAbyssDiscordAlertUsesAnonymousMentionSafePayload(t *testing.T) {
	t.Parallel()
	var captured abyssDiscordPayload
	client := &http.Client{Transport: abyssDiscordRoundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", request.Method)
		}
		if request.URL.String() != "https://discord.com/api/webhooks/123456/token" {
			t.Errorf("URL = %q", request.URL.String())
		}
		if request.Header.Get("Content-Type") != "application/json" {
			t.Errorf("content type = %q", request.Header.Get("Content-Type"))
		}
		if err := json.NewDecoder(request.Body).Decode(&captured); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		return &http.Response{
			StatusCode: http.StatusNoContent,
			Body:       io.NopCloser(strings.NewReader("")),
			Header:     make(http.Header),
			Request:    request,
		}, nil
	})}
	bot := &Bot{
		Cfg: &config.Config{
			AbyssDiscordAlerts: true,
			AbyssWebhookURL:    "https://discord.com/api/webhooks/123456/token",
		},
		abyssDiscordHTTPClient: client,
	}
	err := bot.sendAbyssDiscordAlert(t.Context(), abyssDiscordAlert{
		Kind:       "eternal_drop",
		ItemName:   "Eternal Crown\n@everyone",
		Rarity:     content.RarityEternal,
		Depth:      42,
		OccurredAt: time.Date(2026, time.August, 27, 12, 30, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("sendAbyssDiscordAlert: %v", err)
	}
	for _, want := range []string{"event=eternal_drop", "item=Eternal Crown @everyone", "rarity=Eternal", "depth=42", "at=2026-08-27T12:30:00Z"} {
		if !strings.Contains(captured.Content, want) {
			t.Errorf("payload %q missing %q", captured.Content, want)
		}
	}
	if captured.AllowedMentions.Parse == nil || len(captured.AllowedMentions.Parse) != 0 {
		t.Errorf("allowed mentions = %#v, want explicit empty parse list", captured.AllowedMentions.Parse)
	}
	for _, forbidden := range []string{"client_uid", "nickname", "player"} {
		if strings.Contains(strings.ToLower(captured.Content), forbidden) {
			t.Errorf("payload contains identifier field %q: %q", forbidden, captured.Content)
		}
	}
}

func TestBotSendAbyssDiscordAlertIsOptInAndBlocksRedirects(t *testing.T) {
	t.Parallel()
	calls := 0
	client := &http.Client{Transport: abyssDiscordRoundTripFunc(func(request *http.Request) (*http.Response, error) {
		calls++
		return &http.Response{
			StatusCode: http.StatusFound,
			Body:       io.NopCloser(strings.NewReader("")),
			Header:     http.Header{"Location": []string{"https://example.test/collect"}},
			Request:    request,
		}, nil
	})}
	bot := &Bot{
		Cfg:                    &config.Config{AbyssWebhookURL: "https://discord.com/api/webhooks/123456/token"},
		abyssDiscordHTTPClient: client,
	}
	alert := abyssDiscordAlert{Kind: "milestone", Title: "Threshold Breaker"}
	if err := bot.sendAbyssDiscordAlert(t.Context(), alert); err != nil {
		t.Fatalf("disabled send returned an error: %v", err)
	}
	if calls != 0 {
		t.Fatalf("disabled alert made %d requests", calls)
	}
	bot.Cfg.AbyssDiscordAlerts = true
	if err := bot.sendAbyssDiscordAlert(t.Context(), alert); err == nil {
		t.Fatal("redirect response unexpectedly succeeded")
	}
	if calls != 1 {
		t.Fatalf("redirect followed: requests = %d, want 1", calls)
	}
}

func TestAbyssDiscordEscrowDropHidesUnidentifiedGear(t *testing.T) {
	t.Parallel()
	alert, ok := abyssDiscordEscrowDrop(abyssLootGrant{Type: "gear", Gear: &content.Gear{
		Name: "Secret Crown", Slot: content.SlotHead, Rarity: content.RarityLegendary, Unidentified: true,
	}}, 17)
	if !ok {
		t.Fatal("Legendary gear did not produce an alert")
	}
	if alert.ItemName != "Unidentified Head" || alert.Depth != 17 || alert.Kind != "legendary_drop" {
		t.Fatalf("alert = %+v", alert)
	}
}
