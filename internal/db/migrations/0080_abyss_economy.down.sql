DROP INDEX IF EXISTS idx_auction_house_bid_settlement;
ALTER TABLE auction_house DROP CONSTRAINT IF EXISTS chk_auction_bid_nonnegative;
ALTER TABLE auction_house DROP COLUMN IF EXISTS bidder_uid;
ALTER TABLE auction_house DROP COLUMN IF EXISTS current_bid;

ALTER TABLE abyss_active DROP COLUMN IF EXISTS economy_loan_principal;
ALTER TABLE abyss_active DROP COLUMN IF EXISTS economy_loan_fee;

DROP TABLE IF EXISTS abyss_shop_gifts;
DROP TABLE IF EXISTS abyss_shop_cosmetics;
DROP TABLE IF EXISTS abyss_vendor_sales;
DROP TABLE IF EXISTS abyss_material_orders;
DROP TABLE IF EXISTS abyss_economy_events;
DROP TABLE IF EXISTS abyss_ah_watchlist;
DROP TABLE IF EXISTS abyss_economy_profiles;
