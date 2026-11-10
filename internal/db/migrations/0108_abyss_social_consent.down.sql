DROP INDEX IF EXISTS idx_abyss_mentor_pairs_accepted_mentee;
DELETE FROM abyss_mentor_pairs WHERE accepted_at IS NULL;
ALTER TABLE abyss_mentor_pairs DROP COLUMN IF EXISTS accepted_at;
ALTER TABLE abyss_mentor_pairs ADD CONSTRAINT abyss_mentor_pairs_mentee_uid_key UNIQUE (mentee_uid);

DROP TABLE IF EXISTS abyss_party_invites;
