# Brawl minimum-device performance budget

These are engineering targets for the implementation, not measured support claims
or release approval. Keep failed and unmeasured gates visible. Do not relax a
threshold to make a candidate pass without recording the rationale and review.

## Reference profiles

The physical minimum-device target is an Intel Core i5-8250U laptop with Intel
UHD 620 graphics, 8 GB RAM and SSD storage, plugged in with battery saver off.
Use a supported OS/browser combination on that hardware; record the exact OS,
Chromium version, power mode, display refresh rate and available memory. Fix the
viewport at 1280x900 CSS pixels, DPR 1, and Brawl's Lower power preset (30 FPS).
Keep the page foreground, disable extensions and close unrelated active workloads.
This profile defines a target to test; it is not a claim that this device was used.

For repeatable development comparisons, use the same Chromium build and machine,
1280x900/DPR 1 and CDP CPU slowdown 4. Record the unthrottled host CPU and RAM.
CPU slowdown is a comparison tool, not an equivalence to the physical laptop or
a substitute for testing its GPU. Run a separate mobile layout check at 390x844;
that viewport alone does not establish mobile performance support.

Use three independent runs per scenario. Warm artwork and wait 5 seconds before
frame sampling. Record each run; every run must meet the specified threshold.
Cold readiness has its own fresh-cache procedure below. Exclude hidden-tab time,
not slow foreground frames. Do not discard a slow run without a documented cause.

## Gates

| Workload | Measurement | Pass threshold |
| --- | --- | --- |
| Cold startup | Start enabled from navigation; fresh context/cache, 150ms latency, 200000 B/s down, 93750 B/s up | Median of 3 runs <=20s; every run <=25s; transferred bytes at readiness <=3000000 |
| Warm startup | Same page and build with cached art, same network profile | Every run <=3s to enabled Start |
| Ordinary combat | Three-tier mission 1 with normal inputs, Lower power preset | Completed-frame interval p95 <=50ms and p99 <=100ms; synchronous render p95 <=16ms |
| Crowded rendering | Seeded visual scene, crowd 120, Lower power preset, 60s foreground sampling after warmup | Same frame thresholds; no canvas/context failure |
| Crowded boss combat | Mission 100 final tier with real combat inputs, 60s sample or full encounter if shorter | Same frame thresholds; report actual enemy/projectile peaks and sample duration |
| Input confirmation | At least 50 accepted movement/action inputs under the startup network profile | Recognition-to-authoritative-response p95 <=300ms; maximum <=1000ms; no lost discrete actions |
| Restart memory | 20 expedition start/exit cycles, art warmed before baseline | Retained JS heap growth <=10MiB after consistent GC/settling; final 5 checkpoints show no monotonic retained-object growth |
| Session memory | 30 minutes of repeated tier/mission play after warmup | Same <=10MiB retained JS heap growth; no renderer/audio failure; report browser process memory separately |

The fixed 960x540 backing canvas caps pixel count at 518400 regardless of viewport
or DPR. This limit is already regression-tested but does not itself satisfy any
frame or memory gate. Frame intervals include intentional 30 FPS limiting. Idle
preview runs at 15 FPS in Lower power mode and is excluded from combat thresholds.

## Measurement and evidence

Use the [cold-start command](../../scripts/README-brawl-cold-start.md) for fresh
startup measurements and retain its JSON. It records three fresh contexts with
cache disabled; warm startup needs a separate run that preserves cache.

Use the [crowded fixture](../../tests/e2e/README-brawl-fixtures.md):
`/abyss/rift?scenario=visual&seed=crowded-v1&crowd=120&riftFrameDebug=1`.
Record the fixed camera position and selected class. Keep this synthetic scene
paused; it measures drawing cost, not encounter balance or full gameplay cost.
Use normal resumed encounters for the ordinary/boss combat gates.

The [frame overlay](README-cold-start-baseline.md#in-session-frame-diagnostics)
provides a rolling 120-frame diagnostic window. Its displayed p95 alone is not a
60-second gate result. Capture a browser performance trace for the entire sample
or collect completed-frame samples without gaps into a measurement artifact, then
compute nearest-rank p95/p99 over all frames. Preserve counts and raw durations.
Render duration covers JS canvas submission, not GPU/compositor completion.
Use browser frame/presentation traces to investigate GPU stalls and dropped frames.

For input timing, correlate a recognized input with the response carrying its
submitted action; generic request latency is insufficient. Record rejected or
unconfirmed actions separately. A frame callback or local key highlight is not
server confirmation.

For memory, capture baseline and checkpoints with the same GC protocol through
DevTools; record used JS heap and retained-object counts by type. Sample every 5
restarts or 5 minutes. Retain heap snapshots locally and inspect for actor, DOM,
effect, listener, timer and audio-node retention. Process memory also contains
image/GPU caches and cannot be equated to JS heap. Browser reload is not a valid
substitute for repeated in-page expedition restarts. Use synthetic fixture data.

Every report must include candidate commit and actual server build, scenario URL,
seed/class, device/browser/profile, settings, commands, timestamps, sample counts,
raw artifact paths, summaries, errors and gate result (pass/fail/unmeasured).
Do not include credentials, real saves or private player records. Follow the
[release checklist](README-release.md) for the broader release decision.

## Current evidence

The [constrained-network baseline](README-cold-start-baseline.md) measured median
cold readiness 230.403s and 45949975 transferred bytes at its recorded commit. It
fails both the 20s target and 3MB byte budget. Explicit image priorities do not
establish a new pass; repeat the measurement after reducing startup bytes.

Frame-scheduling, bounded-canvas and cache-retirement regressions prove their
specific invariants. The remaining physical-device, full-window frame, input and
long-session/restart memory gates are unmeasured against this budget. Do not mark
them passed from the existence of the overlay or seeded fixture.

## Adaptive decoration setting

The optional display checkbox `Reduce background particles during slow frames`
scales ambient particle density only. Six consecutive rendered-frame intervals
above1.5 times the selected frame budget halve the scale, to a minimum0.25. Each
120 consecutive healthy intervals restores0.25, up to the selected density.
Disabling the option restores full selected density immediately; reduced motion
and particle-off settings retain priority. Visibility loss clears streak counters.
Combat effects, telegraphs, projectiles and authoritative simulation are unaffected.
Record whether this setting was enabled in performance reports; its existence
does not establish that a device meets the frame budget. It defaults to off.


## Projectile candidate microbenchmark

Run `go test ./internal/rift -run '^$' -bench BenchmarkProjectileCandidates -benchmem -count=2`.
On Windows amd64, Xeon Gold6126@2.60GHz, the September24 development run measured
40 friendly queries per pass, including rebuilding16 horizontal buckets:

| Enemies | Previous ordered scan | Reused buckets |
| --- | --- | --- |
|12|5.16–5.23µs|1.89–1.95µs|
|120|57.31–57.73µs|2.95–2.99µs|
|400|192.06–193.40µs|7.03–7.05µs|

Both paths report zero allocations per warmed pass. The fixture mixes targets
near the last grid row and misses; it measures candidate lookup only, not a full
tick, network latency or rendering. It does not satisfy the physical-device gates.
Buckets retain indices only and rebuild lazily after enemy movement on the first
friendly collision query. Queries preserve enemy-list priority, check live health
and use the skill's exact strict hitbox extents. Projectile hits currently change
health/recoil but not enemy coordinates; future hit-time movement or spawning must
refresh the index before another projectile query. Storage grows to the largest
enemy list encountered by that Run and is excluded from saves and snapshots.
