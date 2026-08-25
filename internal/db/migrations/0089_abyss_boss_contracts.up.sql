ALTER TABLE abyss_active
    ADD COLUMN IF NOT EXISTS boss_contract_wager BIGINT NOT NULL DEFAULT 0 CHECK (boss_contract_wager BETWEEN 0 AND 5),
    ADD COLUMN IF NOT EXISTS boss_contract_depth INTEGER NOT NULL DEFAULT 0 CHECK (boss_contract_depth >= 0);

ALTER TABLE abyss_active
    ADD CONSTRAINT abyss_active_boss_contract_pair CHECK (
        (boss_contract_wager = 0 AND boss_contract_depth = 0)
        OR (boss_contract_wager IN (1, 3, 5) AND boss_contract_depth > 0)
    );
