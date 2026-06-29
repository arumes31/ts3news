-- Items found inside the Abyss are now escrowed (locked) for the duration of a
-- run, exactly like the gold cache: they are applied to the character only when
-- the run is banked, and discarded if the delver dies. This table holds the
-- pending grants for an active run, one row per dropped item.
CREATE TABLE IF NOT EXISTS abyss_escrow_loot (
    id         BIGSERIAL PRIMARY KEY,
    client_uid TEXT        NOT NULL,
    item_type  TEXT        NOT NULL,           -- gear|cons|skill|ultimate|artifact|title|unique|ench|gold
    label      TEXT        NOT NULL,           -- BBCode display string for the combat/bank log
    item_data  JSONB       NOT NULL,           -- serialized payload replayed through the live granters on bank
    depth      INTEGER     NOT NULL DEFAULT 0, -- floor it dropped on (flavour / future UI)
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_abyss_escrow_loot_uid ON abyss_escrow_loot (client_uid);
