DROP TABLE IF EXISTS abyss_achievements;

ALTER TABLE users DROP COLUMN IF EXISTS abyss_tokens;
ALTER TABLE users DROP COLUMN IF EXISTS abyss_lifetime_floors;
ALTER TABLE users DROP COLUMN IF EXISTS abyss_lifetime_banked;
ALTER TABLE users DROP COLUMN IF EXISTS abyss_deaths;
ALTER TABLE users DROP COLUMN IF EXISTS abyss_bank_streak;
ALTER TABLE users DROP COLUMN IF EXISTS abyss_last_descent;
ALTER TABLE users DROP COLUMN IF EXISTS abyss_curse_fights;
ALTER TABLE users DROP COLUMN IF EXISTS abyss_day;
ALTER TABLE users DROP COLUMN IF EXISTS abyss_day_gold;
ALTER TABLE users DROP COLUMN IF EXISTS abyss_up_vigor;
ALTER TABLE users DROP COLUMN IF EXISTS abyss_up_greed;
ALTER TABLE users DROP COLUMN IF EXISTS abyss_up_fortune;
ALTER TABLE users DROP COLUMN IF EXISTS abyss_up_ward;

ALTER TABLE abyss_active DROP COLUMN IF EXISTS tier;
ALTER TABLE abyss_active DROP COLUMN IF EXISTS insured;
ALTER TABLE abyss_active DROP COLUMN IF EXISTS revived;
ALTER TABLE abyss_runs DROP COLUMN IF EXISTS tier;

DELETE FROM arcade_jackpots WHERE game_key='abyss';
