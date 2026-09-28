# Long campaign session memory capture

Run `npx playwright test --config=playwright.session-memory.config.js` for three
independent development-profile samples. Each sample plays mission 1 repeatedly
through all three tiers and the boss using normal keyboard input, banking and
replay controls, without reloading the page or modifying combat state. Warm up
one complete mission before the baseline. Run for at least 30 minutes after that
baseline; finish the current mission before the final checkpoint.

The fixture uses synthetic characters on port 18098 and is freshly compiled.
The report records its revision and tracked patch hash; an accompanying local
patch captures existing modifications. The physical minimum laptop is not tested.
CPU slowdown is 4, viewport 1280x900, DPR 1, and display preset Lower power at 30 FPS.

At completed-mission boundaries following each five-minute interval, settle two
seconds, collect garbage, settle one second and collect again. Record actual
elapsed time, heap usage, DOM counters, reachable object counts and a local heap
snapshot. Mission duration can delay a checkpoint; timestamps expose that delay.
Record CDP-listed browser process working sets and private bytes separately.
Those include non-JS allocations and are not the retained-JS-heap measurement.

The report also records mission IDs, replay seeds, tier completion, errors,
visibility, canvas context loss, rendered-frame liveness and audio voice cleanup.
The instrumentation retains only counters in the page, not an unbounded frame
array. Heap snapshots are local diagnostic artifacts and must not be published.

Set `BRAWL_SESSION_SMOKE=1` for one short harness verification. It still completes
all three tiers, but it never counts as the 30-minute gate. The full harness
requires retained-heap growth at most 10 MiB and no renderer/audio failure.
Passing the numerical check is insufficient: inspect snapshots and retaining
paths for actors, DOM, effects, listeners, timers and audio nodes before declaring
the gate passed. No session-memory pass is currently claimed.

The September 28 smoke run completed a warmup mission plus two replayed complete
missions. Its artifacts are in `test-results/session-memory-smoke-recovery`.
The earlier failed driver attempt is retained in `test-results/session-memory`:
it stopped in tier 2 before baseline. Synchronizing the tier UI and periodically
releasing/repressing held inputs let the normal-control driver complete the run.
These short captures validate the harness only, not the session-memory gate.

After the capture process has finished, run:

```powershell
python scripts/analyze-brawl-restart-memory.py test-results/session-memory-full
node --test tests/performance/heap-retention.test.cjs tests/performance/heap-summary.test.cjs
```

The reviewer accepts both restart-cycle and completed-mission checkpoints. It
reports actor, effect, run and attempt-record shapes, audio/timer counts, detached
nodes and direct strong retainers of run-shaped objects. Weak edges are excluded
from the direct-retainer list because they do not keep their target alive. Each
snapshot must contain only the corresponding current expedition ID; a stale ID
raises an error and requires inspection. Each run also has one shortest path from the synthetic snapshot root through
non-weak edges. Breadth-first traversal handles cycles, and printed paths are
limited to 128 nodes with explicit truncation status. These are snapshot-graph
paths, not V8 ephemeron-liveness proofs or dominator retained sizes. Follow
suspicious owners in the original snapshot before drawing a leak conclusion.

## Failed first full-duration attempt

The first long capture at `329e7e7b` plus the preserved engine diff was stopped
after 35 measured mission replays and 27 renderer errors. Its last complete heap
checkpoint was after 31 replays at 22.45 minutes. It is incomplete and failed,
not a 30-minute pass. Raw reports/heaps and `capture-provenance.json` remain in
`test-results/session-memory-full`; no samples were silently discarded.

Connection recovery returned an area-effect indicator without a `name`. The
renderer uppercased that missing field, throwing on every affected frame. The
previous regression updated only the text HUD. An expanded test now passes the
server-produced recovery snapshot through the actual renderer, reproducing the
crash before the fix and passing once the indicator supplies its name.

Across the five preserved checkpoints, actor-shaped payloads stay at 163,
run-shaped payloads at two, and live effect-shaped payloads at zero. Both run IDs
match the current expedition. Latest root paths lead through the Window globals
to existing loot/renderer closure contexts. Attempt-record shapes grow from two
to 95, so this capture does not establish a retention plateau. Heap grows from
7.071 to 8.35 MiB; low growth does not override the renderer failure.

Future captures stop on the first runtime error, preserve its stack/timestamp,
and stop the remaining sample batch. They write one shared fixture-source record
and copy that original patch into every sample, plus the server's asset-build ID.
Do not change source between fixture launch and its first sample source capture.
Later source edits require a new run before claiming evidence for those edits.

## Replacement capture: first sample completed

The replacement capture in `test-results/session-memory-fixed-full` uses
`4755892b` plus its recorded original engine diff. Sample one completed 40 full
mission replays after warmup over 30.49 measured minutes with zero reported
runtime errors. Retained JS grew by 1.331 MiB (about 7.07 to 8.40 MiB), passing
only the numeric heap-size threshold. The report explicitly retains
`retaining-path review required`. Samples two and three and the final heap
review remain unfinished; this is not a three-sample gate pass. Later terrain
reaction commits are not served by this running fixture and are not covered by
its evidence.

## Shareable aggregate export

Run `node scripts/brawl-session-evidence.cjs <capture-directory>` to write
`public-session-evidence.json` beside the private reports. It exports only
sample numbers, strictly validated source hashes, durations, counts and heap
measurements, plus fixed status labels. No player/run IDs, local paths, patch
contents, heap strings, error text or arbitrary report metadata are copied.
A baseline alone has unknown growth, not zero measured growth.

The exporter recomputes growth from checkpoints and requires complete 30-minute
samples before marking their numeric threshold as passed. Three samples must
share source provenance; even then the result requires retention review and
never claims release readiness or a verified physical device. It is only a
partial release-evidence artifact. Process-memory inspection, raw retention
analysis, other gates and final release verification remain separate work.
The raw reports and heap snapshots stay local.
