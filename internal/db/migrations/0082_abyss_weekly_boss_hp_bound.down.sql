ALTER TABLE abyss_weekly_bosses
    DROP CONSTRAINT IF EXISTS abyss_weekly_bosses_current_hp_check;

ALTER TABLE abyss_weekly_bosses
    ADD CONSTRAINT abyss_weekly_bosses_current_hp_check
    CHECK (current_hp >= 0);
