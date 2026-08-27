package bot

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode"

	"ts3news/internal/content"
)

const (
	abyssDiscordAlertConcurrency = 4
	abyssDiscordAlertTimeout     = 4 * time.Second
)

var abyssDiscordAlertSlots = make(chan struct{}, abyssDiscordAlertConcurrency)

type abyssDiscordAlert struct {
	Kind       string
	Title      string
	ItemName   string
	Rarity     content.Rarity
	Depth      int
	OccurredAt time.Time
}

type abyssDiscordPayload struct {
	Content         string                      `json:"content"`
	AllowedMentions abyssDiscordAllowedMentions `json:"allowed_mentions"`
}

type abyssDiscordAllowedMentions struct {
	Parse []string `json:"parse"`
}

func validAbyssDiscordTokenPart(value string, digitsOnly bool) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if digitsOnly {
			if r < '0' || r > '9' {
				return false
			}
			continue
		}
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '-' && r != '_' && r != '.' {
			return false
		}
	}
	return true
}

func validateAbyssDiscordWebhook(raw string) (*url.URL, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return nil, errors.New("invalid Discord webhook URL")
	}
	if parsed.Scheme != "https" || parsed.Host != "discord.com" || parsed.User != nil ||
		parsed.RawQuery != "" || parsed.Fragment != "" || parsed.RawPath != "" {
		return nil, errors.New("discord webhook URL must use the authorized Discord endpoint")
	}
	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(parts) != 4 || parts[0] != "api" || parts[1] != "webhooks" ||
		!validAbyssDiscordTokenPart(parts[2], true) || !validAbyssDiscordTokenPart(parts[3], false) {
		return nil, errors.New("discord webhook URL has an invalid path")
	}
	return parsed, nil
}

func sanitizeAbyssDiscordField(value string) string {
	value = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, strings.TrimSpace(value))
	value = strings.Join(strings.Fields(value), " ")
	runes := []rune(value)
	if len(runes) > 240 {
		value = string(runes[:239]) + "…"
	}
	return value
}

func abyssDiscordAlertContent(alert abyssDiscordAlert) (string, error) {
	kind := sanitizeAbyssDiscordField(alert.Kind)
	switch kind {
	case "legendary_drop", "eternal_drop", "milestone", "world_first_depth":
	default:
		return "", errors.New("unsupported abyss Discord alert kind")
	}
	parts := []string{"Abyss alert", "event=" + kind}
	if title := sanitizeAbyssDiscordField(alert.Title); title != "" {
		parts = append(parts, "title="+title)
	}
	if item := sanitizeAbyssDiscordField(alert.ItemName); item != "" {
		parts = append(parts, "item="+item)
	}
	if kind == "legendary_drop" || kind == "eternal_drop" {
		parts = append(parts, "rarity="+alert.Rarity.String())
	}
	if alert.Depth > 0 {
		parts = append(parts, fmt.Sprintf("depth=%d", alert.Depth))
	}
	at := alert.OccurredAt
	if at.IsZero() {
		at = time.Now()
	}
	parts = append(parts, "at="+at.UTC().Format(time.RFC3339))
	return strings.Join(parts, " | "), nil
}

func (b *Bot) sendAbyssDiscordAlert(ctx context.Context, alert abyssDiscordAlert) error {
	if b == nil || b.Cfg == nil || !b.Cfg.AbyssDiscordAlerts || b.Cfg.AbyssWebhookURL == "" {
		return nil
	}
	endpoint, err := validateAbyssDiscordWebhook(b.Cfg.AbyssWebhookURL)
	if err != nil {
		return err
	}
	contentText, err := abyssDiscordAlertContent(alert)
	if err != nil {
		return err
	}
	payload, err := json.Marshal(abyssDiscordPayload{
		Content:         contentText,
		AllowedMentions: abyssDiscordAllowedMentions{Parse: []string{}},
	})
	if err != nil {
		return fmt.Errorf("encode Discord alert: %w", err)
	}
	deliveryCtx, cancel := context.WithTimeout(ctx, abyssDiscordAlertTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(deliveryCtx, http.MethodPost, endpoint.String(), bytes.NewReader(payload))
	if err != nil {
		return errors.New("create Discord webhook request")
	}
	req.Header.Set("Content-Type", "application/json")

	baseClient := http.DefaultClient
	if b.abyssDiscordHTTPClient != nil {
		baseClient = b.abyssDiscordHTTPClient
	}
	client := *baseClient
	client.CheckRedirect = func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}
	response, err := client.Do(req)
	if err != nil {
		return errors.New("discord webhook request failed")
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("discord webhook returned status %d", response.StatusCode)
	}
	return nil
}

func (b *Bot) queueAbyssDiscordAlert(alert abyssDiscordAlert) {
	if b == nil || b.Cfg == nil || !b.Cfg.AbyssDiscordAlerts || b.Cfg.AbyssWebhookURL == "" {
		return
	}
	select {
	case abyssDiscordAlertSlots <- struct{}{}:
		go func() {
			defer func() { <-abyssDiscordAlertSlots }()
			if err := b.sendAbyssDiscordAlert(context.Background(), alert); err != nil {
				log.Printf("abyss Discord alert delivery failed for %s", alert.Kind)
			}
		}()
	default:
		log.Printf("abyss Discord alert capacity reached for %s", alert.Kind)
	}
}

func (b *Bot) queueAbyssDiscordDrop(itemName string, rarity content.Rarity, depth int) {
	if rarity < content.RarityLegendary {
		return
	}
	kind := "legendary_drop"
	if rarity >= content.RarityEternal {
		kind = "eternal_drop"
	}
	b.queueAbyssDiscordAlert(abyssDiscordAlert{
		Kind: kind, ItemName: itemName, Rarity: rarity, Depth: depth, OccurredAt: time.Now(),
	})
}

func (b *Bot) queueAbyssDiscordMilestone(title string, depth int) {
	b.queueAbyssDiscordAlert(abyssDiscordAlert{
		Kind: "milestone", Title: title, Depth: depth, OccurredAt: time.Now(),
	})
}

func (b *Bot) queueAbyssDiscordWorldFirst(depth int) {
	b.queueAbyssDiscordAlert(abyssDiscordAlert{
		Kind: "world_first_depth", Depth: depth, OccurredAt: time.Now(),
	})
}

func abyssDiscordEscrowDrop(grant abyssLootGrant, depth int) (abyssDiscordAlert, bool) {
	if grant.Type != "gear" || grant.Gear == nil || grant.Gear.Rarity < content.RarityLegendary {
		return abyssDiscordAlert{}, false
	}
	itemName := grant.Gear.Name
	if grant.Gear.Unidentified {
		itemName = "Unidentified gear"
		if grant.Gear.Slot != "" {
			itemName = "Unidentified " + string(grant.Gear.Slot)
		}
	}
	kind := "legendary_drop"
	if grant.Gear.Rarity >= content.RarityEternal {
		kind = "eternal_drop"
	}
	return abyssDiscordAlert{
		Kind:       kind,
		ItemName:   itemName,
		Rarity:     grant.Gear.Rarity,
		Depth:      max(depth, 0),
		OccurredAt: time.Now(),
	}, true
}
