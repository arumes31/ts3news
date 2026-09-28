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
