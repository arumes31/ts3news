-- Daily Bounties: a single server-chosen objective per calendar day. Progress is
-- recomputed on demand from existing tables (abyss_runs / abyss_boss_kills), so the
-- only state needed is a one-row-per-claimed-day guard that prevents double rewards.
CREATE TABLE IF NOT EXISTS abyss_bounty_claims (
    client_uid TEXT        NOT NULL REFERENCES users(client_uid) ON DELETE CASCADE,
    bounty_day DATE        NOT NULL,
    claimed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (client_uid, bounty_day)
);
