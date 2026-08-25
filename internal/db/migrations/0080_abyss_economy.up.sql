-- Abyss economy tranche (AB-126..150): durable preferences, notifications,
-- orders, gifts, subscriptions, loans and reserved auction bids.

CREATE TABLE IF NOT EXISTS abyss_economy_profiles (
    client_uid TEXT PRIMARY KEY REFERENCES users(client_uid) ON DELETE CASCADE,
    potion_subscription BOOLEAN NOT NULL DEFAULT FALSE,
    potion_delivery_date DATE,
    repair_until TIMESTAMPTZ,
    scratch_date DATE,
    token_bundle_date DATE,
    token_bundle_count INTEGER NOT NULL DEFAULT 0 CHECK (token_bundle_count >= 0),
    bounty_insurance_week TEXT,
    bounty_insurance_used BOOLEAN NOT NULL DEFAULT FALSE,
    tax_rebate_week TEXT
);

CREATE TABLE IF NOT EXISTS abyss_ah_watchlist (
    client_uid TEXT NOT NULL REFERENCES users(client_uid) ON DELETE CASCADE,
    item_id TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (client_uid, item_id)
);

CREATE TABLE IF NOT EXISTS abyss_economy_events (
    id BIGSERIAL PRIMARY KEY,
    client_uid TEXT NOT NULL REFERENCES users(client_uid) ON DELETE CASCADE,
    kind TEXT NOT NULL,
    message TEXT NOT NULL,
    amount BIGINT NOT NULL DEFAULT 0,
    seen BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_abyss_economy_events_unseen
    ON abyss_economy_events (client_uid, created_at DESC) WHERE seen = FALSE;

CREATE TABLE IF NOT EXISTS abyss_material_orders (
    id BIGSERIAL PRIMARY KEY,
    buyer_uid TEXT NOT NULL REFERENCES users(client_uid) ON DELETE CASCADE,
    material TEXT NOT NULL CHECK (material IN ('dust','shard','core')),
    unit_price BIGINT NOT NULL CHECK (unit_price > 0),
    remaining INTEGER NOT NULL CHECK (remaining >= 0 AND remaining <= 10000),
    escrow_gold BIGINT NOT NULL CHECK (escrow_gold >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    closed_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_abyss_material_orders_open
    ON abyss_material_orders (material, unit_price DESC, created_at) WHERE closed_at IS NULL;

CREATE TABLE IF NOT EXISTS abyss_shop_gifts (
    code TEXT PRIMARY KEY,
    sender_uid TEXT NOT NULL REFERENCES users(client_uid) ON DELETE CASCADE,
    recipient_uid TEXT NOT NULL REFERENCES users(client_uid) ON DELETE CASCADE,
    item_key TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    claimed_at TIMESTAMPTZ,
    CHECK (sender_uid <> recipient_uid)
);

CREATE TABLE IF NOT EXISTS abyss_shop_cosmetics (
    client_uid TEXT NOT NULL REFERENCES users(client_uid) ON DELETE CASCADE,
    cosmetic_key TEXT NOT NULL,
    acquired_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (client_uid, cosmetic_key)
);

CREATE TABLE IF NOT EXISTS abyss_vendor_sales (
    client_uid TEXT NOT NULL REFERENCES users(client_uid) ON DELETE CASCADE,
    item_type TEXT NOT NULL,
    sold_count INTEGER NOT NULL DEFAULT 0 CHECK (sold_count >= 0),
    PRIMARY KEY (client_uid, item_type)
);

ALTER TABLE abyss_active ADD COLUMN IF NOT EXISTS economy_loan_fee BIGINT NOT NULL DEFAULT 0;
ALTER TABLE abyss_active ADD COLUMN IF NOT EXISTS economy_loan_principal BIGINT NOT NULL DEFAULT 0;

ALTER TABLE auction_house ADD COLUMN IF NOT EXISTS current_bid BIGINT NOT NULL DEFAULT 0;
ALTER TABLE auction_house ADD COLUMN IF NOT EXISTS bidder_uid TEXT REFERENCES users(client_uid) ON DELETE SET NULL;
DO $$ BEGIN
    ALTER TABLE auction_house ADD CONSTRAINT chk_auction_bid_nonnegative CHECK (current_bid >= 0);
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;
CREATE INDEX IF NOT EXISTS idx_auction_house_bid_settlement
    ON auction_house (expires_at) WHERE sold_at IS NULL AND bidder_uid IS NOT NULL;
