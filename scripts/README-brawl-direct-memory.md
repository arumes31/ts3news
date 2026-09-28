# Direct-browser memory comparison driver

The direct CDP transport and Chromium launcher are groundwork for comparing
network-inspector overhead under the same synthetic Brawl gameplay workload.
They do not provide a completed memory comparison or close ledger0798.

`brawl-direct-cdp.cjs` routes commands and target-session notifications over one
WebSocket. It enables no CDP domains implicitly. Callers must explicitly enable
Runtime/Page and choose whether Network inspection belongs in their sample.
Timeout, protocol-error and disconnect handling reject pending commands; listeners
can be removed after heap snapshot streaming. Three Node tests cover routing,
error handling and timeouts.

`brawl-direct-chromium.cjs` starts the installed Playwright Chromium executable
directly, using an isolated profile under the caller's ignored output directory.
It binds remote debugging to loopback, launches headless with windows hidden,
and closes only its own browser. It does not connect to the user's open browser.
Profiles and diagnostic stderr stay local and must not enter public evidence.

The September28 launch smoke used Chrome153.0.8010.12, created a separate target,
enabled Runtime, evaluated2+2 and confirmed visible document state, then closed
the browser. No Network.enable command was sent. This proves basic transport and
launch, not campaign execution, comparable rendering or stable process memory.
Next reuse the complete three-tier keyboard navigator and checkpoint protocol
for paired Network-enabled/disabled captures with identical launch settings.
Record exceptions and HTTP failures independently in the disabled variant.

## Paired campaign harness

Run `node node_modules/@playwright/test/cli.js test --config=playwright.direct-memory.config.js`
with ABYSS_E2E_PORT=18098. BRAWL_DIRECT_MEMORY_SMOKE=1 selects two short validation
runs; omit it for at least30 measured minutes and60 complete replays per variant.
The fixture starts fresh, variants run sequentially, and each launches a fresh
isolated browser/profile. Variant1 does not enable Network; variant2 does.
Both use Runtime/Page, the same1280x900/DPR1, CPU4x, Lower power settings and the
same mission1 bloodblade keyboard strategy with the shared ledge navigator.
API reads use Node fetch with the synthetic fixture cookie, outside the page's
Network domain in both variants. Runtime/console errors, page fetch/resource
failures and API response statuses remain checked when inspection is disabled.
Do not compare absolute native memory to the earlier Playwright launch as if
launch flags or browser-process composition were identical; compare the paired
variants first, and record remaining instrumentation differences.

Every checkpoint uses2s settling, GC,1s settling, GC, process counters, heap/DOM
counters, render/audio health and a heap snapshot. Full runs explicitly capture
replays49/54/59/60, plus five-minute checkpoints. Driver sources and hashes,
server diff, launch arguments and graphics settings stay with local raw reports.
The heap collector itself remains enabled in both variants; this is a comparison
of Network inspection, not an entirely uninstrumented browser.

The initial paired smoke passed in5.1minutes, with a warmup plus two complete
three-tier replays per variant. Disabled inspection recorded0Network events at
both checkpoints; enabled inspection recorded162then388. Runtime/error checks
passed. Artifacts: test-results/direct-memory-smoke-20260928. This establishes
that both drivers can complete the workload; it does not establish a long-session
plateau or attribute all process-memory growth. Full comparison remains pending.
