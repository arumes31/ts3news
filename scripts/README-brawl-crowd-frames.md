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
return values and thrown errors; two focused Node tests verify those invariants.
It changes instrumentation overhead and cannot prove causality or GPU timing.
Image labels are static asset filenames or canvas dimensions; it retains no
image references, per-frame event arrays, query strings or player records.

The September28 short diagnostic recorded61 save calls >=8ms, all immediately
after rift_props.png draws, totaling3223.7ms across183 save calls in that group.
Maximum was87.3ms. Saves following creature images stayed below2.3ms. This
isolates a prop-associated state boundary for the next experiment, not proof
that save itself is expensive. No production drawing behavior changed.
Raw local evidence: test-results/crowd-canvas-cost-20260928.
