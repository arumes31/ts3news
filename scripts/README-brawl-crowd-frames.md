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
