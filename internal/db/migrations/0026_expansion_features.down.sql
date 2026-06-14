-- 0026_expansion_features.down.sql
DROP TABLE IF EXISTS world_events;
DROP TABLE IF EXISTS arcade_jackpots;
ALTER TABLE users DROP COLUMN IF EXISTS last_daily_spin;
ALTER TABLE users DROP COLUMN IF EXISTS vip_points;
