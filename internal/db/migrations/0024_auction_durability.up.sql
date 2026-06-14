-- Preserve a listed gear piece's current durability so a buyer receives the worn
-- item rather than a freshly repaired one.
ALTER TABLE auction_house ADD COLUMN IF NOT EXISTS durability INT;
