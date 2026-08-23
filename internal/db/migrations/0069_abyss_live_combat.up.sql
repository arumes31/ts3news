CREATE TABLE abyss_combat_sessions (
    session_id TEXT PRIMARY KEY,
    owner_uid TEXT NOT NULL REFERENCES users(client_uid) ON DELETE CASCADE,
    phase TEXT NOT NULL,
    round INTEGER NOT NULL DEFAULT 0,
    version BIGINT NOT NULL DEFAULT 0,
    deadline TIMESTAMPTZ,
    pause_reason TEXT NOT NULL DEFAULT '',
    state JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE abyss_combat_members (
    session_id TEXT NOT NULL REFERENCES abyss_combat_sessions(session_id) ON DELETE CASCADE,
    client_uid TEXT NOT NULL REFERENCES users(client_uid) ON DELETE CASCADE,
    tactic TEXT NOT NULL DEFAULT 'balanced',
    selected_target TEXT NOT NULL DEFAULT '',
    queued_action JSONB,
    state JSONB NOT NULL DEFAULT '{}'::jsonb,
    submitted_round INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (session_id, client_uid),
    UNIQUE (client_uid)
);

CREATE INDEX abyss_combat_sessions_owner_idx ON abyss_combat_sessions(owner_uid);
