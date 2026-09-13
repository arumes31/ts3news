-- Capture actual currency/stock deltas inside the writing transaction, including
-- older application paths without explicit operation annotations. No query text,
-- arguments, credentials, or user profile snapshots are retained.
ALTER TABLE users ADD COLUMN abyss_talent_credit BIGINT NOT NULL DEFAULT 0 CHECK (abyss_talent_credit >= 0);

CREATE TABLE economy_ledger (
    id BIGSERIAL PRIMARY KEY,
    client_uid TEXT NOT NULL,
    currency TEXT NOT NULL,
    balance_before NUMERIC(20,0) NOT NULL,
    balance_after NUMERIC(20,0) NOT NULL,
    delta NUMERIC(21,0) GENERATED ALWAYS AS (balance_after - balance_before) STORED,
    transaction_id BIGINT NOT NULL,
    source TEXT NOT NULL,
    request_id TEXT NOT NULL DEFAULT '',
    run_id TEXT NOT NULL DEFAULT '',
    item_id TEXT NOT NULL DEFAULT '',
    source_table TEXT NOT NULL,
    economy_epoch TEXT NOT NULL,
    revision TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    CHECK (balance_before <> balance_after)
);
CREATE INDEX economy_ledger_player_time ON economy_ledger (client_uid, created_at DESC, id DESC);
CREATE INDEX economy_ledger_transaction ON economy_ledger (transaction_id, id);
CREATE INDEX economy_ledger_epoch_time ON economy_ledger (economy_epoch, created_at DESC);

CREATE FUNCTION record_economy_balance_change() RETURNS TRIGGER LANGUAGE plpgsql AS $$
DECLARE
    before_row JSONB := CASE WHEN TG_OP = 'INSERT' THEN '{}'::jsonb ELSE to_jsonb(OLD) END;
    after_row JSONB := CASE WHEN TG_OP = 'DELETE' THEN '{}'::jsonb ELSE to_jsonb(NEW) END;
    column_name TEXT;
    currency_name TEXT;
    before_amount NUMERIC(20,0);
    after_amount NUMERIC(20,0);
    epoch TEXT;
    build_revision TEXT;
    transfer BOOLEAN := TG_OP = 'UPDATE' AND (
        before_row->>'client_uid' IS DISTINCT FROM after_row->>'client_uid'
        OR before_row->>'mat_id' IS DISTINCT FROM after_row->>'mat_id'
        OR before_row->>'cons_id' IS DISTINCT FROM after_row->>'cons_id');
    side INTEGER;
BEGIN
    -- A stock owner/key change is a debit from the old account and a credit to
    -- the new account, even when the quantity itself remains unchanged.
    FOR side IN 1..CASE WHEN transfer THEN 2 ELSE 1 END LOOP
    IF transfer THEN
        before_row := CASE WHEN side = 1 THEN to_jsonb(OLD) ELSE '{}'::jsonb END;
        after_row := CASE WHEN side = 2 THEN to_jsonb(NEW) ELSE '{}'::jsonb END;
    END IF;
    FOREACH column_name IN ARRAY TG_ARGV LOOP
        before_amount := COALESCE((before_row->>column_name)::numeric, 0);
        after_amount := COALESCE((after_row->>column_name)::numeric, 0);
        IF before_amount = after_amount THEN CONTINUE; END IF;
        IF TG_TABLE_NAME = 'user_materials' THEN
            currency_name := 'material:' || COALESCE(after_row->>'mat_id', before_row->>'mat_id');
        ELSIF TG_TABLE_NAME = 'user_consumables' THEN
            currency_name := 'consumable:' || COALESCE(after_row->>'cons_id', before_row->>'cons_id');
        ELSE
            currency_name := column_name;
        END IF;
        IF epoch IS NULL THEN
            epoch := COALESCE(NULLIF(current_setting('ts3news.economy_epoch', true), ''),
                (SELECT value FROM app_meta WHERE key = 'gold_economy_version'), 'unversioned');
            build_revision := COALESCE(NULLIF(current_setting('ts3news.economy_revision', true), ''),
                (SELECT value FROM app_meta WHERE key = 'economy_deployed_revision'), 'unknown');
        END IF;
        INSERT INTO economy_ledger (client_uid, currency, balance_before, balance_after,
            transaction_id, source, request_id, run_id, item_id, source_table, economy_epoch, revision)
        VALUES (COALESCE(after_row->>'client_uid', before_row->>'client_uid'), currency_name,
            before_amount, after_amount, txid_current(),
            COALESCE(NULLIF(current_setting('ts3news.economy_source', true), ''),
                substring(current_query() FROM '^[[:space:]]*/\* economy:([A-Za-z][A-Za-z0-9_.]{0,127}) \*/'), 'unattributed'),
            COALESCE(current_setting('ts3news.economy_request_id', true), ''),
            COALESCE(current_setting('ts3news.economy_run_id', true), ''),
            COALESCE(NULLIF(current_setting('ts3news.economy_item_id', true), ''),
                after_row->>'mat_id', before_row->>'mat_id', after_row->>'cons_id', before_row->>'cons_id', ''),
            TG_TABLE_NAME, epoch, build_revision);
    END LOOP;
    END LOOP;
    RETURN NULL;
END;
$$;

CREATE TRIGGER users_economy_ledger AFTER INSERT OR UPDATE OR DELETE ON users
    FOR EACH ROW EXECUTE FUNCTION record_economy_balance_change('gold', 'abyss_tokens', 'abyss_boss_tokens', 'scrap_stack', 'abyss_talent_credit');
CREATE TRIGGER materials_economy_ledger AFTER INSERT OR UPDATE OR DELETE ON user_materials
    FOR EACH ROW EXECUTE FUNCTION record_economy_balance_change('count');
CREATE TRIGGER consumables_economy_ledger AFTER INSERT OR UPDATE OR DELETE ON user_consumables
    FOR EACH ROW EXECUTE FUNCTION record_economy_balance_change('remaining_fights');
