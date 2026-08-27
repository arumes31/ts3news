ALTER TABLE abyss_runs
    ADD COLUMN IF NOT EXISTS build_key TEXT NOT NULL DEFAULT 'initiate',
    ADD COLUMN IF NOT EXISTS pact_multiplier NUMERIC(8,3) NOT NULL DEFAULT 1 CHECK (pact_multiplier >= 1),
    ADD COLUMN IF NOT EXISTS ts3_channel_id INTEGER,
    ADD COLUMN IF NOT EXISTS audit_hash TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS audit_data JSONB NOT NULL DEFAULT '{}'::jsonb;

ALTER TABLE abyss_social_profiles
    ADD COLUMN IF NOT EXISTS shame_opt_in BOOLEAN NOT NULL DEFAULT FALSE;

CREATE INDEX IF NOT EXISTS idx_abyss_runs_competition_depth
    ON abyss_runs (tier, created_at DESC, depth DESC, gold_banked DESC);
CREATE INDEX IF NOT EXISTS idx_abyss_runs_competition_speed
    ON abyss_runs (tier, duration_ms, created_at DESC) WHERE depth >= 20;
CREATE INDEX IF NOT EXISTS idx_abyss_runs_competition_channel
    ON abyss_runs (ts3_channel_id, created_at DESC, depth DESC) WHERE ts3_channel_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS abyss_competition_presence (
    client_uid TEXT PRIMARY KEY REFERENCES users(client_uid) ON DELETE CASCADE,
    channel_id INTEGER NOT NULL,
    seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_abyss_competition_presence_channel
    ON abyss_competition_presence (channel_id, seen_at DESC);

CREATE TABLE IF NOT EXISTS abyss_competition_periods (
    period_key TEXT NOT NULL,
    period_kind TEXT NOT NULL CHECK (period_kind IN ('weekly','season')),
    starts_at TIMESTAMPTZ NOT NULL,
    ends_at TIMESTAMPTZ NOT NULL,
    settled_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (period_kind, period_key)
);

CREATE TABLE IF NOT EXISTS abyss_competition_snapshots (
    period_kind TEXT NOT NULL CHECK (period_kind IN ('weekly','season')),
    period_key TEXT NOT NULL,
    tier TEXT NOT NULL,
    rank INTEGER NOT NULL CHECK (rank > 0),
    client_uid TEXT NOT NULL REFERENCES users(client_uid) ON DELETE CASCADE,
    nickname TEXT NOT NULL,
    depth INTEGER NOT NULL CHECK (depth >= 0),
    gold_banked BIGINT NOT NULL DEFAULT 0 CHECK (gold_banked >= 0),
    build_key TEXT NOT NULL DEFAULT 'initiate',
    reward_kind TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (period_kind, period_key, tier, rank),
    UNIQUE (period_kind, period_key, tier, client_uid)
);

CREATE TABLE IF NOT EXISTS abyss_competition_rewards (
    period_kind TEXT NOT NULL CHECK (period_kind IN ('weekly','season')),
    period_key TEXT NOT NULL,
    tier TEXT NOT NULL,
    client_uid TEXT NOT NULL REFERENCES users(client_uid) ON DELETE CASCADE,
    rank INTEGER NOT NULL CHECK (rank > 0),
    reward_kind TEXT NOT NULL,
    tokens INTEGER NOT NULL DEFAULT 0 CHECK (tokens >= 0),
    cosmetic_key TEXT NOT NULL DEFAULT '',
    awarded_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (period_kind, period_key, tier, client_uid)
);

CREATE TABLE IF NOT EXISTS abyss_rank_notifications (
    client_uid TEXT NOT NULL REFERENCES users(client_uid) ON DELETE CASCADE,
    board_key TEXT NOT NULL,
    period_key TEXT NOT NULL,
    rank INTEGER NOT NULL CHECK (rank > 0),
    previous_rank INTEGER NOT NULL CHECK (previous_rank > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (client_uid, board_key, period_key)
);

CREATE TABLE IF NOT EXISTS abyss_wager_entries (
    week_key TEXT NOT NULL,
    bracket INTEGER NOT NULL CHECK (bracket IN (100,1000,10000)),
    client_uid TEXT NOT NULL REFERENCES users(client_uid) ON DELETE CASCADE,
    entry_fee BIGINT NOT NULL CHECK (entry_fee > 0),
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    settled_at TIMESTAMPTZ,
    payout BIGINT NOT NULL DEFAULT 0 CHECK (payout >= 0),
    PRIMARY KEY (week_key, bracket, client_uid)
);
CREATE INDEX IF NOT EXISTS idx_abyss_wager_entries_settlement
    ON abyss_wager_entries (week_key, bracket, settled_at, joined_at);
