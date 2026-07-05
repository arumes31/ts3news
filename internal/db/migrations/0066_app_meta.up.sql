-- 0066: generic key/value app metadata.
-- First use: the Abyss skill-web layout hash. When the generated web layout
-- changes (nodes, edges or point costs), the bot grants everyone a free full
-- respec by wiping user_abyss_tree and recording the new hash.
CREATE TABLE IF NOT EXISTS app_meta (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);
