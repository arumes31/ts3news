# Mission100 boss frame baseline — September28,2026

The three independent authored final-tier captures all fail the development
frame budget. They completed their whole encounters before the60-second maximum.
These are measured failures, not release approval or physical-device evidence.

| Run | Duration | Frames | Interval p95 | Interval p99 | Render p95 | Enemy/projectile peak |
| --- | --- | --- | --- | --- | --- | --- |
|1|22.220s|347|133.3ms|216.7ms|93.3ms|3/0|
|2|23.783s|343|166.6ms|250.0ms|100.7ms|3/1|
|3|24.632s|343|150.1ms|266.7ms|102.0ms|3/0|

Limits remain50ms/100ms/16ms respectively. All three encounters cleared; all had
zero console/page/HTTP errors, no context loss and no hidden-tab time. Raw interval
sums cover22.182s/23.766s/24.599s; only first/last sampling boundaries differ from
elapsed duration. Each capture contains the entire frame history, not a rolling
120-frame summary. The Playwright command passed3 collection tests in2.1minutes;
that does not mean the performance gate passed.

Host: Windows10.0.26200, Xeon Gold6126@2.60GHz,16logical CPUs,64GiB RAM.
Chromium153.0.8010.12, headless,1280x900/DPR1, Lower power30FPS, CDP CPU slowdown4.
ANGLE SwiftShader/SwANGLE5.0.0 reports software canvas/compositing/rasterization.
This cannot substitute for the physical i5-8250U/UHD620/8GB reference. No GPU
presentation trace was collected; render cost is synchronous canvas submission.

Candidate server: fe08cd8e plus the preserved local combat/monsters/stats changes;
report hashes identify that tracked diff, retained locally beside each report.
The fixture uses the production simulation, a synthetic bloodblade build and
seed boss-frames-v1 (replay seed3762941385710976), fixed content time, mission100
tier3. It had Mournroot Prime plus two enemies. Nothing inflated HP or population.
This is the budget's authored boss workload, not a120-enemy crowded stress scene.
The broader crowded-boss improvement remains open pending that additional scope.

[Reproduction command](../../scripts/README-brawl-boss-frames.md).
[Raw fixture-only reports](../../tests/performance/baselines/boss-frames-2026-09-28.json).
Local capture directories: test-results/boss-frames-full; smoke is separately under
boss-frames-smoke and excluded from all results above. Next investigate the slow
submission cost with profiling; do not reduce thresholds or discard slow runs.

## Follow-up CPU profile

One separately instrumented encounter completed in22.602s,342frames, with166.5ms
interval p95/233.3ms p99 and100.1ms render p95. It is not a new gate result.
The entire profiler window sampled24.327s including activation/teardown. Render
inclusive samples total11.625s; three native restore nodes account for8.537s.
Their calling paths are sprite(4.439s), fx(2.271s), catalogActor(1.826s).
The renderer's actor path totals7.107s inclusive. Background drawImage is not the
dominant sampled path. Native attribution can include deferred raster work, so
this identifies a place to experiment, not proof that removing restore is safe
or sufficient. Next compare state handling or bounded sprite caching while
verifying pixels and repeating the original uninstrumented three-run protocol.

[CPU summary](../../tests/performance/baselines/boss-cpu-2026-09-28.json).
Full local profile is under test-results/boss-cpu-profile. Collection passed in
49.9s; no production rendering change or performance improvement is claimed.

## Rejected explicit-state experiment

Replacing save/restore in sprite/catalog/effect drawing initially produced equal
PNG hashes, but explicit matrix restoration changed stored transform precision
under rotated parents. The exact-state comparison rejected that variant.

A narrower fx-only candidate restored just globalAlpha. Exact pixel/state tests
passed(30.1s) with parent opacity, rotation, clipping, brightness and flipped
sprites. Three uninstrumented captures then yielded:

| Run | Duration | Frames | Interval p95 | Interval p99 | Render p95 |
| --- | --- | --- | --- | --- | --- |
|1|24.867s|357|166.7ms|216.7ms|109.2ms|
|2|22.080s|345|166.6ms|216.6ms|99.7ms|
|3|23.254s|343|149.9ms|233.3ms|96.3ms|

All cleared with three enemies, no errors/hidden time/context loss, and all missed
the numeric gates. Median interval p95 was166.6ms versus150.1ms in the original
baseline; median render p95 was99.7ms versus100.7ms. This is not a demonstrated
overall improvement, so the production renderer was restored exactly. A native
restore sample can account for queued drawing work; replacing the state operation
did not remove the underlying expense. No performance gain or gate pass claimed.

[Candidate diff and raw captures](../../tests/performance/baselines/effect-alpha-2026-09-28.json).
[Archived visual comparison harness](../../tests/performance/baselines/effect-alpha-equivalence-2026-09-28.txt)
compares a candidate fx implementation with its save/restore reference; apply the
recorded candidate first to repeat this experiment. It is not an active no-op test
of the restored renderer. The original passing comparison used the whole renderer
at51ac6875 as its reference; the archived harness isolates the identical old fx
body to avoid requiring a historical Git object. Local captures remain under
canvas-state-equivalence(rejected matrix),canvas-alpha-equivalence(passing narrow
variant),and boss-frames-alpha(three measurements). Next investigate bounded
sprite/effect raster reuse rather than assuming save/restore itself is the cause.

## Retained integer atlas-frame cache

The next candidate caches original-resolution integer atlas rectangles in bounded
source canvases. Fractional source rectangles bypass it because the visual audit
found one/two changed edge pixels when reoriginating those crops. Exact state and
pixels then passed for transformed sprites and every hero/effect atlas cell.

| Run | Duration | Frames | Interval p95 | Interval p99 | Render p95 |
| --- | --- | --- | --- | --- | --- | --- |
|1|21.532s|341|133.3ms|233.3ms|14.2ms|
|2|21.834s|344|133.4ms|183.3ms|12.4ms|
|3|21.510s|346|149.9ms|216.7ms|13.0ms|

Every run cleared with three enemies, zero projectile peak and no reported errors.
The three synchronous-render thresholds pass; the overall frame gate still FAILS.
Median render p95 fell from100.7ms to13.0ms(about87%) on the same software-rendered
development profile. Do not equate this to an87% FPS improvement or physical-device
support. The cache adds up to8MiB of calculated source-pixel backing,64entries,
with dimension reset on eviction. Total process/GPU memory has not been measured.

[Implementation and verification scope](README-atlas-frame-cache.md).
[Raw three-run captures](../../tests/performance/baselines/boss-atlas-cache-2026-09-28.json).
Collection passed in1.9minutes. Local pixel/eviction and atlas integration evidence
is under atlas-cache-final,atlas-cache-regressions and atlas-cache-bounds-final.
The exhaustive actor/scene probes now prepare both lazy hero sheets before direct
render calls; unchanged coverage assertions pass for124monsters/32rigs and300scenes.
