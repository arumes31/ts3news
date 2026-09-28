# Crowd120 frame baseline — September 28, 2026

Three independent 60-second paused crowd captures fail the development frame
budget. The measurement harness completed successfully (3 tests, 4.1 minutes);
that is not a performance pass.

| Run | Duration | Frames | Interval p95 | Interval p99 | Render p95 |
| --- | --- | --- | --- | --- | --- |
| 1 | 60.090s | 605 | 133.4ms | 166.7ms | 97.8ms |
| 2 | 60.071s | 629 | 133.3ms | 150.0ms | 94.6ms |
| 3 | 60.129s | 595 | 133.4ms | 166.7ms | 100.3ms |

Limits remain 50ms / 100ms / 16ms. Raw interval sums cover 59.981s, 60.014s and
60.031s; the first/last sample boundaries account for the remaining elapsed time.
All samples are retained, beyond the diagnostic overlay's rolling 120 frames.
All runs had no console/page/HTTP errors, hidden time, context loss or camera drift.

Each scene contained 120 living enemies, including 14 bosses, and no projectiles.
The vanguard fixture used seed crowded-v1, replay seed 3626808109281548, clock 0,
normal camera x=0 and normal culling. This population is not a claim that all 120
sprites were visible. The simulation stayed paused throughout; live AI, attacks,
input and crowded boss combat are outside this capture's scope. Item 0997 remains
open for that broader encounter measurement.

The retained integer atlas cache held four entries and 267288 calculated pixel
bytes in every run, with four misses and 17346–18221 hits. It was not churning at
its 8MiB limit. Fractional crops continue using the original atlas. This evidence
motivates investigating those draws; it does not prove they caused the slowdown.

Host: Windows 10.0.26200, Xeon Gold 6126 @ 2.60GHz, 16 logical CPUs, 64GiB RAM.
Chromium 153.0.8010.12, headless, 1280x900/DPR1, Lower power 30FPS, CPU slowdown 4.
ANGLE SwiftShader/SwANGLE reports software canvas/compositing/rasterization.
These results do not substitute for the physical i5/UHD620 target. Render duration
measures synchronous canvas submission; no GPU presentation trace was captured.

Candidate server: 5d67dcd9 plus the preserved combat/monsters/stats working changes.
All reports record the same tracked-diff hash, with its patch retained locally.
The fixture contains synthetic data only. No user server or player save was used.

[Reproduction command](../../scripts/README-brawl-crowd-frames.md).
[Raw reports](../../tests/performance/baselines/crowd-frames-2026-09-28.json).
Local artifacts are in test-results/crowd-frames-full. The separate 10-second smoke
capture is excluded from these results. No runtime optimization is claimed here.

## Profile and rejected cache follow-ups

A separately instrumented 61.491s capture sampled 61.397s of CPU time. The render
path accounts for 44.792s inclusive; 22.201s is attributed to native restore in
the depth-sorted scenery callback, compared with 0.115s under catalogActor.
Native samples may include deferred drawing work. This directs investigation to
scenery, rather than proving that restore itself is expensive. The instrumented
frame timings are not a gate result.

[CPU summary and capture](../../tests/performance/baselines/crowd-cpu-2026-09-28.json).
The full local profile is under test-results/crowd-cpu-profile. The reproduction
command now supports BRAWL_FRAME_PROFILE=1 and explicitly excludes profiling
captures from gate verdicts.

Three candidate approaches were rejected:

- Full-width rows for fractional crops still changed one pixel in a flipped,
  rotated actor comparison. The archived experiment covers all 32 shared rigs;
  restore it to tests/e2e/rift-row-cache-experiment.spec.js to reproduce. It stays
  outside the active suite because its candidate deliberately fails equality.
- Routing terrain-cover draws through the existing cache created no new entries
  in this scene. Its three render p95 results were 94.6/111.2/95.3ms; interval p95
  was 133.3/150.0/133.3ms. It did not address the measured scenery workload.
- Caching the entire prop atlas preserved exact pixels and passed 12 browser
  checks, but added 6294152 calculated pixel bytes without a frame improvement.
  Render p95 was 98.8/100.0/97.3ms, interval p95 133.4/133.4/133.3ms, and p99
  150.0/166.6/150.1ms. All three 60-second runs failed the unchanged thresholds.
  Five entries occupied 6561440 bytes. No churn occurred in this scene.

A final smaller prop-only fractional crop failed exact pixels: six changed pixels
in one transformed panel and eleven in its flipped panel. It was not benchmarked.
The runtime renderer is restored to the retained integer-only cache. No new
performance improvement or memory-cost increase is shipped by these experiments.

[Rejected terrain measurements](../../tests/performance/baselines/crowd-terrain-rejected-2026-09-28.json),
[whole-prop candidate and measurements](../../tests/performance/baselines/crowd-whole-prop-rejected-2026-09-28.json),
[prop-crop candidate and pixel differences](../../tests/performance/baselines/prop-crop-rejected-2026-09-28.json),
[archived row experiment](../../tests/performance/baselines/row-cache-equivalence-2026-09-28.txt).

The active equality test now includes all eight prop cells under clipping,
rotation, opacity, brightness and flips. Fade/culling probes follow atlas identity
through source canvases, preserving their original behavior assertions. A disk-full
Go linker failure interrupted the first prop-crop test before it could run;
clearing the reproducible Go build cache allowed the actual comparison to finish.
No source, player data or measurement artifacts were removed.

## Current profile after shelter verification

A new instrumented capture at1aa78c5b lasted61.471s with405 rendered frames.
Of61.462s sampled CPU time,22.512s self time was attributed to native save under
actor; actor's inclusive time was38.281s and catalogActor's8.167s. Earlier native
restore cost appeared under the scenery callback. The dominant attribution thus
moved after the retained cover-alpha change. Canvas may charge deferred drawing
to a subsequent state operation; this is not evidence that removing actor saves
would remove22.5s of rendering work. Next isolate the preceding image draws and
state boundaries with explicitly diagnostic timing before changing more rendering.

Only four cache entries /267288 calculated bytes were retained (12571hits,
4misses). There was no cache churn. Instrumented interval p95/p99 were216.7/266.7ms,
render p95 was167.9ms. These are diagnostic observations, not a new gate result.
The local raw profile is in test-results/crowd-profile-after-shelters-20260928;
its compact aggregate is tests/performance/baselines/crowd-after-shelters-profile-2026-09-28.json.
No performance improvement or physical-device result is claimed by this capture.


## Origin-preserving prop cache (2026-09-28)

The retained candidate copies a bounded prefix of the prop atlas from (0,0), so
fractional source coordinates remain unchanged. Other fractional crops stay native.
Prefixes over2MiB fall back to native drawing; the existing8MiB/64-entry LRU and
zero-sized evicted canvases remain. All eight prop cells passed exact pixel checks
under transforms, clips, opacity, brightness and flips, alongside existing hero
and effect panels. A new fallback/origin regression failed before the change and
passed on the production path; the same check verifies eviction and memory bounds.

Three60-second candidate samples followed by three fresh controls used commit
9d7fd9b9 and the same user simulation diff, Chrome153,1280x900/DPR1,CPU4x,Lower power.
The candidate source hash is recorded in each public report. Production applies
the same two guard/origin changes; only comments differ from the injected source.

| p95/p99 measurement (ms) | Candidate samples | Control samples |
| --- | --- | --- |
| Synchronous render p95 | 50.6 / 47.1 / 48.6 | 98.6 / 94.0 / 91.5 |
| Completed-frame interval p95 | 133.3 / 116.7 / 133.3 | 133.4 / 133.3 / 133.2 |
| Completed-frame interval p99 | 150.0 / 133.4 / 150.0 | 150.1 / 150.0 / 150.1 |

Median render p95 fell from94.0 to48.6ms (48.3%); all candidate render samples
beat all controls. Median frame intervals did not materially improve. Keep this
as a synchronous drawing-cost improvement, not a frame-stability or release pass.
All six samples still fail the unchanged50/100/16ms budgets. Grouped sequential
order is a limitation; physical-device and GPU presentation evidence remain absent.

Cache occupancy increased from4 entries/267288 bytes to5/1059388, without churn.
A separate instrumented smoke saw save calls following the445x445 prop prefix at
0.3ms maximum with zero calls>=8ms, but its frame metrics are not gate evidence.
The temporary injection switch was removed when applying the production change.

Public aggregates: tests/performance/baselines/prop-origin-crowd-2026-09-28.json
and prop-origin-control-2026-09-28.json. Raw outputs remain in corresponding
 test-results/prop-origin-*-full-20260928 directories. Pixel/state/bounds artifacts
are in prop-origin-production-20260928; the expected pre-change failure is in
prop-origin-regression-red-20260928. No real player data is included.

Nine production cover fading/culling checks passed in31.0s; artifacts are in
test-results/prop-origin-cover-regressions-20260928.
