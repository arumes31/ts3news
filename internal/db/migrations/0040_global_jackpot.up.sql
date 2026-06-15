-- 0040: Add Global Jackpot
INSERT INTO arcade_jackpots (game_key, amount) VALUES ('global', 50000)
ON CONFLICT DO NOTHING;
