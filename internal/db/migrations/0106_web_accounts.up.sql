-- Readable, permanent login names. The sequence makes concurrent inserts and
-- backfills collision-free; names are identifiers, never authentication secrets.
CREATE SEQUENCE web_username_seq MINVALUE 0 START 0;
CREATE FUNCTION next_web_username() RETURNS TEXT LANGUAGE plpgsql VOLATILE AS $$
DECLARE
    n BIGINT := nextval('web_username_seq');
    animals TEXT[] := ARRAY['fox','owl','bear','lynx','wolf','hare','deer','hawk','seal','dove','swan','wren','mole','puma','orca','ibis','kiwi','crab','frog','newt','moth','duck','goat','lion','yak','finch','otter','panda','robin','raven','koala','tiger'];
    places TEXT[] := ARRAY['river','forest','meadow','valley','grove','shore','ridge','brook','hill','lake','cloud','field','glade','dune','reef','island','creek','marsh','cove','peak','trail','wood','cliff','dawn','rain','snow','moon','star','leaf','fern','moss','pine'];
BEGIN
    RETURN animals[1 + (n % 32)::INT] || '-' || places[1 + ((n / 32) % 32)::INT] || '-' || (10 + n / 1024)::TEXT;
END;
$$;

ALTER TABLE users ADD COLUMN web_username TEXT DEFAULT next_web_username();
ALTER TABLE users ALTER COLUMN web_username SET NOT NULL;
CREATE UNIQUE INDEX users_web_username_unique ON users (lower(web_username));
ALTER TABLE users ADD COLUMN web_password_hash TEXT;
ALTER TABLE users ADD COLUMN web_recovery_hash TEXT;
ALTER TABLE users ADD COLUMN web_recovery_expires TIMESTAMPTZ;
ALTER TABLE users ADD COLUMN web_recovery_issued TIMESTAMPTZ;
CREATE UNIQUE INDEX users_web_recovery_unique ON users (web_recovery_hash) WHERE web_recovery_hash IS NOT NULL;

CREATE TABLE web_sessions (
    token_hash TEXT PRIMARY KEY,
    client_uid TEXT NOT NULL REFERENCES users(client_uid) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX web_sessions_user ON web_sessions(client_uid);
CREATE INDEX web_sessions_expiry ON web_sessions(expires_at);
-- Preserve active legacy cookies for a bounded transition. New logins always
-- receive an independent random session; legacy credentials never gain reset rights.
INSERT INTO web_sessions (token_hash, client_uid, expires_at)
SELECT encode(sha256(convert_to(web_token, 'UTF8')), 'hex'), client_uid,
       COALESCE(web_token_expires, NOW() + INTERVAL '90 days')
FROM users WHERE web_token IS NOT NULL AND web_token <> ''
AND (web_token_expires IS NULL OR web_token_expires > NOW());

CREATE TABLE web_auth_attempts (
    key_hash TEXT PRIMARY KEY,
    attempts INTEGER NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX web_auth_attempts_expiry ON web_auth_attempts(expires_at);
