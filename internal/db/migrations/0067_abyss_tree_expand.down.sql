-- Revert to the fixed 1..1000 node-id range. Drop any rows outside it first so
-- the narrower CHECK can be re-added (mirrors migration 0065's cleanup).
ALTER TABLE user_abyss_tree
    DROP CONSTRAINT IF EXISTS user_abyss_tree_node_id_positive;

DELETE FROM user_abyss_tree WHERE node_id < 1 OR node_id > 1000;

ALTER TABLE user_abyss_tree
    ADD CONSTRAINT user_abyss_tree_node_id_range CHECK (node_id BETWEEN 1 AND 1000);
