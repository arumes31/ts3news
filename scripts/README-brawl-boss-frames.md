# Crowded boss frame capture

Run on an unused loopback port:

```powershell
$env:ABYSS_E2E_PORT='18098'
node node_modules/@playwright/test/cli.js test --config=playwright.boss-performance.config.js
```

The dedicated configuration builds a fresh fixture, uses three separate Chromium
contexts at1280x900/DPR1 with CPU slowdown4 and the Lower power preset, and enters
mission100 tier3 with the synthetic bloodblade build. It does not alter enemy
counts, health or combat damage. Artwork is ready and warmed for five seconds
before sampling. Keyboard movement/jump/attacks/class skills drive actual combat.
Each capture lasts sixty seconds or the complete encounter if it ends earlier.

The opt-in frame diagnostic array is observed in the test to retain every sample
before its120-entry rolling truncation. No production instrumentation is added.
Reports include raw frame intervals/render costs, nearest-rank p95/p99, snapshot
and enemy/projectile peaks, actual duration, result, settings, host and browser.
A passing Playwright run confirms collection; read each JSON gate field for the
performance verdict. Defeat and short encounters must remain explicit in review.

`BRAWL_FRAME_SMOKE=1` limits this to one five-second capture and cannot pass a gate.
Headless CDP slowdown is a development comparison, not physical i5/UHD620 evidence.
Synchronous render time excludes GPU completion. Frame traces may be needed to
investigate presentation stalls. Raw reports use fixture data only and stay under
ignored test-results. Preserve failed/slow runs; do not discard them to pass.

## CPU investigation

Set `BRAWL_FRAME_PROFILE=1` to sample one complete encounter with CDP's CPU
profiler. This writes boss.cpuprofile beside the report. The JSON gate is marked
unmeasured because profiling instrumentation can alter timings. Remove the
variable before collecting normal three-run comparisons.

Run `python scripts/analyze-brawl-cpu-profile.py <path-to-boss.cpuprofile>` to
produce weighted self/inclusive timings. They describe sampled main-thread time,
not GPU presentation. Native canvas calls may include queued rasterization work;
large restore timings do not alone prove that state restoration is the cause.
