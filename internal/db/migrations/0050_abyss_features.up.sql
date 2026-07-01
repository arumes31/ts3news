-- The Abyss feature expansion: three new Deep-Delver upgrade nodes that apply in
-- the Abyss reward/escrow/XP layer (no combat-engine surgery needed). The existing
-- Fortune node is finally wired into the loot roller in code; these columns add new
-- token sinks alongside it. All are authoritative counters, so non-negative.

-- Compounding: +0.5% escrow interest per level (rides on the let-it-ride curve).
ALTER TABLE users ADD COLUMN IF NOT EXISTS abyss_up_interest INTEGER NOT NULL DEFAULT 0 CHECK (abyss_up_interest >= 0);
-- Tribute: +10% Abyss Tokens earned on bank per level.
ALTER TABLE users ADD COLUMN IF NOT EXISTS abyss_up_tribute  INTEGER NOT NULL DEFAULT 0 CHECK (abyss_up_tribute >= 0);
-- Insight: +5% combat reward XP from cleared floors per level.
ALTER TABLE users ADD COLUMN IF NOT EXISTS abyss_up_insight  INTEGER NOT NULL DEFAULT 0 CHECK (abyss_up_insight >= 0);
