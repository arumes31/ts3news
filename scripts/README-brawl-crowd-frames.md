# Crowded frame capture

```powershell
$env:ABYSS_E2E_PORT='18098'
node node_modules/@playwright/test/cli.js test --config=playwright.crowd-performance.config.js
```

This starts a fresh managed fixture and makes three independent Chromium contexts.
Each loads scenario=visual,seed=crowded-v1,crowd=120, selects Lower power30FPS at
1280x900/DPR1 with CDP CPU slowdown4, warms art for5s, then collects every completed
frame for at least60s. The fixture remains paused, at its fixed normal camera:
this is the performance budget's drawing-only workload, not a live AI/combat test.
Normal camera culling stays enabled; population120 does not mean120 sprites are
simultaneously visible. The report records alive/boss counts, camera, simulation
clock, cache occupancy/counters, graphics backend, raw timings and exact duration.

Pass/fail thresholds remain frame interval p95<=50ms,p99<=100ms, synchronous render
p95<=16ms. A passing collection test is distinct from the JSON performance verdict.
No errors, context loss, hidden time or camera drift are accepted. No private saves,
cookies or inventory are recorded; the fixture uses a synthetic character.

BRAWL_FRAME_SMOKE=1 collects one10s sample and cannot pass a gate. Remove it for
full capture. CPU slowdown/headless software graphics are development comparisons,
not substitutes for physical i5/UHD620 testing or GPU presentation traces. The
separate boss harness uses real inputs in mission100's final tier.

Set BRAWL_FRAME_PROFILE=1 for one separately instrumented capture. It writes
crowd.cpuprofile beside crowd-report.json and marks the gate unmeasured. Summarize
it with `python scripts/analyze-brawl-cpu-profile.py <path-to-crowd.cpuprofile>`.
Remove both profiling and smoke environment flags before collecting gate evidence.

## Canvas operation attribution

Set BRAWL_CANVAS_COST=1 to install the test-only canvas timing probe after warmup.
It records drawImage/save/restore counts, total/max duration and calls >=8ms,
grouped by the most recent image draw. BRAWL_FRAME_SMOKE=1 limits collection to
10 seconds; without it the diagnostic uses60 seconds. Either way the canvas-cost
flag forces diagnostic/profiling mode, never a numeric performance-gate pass.
The probe restores original method descriptors after collection. It preserves
return values and thrown errors; three focused Node tests verify those invariants and preceding-draw state capture.
It changes instrumentation overhead and cannot prove causality or GPU timing.
Image labels are static asset filenames or canvas dimensions; it retains no
image references, per-frame event arrays, query strings or player records.

The September28 short diagnostic recorded61 save calls >=8ms, all immediately
after rift_props.png draws, totaling3223.7ms across183 save calls in that group.
Maximum was87.3ms. Saves following creature images stayed below2.3ms. This
isolates a prop-associated state boundary for the next experiment, not proof
that save itself is expensive. No production drawing behavior changed.
Raw local evidence: test-results/crowd-canvas-cost-20260928.

## Rejected full-size prop ImageBitmap experiment

BRAWL_PROP_BITMAP_EXPERIMENT=1 injects a test-only readiness step that replaces
images.props with createImageBitmap(images.props). The exact experimental
renderer and its SHA-256 are retained with the report; production is unchanged.
Combine with BRAWL_CANVAS_COST=1 and BRAWL_FRAME_SMOKE=1 for the diagnostic.

Twelve image/state comparisons passed across all eight prop cells, both cover
heights, opacity, clipping, rotation and brightness. The archived probe is
 tests/performance/baselines/prop-bitmap-equivalence-2026-09-28.txt;
restore it as tests/e2e/rift-prop-bitmap-experiment.spec.js to reproduce.

The short instrumented candidate still attributed3155.6ms to saves following
the1774x887 bitmap (144 calls,48 >=8ms,max102.3ms), versus3223.7ms across183calls
and61slow calls for the earlier image baseline. Each had one slow boundary per
rendered frame; differing frame counts and host conditions prevent treating raw
total time as a speedup. Candidate render p95 was273.7ms versus183.4ms in the
baseline diagnostic. These are instrumented short captures, not gate results or
a controlled regression estimate. No useful improvement was established, so the
candidate is not shipped. Local evidence is in prop-bitmap-equivalence-20260928
and prop-bitmap-cost-20260928 beneath test-results.

## Prop draw state and rejected full-atlas clip

The diagnostic now groups costs by the preceding draw's filter, opacity and
compositing mode. In prop-draw-state-cost-20260928 every prop draw used filter
none, alpha1 and source-over. There were57 prop draws;57 subsequent saves took
at least8ms, totaling2868.4ms across171 grouped saves (max110.3ms). This rules out
an active CSS canvas filter or fractional opacity on those particular draws,
not every possible rendering cost or every gameplay scene.

A separate candidate used a Path2D destination clip with a full-atlas draw,
retaining the source atlas's coordinate origin instead of a fractional source
rectangle. All12 comparison panels changed pixels under the existing transformed,
clipped and opacity/brightness matrix. It was rejected without benchmarking and
never applied to production. Restore the archived
 tests/performance/baselines/prop-clip-equivalence-rejected-2026-09-28.txt
as tests/e2e/rift-prop-clip-experiment.spec.js to reproduce. Failure artifacts remain
under test-results/prop-clip-equivalence-20260928.


## Browser timeline diagnostic

Set BRAWL_FRAME_TRACE=1 for one diagnostic sample; add BRAWL_FRAME_SMOKE=1 for
10 seconds or leave it unset for60 seconds. CPU sampling is separate: it remains
off unless BRAWL_FRAME_PROFILE or BRAWL_CANVAS_COST is also enabled. Trace mode
always marks the capture as profiling and cannot pass a numeric gate.

The collector uses [Tracing.start and tracingComplete](https://raw.githubusercontent.com/ChromeDevTools/devtools-protocol/master/pdl/domains/Tracing.pdl)
with ReturnAsStream and JSON output, then [IO.read and IO.close](https://raw.githubusercontent.com/ChromeDevTools/devtools-protocol/master/pdl/domains/IO.pdl).
The trace buffer is64MiB and disk output is bounded to256MiB; known data loss
invalidates the test. Completion waits at most30 seconds. Five focused tests cover
mixed encoding, cleanup after read/size errors, missing streams and timeout.
The ignored crowd.timeline.json can contain URLs and event arguments; do not
publish it as release evidence. Public summaries must use selected numeric fields
and fixed event names, excluding raw args, process IDs and paths.

The first smoke completed without data loss (54,096 events,11,808,195 bytes).
Main-thread LayerTreeHost::DoUpdateLayers events summed5,082.282ms and animation
callbacks3,921.205ms. These are nested inclusive timings, not additive CPU totals;
see README-crowd-frame-baseline.md for interpretation and remaining work.
