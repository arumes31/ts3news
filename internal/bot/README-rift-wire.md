# Lean snapshot projection prototype

riftWireSnapshot is a server-side wire projection, currently not connected to
HTTP handlers or the browser. It never changes the database Run. Fifteen large
sections are retained by content token; all other fields serialize from the
current Run. A SHA256 token covers the retained sections and run ID. Only matching
step requests in fighting state can receive lean-v1; other cases receive full.
The client must still be integrated with exact-baseline hydration and existing
protocol/revision checks before enabling this path or completing0778.

The full-campaign receipt fixture reconstructs exactly after restoring only the
retained fields:340469 run-body bytes become7350 bytes, a97.84% reduction. This is
a synthetic large receipt, not a claim about every live run or HTTP wire framing.
Tests also cover unknown baseline, changed build, different run ID, non-step and
terminal-state fallback, and ensure projection does not mutate saved data.

Three serialization benchmark samples on Windows/amd64 Xeon Gold6126 at2.60GHz:
full1.568/1.603/1.669ms, about531–540KB allocated,197–198 allocations;
lean including token2.720/2.632/2.619ms,585–600KB,269 allocations.
Token generation currently reserializes retained content, so this prototype trades
about1ms extra server work for fewer transferred bytes on large records. It is not
a CPU optimization and establishes no end-to-end performance gate. Integration,
ordinary-run measurements, client recovery and browser journeys remain pending.

Reproduce:
    go test ./internal/bot -run TestRiftLeanSnapshot -count=1 -v
    go test ./internal/bot -run '^$' -bench BenchmarkRiftWireSnapshot -benchmem -count=3
