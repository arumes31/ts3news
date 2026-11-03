-- Login links created before web_token_expires existed were preserved with a
-- NULL expiry. Their browser-session transition is handled by web_sessions
-- (migration 0106), so the link credential itself must not stay valid forever:
-- retire every link without a finite expiry. The next bot cycle PMs a fresh,
-- expiring link to online players.
UPDATE users SET web_token = NULL
WHERE web_token IS NOT NULL AND web_token_expires IS NULL;
