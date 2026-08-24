CREATE TABLE IF NOT EXISTS abyss_tree_mutation_audit (
    id BIGSERIAL PRIMARY KEY,
    client_uid TEXT NOT NULL REFERENCES users(client_uid) ON DELETE CASCADE,
    action TEXT NOT NULL,
    node_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
    succeeded BOOLEAN NOT NULL,
    request_key TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT abyss_tree_mutation_audit_action_length CHECK (char_length(action) BETWEEN 1 AND 32),
    CONSTRAINT abyss_tree_mutation_audit_request_key_length CHECK (char_length(request_key) <= 128)
);

CREATE INDEX IF NOT EXISTS idx_abyss_tree_mutation_audit_user_recent
    ON abyss_tree_mutation_audit (client_uid, id DESC);
