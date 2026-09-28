# Brawl cached-startup measurement

Run from the repository root:

```powershell
node node_modules/@playwright/test/cli.js test --config=playwright.warm-performance.config.js
```

The configuration starts and stops an isolated synthetic Go fixture on port 18098.
It refuses to reuse an existing server. The development server on 18089 is not used.
Three tests each receive a fresh browser context. Each warms the same Brawl page
and character build without network throttling, selects the Lower power preset,
waits for outstanding requests, navigates to about:blank, and navigates back with
HTTP cache preserved. No request interception is used because it can disable cache.

Measured navigation uses 1280x900, DPR 1, CPU slowdown 4, 150ms latency,
200000 B/s download and 93750 B/s upload. Service workers are blocked. A document
mutation observer records performance.now() when Start becomes enabled and the
base atlas counter is ready. This includes the initial API response and selected
character-art readiness; it is not time to first combat input. Every measured
PNG must have zero Resource Timing transfer bytes and nonzero encoded body bytes.
This validates cache use rather than inferring it from a fast timing.

Reports in test-results/warm-start include navigation/resource timings, cached
image counts, transferred bytes, host/browser, source revision and tracked diff
hash, fixture provenance, settings, errors and gate result. Failed measurements
are retained. The harness passing means the capture was valid; read each report's
gate field to assess the <=3000ms target. All three must pass. CPU emulation and
headless Chromium are development comparisons, not physical minimum-device proof.

The warmup is deliberately unthrottled: it populates the cache, not a cold-start
sample. This test does not replace the separately constrained cold-start budget,
frame-time checks, input-latency checks, memory checks or physical-device test.

## September 28 development result

At candidate `09cd719c`, with the three existing uncommitted engine changes
identified by the report's tracked-diff hash, all three valid captures passed:

| Sample | Enabled Start | Transferred bytes | Cached PNGs |
| --- | ---: | ---: | ---: |
| 1 | 1712.9 ms | 98629 | 20 |
| 2 | 2046.5 ms | 98629 | 20 |
| 3 | 1740.3 ms | 98629 | 20 |

No page or request errors were recorded. The harness completed in 37.8 seconds.
Host: Windows 10.0.26200, Xeon Gold 6126 at 2.60 GHz, 16 logical CPUs, 64 GiB RAM;
Chromium 153.0.8010.12 headless with the development profile above. This establishes
only the cached-startup development gate. Cold startup remains failed and the
physical minimum-device test remains unmeasured.

Raw reports: [warm-start-2026-09-28.json](../../tests/performance/baselines/warm-start-2026-09-28.json).
Full local captures and exact tracked source patches: `test-results/warm-start/`.
An initial configuration mismatch started the fixture on its default port 18082
while the runner waited for 18098; it produced no measurements. The committed
configuration explicitly passes the port to the fixture. That unused first
fixture was terminated during the corrected capture; no measured run was discarded.
