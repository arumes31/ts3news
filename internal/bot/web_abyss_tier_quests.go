package bot

import (
	"context"
	"fmt"
)

type abyssTierQuest struct {
	Tier        string
	SourceTier  string
	TargetDepth int
	BestDepth   int
	Complete    bool
}

func abyssTierQuestCatalog() []abyssTierQuest {
	return []abyssTierQuest{
		{Tier: "normal", Complete: true},
		{Tier: "nightmare", SourceTier: "normal", TargetDepth: 10},
		{Tier: "hell", SourceTier: "nightmare", TargetDepth: 20},
		{Tier: "insanity", SourceTier: "hell", TargetDepth: 30},
	}
}

func (quest abyssTierQuest) requirement() string {
	if quest.Tier == "normal" {
		return "Open expedition"
	}
	name := abyssTiers[quest.SourceTier].Name
	return fmt.Sprintf("Bank a %s run at depth %d+", name, quest.TargetDepth)
}

func (quest abyssTierQuest) progress() string {
	if quest.Complete {
		return "Quest complete"
	}
	return fmt.Sprintf("Best banked %d / %d", quest.BestDepth, quest.TargetDepth)
}

func (b *Bot) abyssTierQuests(ctx context.Context, uid string) ([]abyssTierQuest, error) {
	quests := abyssTierQuestCatalog()
	rows, err := b.DB.QueryContext(ctx, `SELECT COALESCE(tier,'normal'),COALESCE(MAX(depth),0)
		FROM abyss_runs WHERE client_uid=$1 AND victory=TRUE
		GROUP BY COALESCE(tier,'normal')`, uid)
	if err != nil {
		return quests, fmt.Errorf("querying Abyss tier quests: %w", err)
	}
	defer func() { _ = rows.Close() }()

	bestByTier := make(map[string]int, len(abyssTierOrder))
	for rows.Next() {
		var tier string
		var depth int
		if err := rows.Scan(&tier, &depth); err != nil {
			return quests, fmt.Errorf("scanning Abyss tier quest: %w", err)
		}
		bestByTier[tier] = max(depth, 0)
	}
	if err := rows.Err(); err != nil {
		return quests, fmt.Errorf("iterating Abyss tier quests: %w", err)
	}
	for index := range quests {
		quest := &quests[index]
		if quest.Tier == "normal" {
			continue
		}
		quest.BestDepth = bestByTier[quest.SourceTier]
		quest.Complete = quest.BestDepth >= quest.TargetDepth
	}
	return quests, nil
}

func abyssTierQuestByKey(quests []abyssTierQuest, key string) (abyssTierQuest, bool) {
	for _, quest := range quests {
		if quest.Tier == key {
			return quest, true
		}
	}
	return abyssTierQuest{}, false
}

func abyssTierViewsWithQuests(
	quests []abyssTierQuest,
	rates []abyssTierRateView,
) []abyssTierView {
	views := abyssTierListWithRates(0, rates)
	for index := range views {
		quest, ok := abyssTierQuestByKey(quests, views[index].Key)
		if !ok {
			views[index].Unlocked = views[index].Key == "normal"
			views[index].UnlockQuest = "Quest progress unavailable"
			continue
		}
		views[index].Unlocked = quest.Complete
		views[index].UnlockQuest = quest.requirement()
		views[index].QuestProgress = quest.progress()
	}
	return views
}
