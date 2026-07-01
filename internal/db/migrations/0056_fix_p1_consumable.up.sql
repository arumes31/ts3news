-- Legacy Abyss loot granted the small health potion under the id 'P1', which is not
-- in the consumable catalog (GetConsumableByID) — so those potions rendered as the
-- raw id "P1" and could not be used ("invalid consumable"). Fold existing rows into
-- the catalog id 'small_health_potion', merging charge counts where a row exists.
INSERT INTO user_consumables (client_uid, cons_id, remaining_fights)
SELECT client_uid, 'small_health_potion', remaining_fights
  FROM user_consumables WHERE cons_id = 'P1'
ON CONFLICT (client_uid, cons_id)
DO UPDATE SET remaining_fights = user_consumables.remaining_fights + EXCLUDED.remaining_fights;

DELETE FROM user_consumables WHERE cons_id = 'P1';
