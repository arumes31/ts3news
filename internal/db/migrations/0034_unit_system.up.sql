-- Unit System Migration
-- Adds enhanced unit stats, abilities, and upgrades

-- Add unit abilities table
CREATE TABLE IF NOT EXISTS unit_abilities (
    id TEXT PRIMARY KEY,
    unit_key TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    icon TEXT,
    ability_type TEXT NOT NULL, -- 'active', 'passive', 'on_hit', 'on_death'
    cooldown INTEGER DEFAULT 0,
    mana_cost INTEGER DEFAULT 0,
    damage INTEGER DEFAULT 0,
    damage_type TEXT DEFAULT 'physical',
    range INTEGER DEFAULT 1,
    aoe BOOLEAN DEFAULT FALSE,
    aoe_radius INTEGER DEFAULT 0,
    effects JSONB DEFAULT '[]'
);

-- Add unit upgrades table
CREATE TABLE IF NOT EXISTS unit_upgrades (
    id TEXT PRIMARY KEY,
    unit_key TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    cost INTEGER NOT NULL,
    star_req INTEGER DEFAULT 1,
    effect TEXT NOT NULL,
    value INTEGER NOT NULL
);

-- Create indexes for faster lookups
CREATE INDEX IF NOT EXISTS idx_unit_abilities_unit_key ON unit_abilities(unit_key);
CREATE INDEX IF NOT EXISTS idx_unit_upgrades_unit_key ON unit_upgrades(unit_key);

-- Insert default abilities for existing units
INSERT INTO unit_abilities (id, unit_key, name, description, icon, ability_type, cooldown, mana_cost, damage, damage_type, range, aoe, aoe_radius, effects) VALUES
-- Brute abilities
('brute_rage', 'brute', 'Berserker Rage', 'Gain +50% ATK but -20% DEF for 5 ticks', '💢', 'active', 10, 30, 0, 'physical', 1, FALSE, 0, '[{"type":"buff","stat":"atk","value":50,"duration":5},{"type":"debuff","stat":"def","value":-20,"duration":5}]'),
('brute_slam', 'brute', 'Ground Slam', 'Slam the ground dealing 80 damage to nearby enemies', '💥', 'active', 8, 40, 80, 'physical', 1, TRUE, 1, '[]'),

-- Wolf abilities
('wolf_howl', 'wolf', 'Pack Howl', 'Increase ATK of all wolves by 20% for 4 ticks', '🐺', 'active', 12, 35, 0, 'physical', 0, FALSE, 0, '[{"type":"buff","stat":"atk","value":20,"duration":4,"target":"all_wolves"}]'),
('wolf_pounce', 'wolf', 'Pounce', 'Leap to target dealing 60 damage and stunning for 1 tick', '🦘', 'active', 6, 25, 60, 'physical', 2, FALSE, 0, '[{"type":"stun","duration":1}]'),

-- Archer abilities
('archer_multishot', 'archer', 'Multi-Shot', 'Fire arrows at up to 3 targets dealing 70% damage each', '🏹', 'active', 7, 30, 0, 'physical', 3, FALSE, 0, '[{"type":"multishot","targets":3,"damage_mult":0.7}]'),
('archer_piercing', 'archer', 'Piercing Shot', 'Fire a piercing arrow that ignores 50% DEF', '🎯', 'active', 9, 40, 100, 'physical', 4, FALSE, 0, '[{"type":"armor_pen","value":50}]'),

-- Mage abilities
('mage_ice_nova', 'mage', 'Ice Nova', 'Explode with ice dealing 150 magic damage and slowing enemies', '❄️', 'active', 8, 50, 150, 'magic', 2, TRUE, 2, '[{"type":"slow","value":30,"duration":3}]'),
('mage_frostbolt', 'mage', 'Frostbolt', 'Launch a frostbolt dealing 120 magic damage', '🧊', 'active', 5, 35, 120, 'magic', 3, FALSE, 0, '[]'),

-- Knight abilities
('knight_shield_wall', 'knight', 'Shield Wall', 'Gain +30% DEF and taunt enemies for 3 ticks', '🛡️', 'active', 10, 40, 0, 'physical', 1, FALSE, 0, '[{"type":"buff","stat":"def","value":30,"duration":3},{"type":"taunt","duration":3}]'),
('knight_charge', 'knight', 'Shield Charge', 'Charge at target dealing 90 damage and stunning for 1 tick', '⚔️', 'active', 8, 35, 90, 'physical', 2, FALSE, 0, '[{"type":"stun","duration":1}]'),

-- Rogue abilities
('rogue_backstab', 'rogue', 'Backstab', 'Deal 200% damage when attacking from behind', '🗡️', 'passive', 0, 0, 0, 'physical', 1, FALSE, 0, '[{"type":"backstab","damage_mult":2.0}]'),
('rogue_shadow_step', 'rogue', 'Shadow Step', 'Teleport behind target and gain +50% crit chance for 2 ticks', '👤', 'active', 6, 30, 0, 'physical', 3, FALSE, 0, '[{"type":"teleport","position":"behind"},{"type":"buff","stat":"crit_chance","value":50,"duration":2}]'),

-- Golem abilities
('golem_fortify', 'golem', 'Fortify', 'Gain +50% DEF and +200 HP for 5 ticks', '🗿', 'active', 12, 50, 0, 'physical', 1, FALSE, 0, '[{"type":"buff","stat":"def","value":50,"duration":5},{"type":"buff","stat":"hp","value":200,"duration":5}]'),
('golem_earthquake', 'golem', 'Earthquake', 'Stomp the ground dealing 100 damage to all enemies', '🌋', 'active', 15, 60, 100, 'physical', 0, TRUE, 99, '[]'),

-- Sorcerer abilities
('sorcerer_arcane_blast', 'sorcerer', 'Arcane Blast', 'Unleash arcane energy dealing 200 magic damage', '🔮', 'active', 7, 60, 200, 'magic', 3, FALSE, 0, '[]'),
('sorcerer_mana_shield', 'sorcerer', 'Mana Shield', 'Convert mana to shield absorbing 150 damage', '✨', 'active', 10, 50, 0, 'magic', 1, FALSE, 0, '[{"type":"shield","value":150}]'),

-- Dragon abilities
('dragon_breath', 'dragon', 'Dragon Breath', 'Breathe fire dealing 250 magic damage in a cone', '🐉', 'active', 8, 70, 250, 'magic', 2, TRUE, 2, '[{"type":"burn","damage":30,"duration":3}]'),
('dragon_fly', 'dragon', 'Take Flight', 'Fly up becoming untargetable for 2 ticks', '🪽', 'active', 15, 50, 0, 'physical', 1, FALSE, 0, '[{"type":"untargetable","duration":2}]'),

-- Titan abilities
('titan_smash', 'titan', 'Titan Smash', 'Smash target dealing 300 damage and reducing their DEF by 30%', '👊', 'active', 10, 80, 300, 'physical', 1, FALSE, 0, '[{"type":"debuff","stat":"def","value":-30,"duration":5}]'),
('titan_roar', 'titan', 'Intimidating Roar', 'Reduce ATK of all enemies by 20% for 4 ticks', '📢', 'active', 12, 60, 0, 'physical', 0, TRUE, 99, '[{"type":"debuff","stat":"atk","value":-20,"duration":4}]')
ON CONFLICT (id) DO NOTHING;

-- Insert default upgrades for existing units
INSERT INTO unit_upgrades (id, unit_key, name, description, cost, star_req, effect, value) VALUES
-- Common upgrades (all units)
('upgrade_hp_1', 'all', 'Vitality I', '+100 HP', 10, 1, 'hp', 100),
('upgrade_hp_2', 'all', 'Vitality II', '+200 HP', 20, 2, 'hp', 200),
('upgrade_hp_3', 'all', 'Vitality III', '+400 HP', 40, 3, 'hp', 400),
('upgrade_atk_1', 'all', 'Strength I', '+10 ATK', 10, 1, 'atk', 10),
('upgrade_atk_2', 'all', 'Strength II', '+25 ATK', 25, 2, 'atk', 25),
('upgrade_atk_3', 'all', 'Strength III', '+50 ATK', 50, 3, 'atk', 50),
('upgrade_def_1', 'all', 'Toughness I', '+5 DEF', 15, 1, 'def', 5),
('upgrade_def_2', 'all', 'Toughness II', '+15 DEF', 30, 2, 'def', 15),
('upgrade_spd_1', 'all', 'Agility I', '+1 SPD', 20, 2, 'spd', 1),
('upgrade_crit_1', 'all', 'Precision I', '+10% Crit Chance', 25, 2, 'crit_chance', 10),

-- Unit-specific upgrades
('brute_upgrade_1', 'brute', 'Berserker Training', '+20% ATK when below 50% HP', 30, 2, 'passive_low_hp_atk', 20),
('wolf_upgrade_1', 'wolf', 'Pack Leader', '+15% ATK for each nearby wolf', 25, 2, 'pack_bonus', 15),
('archer_upgrade_1', 'archer', 'Eagle Eye', '+1 Range', 30, 2, 'range', 1),
('mage_upgrade_1', 'mage', 'Arcane Mastery', '+20% Magic Damage', 35, 2, 'magic_damage', 20),
('knight_upgrade_1', 'knight', 'Unbreakable', '+10% Damage Reduction', 40, 2, 'damage_reduction', 10),
('rogue_upgrade_1', 'rogue', 'Assassin', '+50% Crit Damage', 35, 2, 'crit_damage', 50),
('golem_upgrade_1', 'golem', 'Stone Skin', '+15% Damage Reduction', 45, 2, 'damage_reduction', 15),
('sorcerer_upgrade_1', 'sorcerer', 'Mana Font', '+30 Mana Regen', 40, 2, 'mana_regen', 30),
('dragon_upgrade_1', 'dragon', 'Ancient Power', '+100 HP, +50 ATK', 60, 3, 'multi_stat', 0),
('titan_upgrade_1', 'titan', 'Colossus', '+500 HP, +20% Damage Reduction', 80, 3, 'colossus', 0)
ON CONFLICT (id) DO NOTHING;
