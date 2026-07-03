-- Depth-locked cosmetic badge: players can display one earned Abyss achievement
-- code as a badge next to their name. NULL = no badge selected.
ALTER TABLE users ADD COLUMN IF NOT EXISTS abyss_active_badge TEXT;
