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


## Drawing omission diagnostics

BRAWL_RENDER_ABLATION=actors or background injects a temporary renderer that
omits that component. The injected source and hash are retained locally; both
variants always count as profiling and cannot pass the performance gate. Unknown
values and combinations with the bitmap experiment are rejected. Production
rendering is unchanged. These variants help locate cost, not propose removals.

Sequential ten-second captures on2026-09-28 produced:

| Omitted drawing | Render p95 | Frame interval p95 | Frame interval p99 |
| --- | --- | --- | --- |
| Actors | 25.5ms | 99.9ms | 116.6ms |
| Background | 66.4ms | 150.1ms | 216.6ms |
| None (control) | 60.4ms | 150.0ms | 216.6ms |

Background omission did not establish an improvement. Actor omission reduced
drawing time but left substantial frame delays. Single short sequential samples
are insufficient to estimate a production speedup; the control also varied from
the earlier full-length captures. Next investigate actor drawing and browser
layer work while preserving all visible content. The five timeline collector
tests still pass with the deeper Blink/Skia categories enabled. Trace durations
remain overlapping inclusive measurements, not additive CPU or GPU totals.

Public numeric evidence: tests/performance/baselines/crowd-drawing-omissions-2026-09-28.json.
Raw reports and injected sources remain in the corresponding ignored
test-results/crowd-without-actors-20260928, crowd-without-background-20260928 and
crowd-omission-control-20260928 directories. No gate or ledger item is closed.


## Rejected duplicate-profile lookup candidate

Passing the actor function's already resolved profile as the seventh catalogActor
argument removed one duplicate lookup per living or defeated monster draw. The
helper defaulted that argument to bestiary.profile(unit) for direct callers.
The regression failed before the change (two lookups) and passed after it (one).
Three browser checks passed, including124 monsters,32 rigs,7020 frame definitions,
14688 actor draws, and existing atlas pixel/state/cache checks.

Three60-second candidate and three fresh control captures produced:

| Metric | Candidate | Control |
| --- | --- | --- |
| Render p95 (ms) | 64.1 / 67.3 / 89.7 | 70.8 / 66.6 / 76.7 |
| Frame interval p95 (ms) | 166.7 / 166.7 / 233.4 | 183.3 / 166.6 / 183.4 |
| Frame interval p99 (ms) | 216.7 / 199.9 / 283.3 | 216.7 / 216.7 / 249.9 |

All six fail the unchanged frame budget. The distributions overlap and vary
substantially; grouped sequential order limits inference. Median render p95 was
only4.9% lower, with a slower third candidate sample. This does not establish a
repeatable improvement, so the production change was reverted. Do not count
reduced lookup calls as proven frame improvement.

Public aggregates: tests/performance/baselines/actor-profile-reuse-{candidate,control}-2026-09-28.json.
The regression is archived as actor-profile-reuse-rejected-2026-09-28.txt in that
directory, since it intentionally fails against the retained original renderer.
Raw candidate/control output is under test-results/actor-profile-reuse-{full,control-full}-20260928;
the candidate directory also preserves candidate-renderer.js. No ledger item closes.


Deeper Blink/Skia trace analysis found64 layer-update events totaling5977.561ms.
Wholly contained events included17664 drawPath calls (1833.983ms),21504 drawRect
calls (824.316ms),3584 drawTextBlob calls (222.459ms), and64 canvas-resource
production calls (170.252ms). This locates drawing work inside layer updates;
it does not identify which gameplay primitive caused each call or prove GPU cost.
Inclusive timings can overlap. Public aggregate: crowd-layer-drawing-2026-09-28.json
in tests/performance/baselines; ignored raw trace: crowd-timeline-canvas-20260928
in test-results. No performance gate changes.


## Rejected opaque main-canvas experiment

BRAWL_OPAQUE_CANVAS_EXPERIMENT=1 injects alpha:false only into the main canvas
context. It cannot combine with other rendering experiments. The actual context
attributes are asserted and retained with each crowd report. This implements the
[HTML canvas alpha setting](https://html.spec.whatwg.org/multipage/canvas.html#concept-canvas-alpha);
it does not alter transparency of sprite sources or cache canvases. Production
still uses its original context settings.

The atlas pixel/state comparison supports the same switch and compares ordinary
versus opaque contexts with unchanged caching. Initial comparison passed. In a
sequential ten-second smoke pair, opaque render p95 was62.8ms versus60.8ms for
control; both interval p95 values were150ms, with p99 200ms versus183.3ms. No
benefit was established, so this candidate was rejected without longer captures.
These smoke captures cannot pass the gate. Public numeric evidence is
tests/performance/baselines/opaque-canvas-smoke-2026-09-28.json. Raw output is in
test-results/opaque-canvas-{smoke,control-smoke,equivalence}-20260928.

The follow-up pixel/state comparison also passed with explicit assertions for
alpha:true on control and alpha:false on candidate (19.7s including setup).
Artifacts: test-results/opaque-canvas-attributes-equivalence-20260928.


## Rejected shared monster atlas cache experiments

The shared1254x1254 atlases have156.75px columns, which bypass the retained
integer-only cache. Test-only origin-preserving per-crop prefixes kept pixels
unchanged but churned the8MiB cache:3810 misses and334.9ms render p95 in a10s
smoke. That layout was rejected. Its injected renderer is preserved with the raw
capture in test-results/shared-origin-smoke-20260928.

BRAWL_SHARED_ORIGIN_EXPERIMENT=1 now evaluates the second candidate from
scripts/brawl-shared-origin-experiment.cjs: one315x1254 origin-preserving strip
for the first two columns of each shared atlas. Other columns retain native
drawing, and a new strip is rejected when free cache space is insufficient.
Both the8MiB overall and2MiB single-entry bounds stay unchanged.

The revised candidate matched all256 atlas cells across flips, fractional
translation, opacity and brightness, alongside the existing hero/effect/prop
pixel and state checks. Cache admission with free space and fallback when full
also passed. An initial admission assertion placed after the cache-fill test
correctly returned null; the follow-up tests both states separately. Evidence:
test-results/shared-idle-strip-admission-equivalence-20260928 (25.4s).

Despite eliminating churn (nine misses, nine entries,7379548 bytes), the strip
candidate measured87.1ms render p95 and200ms interval p95 versus fresh control
62.5ms and149.9ms. Their interval p99 values were233.4ms and200ms. Single
sequential smokes do not quantify a reliable regression, but give no reason to
retain this candidate or run longer captures. Production remains unchanged.
Public evidence: tests/performance/baselines/shared-origin-smoke-2026-09-28.json.
Raw strip/control captures: test-results/shared-idle-strip-{smoke,control-smoke}-20260928.
The exact injected renderer hash is included in each candidate report.
