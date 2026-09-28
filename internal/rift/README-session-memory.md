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
raises an error and requires inspection. This is a diagnostic check, not a full
GC-root path search or dominator retained-size calculation. Follow suspicious
owners in the original snapshot before drawing a leak conclusion.
