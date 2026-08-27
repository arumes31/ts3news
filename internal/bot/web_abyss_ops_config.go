package bot

import (
	"errors"
	"hash/fnv"
	"math"
	"sync"
)

const (
	abyssRewardBaseBPS     = 10_000
	abyssRewardMaxBonusBPS = 2_500
)

// abyssFeatureConfig keeps independently reversible Abyss subsystems and the
// reward experiment under one lock. Runtime changes are process-local; the
// configured environment defaults are restored on restart.
type abyssFeatureConfig struct {
	mu sync.RWMutex

	liveActions   bool
	social        bool
	tree          bool
	forge         bool
	liveRollout   int
	socialRollout int
	treeRollout   int
	forgeRollout  int
	opsToken      string

	rewardExperiment bool
	rewardRollout    int
	rewardBonusBPS   int
	revision         uint64
	experimentRev    uint64
}

type abyssFeatureSnapshot struct {
	LiveActions              bool   `json:"live_actions"`
	Social                   bool   `json:"social"`
	TreeEnhancements         bool   `json:"tree_enhancements"`
	ForgeWorkbench           bool   `json:"forge_workbench"`
	RolloutPercent           int    `json:"rollout_percent"`
	SocialRolloutPercent     int    `json:"social_rollout_percent"`
	TreeRolloutPercent       int    `json:"tree_rollout_percent"`
	ForgeRolloutPercent      int    `json:"forge_rollout_percent"`
	RewardExperimentEnabled  bool   `json:"reward_experiment_enabled"`
	RewardExperimentRollout  int    `json:"reward_experiment_rollout_percent"`
	RewardTreatmentBonusBPS  int    `json:"reward_treatment_bonus_bps"`
	Revision                 uint64 `json:"revision"`
	RewardExperimentRevision uint64 `json:"reward_experiment_revision"`
}

type abyssRewardAssignment struct {
	Cohort        string `json:"cohort"`
	MultiplierBPS int    `json:"multiplier_bps"`
	Revision      uint64 `json:"revision"`
}

type abyssFeatureUpdate struct {
	Feature  string `json:"feature"`
	Enabled  *bool  `json:"enabled,omitempty"`
	Percent  *int   `json:"percent,omitempty"`
	BonusBPS *int   `json:"bonus_bps,omitempty"`
}

func newAbyssFeatureConfig(b *Bot) *abyssFeatureConfig {
	cfg := &abyssFeatureConfig{
		liveActions:    true,
		social:         true,
		tree:           true,
		forge:          true,
		liveRollout:    100,
		socialRollout:  100,
		treeRollout:    100,
		forgeRollout:   100,
		rewardRollout:  100,
		rewardBonusBPS: 500,
		revision:       1,
		experimentRev:  1,
	}
	if b != nil && b.Cfg != nil {
		cfg.liveActions = b.Cfg.AbyssLiveActions
		cfg.social = b.Cfg.AbyssSocial
		cfg.tree = b.Cfg.AbyssTreeEnhancements
		cfg.forge = b.Cfg.AbyssForgeWorkbench
		rollout := min(100, max(0, b.Cfg.AbyssLiveRolloutPercent))
		cfg.liveRollout = rollout
		cfg.socialRollout = rollout
		cfg.treeRollout = rollout
		cfg.forgeRollout = rollout
		cfg.opsToken = b.Cfg.AbyssOpsToken
	}
	return cfg
}

func stableAbyssBucket(uid, salt string) int {
	h := fnv.New32a()
	if salt != "" {
		_, _ = h.Write([]byte(salt))
		_, _ = h.Write([]byte{0})
	}
	_, _ = h.Write([]byte(uid))
	return int(h.Sum32() % 100)
}

func (c *abyssFeatureConfig) enabled(feature, uid string) bool {
	if c == nil {
		return false
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	switch feature {
	case "social":
		return c.social && stableAbyssBucket(uid, "") < c.socialRollout
	case "live_actions":
		return c.liveActions && stableAbyssBucket(uid, "") < c.liveRollout
	case "tree":
		return c.tree && stableAbyssBucket(uid, "tree") < c.treeRollout
	case "forge":
		return c.forge && stableAbyssBucket(uid, "forge") < c.forgeRollout
	default:
		return false
	}
}

func (c *abyssFeatureConfig) token() string {
	if c == nil {
		return ""
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.opsToken
}

func (c *abyssFeatureConfig) snapshot() abyssFeatureSnapshot {
	if c == nil {
		return abyssFeatureSnapshot{}
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return abyssFeatureSnapshot{
		LiveActions:              c.liveActions,
		Social:                   c.social,
		TreeEnhancements:         c.tree,
		ForgeWorkbench:           c.forge,
		RolloutPercent:           c.liveRollout,
		SocialRolloutPercent:     c.socialRollout,
		TreeRolloutPercent:       c.treeRollout,
		ForgeRolloutPercent:      c.forgeRollout,
		RewardExperimentEnabled:  c.rewardExperiment,
		RewardExperimentRollout:  c.rewardRollout,
		RewardTreatmentBonusBPS:  c.rewardBonusBPS,
		Revision:                 c.revision,
		RewardExperimentRevision: c.experimentRev,
	}
}

func (c *abyssFeatureConfig) update(update abyssFeatureUpdate) (abyssFeatureSnapshot, bool, error) {
	if c == nil {
		return abyssFeatureSnapshot{}, false, errors.New("feature configuration unavailable")
	}
	c.mu.Lock()
	experimentChanged := false
	switch update.Feature {
	case "live_actions", "social", "tree_enhancements", "forge_workbench", "reward_experiment":
		if update.Enabled == nil || update.Percent != nil || update.BonusBPS != nil {
			c.mu.Unlock()
			return abyssFeatureSnapshot{}, false, errors.New("enabled is required for this feature")
		}
		switch update.Feature {
		case "live_actions":
			c.liveActions = *update.Enabled
		case "social":
			c.social = *update.Enabled
		case "tree_enhancements":
			c.tree = *update.Enabled
		case "forge_workbench":
			c.forge = *update.Enabled
		case "reward_experiment":
			c.rewardExperiment = *update.Enabled
			experimentChanged = true
		}
	case "live_rollout", "social_rollout", "tree_rollout", "forge_rollout", "reward_experiment_rollout":
		if update.Percent == nil || update.Enabled != nil || update.BonusBPS != nil || *update.Percent < 0 || *update.Percent > 100 {
			c.mu.Unlock()
			return abyssFeatureSnapshot{}, false, errors.New("percent must be between 0 and 100")
		}
		switch update.Feature {
		case "live_rollout":
			c.liveRollout = *update.Percent
		case "social_rollout":
			c.socialRollout = *update.Percent
		case "tree_rollout":
			c.treeRollout = *update.Percent
		case "forge_rollout":
			c.forgeRollout = *update.Percent
		default:
			c.rewardRollout = *update.Percent
			experimentChanged = true
		}
	case "reward_treatment_bonus":
		if update.BonusBPS == nil || update.Enabled != nil || update.Percent != nil || *update.BonusBPS < 0 || *update.BonusBPS > abyssRewardMaxBonusBPS {
			c.mu.Unlock()
			return abyssFeatureSnapshot{}, false, errors.New("bonus_bps must be between 0 and 2500")
		}
		c.rewardBonusBPS = *update.BonusBPS
		experimentChanged = true
	default:
		c.mu.Unlock()
		return abyssFeatureSnapshot{}, false, errors.New("unknown feature")
	}
	c.revision++
	if experimentChanged {
		c.experimentRev++
	}
	snapshot := abyssFeatureSnapshot{
		LiveActions:              c.liveActions,
		Social:                   c.social,
		TreeEnhancements:         c.tree,
		ForgeWorkbench:           c.forge,
		RolloutPercent:           c.liveRollout,
		SocialRolloutPercent:     c.socialRollout,
		TreeRolloutPercent:       c.treeRollout,
		ForgeRolloutPercent:      c.forgeRollout,
		RewardExperimentEnabled:  c.rewardExperiment,
		RewardExperimentRollout:  c.rewardRollout,
		RewardTreatmentBonusBPS:  c.rewardBonusBPS,
		Revision:                 c.revision,
		RewardExperimentRevision: c.experimentRev,
	}
	c.mu.Unlock()
	return snapshot, experimentChanged, nil
}

func (c *abyssFeatureConfig) rewardAssignment(uid string) abyssRewardAssignment {
	if c == nil {
		return abyssRewardAssignment{Cohort: "off", MultiplierBPS: abyssRewardBaseBPS}
	}
	c.mu.RLock()
	enabled, rollout, bonusBPS, revision := c.rewardExperiment, c.rewardRollout, c.rewardBonusBPS, c.experimentRev
	c.mu.RUnlock()
	assignment := abyssRewardAssignment{Cohort: "off", MultiplierBPS: abyssRewardBaseBPS, Revision: revision}
	if !enabled {
		return assignment
	}
	if stableAbyssBucket(uid, "reward-enrollment") >= rollout {
		assignment.Cohort = "holdout"
		return assignment
	}
	assignment.Cohort = "control"
	if stableAbyssBucket(uid, "reward-variant") < 50 {
		assignment.Cohort = "treatment"
		assignment.MultiplierBPS += bonusBPS
	}
	return assignment
}

func applyAbyssRewardAssignment(bonus int64, assignment abyssRewardAssignment) int64 {
	if bonus <= 0 || assignment.MultiplierBPS <= abyssRewardBaseBPS {
		return bonus
	}
	multiplier := int64(assignment.MultiplierBPS)
	if bonus > math.MaxInt64/multiplier {
		return bonus
	}
	return bonus * multiplier / abyssRewardBaseBPS
}
