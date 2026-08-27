DROP TABLE IF EXISTS abyss_wager_entries;
DROP TABLE IF EXISTS abyss_rank_notifications;
DROP TABLE IF EXISTS abyss_competition_rewards;
DROP TABLE IF EXISTS abyss_competition_snapshots;
DROP TABLE IF EXISTS abyss_competition_periods;
DROP TABLE IF EXISTS abyss_competition_presence;

DROP INDEX IF EXISTS idx_abyss_runs_competition_channel;
DROP INDEX IF EXISTS idx_abyss_runs_competition_speed;
DROP INDEX IF EXISTS idx_abyss_runs_competition_depth;

ALTER TABLE abyss_social_profiles
    DROP COLUMN IF EXISTS shame_opt_in;

ALTER TABLE abyss_runs
    DROP COLUMN IF EXISTS audit_data,
    DROP COLUMN IF EXISTS audit_hash,
    DROP COLUMN IF EXISTS ts3_channel_id,
    DROP COLUMN IF EXISTS pact_multiplier,
    DROP COLUMN IF EXISTS build_key;
