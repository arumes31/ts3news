-- Item System Migration
-- Adds item components, crafted items, recipes, and player inventory

-- Add item components table
CREATE TABLE IF NOT EXISTS item_components (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    icon TEXT,
    tier INTEGER NOT NULL DEFAULT 1,
    stats JSONB NOT NULL DEFAULT '{}'
);

-- Add crafted items table
CREATE TABLE IF NOT EXISTS crafted_items (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    icon TEXT,
    tier INTEGER NOT NULL DEFAULT 4,
    components JSONB NOT NULL DEFAULT '[]',
    stats JSONB NOT NULL DEFAULT '{}',
    effects JSONB NOT NULL DEFAULT '[]'
);

-- Add crafting recipes table
CREATE TABLE IF NOT EXISTS crafting_recipes (
    id TEXT PRIMARY KEY,
    result_id TEXT NOT NULL REFERENCES crafted_items(id),
    components JSONB NOT NULL DEFAULT '[]',
    gold_cost INTEGER NOT NULL DEFAULT 0
);

-- Add player items table (inventory)
CREATE TABLE IF NOT EXISTS player_items (
    id TEXT PRIMARY KEY,
    client_uid TEXT NOT NULL,
    item_id TEXT NOT NULL,
    item_type TEXT NOT NULL, -- 'component' or 'crafted'
    equipped_to TEXT,        -- Unit ID if equipped
    created_at TIMESTAMP DEFAULT NOW()
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_player_items_client_uid ON player_items(client_uid);
CREATE INDEX IF NOT EXISTS idx_player_items_equipped_to ON player_items(equipped_to);
CREATE INDEX IF NOT EXISTS idx_crafting_recipes_result_id ON crafting_recipes(result_id);

-- Insert basic components (Tier 1)
INSERT INTO item_components (id, name, description, icon, tier, stats) VALUES
('comp_iron_sword', 'Iron Sword', 'A basic iron blade', '⚔️', 1, '{"atk": 10}'),
('comp_leather_armor', 'Leather Armor', 'Basic protective gear', '🛡️', 1, '{"hp": 50, "def": 5}'),
('comp_magic_ring', 'Magic Ring', 'A ring imbued with magic', '💍', 1, '{"atk": 15, "mdef": 10}'),
('comp_swift_boots', 'Swift Boots', 'Boots that enhance speed', '👢', 1, '{"spd": 1}'),
('comp_lucky_charm', 'Lucky Charm', 'A charm that brings fortune', '🍀', 1, '{"crit_chance": 5}')
ON CONFLICT (id) DO NOTHING;

-- Insert advanced components (Tier 2)
INSERT INTO item_components (id, name, description, icon, tier, stats) VALUES
('comp_steel_blade', 'Steel Blade', 'A sharp steel weapon', '🗡️', 2, '{"atk": 25}'),
('comp_chain_mail', 'Chain Mail', 'Interlocking metal rings', '⛓️', 2, '{"hp": 150, "def": 15}'),
('comp_arcane_orb', 'Arcane Orb', 'A sphere of pure magic', '🔮', 2, '{"atk": 40, "mdef": 25}'),
('comp_wind_talisman', 'Wind Talisman', 'Harness the power of wind', '🌪️', 2, '{"spd": 2}'),
('comp_critical_gem', 'Critical Gem', 'A gem that enhances precision', '💎', 2, '{"crit_chance": 15}')
ON CONFLICT (id) DO NOTHING;

-- Insert rare components (Tier 3)
INSERT INTO item_components (id, name, description, icon, tier, stats) VALUES
('comp_mythril_edge', 'Mythril Edge', 'A blade forged from mythril', '⚔️', 3, '{"atk": 50, "crit_damage": 10}'),
('comp_dragon_scale', 'Dragon Scale', 'Scales from an ancient dragon', '🐉', 3, '{"hp": 400, "def": 30, "mdef": 20}'),
('comp_void_crystal', 'Void Crystal', 'A crystal from the void', '🌑', 3, '{"atk": 80, "mdef": 50, "lifesteal": 15}'),
('comp_phoenix_feather', 'Phoenix Feather', 'A feather from a phoenix', '🔥', 3, '{"spd": 3, "dodge": 10}'),
('comp_chaos_stone', 'Chaos Stone', 'A stone of pure chaos', '💥', 3, '{"crit_chance": 25, "crit_damage": 50}')
ON CONFLICT (id) DO NOTHING;

-- Insert epic crafted items (Tier 4)
INSERT INTO crafted_items (id, name, description, icon, tier, components, stats, effects) VALUES
('craft_blade_fallen', 'Blade of the Fallen', 'A blade that grows stronger with each kill', '⚔️', 4, 
 '["comp_iron_sword", "comp_iron_sword", "comp_steel_blade"]',
 '{"atk": 60, "crit_chance": 20}',
 '[{"type": "on_kill", "trigger": "kill", "chance": 100, "cooldown": 0, "effect": "buff", "value": 30, "duration": 3}]'),

('craft_aegis_valor', 'Aegis of Valor', 'A shield that protects the worthy', '🛡️', 4,
 '["comp_leather_armor", "comp_leather_armor", "comp_chain_mail"]',
 '{"hp": 500, "def": 40}',
 '[{"type": "passive", "trigger": "low_hp", "chance": 100, "cooldown": 10, "effect": "buff_def", "value": 50, "duration": 5}]'),

('craft_staff_eternity', 'Staff of Eternity', 'A staff that channels arcane power', '🔮', 4,
 '["comp_magic_ring", "comp_magic_ring", "comp_arcane_orb"]',
 '{"atk": 120, "mdef": 60}',
 '[{"type": "on_hit", "trigger": "attack", "chance": 20, "cooldown": 3, "effect": "magic_damage", "value": 150, "duration": 0}]')
ON CONFLICT (id) DO NOTHING;

-- Insert legendary crafted items (Tier 5)
INSERT INTO crafted_items (id, name, description, icon, tier, components, stats, effects) VALUES
('craft_godslayer', 'Godslayer', 'A weapon capable of slaying gods', '⚔️', 5,
 '["comp_mythril_edge", "craft_blade_fallen"]',
 '{"atk": 150, "crit_chance": 30, "crit_damage": 100}',
 '[{"type": "on_crit", "trigger": "crit", "chance": 100, "cooldown": 0, "effect": "true_damage", "value": 300, "duration": 0}]'),

('craft_immortal_plate', 'Immortal Plate', 'Armor that defies death itself', '🛡️', 5,
 '["comp_dragon_scale", "craft_aegis_valor"]',
 '{"hp": 1500, "def": 80, "mdef": 50}',
 '[{"type": "on_death", "trigger": "death", "chance": 100, "cooldown": 0, "effect": "revive", "value": 50, "duration": 0}]'),

('craft_void_reaper', 'Void Reaper', 'A weapon that consumes souls', '🌑', 5,
 '["comp_void_crystal", "craft_staff_eternity"]',
 '{"atk": 250, "mdef": 100, "lifesteal": 30}',
 '[{"type": "on_kill", "trigger": "kill", "chance": 100, "cooldown": 0, "effect": "heal_percent", "value": 20, "duration": 0}]')
ON CONFLICT (id) DO NOTHING;

-- Insert crafting recipes
INSERT INTO crafting_recipes (id, result_id, components, gold_cost) VALUES
('recipe_blade_fallen', 'craft_blade_fallen', '["comp_iron_sword", "comp_iron_sword", "comp_steel_blade"]', 50),
('recipe_aegis_valor', 'craft_aegis_valor', '["comp_leather_armor", "comp_leather_armor", "comp_chain_mail"]', 60),
('recipe_staff_eternity', 'craft_staff_eternity', '["comp_magic_ring", "comp_magic_ring", "comp_arcane_orb"]', 70),
('recipe_godslayer', 'craft_godslayer', '["comp_mythril_edge", "craft_blade_fallen"]', 150),
('recipe_immortal_plate', 'craft_immortal_plate', '["comp_dragon_scale", "craft_aegis_valor"]', 180),
('recipe_void_reaper', 'craft_void_reaper', '["comp_void_crystal", "craft_staff_eternity"]', 200)
ON CONFLICT (id) DO NOTHING;
