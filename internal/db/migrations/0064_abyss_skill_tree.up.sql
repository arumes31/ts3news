-- Abyss Skill Web: PoE-style passive tree with 1000 allocatable nodes.
-- Allocated node IDs are stored per player; the tree itself is generated
-- deterministically in code (internal/content/abysstree.go).
CREATE TABLE IF NOT EXISTS user_abyss_tree (
    client_uid   TEXT        NOT NULL,
    node_id      INTEGER     NOT NULL,
    allocated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (client_uid, node_id)
);

CREATE INDEX IF NOT EXISTS idx_user_abyss_tree_uid ON user_abyss_tree (client_uid);
