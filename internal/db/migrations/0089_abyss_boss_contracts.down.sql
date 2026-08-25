ALTER TABLE abyss_active
    DROP CONSTRAINT IF EXISTS abyss_active_boss_contract_pair,
    DROP COLUMN IF EXISTS boss_contract_depth,
    DROP COLUMN IF EXISTS boss_contract_wager;
