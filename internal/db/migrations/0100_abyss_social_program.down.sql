DROP TABLE IF EXISTS abyss_tournament_members;
DROP TABLE IF EXISTS abyss_tournament_teams;
DROP TABLE IF EXISTS abyss_raid_members;
DROP TABLE IF EXISTS abyss_raid_lobbies;
DROP TABLE IF EXISTS abyss_referrals;
DROP TABLE IF EXISTS abyss_referral_codes;
DROP TABLE IF EXISTS abyss_kudos;
DROP TABLE IF EXISTS abyss_floor_messages;
DROP TABLE IF EXISTS abyss_duels;
DROP TABLE IF EXISTS abyss_mentor_pairs;
DROP TABLE IF EXISTS abyss_consumable_trades;
DROP TABLE IF EXISTS abyss_shoutbox_messages;
DROP TABLE IF EXISTS abyss_guild_weekly_progress;
DROP TABLE IF EXISTS abyss_guild_members;
DROP TABLE IF EXISTS abyss_guilds;
DROP TABLE IF EXISTS abyss_rescue_missions;
DROP TABLE IF EXISTS abyss_friend_cheers;
DROP TABLE IF EXISTS abyss_friendships;
DROP TABLE IF EXISTS abyss_party_members;

ALTER TABLE abyss_escrow_loot
    DROP COLUMN IF EXISTS party_recipient_uid;

ALTER TABLE abyss_deaths
    DROP COLUMN IF EXISTS rescued_at,
    DROP COLUMN IF EXISTS lost_cache;
