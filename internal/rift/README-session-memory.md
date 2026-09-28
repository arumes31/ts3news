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

## Replacement capture: two complete samples, third failed

The replacement capture in `test-results/session-memory-fixed-full` uses
`4755892b` plus its recorded original engine diff. Sample one completed 40 full
mission replays after warmup over 30.49 measured minutes with zero reported
runtime errors. Retained JS grew by 1.331 MiB (about 7.07 to 8.40 MiB). Sample
two completed 45 replays over 30.74 minutes, with zero reported runtime errors
and 1.363 MiB JS growth (about 7.11 to 8.47 MiB). Both pass only the numeric
heap-size threshold and retain `retaining-path review required`. Sample three
failed on character death after 30 completed replays; its last heap checkpoint
was at 21.46 minutes. The batch exited with two passes and one failure. It is
not a three-sample gate pass. Later terrain reaction commits were not served
by this fixture and are not covered by its evidence.

The failed driver was below the mission-one ledge at (297.337,425.019), with its
target at (342.411,394.536). Upward movement crosses the blocked landing edge;
the target's depth difference also exceeds the driver's attack threshold.
The driver now follows keyboard waypoints around an end before ascending,
using the target's side when it lies outside the ledge. Three focused Node
checks and a real-keyboard browser test (`test-results/session-driver-ledge`)
pass. Character stats, damage, collisions and the memory thresholds are unchanged.
This repairs a demonstrated driver trap, not proof that future full captures pass.

The completed heap analysis found 163 actor shapes, two current-expedition run
shapes and zero effect shapes at every captured checkpoint. No stale run ID was
found. History shapes grew while records accumulated (sample two ended at 137).
Counts alone do not establish retention ownership or a plateau; root-path review
and process-memory investigation remain required. The aggregate export preserves
the third sample as incomplete and release readiness as false.

Renderer process memory needs separate investigation. Sample one's private
bytes rose from 136.42 to 340.51 MiB, increasing at every recorded checkpoint;
working set rose from 372.71 to 489.92 MiB. Sample two's private bytes rose from
141.09 to 380.64 MiB, also increasing at every checkpoint; working set rose from
385.96 to 544.20 MiB. These are the OS measurements in `browserProcesses` for
the renderer, not JS heap sizes or proven application leak sizes. The capture
uses repeated heap snapshots and browser automation, so allocation ownership
and measurement overhead must be investigated before attributing the growth.
The low JS growth does not establish stable process memory.

## Retention ownership review of completed sample two

The final original heap contains 137 history-shaped objects. Shortest non-weak
root paths divide into 46 records through the current run, 46 through the
protocol response base (`snapshotBase.fields.attempt_history`), and 45 through
the renderer's previous snapshot. The source caps each attempt history at 50
and validates that cap in the client. This is consistent with three bounded
views still filling; it does not prove a post-cap plateau. The two run-shaped
objects belong to the current expedition and are held by current presentation
closures and the renderer's previous snapshot, not earlier expedition IDs.
Local path details are in `history-retention-review.json` and the original
`retention-review.json` under this capture directory.

Native `blink::NetworkResourcesData::ResourceData` objects grow from 183 to
5,591, with reported shallow size from 52,704 to 1,610,208 bytes. Inspected paths
in both original heaps lead from C++ persistent roots through DevToolsSession,
InspectorNetworkAgent and NetworkResourcesData. This identifies browser
inspection bookkeeping as a contributor; shallow sizes do not explain the
entire 239.55 MiB renderer private-byte increase. No claim that all native
growth is instrumentation, or that process memory is stable, follows from this.
Local example paths are retained in `network-retention-review.json`. Raw paths
are excluded from the public aggregate export.

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


## Completed ledge-routing capture (2026-09-28)

`test-results/session-memory-ledgeroute-full` completed all three full samples
with no recorded runtime errors. Each replay cleared all three tiers of mission
1 using keyboard input; the corrected ledge navigator avoided the earlier stall.

| Sample | Measured minutes | Completed replays | JS heap growth |
| --- | ---: | ---: | ---: |
| 1 | 30.69 | 38 | 1,334,812 bytes |
| 2 | 30.44 | 40 | 1,856,752 bytes |
| 3 | 30.34 | 40 | 1,487,536 bytes |

The numeric heap limit passed in all three samples. The privacy-safe aggregate
reports `retention_review_required`, not release readiness. Original heap
retaining paths and separate process-memory measurements still require review.
The earlier failed capture remains preserved; these results do not replace its
failure evidence.

This fixture used revision `45f5359ddc1d67af94c6b89922e249cee3bdd318` plus tracked
diff SHA-256 `6aab0502cd21b63391b02389260b4bc983fde1f96ab9f4ffd6da94966d0833ae`.
It predates the new potion, boss ward, bridge and circular-court changes. Do not
attribute this memory evidence to those newer changes or to physical target
hardware. The source snapshot remains beside the local capture.

A separate review of process counters from this capture found renderer private
bytes increasing from 153,321,472 to 352,280,576 in sample 1, 141,766,656 to
378,273,792 in sample 2, and 130,072,576 to 374,468,608 in sample 3. Detached-node
counts ended at 78 in every sample (starting at 69, 52 and 70 respectively).
These measurements do not identify the retaining owner. They prevent treating
the JS heap threshold as evidence that total renderer memory stays stable;
original retaining paths and instrumentation overhead still need investigation.

## Retaining-path review of the completed capture

The extended local analyzer inspected all 21 saved heaps. Retained run IDs
matched each checkpoint's current expedition. Actor-shaped objects settled at
163 (sample 3 began at 171); effect-shaped objects were zero at the settled
checkpoints. Listener count remained 606, with unchanged audio-node counts.

Detached nodes reached 78 at the first measured checkpoint and stayed there in
all three samples. Every final detached-node path was inspected: 64 canvases
belong to the bounded renderer atlas cache, nine images to renderer assets, four
images to the catalog cache and one image to the chest renderer. Detached here
means off-document image/canvas objects, not discarded UI trees accumulating.

Final attempt-record counts were 116, 122 and 122. Their paths divide into
current run / protocol snapshotBase / renderer previous snapshot respectively:
39/39/38, 41/41/40 and 41/41/40. The server caps each attempt history at 50, but
these captures never reached that cap. They cannot demonstrate a post-cap
plateau; a longer or suitably warmed capture remains necessary.

Native NetworkResourcesData::ResourceData counts rose from 168 to 4,753,
167 to 5,070 and 176 to 5,063. Final shallow sizes were 1,368,864, 1,460,160
and 1,458,144 bytes. At every checkpoint, three positional example paths led
through DevToolsSession, InspectorNetworkAgent and NetworkResourcesData.
This establishes inspector retention for the sampled resources, not for every
native allocation. Shallow sizes do not explain all renderer private bytes.
A controlled comparison of instrumentation overhead remains necessary before
attributing total process growth to the application or declaring it stable.

Original paths, IDs and heap strings remain only in the ignored local
retention-review.json. No raw retaining paths belong in public release evidence.
Ledger 0798 remains open for post-cap and process-memory verification.


## Post-history-cap diagnostic

Set `BRAWL_SESSION_POST_CAP=1` (leave `BRAWL_SESSION_SMOKE` unset) and run the
same session-memory Playwright config with a new output directory. This performs
one warmed sample, at least30 measured minutes and at least60 complete three-tier
replays. It has a60-minute timeout; a timeout is incomplete evidence, not a pass.
Normal mode remains three30-minute samples. Smoke and post-cap are mutually
exclusive, and the mode is recorded explicitly in the report.

The ordinary five-minute checkpoints remain, with mandatory additional snapshots
after replay49,54,59 and60. Including the completed warmup, replay49 reaches the
50-entry attempt-history limit. Every checkpoint records the actual history count;
the four mandatory checkpoints must each contain50 entries. This establishes that
the capture exercised pruning rather than assuming elapsed time reached the cap.
Retained object paths and separate process memory must still be reviewed after
the capture; numeric JS heap growth alone never closes0798. The public exporter recognizes this mode separately and never classifies it as
a standard three-sample gate result. It verifies a completed three-tier warmup,
contiguous complete replays, actual history counts, mandatory cap checkpoints,
ordered times, duration and source hashes. It derives heap growth from checkpoints
and exports only allowlisted numeric aggregates, including separate renderer
private bytes when available. Its best status is post_cap_review_required; it
never claims release readiness. Preserve raw heaps and identifiers locally.

    node --test tests/performance/session-options.test.cjs

Standard evidence exports also validate mission completion: a completed warmup
must precede contiguous numbered replays, and each record must contain completed
status and tiers 0, 1, and 2 in order. Missing, partial, duplicate, or malformed
mission records remain incomplete even when the numeric heap threshold passes.
The existing three captures retain their review-required status with 38, 40,
and 40 verified replays. The focused exporter/options suite contains 13 checks.

## Completed post-cap diagnostic (2026-09-28)

The single post-cap capture completed 60 full three-tier replays in 2,580,290ms
of measured play (43.00 minutes; 44.2 minutes including fixture/warmup). It recorded
zero runtime errors. Source revision was b4531bca95614db6d20b8558a74accf8287cf787,
with tracked-diff hash 1c131dbc2231b54da968f996b1063a30e81b3dca27fec709c060cb47aaa42a5b.
Later rendering and simulation changes are outside this capture's source scope.

History counts at replays 49, 54, 59 and 60 were all 50. Retained attempt objects
were 149, 150, 150 and 150. Final root paths split exactly into 50 current-run,
50 protocol snapshotBase and 50 previous-renderer records. All eleven heaps
retained only their current expedition IDs. Actor counts were 163 except for a
transient 171 at replay 32; effect-shaped counts stayed zero, with two run shapes.
Detached objects plateaued at 78 from replay 8 onward. Audio and listener counts
fluctuated within bounded observed ranges rather than increasing per replay.

JS heap grew 1,411,320 bytes overall, below the 10MiB limit. From replay 49 to 60
it decreased by 21,976 bytes. This establishes a bounded history plateau in this
capture, not a complete process-memory pass: renderer private bytes rose from
171,667,456 to 487,350,272, including growth after the history cap. Native network
resource counts rose from 168 to 7,589; three final example paths led through
DevToolsSession, InspectorNetworkAgent and NetworkResourcesData. A controlled
instrumentation comparison is still required to attribute total growth.

A disk-full build failure during the run was resolved by clearing the verified
Go build cache. The fixture was not restarted. The local environment-events.json
records this event and concurrent bounded build work; consider this host activity
when interpreting process memory. Raw heaps and retaining paths remain ignored
under test-results/session-memory-post-cap-20260928. The allowlisted public
aggregate is tests/performance/baselines/session-post-cap-2026-09-28.json.
Ledger 0798 remains open for process-memory/instrumentation verification.


## Completed direct inspection comparison (2026-09-28)

Both sequential isolated Chromium runs completed60 measured three-tier replays
plus warmup with zero runtime errors. Fixture ecf7750d2370e56c95cc864fd322eff1dcb42477,
tracked-diff1c131dbc2231b54da968f996b1063a30e81b3dca27fec709c060cb47aaa42a5b,
and driver/browser/graphics/profile/settings hashes matched. Public evidence:
tests/performance/baselines/direct-memory-2026-09-28.json.

| Measurement | Network inspection off | Network inspection on |
| --- | ---: | ---: |
| Measured duration (ms) | 2,629,310 | 2,406,396 |
| Overall JS heap growth (bytes) | 1,074,816 | 1,232,596 |
| Post-cap JS heap growth, replay49 to60 (bytes) | 43,304 | 141,940 |
| Overall renderer private growth (bytes) | 87,605,248 | 317,341,696 |
| Post-cap renderer private growth (bytes) | 9,515,008 | 49,803,264 |
| Final network resource records | 0 | 7,183 |

All20 heaps retained only their current expedition IDs. Actor-shaped objects
stayed163, effect-shaped objects zero and run-shaped objects two. Attempt records
were149 at49 and150 at54/59/60 in both runs; final paths split into50 current-run,
50 protocol snapshotBase and50 renderer previous records. Audio/timer/listener
counts fluctuated within observed bounds. Detached counts plateaued at79 with
inspection off and77-78 after warmup with it on; counts alone are not a complete
ownership audit of detached objects.

Inspection-off heaps contained zero NetworkResourcesData::ResourceData objects.
Inspection-on counts rose165 to7183; three final positional example paths passed
through DevToolsSession and InspectorNetworkAgent. Total shallow resource size
was2,068,704 bytes, insufficient to explain all process growth. Renderer growth
was229,736,448 bytes greater with inspection enabled. This supports an inspector
contribution, but sequential order, different durations and heap collection in
both variants prevent full causal attribution from one pair. Remaining9,515,008
bytes of post-cap growth with inspection off is neither a proven application
leak nor a demonstrated plateau.

Ledger0798 remains open for remaining process-memory attribution. A next diagnostic
can keep Network disabled and compare process counters with fewer heap snapshots
to isolate collector contributions, preserving gameplay and thresholds. Raw heaps,
profiles and paths remain ignored under test-results/direct-memory-full-20260928.
Capture and retention analysis completed successfully. No other browser workload
ran during this paired capture. This is review evidence, not a release pass.
