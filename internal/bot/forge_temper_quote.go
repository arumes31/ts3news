package bot

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"ts3news/internal/content"
)

func (s *WebServer) temperQuoteChance(ctx context.Context, uid string, gear content.Gear, request abyssForgeQuoteRequest) (float64, string, string, error) {
	var stacks int
	if err := s.bot.DB.QueryRowContext(ctx, "SELECT temper_fail_stacks FROM users WHERE client_uid=$1", uid).Scan(&stacks); err != nil {
		return 0, "", "", fmt.Errorf("temper pity: %w", err)
	}
	var guard string
	if err := s.bot.DB.QueryRowContext(ctx, "SELECT value FROM app_meta WHERE key=$1", forge4TemperGuardKey(uid)).Scan(&guard); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return 0, "", "", fmt.Errorf("temper guard: %w", err)
	}
	chance := temperChance(gear.Temper, stacks)
	pity := fmt.Sprintf("Includes %d current failed-attempt pity stacks.", stacks)
	if guard != "" && guard == forge4ItemKey(request.InvID, request.Slot) {
		chance = 1
		pity += " This item's temper guard converts a failed roll into success and is consumed only on failure."
	}
	return chance, fmt.Sprintf("%.1f%% effective success chance for this item, including current pity and protection.", chance*100), pity, nil
}
