# Abyss skill-tree API contract

The skill tree is an append-only, server-authoritative catalog. Existing node IDs are never reassigned. Clients must treat node names, descriptions, and effect metadata as display data rather than executable content.

## Compatibility

- `GET /abyss/tree` embeds `TREE_CATALOG.schema_version`, `layout_hash`, and `balance_hash`.
- Every tree mutation accepts `X-Abyss-Tree-Layout`. A missing header remains supported for legacy clients. A non-empty stale hash is rejected before a handler runs and the response includes the current `layout_hash`.
- Layout schema version `2` distinguishes topology compatibility from balance changes. `layout_hash` changes for node/edge topology; `balance_hash` also changes for names, stats, costs, and effects.
- New nodes are appended after legacy nodes. Startup validates unique IDs, known effects, finite values, valid kinds, bidirectional edges, and reachability.

## Mutation safety

All routes below are `POST`, use JSON, and are serialized by authenticated user ID.

| Route | Request | Behavior |
|---|---|---|
| `/api/abyss/tree/allocate` | `{"node_id": 123}` | Allocates one adjacent, affordable, depth-eligible node. |
| `/api/abyss/tree/batch_allocate` | `{"ids":[1,2]}` | Validates queue order, then commits all nodes atomically. |
| `/api/abyss/tree/refund` | `{"node_id":123}` | Refunds the node and dependent branch using one transaction. |
| `/api/abyss/tree/refund_preview` | `{"node_id":123}` | Quotes the exact cascade, node costs, refunded points, and gold fee without writing. |
| `/api/abyss/tree/respec` | `{}` | Clears the allocation using the weekly-free/token rule. |
| `/api/abyss/tree/respec_quote` | `{}` | Quotes affected nodes and the exact free/token cost without writing. |
| `/api/abyss/tree/plan_preview` | `{"ids":[1,2]}` | Read-only exact quote: canonical IDs, cost, gates, connectivity, and stat/effect delta. |
| `/api/abyss/tree/plan_draft` | `{"action":"save|list|delete","slot":1,"name":"…","ids":[…]}` | Stores at most five named drafts. |
| `/api/abyss/tree/build_import` | `{"ids":[…]}` or `{"code":"…"}` | Applies the same validation model used by preview. |

Every wrapped response contains `ok`, `error`, `msg`, `mutation`, `schema_version`, and `layout_hash`. Successful mutation responses contain `"ok":true`; rejected mutations contain `"ok":false` and a stable human-readable `error`. Allocation responses include exact node costs and `undo_expires_at`; refund responses include affected-node count, point total, per-node costs, and gold. Preview validity and commit acceptance use the same analyzer.

Request bodies are capped at 256 KiB and batch allocation accepts at most 100 submitted IDs. Duplicate IDs are canonicalized by preserving their first occurrence. Database reads and writes use the request context, and multi-node replacements roll back if any statement or commit fails.

## Idempotency

Mutation clients may send `Idempotency-Key` (maximum 128 bytes). A successful response is replayable for 15 minutes for the same authenticated user, route, key, and request body. A reused key with a different body is rejected. Replays carry `X-Idempotent-Replay: true`. Failed mutations are never cached.

## Build codes

Version 1 codes are standard padded Base64 containing JSON:

```json
{"v":1,"schema":2,"layout":"<topology hash>","ids":[1,2,3]}
```

The decoded payload is capped at 128 KiB, rejects unknown fields and trailing data, and contains at most 6,000 IDs. Commit requires the current layout hash; a stale code must be previewed instead.

## Rollout and telemetry

`ABYSS_TREE_ENHANCEMENTS_ENABLED=false` hides enhanced navigation, planning, inspection, progression, and accessibility layers and rejects enhancement-only APIs. The base skill tree remains available.

The private Abyss operations snapshot includes aggregate tree preview, commit, refund, respec, and failure counts; server mutation latency; optional client render latency from `X-Abyss-Tree-Render-Ms`; and aggregate sector/node-kind popularity. It stores no user ID, nickname, build code, or complete allocation.

Successful and rejected operations append a compact per-user audit record containing only the operation, canonical referenced node IDs, success flag, and idempotency key. Retention is bounded to the newest 50 records per user.
