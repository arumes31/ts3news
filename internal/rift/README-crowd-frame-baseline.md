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
