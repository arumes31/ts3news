# Restart memory baseline — September 27, 2026

The three 20-restart captures completed. The <=10 MiB retained JS heap growth
limit passes in every sample. The separate no-monotonic-object-growth criterion
is **not met** by this fresh-history workload: attempt records are still being
added. This is measurement evidence, not an overall performance pass or a claim
that the application is leak-free.

## Build, machine and workload

Candidate/server source: `0b7a81e3` plus the preserved working-tree engine diff,
SHA-256 `4d310e18ec0adccc747f4fab987ebc2c408b1903f147d2a643d01fbe4d78ff36`.
Each sample retains `server-tracked-diff.patch`. The managed fixture was built
fresh, with no server reuse. No production saves or user app process were used.

Windows10.0.26200, Intel Xeon Gold6126@2.60GHz,16 logical CPUs,68718370816 bytes
RAM; Chromium153.0.8010.12, headless,1280x900/DPR1, CPU slowdown4, Lower power
preset30FPS, adaptive particles off, default audio. This is a development host,
not the physical minimum-device target. Browser process/GPU memory was not sampled.

Each context loaded `/abyss/rift?subclass=bloodblade` once, fought mission1's first
tier using normal keyboard inputs, and exited at its cleared checkpoint. One full
expedition and five seconds of settling warmed artwork/audio before baseline.
Then20 distinct expeditions completed on the same page, without reloading,
resetting fixture state, or directly modifying combat. Seeds and run IDs are in
each report. Three contexts ran sequentially:63 real fights including warmups,
60 measured restarts,63 GC checkpoints,15 complete heap snapshots.

GC protocol at every checkpoint: settle2s, collect garbage, settle1s, collect
again, read JS heap and DOM counters. Full snapshots follow baseline and cycles
5/10/15/20. Capture ran21:04:19–21:19:56 UTC; Playwright finished3 tests in15.9min.
No captured page, console or HTTP errors occurred. Failed-request events were not
separately collected; this does not assert that every network request succeeded.

## Results

| Sample | Baseline bytes | After20 bytes | Growth bytes | Growth MiB |
| --- | ---: | ---: | ---: | ---: |
|1|6854500|7734224|879724|0.839|
|2|6890432|7846124|955692|0.911|
|3|6805388|7712844|907456|0.865|

All15 snapshots contain152 actor-shaped payloads, two run-shaped payloads and
zero effect-shaped payloads. Both retained run IDs match the current expedition
at every snapshot. Direct context retainers include the game's current run,
renderer `snapshot` and renderer `previous`; historical expedition run graphs
were not found by this check. Empty reusable effect shells are intentionally
pooled with a40-object limit and are not counted as live effect payloads.

Detached nodes remain20 in all15 snapshots. Event listener count is616 at every
full snapshot except sample1/cycle15, which temporarily has618; it returns to616.
Audio nodes normally total11 GainNode,4 OscillatorNode,2 AudioBuffer and2
AudioBufferSourceNode objects. The same checkpoint temporarily adds two gains and
two oscillators, returning to baseline at cycle20. DOM timer coordinator count
stays1. These checks find no accumulating actor, effect, listener or audio graph
in the measured workload; they do not prove other workloads behave identically.

Attempt-shaped payload counts are1/11/21/31/41 at checkpoints0/5/10/15/20 in every
sample: current and previous snapshots retain their respective attempt histories.
The engine caps history at50 attempts, but these captures stop before that cap.
The records UI renders a row for every retained attempt. Therefore growing
record objects and DOM rows must not be presented as a demonstrated plateau.
A further capture must exercise the full history before claiming steady state.

Other growth includes inspector-owned network records. In sample1 they rise from
87 to770. Incoming references lead through the backing map to
`blink::NetworkResourcesData` and `blink::InspectorNetworkAgent`. Browser timing
entries and compiled-function bookkeeping also change. These are included in the
raw evidence; no subtraction or relaxed threshold was used to manufacture a pass.

## Reproduce and inspect

```powershell
$env:ABYSS_E2E_PORT='18098'
Remove-Item Env:BRAWL_MEMORY_SMOKE -ErrorAction SilentlyContinue
node node_modules/@playwright/test/cli.js test --config=playwright.memory.config.js --output=test-results/restart-memory-full
python scripts/analyze-brawl-restart-memory.py test-results/restart-memory-full
node --test tests/performance/heap-summary.test.cjs
```

Each `rift-restart-memory-restart-memory-sample-N` directory retains its JSON
report, source diff and five `.heapsnapshot` files. `retention-review.json` records
payload counts, current-run identity checks and immediate retaining references.
Keep the raw heaps local; use Chromium's heap snapshot viewer for deeper paths.
The analyzer identifies actors by `hp/max_hp/kind`, effects by `x/y/started/kind`,
runs by `player/enemies/status`, and attempts by
`mission/outcome/at_ms/splits/hp/class`. These are reachable shape counts, not
constructor guarantees or dominator retained sizes. The separate counter test
covers prototype-like constructor names that initially corrupted summaries.

The30-minute session, full frame-window, constrained-network and physical-device
gates remain separate. Completing this measurement does not complete those gates.
