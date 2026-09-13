DROP TRIGGER IF EXISTS consumables_economy_ledger ON user_consumables;
DROP TRIGGER IF EXISTS materials_economy_ledger ON user_materials;
DROP TRIGGER IF EXISTS users_economy_ledger ON users;
DROP FUNCTION IF EXISTS record_economy_balance_change();
DROP TABLE IF EXISTS economy_ledger;
ALTER TABLE users DROP COLUMN IF EXISTS abyss_talent_credit;
