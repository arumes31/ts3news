# Lean movement snapshots

The client opts into lean-v1 using X-Rift-Snapshot and sends its last validated
X-Rift-Snapshot-Base on step requests. GET/start/pause/advance and terminal states
return full responses. Legacy callers without opt-in keep the original contract.
Database snapshots remain complete and unchanged.

Fifteen explicit sections (see riftRetainedFields) are identified by a SHA256 token
covering their contents and run identity. Only a matching step in fighting state
receives a compact projection. Any retained change, missing/stale token or other
run falls back to full. The projection does not mutate the saved Run.

The client retains a detached copy after response/revision validation. Hydration
copies only those sections into the newly received run; all dynamic fields are
replaced, so absent effects/receipts clear normally. Missing/wrong baseline, run,
action, unknown wire version or a mixed full/lean payload fails closed and uses
the existing explicit recovery path. Full recovery establishes a new baseline.
The retained sections are read-only to game rendering/UI code. Payload diagnostics
measure serialized wire run bytes before hydration, not reconstructed size.

The synthetic full-campaign receipt reconstructs exactly:340469 run-body bytes
become7350 (97.84% reduction). This is not a claim about every run or HTTP framing.
Three benchmarks on Windows/amd64 Xeon Gold6126 at2.60GHz measured full serialization
1.568/1.603/1.669ms,531–540KB allocated,197–198 allocations; lean including baseline
serialization/hash2.720/2.632/2.619ms,585–600KB,269 allocations. This trades about1ms
extra server work for fewer transferred bytes on large records; no end-to-end
performance gate is established by these measurements.

Unit checks cover exact reconstruction, non-mutation, header opt-in, legacy API,
content/build/level/encounter/receipt invalidation and terminal/full fallback.
Client tests cover clearing absent dynamic state and rejecting invalid baselines.
Browser checks cover live movement, full reload, corruption recovery, two complete
three-tier banking journeys, revision conflict, stale response, session expiry,
and bounded UTF-8 payload diagnostics. The representative browser fixture measured
a14695-byte full POST response versus6448-byte lean movement response (~56% less).
These are different moments in one run, not an identical-state comparison. Exact
identical-state reconstruction is covered by the large fixture unit test.
Bot TestRift5.600s; focused HTTP/projection tests3.334s; three client unit tests;
seven journey/lean/diagnostic browser tests1.6m and eight recovery/revision/session
tests51.5s passed. Final ordinary-payload measurement passed34.8s. Implements0778.

Reproduce:
    node --test tests/rift-wire.test.cjs
    go test ./internal/bot -run 'TestRiftLeanSnapshot|TestRiftHTTPProjection' -count=1 -v
    go test ./internal/bot -run '^$' -bench BenchmarkRiftWireSnapshot -benchmem -count=3
