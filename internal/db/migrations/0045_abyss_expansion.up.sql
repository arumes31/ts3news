-- The Abyss expansion: difficulty tiers, escrow insurance, meta-progression
-- (tokens + Deep Delver upgrades), lifetime stats, streaks, daily bonus, a
-- per-day gold guard, a banked-but-cursed debuff, achievements, and a shared
-- "deep cache" jackpot fed by forfeited caches.

-- Per-run state on the active descent. tier is constrained to the supported set
-- so a bad value can't break unlock/leaderboard logic.
ALTER TABLE abyss_active ADD COLUMN IF NOT EXISTS tier    TEXT    NOT NULL DEFAULT 'normal' CHECK (tier IN ('normal','nightmare','hell'));
ALTER TABLE abyss_active ADD COLUMN IF NOT EXISTS insured INTEGER NOT NULL DEFAULT 0 CHECK (insured BETWEEN 0 AND 100); -- % of cache protected on death
ALTER TABLE abyss_active ADD COLUMN IF NOT EXISTS revived BOOLEAN NOT NULL DEFAULT FALSE; -- double-or-nothing revival already used

-- Per-finished-run tier, so leaderboards can be split by difficulty.
ALTER TABLE abyss_runs ADD COLUMN IF NOT EXISTS tier TEXT NOT NULL DEFAULT 'normal' CHECK (tier IN ('normal','nightmare','hell'));

-- Meta-progression + lifetime stats + economy guards on the player row. These are
-- authoritative balances/counters, so they're constrained to be non-negative.
ALTER TABLE users ADD COLUMN IF NOT EXISTS abyss_tokens          BIGINT  NOT NULL DEFAULT 0 CHECK (abyss_tokens >= 0);
ALTER TABLE users ADD COLUMN IF NOT EXISTS abyss_lifetime_floors BIGINT  NOT NULL DEFAULT 0 CHECK (abyss_lifetime_floors >= 0);
ALTER TABLE users ADD COLUMN IF NOT EXISTS abyss_lifetime_banked BIGINT  NOT NULL DEFAULT 0 CHECK (abyss_lifetime_banked >= 0);
ALTER TABLE users ADD COLUMN IF NOT EXISTS abyss_deaths          INTEGER NOT NULL DEFAULT 0 CHECK (abyss_deaths >= 0);
ALTER TABLE users ADD COLUMN IF NOT EXISTS abyss_bank_streak     INTEGER NOT NULL DEFAULT 0 CHECK (abyss_bank_streak >= 0);
ALTER TABLE users ADD COLUMN IF NOT EXISTS abyss_last_descent    DATE;
ALTER TABLE users ADD COLUMN IF NOT EXISTS abyss_curse_fights    INTEGER NOT NULL DEFAULT 0 CHECK (abyss_curse_fights >= 0);
ALTER TABLE users ADD COLUMN IF NOT EXISTS abyss_day             DATE;
ALTER TABLE users ADD COLUMN IF NOT EXISTS abyss_day_gold        BIGINT  NOT NULL DEFAULT 0 CHECK (abyss_day_gold >= 0);

-- Deep Delver upgrade tree (token-bought permanent bonuses).
ALTER TABLE users ADD COLUMN IF NOT EXISTS abyss_up_vigor   INTEGER NOT NULL DEFAULT 0 CHECK (abyss_up_vigor >= 0); -- start each run with bonus max HP %
ALTER TABLE users ADD COLUMN IF NOT EXISTS abyss_up_greed   INTEGER NOT NULL DEFAULT 0 CHECK (abyss_up_greed >= 0); -- +escrow bonus %
ALTER TABLE users ADD COLUMN IF NOT EXISTS abyss_up_fortune INTEGER NOT NULL DEFAULT 0 CHECK (abyss_up_fortune >= 0); -- +loot quality
ALTER TABLE users ADD COLUMN IF NOT EXISTS abyss_up_ward    INTEGER NOT NULL DEFAULT 0 CHECK (abyss_up_ward >= 0); -- cheaper insurance / start insured

-- Earned achievements (depth milestones, boss kills, etc.).
CREATE TABLE IF NOT EXISTS abyss_achievements (
    client_uid TEXT        NOT NULL REFERENCES users(client_uid) ON DELETE CASCADE,
    code       TEXT        NOT NULL,
    earned_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (client_uid, code)
);

-- Shared "deep cache" jackpot, fed by a slice of every forfeited cache and paid
-- out on a rare deep bank. Reuses the arcade jackpot machinery.
INSERT INTO arcade_jackpots (game_key, amount) VALUES ('abyss', 25000)
ON CONFLICT DO NOTHING;
