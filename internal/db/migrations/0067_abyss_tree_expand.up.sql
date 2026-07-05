-- The skill web grew past its original 1000 nodes: the outer "Ascendant Rim"
-- signature notables (IDs 1001..1100, generated in internal/content/abysstree.go).
-- Drop the fixed 1..1000 range CHECK from migration 0065 and replace it with a
-- simple positive-id check, so future web growth needs no further migration.
-- Exact node existence and path connectivity are still validated in the app.
ALTER TABLE user_abyss_tree
    DROP CONSTRAINT IF EXISTS user_abyss_tree_node_id_range;

ALTER TABLE user_abyss_tree
    ADD CONSTRAINT user_abyss_tree_node_id_positive CHECK (node_id >= 1);
