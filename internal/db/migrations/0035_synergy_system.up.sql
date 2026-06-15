-- Synergy/Traits System Migration
-- Adds trait definitions and synergy tracking

-- Add trait definitions table
CREATE TABLE IF NOT EXISTS trait_definitions (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    icon TEXT,
    thresholds JSONB NOT NULL DEFAULT '[]'
);

-- Create index for faster lookups
CREATE INDEX IF NOT EXISTS idx_trait_definitions_id ON trait_definitions(id);

-- Insert default trait definitions
INSERT INTO trait_definitions (id, name, description, icon, thresholds) VALUES
('warrior', 'Warrior', 'Warriors gain bonus attack damage', '⚔️', 
 '[{"count":2,"bonus":"+20 ATK","effects":[{"type":"stat_bonus","target":"trait_units","stat":"atk","value":20}]},{"count":4,"bonus":"+50 ATK, +10% Lifesteal","effects":[{"type":"stat_bonus","target":"trait_units","stat":"atk","value":50},{"type":"lifesteal","target":"trait_units","value":10}]}]'),

('tank', 'Tank', 'Tanks gain bonus health and defense', '🛡️',
 '[{"count":2,"bonus":"+150 HP","effects":[{"type":"stat_bonus","target":"trait_units","stat":"hp","value":150}]},{"count":4,"bonus":"+400 HP, +20 DEF","effects":[{"type":"stat_bonus","target":"trait_units","stat":"hp","value":400},{"type":"stat_bonus","target":"trait_units","stat":"def","value":20}]}]'),

('assassin', 'Assassin', 'Assassins gain critical strike chance', '🗡️',
 '[{"count":2,"bonus":"+15% Crit","effects":[{"type":"crit_chance","target":"trait_units","value":15}]},{"count":4,"bonus":"+30% Crit, +50% Crit Dmg","effects":[{"type":"crit_chance","target":"trait_units","value":30},{"type":"crit_damage","target":"trait_units","value":50}]}]'),

('mage', 'Mage', 'Mages gain bonus attack power', '🔮',
 '[{"count":2,"bonus":"+30 ATK","effects":[{"type":"stat_bonus","target":"trait_units","stat":"atk","value":30}]},{"count":4,"bonus":"+80 ATK, -20% Mana Cost","effects":[{"type":"stat_bonus","target":"trait_units","stat":"atk","value":80},{"type":"mana_cost_reduction","target":"trait_units","value":20}]}]'),

('dragon', 'Dragon', 'Dragons are powerful unique units', '🐉',
 '[{"count":1,"bonus":"+1000 HP, +100 ATK","effects":[{"type":"stat_bonus","target":"trait_units","stat":"hp","value":1000},{"type":"stat_bonus","target":"trait_units","stat":"atk","value":100}]}]'),

('titan', 'Titan', 'Titans gain damage reduction', '⚡',
 '[{"count":2,"bonus":"50% Damage Reduction","effects":[{"type":"damage_reduction","target":"trait_units","value":50}]}]'),

('brute', 'Brute', 'Brutes gain attack speed', '💪',
 '[{"count":2,"bonus":"+10% Attack Speed","effects":[{"type":"attack_speed","target":"trait_units","value":10}]},{"count":4,"bonus":"+25% Attack Speed, +30% ATK","effects":[{"type":"attack_speed","target":"trait_units","value":25},{"type":"stat_bonus","target":"trait_units","stat":"atk","value":30,"percentage":true}]}]'),

('wild', 'Wild', 'Wild units gain lifesteal', '🐺',
 '[{"count":2,"bonus":"+5% Lifesteal","effects":[{"type":"lifesteal","target":"trait_units","value":5}]},{"count":4,"bonus":"+15% Lifesteal, +20% ATK","effects":[{"type":"lifesteal","target":"trait_units","value":15},{"type":"stat_bonus","target":"trait_units","stat":"atk","value":20,"percentage":true}]}]'),

('scout', 'Scout', 'Scouts gain bonus range', '👁️',
 '[{"count":2,"bonus":"+1 Range","effects":[{"type":"range_bonus","target":"trait_units","value":1}]},{"count":4,"bonus":"+2 Range, +20% Attack Speed","effects":[{"type":"range_bonus","target":"trait_units","value":2},{"type":"attack_speed","target":"trait_units","value":20}]}]'),

('ranger', 'Ranger', 'Rangers gain attack speed', '🏹',
 '[{"count":2,"bonus":"+15% Attack Speed","effects":[{"type":"attack_speed","target":"trait_units","value":15}]},{"count":4,"bonus":"+35% Attack Speed, +1 Range","effects":[{"type":"attack_speed","target":"trait_units","value":35},{"type":"range_bonus","target":"trait_units","value":1}]}]'),

('elemental', 'Elemental', 'Elementals gain health and elemental power', '🌊',
 '[{"count":2,"bonus":"+100 HP","effects":[{"type":"stat_bonus","target":"trait_units","stat":"hp","value":100}]},{"count":4,"bonus":"+300 HP, +30% Elemental Dmg","effects":[{"type":"stat_bonus","target":"trait_units","stat":"hp","value":300},{"type":"elemental_damage","target":"trait_units","value":30}]}]'),

('knight', 'Knight', 'Knights gain block chance', '⚔️',
 '[{"count":2,"bonus":"+10% Block","effects":[{"type":"block_chance","target":"trait_units","value":10}]},{"count":4,"bonus":"+25% Block, +20 DEF","effects":[{"type":"block_chance","target":"trait_units","value":25},{"type":"stat_bonus","target":"trait_units","stat":"def","value":20}]}]'),

('rogue', 'Rogue', 'Rogues gain critical damage', '🎭',
 '[{"count":2,"bonus":"+20% Crit Damage","effects":[{"type":"crit_damage","target":"trait_units","value":20}]},{"count":4,"bonus":"+50% Crit Damage, +15% Crit","effects":[{"type":"crit_damage","target":"trait_units","value":50},{"type":"crit_chance","target":"trait_units","value":15}]}]'),

('golem', 'Golem', 'Golems gain health and tenacity', '🗿',
 '[{"count":2,"bonus":"+100 HP, +10% Tenacity","effects":[{"type":"stat_bonus","target":"trait_units","stat":"hp","value":100},{"type":"tenacity","target":"trait_units","value":10}]},{"count":4,"bonus":"+400 HP, +30% Tenacity","effects":[{"type":"stat_bonus","target":"trait_units","stat":"hp","value":400},{"type":"tenacity","target":"trait_units","value":30}]}]'),

('mystic', 'Mystic', 'Mystics gain magic resistance', '✨',
 '[{"count":2,"bonus":"+15% Magic Resist","effects":[{"type":"magic_resist","target":"trait_units","value":15}]},{"count":4,"bonus":"+35% Magic Resist, +50 HP","effects":[{"type":"magic_resist","target":"trait_units","value":35},{"type":"stat_bonus","target":"trait_units","stat":"hp","value":50}]}]')
ON CONFLICT (id) DO NOTHING;
