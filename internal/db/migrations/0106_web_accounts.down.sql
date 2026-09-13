DROP TABLE web_auth_attempts;
DROP TABLE web_sessions;
ALTER TABLE users DROP COLUMN web_recovery_issued, DROP COLUMN web_recovery_expires,
    DROP COLUMN web_recovery_hash, DROP COLUMN web_password_hash, DROP COLUMN web_username;
DROP FUNCTION next_web_username();
DROP SEQUENCE web_username_seq;
