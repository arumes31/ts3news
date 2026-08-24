CREATE TABLE IF NOT EXISTS abyss_forge_mutation_audit (
    id           BIGSERIAL PRIMARY KEY,
    client_uid   TEXT NOT NULL REFERENCES users(client_uid) ON DELETE CASCADE,
    operation    TEXT NOT NULL,
    before_state JSONB NOT NULL DEFAULT 'null'::jsonb,
    after_state  JSONB NOT NULL DEFAULT 'null'::jsonb,
    succeeded    BOOLEAN NOT NULL,
    request_key  TEXT NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_abyss_forge_mutation_audit_user
    ON abyss_forge_mutation_audit (client_uid, id DESC);

CREATE TABLE IF NOT EXISTS abyss_forge_progression (
    client_uid       TEXT NOT NULL REFERENCES users(client_uid) ON DELETE CASCADE,
    discipline       TEXT NOT NULL,
    mastery_xp       BIGINT NOT NULL DEFAULT 0 CHECK (mastery_xp >= 0),
    first_craft_date DATE,
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (client_uid, discipline)
);

CREATE TABLE IF NOT EXISTS abyss_forge_receipts (
    id          BIGSERIAL PRIMARY KEY,
    client_uid  TEXT NOT NULL REFERENCES users(client_uid) ON DELETE CASCADE,
    operation   TEXT NOT NULL,
    item_name   TEXT NOT NULL DEFAULT '',
    result      JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_abyss_forge_receipts_user
    ON abyss_forge_receipts (client_uid, id DESC);

CREATE TABLE IF NOT EXISTS abyss_forge_material_flow (
    id         BIGSERIAL PRIMARY KEY,
    client_uid TEXT NOT NULL REFERENCES users(client_uid) ON DELETE CASCADE,
    mat_id     TEXT NOT NULL,
    direction  TEXT NOT NULL CHECK (direction IN ('source', 'sink')),
    amount     INTEGER NOT NULL CHECK (amount > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_abyss_forge_material_flow_user_time
    ON abyss_forge_material_flow (client_uid, created_at DESC);

CREATE TABLE IF NOT EXISTS abyss_forge_recipe_crafts (
    client_uid TEXT NOT NULL REFERENCES users(client_uid) ON DELETE CASCADE,
    recipe_id  TEXT NOT NULL,
    craft_count BIGINT NOT NULL DEFAULT 0 CHECK (craft_count >= 0),
    first_crafted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_crafted_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (client_uid, recipe_id)
);

CREATE TABLE IF NOT EXISTS abyss_forge_milestones (
    client_uid  TEXT NOT NULL REFERENCES users(client_uid) ON DELETE CASCADE,
    milestone_id TEXT NOT NULL,
    progress    BIGINT NOT NULL DEFAULT 0 CHECK (progress >= 0),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (client_uid, milestone_id)
);
