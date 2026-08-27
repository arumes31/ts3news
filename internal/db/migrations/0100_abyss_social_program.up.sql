ALTER TABLE abyss_deaths
    ADD COLUMN IF NOT EXISTS lost_cache BIGINT NOT NULL DEFAULT 0 CHECK (lost_cache >= 0),
    ADD COLUMN IF NOT EXISTS rescued_at TIMESTAMPTZ;

ALTER TABLE abyss_escrow_loot
    ADD COLUMN IF NOT EXISTS party_recipient_uid TEXT REFERENCES users(client_uid) ON DELETE SET NULL;

CREATE TABLE IF NOT EXISTS abyss_party_members (
    owner_uid TEXT NOT NULL REFERENCES users(client_uid) ON DELETE CASCADE,
    member_uid TEXT NOT NULL REFERENCES users(client_uid) ON DELETE CASCADE,
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (owner_uid, member_uid),
    UNIQUE (member_uid),
    CHECK (owner_uid <> member_uid)
);

CREATE TABLE IF NOT EXISTS abyss_friendships (
    uid_low TEXT NOT NULL REFERENCES users(client_uid) ON DELETE CASCADE,
    uid_high TEXT NOT NULL REFERENCES users(client_uid) ON DELETE CASCADE,
    requested_by TEXT NOT NULL REFERENCES users(client_uid) ON DELETE CASCADE,
    accepted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (uid_low, uid_high),
    CHECK (uid_low < uid_high),
    CHECK (requested_by = uid_low OR requested_by = uid_high)
);

CREATE TABLE IF NOT EXISTS abyss_friend_cheers (
    sender_uid TEXT NOT NULL REFERENCES users(client_uid) ON DELETE CASCADE,
    recipient_uid TEXT NOT NULL REFERENCES users(client_uid) ON DELETE CASCADE,
    cheer_date DATE NOT NULL DEFAULT CURRENT_DATE,
    remaining_fights SMALLINT NOT NULL DEFAULT 3 CHECK (remaining_fights BETWEEN 0 AND 3),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (sender_uid, cheer_date),
    CHECK (sender_uid <> recipient_uid)
);
CREATE INDEX IF NOT EXISTS idx_abyss_friend_cheers_recipient
    ON abyss_friend_cheers (recipient_uid, remaining_fights) WHERE remaining_fights > 0;

CREATE TABLE IF NOT EXISTS abyss_rescue_missions (
    death_id BIGINT PRIMARY KEY REFERENCES abyss_deaths(id) ON DELETE CASCADE,
    rescuer_uid TEXT NOT NULL REFERENCES users(client_uid) ON DELETE CASCADE,
    owner_uid TEXT NOT NULL REFERENCES users(client_uid) ON DELETE CASCADE,
    depth INTEGER NOT NULL CHECK (depth > 0),
    lost_cache BIGINT NOT NULL CHECK (lost_cache > 0),
    accepted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    CHECK (rescuer_uid <> owner_uid)
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_abyss_rescue_missions_active_rescuer
    ON abyss_rescue_missions (rescuer_uid) WHERE completed_at IS NULL;

CREATE TABLE IF NOT EXISTS abyss_guilds (
    guild_id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE CHECK (char_length(name) BETWEEN 3 AND 32),
    tag TEXT NOT NULL UNIQUE CHECK (char_length(tag) BETWEEN 2 AND 5),
    owner_uid TEXT NOT NULL UNIQUE REFERENCES users(client_uid) ON DELETE CASCADE,
    invite_code TEXT NOT NULL UNIQUE,
    banner TEXT NOT NULL DEFAULT 'standard' CHECK (banner IN ('standard','ember','tide','verdant','void')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS abyss_guild_members (
    guild_id BIGINT NOT NULL REFERENCES abyss_guilds(guild_id) ON DELETE CASCADE,
    client_uid TEXT NOT NULL UNIQUE REFERENCES users(client_uid) ON DELETE CASCADE,
    role TEXT NOT NULL DEFAULT 'member' CHECK (role IN ('owner','officer','member')),
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (guild_id, client_uid)
);

CREATE TABLE IF NOT EXISTS abyss_guild_weekly_progress (
    guild_id BIGINT NOT NULL REFERENCES abyss_guilds(guild_id) ON DELETE CASCADE,
    week_key TEXT NOT NULL,
    floors INTEGER NOT NULL DEFAULT 0 CHECK (floors >= 0),
    goal INTEGER NOT NULL DEFAULT 500 CHECK (goal > 0),
    reward_claimed_at TIMESTAMPTZ,
    PRIMARY KEY (guild_id, week_key)
);

CREATE TABLE IF NOT EXISTS abyss_shoutbox_messages (
    message_id BIGSERIAL PRIMARY KEY,
    sender_uid TEXT NOT NULL REFERENCES users(client_uid) ON DELETE CASCADE,
    message TEXT NOT NULL CHECK (char_length(message) BETWEEN 1 AND 240),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    relayed_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_abyss_shoutbox_recent
    ON abyss_shoutbox_messages (created_at DESC);

CREATE TABLE IF NOT EXISTS abyss_consumable_trades (
    trade_id BIGSERIAL PRIMARY KEY,
    sender_uid TEXT NOT NULL REFERENCES users(client_uid) ON DELETE CASCADE,
    recipient_uid TEXT NOT NULL REFERENCES users(client_uid) ON DELETE CASCADE,
    offer_cons_id TEXT NOT NULL,
    offer_quantity INTEGER NOT NULL CHECK (offer_quantity BETWEEN 1 AND 99),
    request_cons_id TEXT NOT NULL,
    request_quantity INTEGER NOT NULL CHECK (request_quantity BETWEEN 1 AND 99),
    status TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open','accepted','cancelled','expired')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL DEFAULT NOW() + INTERVAL '24 hours',
    CHECK (sender_uid <> recipient_uid)
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_abyss_consumable_trades_open_sender
    ON abyss_consumable_trades (sender_uid) WHERE status = 'open';

CREATE TABLE IF NOT EXISTS abyss_mentor_pairs (
    mentor_uid TEXT NOT NULL REFERENCES users(client_uid) ON DELETE CASCADE,
    mentee_uid TEXT NOT NULL UNIQUE REFERENCES users(client_uid) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    PRIMARY KEY (mentor_uid, mentee_uid),
    CHECK (mentor_uid <> mentee_uid)
);

CREATE TABLE IF NOT EXISTS abyss_duels (
    duel_id BIGSERIAL PRIMARY KEY,
    challenger_uid TEXT NOT NULL REFERENCES users(client_uid) ON DELETE CASCADE,
    opponent_uid TEXT NOT NULL REFERENCES users(client_uid) ON DELETE CASCADE,
    wager_tokens INTEGER NOT NULL CHECK (wager_tokens BETWEEN 1 AND 100),
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','accepted','declined','expired')),
    winner_uid TEXT REFERENCES users(client_uid) ON DELETE SET NULL,
    combat_log JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at TIMESTAMPTZ,
    CHECK (challenger_uid <> opponent_uid)
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_abyss_duels_pending_pair
    ON abyss_duels (LEAST(challenger_uid,opponent_uid), GREATEST(challenger_uid,opponent_uid))
    WHERE status = 'pending';

CREATE TABLE IF NOT EXISTS abyss_floor_messages (
    message_id BIGSERIAL PRIMARY KEY,
    sender_uid TEXT NOT NULL REFERENCES users(client_uid) ON DELETE CASCADE,
    run_started_at TIMESTAMPTZ NOT NULL,
    depth INTEGER NOT NULL CHECK (depth > 0),
    kind TEXT NOT NULL CHECK (kind IN ('hint','taunt')),
    message TEXT NOT NULL CHECK (char_length(message) BETWEEN 1 AND 160),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (sender_uid, run_started_at)
);
CREATE INDEX IF NOT EXISTS idx_abyss_floor_messages_depth
    ON abyss_floor_messages (depth, created_at DESC);

CREATE TABLE IF NOT EXISTS abyss_kudos (
    week_key TEXT NOT NULL,
    sender_uid TEXT NOT NULL REFERENCES users(client_uid) ON DELETE CASCADE,
    recipient_uid TEXT NOT NULL REFERENCES users(client_uid) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (week_key, sender_uid, recipient_uid),
    CHECK (sender_uid <> recipient_uid)
);

CREATE TABLE IF NOT EXISTS abyss_referral_codes (
    client_uid TEXT PRIMARY KEY REFERENCES users(client_uid) ON DELETE CASCADE,
    code TEXT NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS abyss_referrals (
    referrer_uid TEXT NOT NULL REFERENCES users(client_uid) ON DELETE CASCADE,
    referred_uid TEXT PRIMARY KEY REFERENCES users(client_uid) ON DELETE CASCADE,
    referral_code TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    rewarded_at TIMESTAMPTZ,
    CHECK (referrer_uid <> referred_uid)
);
CREATE INDEX IF NOT EXISTS idx_abyss_referrals_referrer
    ON abyss_referrals (referrer_uid, created_at DESC);

CREATE TABLE IF NOT EXISTS abyss_raid_lobbies (
    lobby_code TEXT PRIMARY KEY,
    owner_uid TEXT NOT NULL REFERENCES users(client_uid) ON DELETE CASCADE,
    week_key TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open','resolved','closed')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS abyss_raid_members (
    lobby_code TEXT NOT NULL REFERENCES abyss_raid_lobbies(lobby_code) ON DELETE CASCADE,
    week_key TEXT NOT NULL,
    client_uid TEXT NOT NULL REFERENCES users(client_uid) ON DELETE CASCADE,
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (lobby_code, client_uid),
    UNIQUE (week_key, client_uid)
);

CREATE TABLE IF NOT EXISTS abyss_tournament_teams (
    week_key TEXT NOT NULL,
    team_code TEXT NOT NULL,
    owner_uid TEXT NOT NULL REFERENCES users(client_uid) ON DELETE CASCADE,
    name TEXT NOT NULL CHECK (char_length(name) BETWEEN 3 AND 24),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (week_key, team_code),
    UNIQUE (week_key, owner_uid)
);

CREATE TABLE IF NOT EXISTS abyss_tournament_members (
    week_key TEXT NOT NULL,
    team_code TEXT NOT NULL,
    client_uid TEXT NOT NULL REFERENCES users(client_uid) ON DELETE CASCADE,
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (week_key, team_code, client_uid),
    UNIQUE (week_key, client_uid),
    FOREIGN KEY (week_key, team_code) REFERENCES abyss_tournament_teams(week_key, team_code) ON DELETE CASCADE
);
