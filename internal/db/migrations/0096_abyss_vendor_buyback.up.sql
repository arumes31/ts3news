CREATE TABLE IF NOT EXISTS abyss_vendor_buybacks (
    id BIGSERIAL PRIMARY KEY,
    client_uid TEXT NOT NULL REFERENCES users(client_uid) ON DELETE CASCADE,
    gear_id TEXT NOT NULL,
    durability INTEGER NOT NULL CHECK (durability >= 0),
    item_data JSONB,
    acquired_at TIMESTAMPTZ NOT NULL,
    sale_value BIGINT NOT NULL CHECK (sale_value > 0),
    buyback_cost BIGINT NOT NULL CHECK (buyback_cost >= sale_value),
    sold_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_abyss_vendor_buybacks_recent
    ON abyss_vendor_buybacks (client_uid, sold_at DESC, id DESC);
