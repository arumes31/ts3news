# Direct-browser memory comparison driver

The direct CDP transport and Chromium launcher are groundwork for comparing
network-inspector overhead under the same synthetic Brawl gameplay workload.
They do not provide a completed memory comparison or close ledger0798.

`brawl-direct-cdp.cjs` routes commands and target-session notifications over one
WebSocket. It enables no CDP domains implicitly. Callers must explicitly enable
Runtime/Page and choose whether Network inspection belongs in their sample.
Timeout, protocol-error and disconnect handling reject pending commands; listeners
can be removed after heap snapshot streaming. Three Node tests cover routing,
error handling and timeouts.

`brawl-direct-chromium.cjs` starts the installed Playwright Chromium executable
directly, using an isolated profile under the caller's ignored output directory.
It binds remote debugging to loopback, launches headless with windows hidden,
and closes only its own browser. It does not connect to the user's open browser.
Profiles and diagnostic stderr stay local and must not enter public evidence.

The September28 launch smoke used Chrome153.0.8010.12, created a separate target,
enabled Runtime, evaluated2+2 and confirmed visible document state, then closed
the browser. No Network.enable command was sent. This proves basic transport and
launch, not campaign execution, comparable rendering or stable process memory.
Next reuse the complete three-tier keyboard navigator and checkpoint protocol
for paired Network-enabled/disabled captures with identical launch settings.
Record exceptions and HTTP failures independently in the disabled variant.

## Paired campaign harness

Run `node node_modules/@playwright/test/cli.js test --config=playwright.direct-memory.config.js`
with ABYSS_E2E_PORT=18098. BRAWL_DIRECT_MEMORY_SMOKE=1 selects two short validation
runs; omit it for at least30 measured minutes and60 complete replays per variant.
The fixture starts fresh, variants run sequentially, and each launches a fresh
isolated browser/profile. Variant1 does not enable Network; variant2 does.
Both use Runtime/Page, the same1280x900/DPR1, CPU4x, Lower power settings and the
same mission1 bloodblade keyboard strategy with the shared ledge navigator.
API reads use Node fetch with the synthetic fixture cookie, outside the page's
Network domain in both variants. Runtime/console errors, page fetch/resource
failures and API response statuses remain checked when inspection is disabled.
Do not compare absolute native memory to the earlier Playwright launch as if
launch flags or browser-process composition were identical; compare the paired
variants first, and record remaining instrumentation differences.

By default every checkpoint uses2s settling, GC,1s settling, GC, process counters, heap/DOM
counters, render/audio health and a heap snapshot. Full runs explicitly capture
replays49/54/59/60, plus five-minute checkpoints. Driver sources and hashes,
server diff, launch arguments and graphics settings stay with local raw reports.
The heap collector itself remains enabled in both variants; this is a comparison
of Network inspection, not an entirely uninstrumented browser.

The initial paired smoke passed in5.1minutes, with a warmup plus two complete
three-tier replays per variant. Disabled inspection recorded0Network events at
both checkpoints; enabled inspection recorded162then388. Runtime/error checks
passed. Artifacts: test-results/direct-memory-smoke-20260928. This establishes
that both drivers can complete the workload; it does not establish a long-session
plateau or attribute all process-memory growth. The full comparison is documented in internal/rift/README-session-memory.md.

## Public aggregate export

`node scripts/brawl-direct-memory-evidence.cjs <capture-directory>` writes
public-direct-memory-evidence.json using fixed labels, validated hashes and
numeric aggregates only. Browser profiles/cookies, launch paths, heap strings,
run IDs, driver source and error text are excluded. A complete pair requires
all60replays, all post-cap checkpoints, event evidence for the chosen inspection
mode, valid process counters and matching fixture/driver/browser/settings hashes.
Smoke and partial runs remain incomplete. The original two smoke reports were
exported and correctly remain incomplete; their numeric observations are retained.

Renderer growth is calculated from checkpoint counters, with separate overall
and post-cap changes. The enabled-minus-disabled growth difference is reported
only for a complete matching pair. Durations, sequential order, GC/heap capture
and remaining instrumentation still require review before interpreting that
difference; one pair cannot establish causality. Release readiness and physical
hardware verification are always false. Four paired-export tests plus ten shared
session-export tests passed. The full pair completed; see internal/rift/README-session-memory.md for results and limits.


## Heap-snapshot frequency diagnostic

BRAWL_DIRECT_SNAPSHOT_ENDPOINTS=1 retains heap snapshots only at the baseline and
final checkpoint. Intermediate checkpoints still perform the same settling, two
explicit garbage collections, JS heap/DOM and browser-process counters, and
render/audio/error checks. Reports record heapSnapshotPolicy, snapshotTaken and
final explicitly; missing intermediate retaining paths are intentional and cannot
support a retaining-path plateau claim. Default behavior still snapshots every
checkpoint. No production behavior or release budget changes.

For the frequency comparison, use --grep 'inspection false' for both runs so
Network remains disabled. Run a fresh all-snapshot control with the endpoint flag
unset, then a fresh endpoint-only capture with it set, in distinct output folders.
Keep the same source and driver, graphics/settings and complete60-replay workload.
Both require30 measured minutes and checkpoints49/54/59/60. This comparison can
help attribute snapshot-collector overhead but does not remove all instrumentation
or establish causality from one sequential pair.

Smoke mode now completes at least two measured replays and includes a checkpoint
after replay1. This tests an intermediate checkpoint independently of the final
one. Endpoint-only smoke must have exactly two snapshots and at least one skipped
snapshot; the all-snapshot variant must capture every checkpoint.

The network-overhead exporter rejects explicit endpoint-only or unknown snapshot
policies, even if a report claims its usual paired mode. Historical reports with
no policy field retain their original all-checkpoint interpretation. Endpoint
frequency reports need separate comparison evidence; they are not a Network pair
and must not be relabeled to pass that exporter.

Endpoint-only smoke passed in3.6minutes and all-checkpoint smoke in4.2minutes.
Both completed warmup plus two measured three-tier replays with0Network events
and no runtime errors. Snapshot flags at checkpoints0/1/2 were true/false/true
and true/true/true respectively; the final flag was set only at checkpoint2.
These method-validation runs used different tracked-diff hashes during harness
development and must not be treated as a matched memory-growth pair. Public
aggregate: tests/performance/baselines/heap-frequency-smoke-2026-09-28.json.
The new policy-export regression failed before the fix, then all five paired
export tests passed. No performance or memory gate is closed by these smokes.


## Replay acknowledgement and first frequency attempt

The first full all-snapshot capture stopped after checkpoint54 (54 measured
replays,42.2minutes including setup) when replay55 observed the previous run
still complete. Direct mouse dispatch does not await the asynchronous begin
handler or its API response. The endpoint-only variant never started because
the sequential shell stops on failure. Keep the partial capture; it cannot
satisfy60-replay/post-cap comparison requirements. Public aggregate:
tests/performance/baselines/heap-frequency-partial-2026-09-28.json.

startDirectExpedition now waits for the prior UI request to settle before
clicking, then waits for an authoritative unpaused fighting run. Replay requires
a different expedition ID; initial start/resume can retain its identity. Runtime
errors still fail immediately, and no transition still times out. Four direct
start tests cover delayed old state, paused replacements, resume, failed starts
and errors; they failed before the acknowledgement fix and pass after it.

The separate frequency exporter is scripts/brawl-heap-frequency-evidence.cjs.
Pass two capture directories with one report each. It requires matching complete
workloads, source/driver/browser/settings/graphics, Network OFF, render/audio
health and the declared snapshot pattern. It publishes fixed numeric fields
and hashes, excludes raw heap strings/paths/error messages, and never grants
release readiness. Seven unit/CLI tests also verify malformed-report privacy.
The capture-failure error count is not proof of an application memory defect.

The fixed-driver endpoint-only smoke passed in3.1minutes: warmup plus two
complete measured three-tier replays, zero Network events, and the required
baseline/intermediate/final snapshot assertions. Raw evidence:
test-results/direct-start-ack-smoke-20260928. Restart the full comparison in
fresh directories; the partial attempt cannot be combined with the new driver.
