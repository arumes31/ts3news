-- Migration 0039: Augment System
-- Creates tables for TFT-style augment selection and management

-- Augment definitions table
CREATE TABLE IF NOT EXISTS tft_augments (
    id VARCHAR(64) PRIMARY KEY,
    key VARCHAR(64) UNIQUE NOT NULL,
    name VARCHAR(128) NOT NULL,
    description TEXT NOT NULL,
    tier INTEGER NOT NULL CHECK (tier IN (1, 2, 3)),
    type VARCHAR(32) NOT NULL CHECK (type IN ('economy', 'combat', 'utility', 'unit')),
    effect_data JSONB NOT NULL DEFAULT '{}',
    icon VARCHAR(255) DEFAULT '',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Player augment selections table
CREATE TABLE IF NOT EXISTS tft_player_augments (
    id SERIAL PRIMARY KEY,
    user_id VARCHAR(64) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    augment_id VARCHAR(64) NOT NULL REFERENCES tft_augments(id) ON DELETE CASCADE,
    selected_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    game_id VARCHAR(64) DEFAULT '',
    UNIQUE(user_id, augment_id, game_id)
);

-- Augment offers table (current offers for selection)
CREATE TABLE IF NOT EXISTS tft_augment_offers (
    id SERIAL PRIMARY KEY,
    user_id VARCHAR(64) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    offer_index INTEGER NOT NULL CHECK (offer_index IN (0, 1, 2)),
    augment_id VARCHAR(64) NOT NULL REFERENCES tft_augments(id) ON DELETE CASCADE,
    stage INTEGER NOT NULL DEFAULT 1,
    round INTEGER NOT NULL DEFAULT 1,
    rerolled BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, offer_index, stage, round)
);

-- Player augment state (reroll count, etc.)
CREATE TABLE IF NOT EXISTS tft_augment_state (
    user_id VARCHAR(64) PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    reroll_count INTEGER DEFAULT 0,
    last_offer_stage INTEGER DEFAULT 0,
    last_offer_round INTEGER DEFAULT 0,
    augments_selected INTEGER DEFAULT 0,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_tft_player_augments_user ON tft_player_augments(user_id);
CREATE INDEX IF NOT EXISTS idx_tft_player_augments_game ON tft_player_augments(game_id);
CREATE INDEX IF NOT EXISTS idx_tft_augment_offers_user ON tft_augment_offers(user_id);
CREATE INDEX IF NOT EXISTS idx_tft_augment_offers_stage ON tft_augment_offers(stage, round);

-- Insert default augment definitions
-- Tier 1 (Silver) Augments
INSERT INTO tft_augments (id, key, name, description, tier, type, effect_data, icon) VALUES
('aug_t1_rich', 'rich_get_richer', 'Rich Get Richer', 'Gain 10 gold immediately.', 1, 'economy', '{"type": "immediate", "effect": "grant_gold", "value": 10}', '/icons/augments/gold.png'),
('aug_t1_level', 'level_up', 'Level Up!', 'Gain 4 XP immediately.', 1, 'utility', '{"type": "immediate", "effect": "grant_xp", "value": 4}', '/icons/augments/xp.png'),
('aug_t1_refresh', 'refresh_expert', 'Refresh Expert', 'Shop refreshes cost 1 gold for the rest of the game.', 1, 'economy', '{"type": "passive", "effect": "refresh_discount", "value": 1}', '/icons/augments/refresh.png'),
('aug_t1_heal', 'minor_heal', 'Minor Heal', 'Restore 2 player HP.', 1, 'utility', '{"type": "immediate", "effect": "heal_player", "value": 2}', '/icons/augments/heal.png'),
('aug_t1_attack', 'minor_attack', 'Minor Attack', 'All units gain 5% attack damage.', 1, 'combat', '{"type": "passive", "effect": "attack_damage_percent", "value": 5}', '/icons/augments/sword.png'),
('aug_t1_armor', 'minor_armor', 'Minor Armor', 'All units gain 5 armor.', 1, 'combat', '{"type": "passive", "effect": "armor_flat", "value": 5}', '/icons/augments/shield.png'),
('aug_t1_mana', 'minor_mana', 'Minor Mana', 'All units start combat with 10 mana.', 1, 'combat', '{"type": "passive", "effect": "starting_mana", "value": 10}', '/icons/augments/mana.png'),
('aug_t1_bench', 'extra_bench', 'Extra Bench', 'Gain 2 extra bench slots.', 1, 'utility', '{"type": "passive", "effect": "bench_slots", "value": 2}', '/icons/augments/bench.png')
ON CONFLICT (id) DO NOTHING;

-- Tier 2 (Gold) Augments
INSERT INTO tft_augments (id, key, name, description, tier, type, effect_data, icon) VALUES
('aug_t2_highroller', 'high_roller', 'High Roller', 'Gain 25 gold immediately.', 2, 'economy', '{"type": "immediate", "effect": "grant_gold", "value": 25}', '/icons/augments/gold_bag.png'),
('aug_t2_trade', 'trade_sector', 'Trade Sector', 'Gain a free shop refresh each round.', 2, 'economy', '{"type": "passive", "effect": "free_refresh", "value": 1}', '/icons/augments/trade.png'),
('aug_t2_academy', 'battle_academy', 'Battle Academy', 'Gain 8 XP immediately.', 2, 'utility', '{"type": "immediate", "effect": "grant_xp", "value": 8}', '/icons/augments/book.png'),
('aug_t2_iron', 'iron_will', 'Iron Will', 'All units gain 15 armor.', 2, 'combat', '{"type": "passive", "effect": "armor_flat", "value": 15}', '/icons/augments/iron_shield.png'),
('aug_t2_attack', 'power_strike', 'Power Strike', 'All units gain 15% attack damage.', 2, 'combat', '{"type": "passive", "effect": "attack_damage_percent", "value": 15}', '/icons/augments/power_sword.png'),
('aug_t2_interest', 'interest_boost', 'Interest Boost', 'Gain +1 interest per 10 gold (max +5).', 2, 'economy', '{"type": "passive", "effect": "interest_bonus", "value": 1}', '/icons/augments/interest.png'),
('aug_t2_streak', 'streak_master', 'Streak Master', 'Win/loss streaks grant +1 gold.', 2, 'economy', '{"type": "passive", "effect": "streak_bonus", "value": 1}', '/icons/augments/streak.png'),
('aug_t2_item', 'item_component', 'Item Component', 'Gain a random item component.', 2, 'utility', '{"type": "immediate", "effect": "grant_item_component", "value": 1}', '/icons/augments/component.png')
ON CONFLICT (id) DO NOTHING;

-- Tier 3 (Prismatic) Augments
INSERT INTO tft_augments (id, key, name, description, tier, type, effect_data, icon) VALUES
('aug_t3_jackpot', 'jackpot', 'Jackpot', 'Gain 50 gold immediately.', 3, 'economy', '{"type": "immediate", "effect": "grant_gold", "value": 50}', '/icons/augments/jackpot.png'),
('aug_t3_scholar', 'scholar', 'Scholar', 'Gain 15 XP immediately.', 3, 'utility', '{"type": "immediate", "effect": "grant_xp", "value": 15}', '/icons/augments/scholar.png'),
('aug_t3_crafter', 'master_crafter', 'Master Crafter', 'Gain a random completed item.', 3, 'utility', '{"type": "immediate", "effect": "grant_completed_item", "value": 1}', '/icons/augments/crafted_item.png'),
('aug_t3_unstoppable', 'unstoppable_force', 'Unstoppable Force', 'All units gain 25% attack speed.', 3, 'combat', '{"type": "passive", "effect": "attack_speed_percent", "value": 25}', '/icons/augments/boots.png'),
('aug_t3_invincible', 'invincible', 'Invincible', 'All units gain 30 armor and magic resist.', 3, 'combat', '{"type": "passive", "effect": "all_resist", "value": 30}', '/icons/augments/invincible.png'),
('aug_t3_infinite', 'infinite_interest', 'Infinite Interest', 'No cap on interest earnings.', 3, 'economy', '{"type": "passive", "effect": "uncapped_interest", "value": 1}', '/icons/augments/infinite.png'),
('aug_t3_legendary', 'legendary_unit', 'Legendary Unit', 'Gain a random 5-cost unit.', 3, 'utility', '{"type": "immediate", "effect": "grant_unit", "value": 5}', '/icons/augments/legendary.png'),
('aug_t3_heal', 'major_heal', 'Major Heal', 'Restore 5 player HP.', 3, 'utility', '{"type": "immediate", "effect": "heal_player", "value": 5}', '/icons/augments/major_heal.png')
ON CONFLICT (id) DO NOTHING;
