-- Abyss expansion 2: crafting materials, rune library, recipes, forge history,
-- new Deep-Delver talents, specializations, sanctuary upgrades, and run-state
-- for momentum / Last Stand / checkpoint & express starts.

-- #1 daily free descent
ALTER TABLE users ADD COLUMN IF NOT EXISTS abyss_free_entry_date DATE;

-- #24 comeback buff: deaths tracked per calendar day
ALTER TABLE users ADD COLUMN IF NOT EXISTS abyss_deaths_today INTEGER NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN IF NOT EXISTS abyss_deaths_date DATE;

-- #154-158 new Deep-Delver talent nodes
ALTER TABLE users ADD COLUMN IF NOT EXISTS abyss_up_swiftness     INTEGER NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN IF NOT EXISTS abyss_up_scavenger     INTEGER NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN IF NOT EXISTS abyss_up_mercy         INTEGER NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN IF NOT EXISTS abyss_up_cartographer  INTEGER NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN IF NOT EXISTS abyss_up_quartermaster INTEGER NOT NULL DEFAULT 0;

-- #161 specialization: '', 'delver', 'plunderer', 'warden'
ALTER TABLE users ADD COLUMN IF NOT EXISTS abyss_spec TEXT NOT NULL DEFAULT '';

-- #114 artisan reputation (1 point per forge action, discounts at breakpoints)
ALTER TABLE users ADD COLUMN IF NOT EXISTS forge_rep INTEGER NOT NULL DEFAULT 0;

-- #116 one undoable forge action per day (previous item snapshot)
ALTER TABLE users ADD COLUMN IF NOT EXISTS forge_undo JSONB;
ALTER TABLE users ADD COLUMN IF NOT EXISTS forge_undo_date DATE;

-- #107 temper fail-stack pity
ALTER TABLE users ADD COLUMN IF NOT EXISTS temper_fail_stacks INTEGER NOT NULL DEFAULT 0;

-- #125 auto-repair before each descent (opt-in)
ALTER TABLE users ADD COLUMN IF NOT EXISTS abyss_auto_repair BOOLEAN NOT NULL DEFAULT FALSE;

-- #105 weekly crafting quest (ISO week key + progress)
ALTER TABLE users ADD COLUMN IF NOT EXISTS craft_quest_week TEXT;
ALTER TABLE users ADD COLUMN IF NOT EXISTS craft_quest_done INTEGER NOT NULL DEFAULT 0;

-- #38/#113 sanctuary upgrades (persistent rest-floor perks), stored as
-- {"discount":1,"forge":1,...} level map
ALTER TABLE users ADD COLUMN IF NOT EXISTS abyss_sanctuary JSONB NOT NULL DEFAULT '{}';

-- Run-state additions
-- #7 momentum: consecutive floors without consumable use
ALTER TABLE abyss_active ADD COLUMN IF NOT EXISTS momentum INTEGER NOT NULL DEFAULT 0;
-- #15 Last Stand: token revive locks banking for the next N floors
ALTER TABLE abyss_active ADD COLUMN IF NOT EXISTS bank_locked_floors INTEGER NOT NULL DEFAULT 0;
ALTER TABLE abyss_active ADD COLUMN IF NOT EXISTS last_stand_used BOOLEAN NOT NULL DEFAULT FALSE;
-- #15 double-or-nothing is now offered on only some downs; TRUE = not offered this run
ALTER TABLE abyss_active ADD COLUMN IF NOT EXISTS revive_locked BOOLEAN NOT NULL DEFAULT FALSE;
-- #2/#3 checkpoint / express-elevator starts
ALTER TABLE abyss_active ADD COLUMN IF NOT EXISTS checkpoint_start INTEGER NOT NULL DEFAULT 0;
ALTER TABLE abyss_active ADD COLUMN IF NOT EXISTS express_until INTEGER NOT NULL DEFAULT 0;
-- #24 comeback buff applied to this run (label + stat bonus)
ALTER TABLE abyss_active ADD COLUMN IF NOT EXISTS comeback BOOLEAN NOT NULL DEFAULT FALSE;
-- #35 rift peek: pre-rolled upcoming floor types consumed by descend
ALTER TABLE abyss_active ADD COLUMN IF NOT EXISTS floor_queue JSONB;

-- #101 crafting materials
CREATE TABLE IF NOT EXISTS user_materials (
    client_uid TEXT   NOT NULL REFERENCES users(client_uid) ON DELETE CASCADE,
    mat_id     TEXT   NOT NULL,
    count      BIGINT NOT NULL DEFAULT 0 CHECK (count >= 0),
    PRIMARY KEY (client_uid, mat_id)
);

-- #118 rune library: once etched, a rune type is unlocked for cheaper re-etching
CREATE TABLE IF NOT EXISTS user_runes (
    client_uid TEXT NOT NULL REFERENCES users(client_uid) ON DELETE CASCADE,
    rune       TEXT NOT NULL,
    PRIMARY KEY (client_uid, rune)
);

-- #104 recipe discovery
CREATE TABLE IF NOT EXISTS user_recipes (
    client_uid TEXT NOT NULL REFERENCES users(client_uid) ON DELETE CASCADE,
    recipe_id  TEXT NOT NULL,
    obtained   TIMESTAMP NOT NULL DEFAULT NOW(),
    PRIMARY KEY (client_uid, recipe_id)
);

-- #123 forge history (last actions, also backs the daily undo)
CREATE TABLE IF NOT EXISTS forge_history (
    id         BIGSERIAL PRIMARY KEY,
    client_uid TEXT NOT NULL REFERENCES users(client_uid) ON DELETE CASCADE,
    action     TEXT NOT NULL,
    detail     TEXT NOT NULL DEFAULT '',
    cost       TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS forge_history_uid_idx ON forge_history (client_uid, id DESC);
