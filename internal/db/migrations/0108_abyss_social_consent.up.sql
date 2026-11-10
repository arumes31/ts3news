-- Co-op party enrollment and mentorship both bind another account. Both now
-- require that account's own acceptance: the inviter creates a pending row and
-- only the invited account can accept (or decline) it.

-- Pending co-op invites. Membership in abyss_party_members is created only
-- when the invited member accepts.
CREATE TABLE IF NOT EXISTS abyss_party_invites (
    owner_uid TEXT NOT NULL REFERENCES users(client_uid) ON DELETE CASCADE,
    member_uid TEXT NOT NULL REFERENCES users(client_uid) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (owner_uid, member_uid),
    CHECK (owner_uid <> member_uid)
);
CREATE INDEX IF NOT EXISTS idx_abyss_party_invites_member
    ON abyss_party_invites (member_uid, created_at DESC);

-- Mentorship offers are pending until the mentee accepts. Existing pairs keep
-- working (they are treated as accepted) and can now be dissolved by the mentee.
ALTER TABLE abyss_mentor_pairs ADD COLUMN IF NOT EXISTS accepted_at TIMESTAMPTZ;
UPDATE abyss_mentor_pairs SET accepted_at = created_at WHERE accepted_at IS NULL;
ALTER TABLE abyss_mentor_pairs DROP CONSTRAINT IF EXISTS abyss_mentor_pairs_mentee_uid_key;
CREATE UNIQUE INDEX IF NOT EXISTS idx_abyss_mentor_pairs_accepted_mentee
    ON abyss_mentor_pairs (mentee_uid) WHERE accepted_at IS NOT NULL;
