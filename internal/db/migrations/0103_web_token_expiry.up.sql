-- Server-side session expiry and logout invalidation for the web portal.
-- web_token previously stayed valid forever; the 90-day limit was only a
-- browser cookie hint. web_token_expires lets the server reject expired and
-- invalidated tokens.
ALTER TABLE users ADD COLUMN IF NOT EXISTS web_token_expires TIMESTAMPTZ;
