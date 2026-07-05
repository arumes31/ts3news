-- Constrain allocated skill-web node IDs to the generated tree's range
-- (1..1000; the root node 0 is implicit and never stored). Rows outside the
-- range could otherwise consume points without contributing any bonus.
-- Kept as a range CHECK (not a FK) because the tree itself lives in code
-- (internal/content/abysstree.go), not in a table; the app validates exact
-- node existence and path connectivity on allocation.
DELETE FROM user_abyss_tree WHERE node_id < 1 OR node_id > 1000;

ALTER TABLE user_abyss_tree
    ADD CONSTRAINT user_abyss_tree_node_id_range CHECK (node_id BETWEEN 1 AND 1000);
