ALTER TABLE users ADD COLUMN IF NOT EXISTS gold BIGINT NOT NULL DEFAULT 0;

CREATE TABLE IF NOT EXISTS auction_house (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    seller_uid TEXT NOT NULL REFERENCES users(client_uid) ON DELETE CASCADE,
    item_type TEXT NOT NULL, -- 'gear', 'skill', 'artifact', 'unique', 'ultimate'
    item_id TEXT NOT NULL,
    item_name TEXT NOT NULL,
    item_data JSONB, -- stores stats, rarity, durability, etc.
    price BIGINT NOT NULL,
    listed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    buyer_uid TEXT REFERENCES users(client_uid) ON DELETE SET NULL,
    sold_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_auction_house_active ON auction_house (expires_at) WHERE sold_at IS NULL;
