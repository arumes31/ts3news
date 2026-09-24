# Measure Brawl cold-start readiness

Run against an isolated Brawl fixture built from the candidate sources:

```powershell
node scripts/measure-brawl-cold-start.cjs http://127.0.0.1:18096/abyss/rift > .tmp/brawl-cold-start.json
```

Requires the project's Playwright package and Chromium installation. The example
port must be replaced with your running fixture's port. The command does not
start a server, authenticate or send combat actions.

Three sequential samples each use a fresh browser context, disabled HTTP cache,
blocked service workers, 1280x900 viewport, en-US locale and UTC time zone. Chromium
network emulation uses 150 ms latency, 200,000 bytes/second download (1.6 Mbps) and
93,750 bytes/second upload (0.75 Mbps). CPU is not throttled. These are synthetic
network conditions, not measurements from a physical mobile connection.

Readiness is measured with performance.now() from navigation time origin at the
first DOM mutation where the start button is enabled, the critical atlas counter
is ready and the button is not offering Retry loading. This includes frontend
initialization and the initial API response. It is not LCP or interaction latency.
Each sample is allowed six minutes to become ready; the limit is a diagnostic
timeout, not an acceptable performance budget. A timeout, loading failure or unexpected browser/request error leaves the report
incomplete and returns exit 1. The failure report retains the button/atlas state
and completed resource timings for diagnosis. Setup errors return exit 2.

The JSON report includes browser/platform, checkout revision (not a verified
server revision), exact network settings, sample timings, completed-at-readiness
resource counts and byte totals, the ten largest completed resources, and browser/
request errors. Transfer totals include headers as reported by Resource Timing;
encoded body totals exclude headers. Requests still in flight at readiness are
excluded. With three successful samples, min/median/max readiness is included.
Progress goes to stderr; stdout remains a single JSON report.

Record the actual fixture build revision separately. Use synthetic characters and
avoid authenticated production captures. Commit/source, browser and local-machine
conditions must be comparable when evaluating an optimization. Keep timed-out
reports as evidence; do not treat a missing readiness value as zero or a pass.

A single aborted initial GET followed by confirmed readiness is recorded as
`recoveredInitialReadTimeout: true`. Its request error stays in the sample. This
is Brawl's bounded cold-load recovery, not an error-free startup; compare its
frequency as well as readiness time when optimizing. Other errors or multiple
aborts still fail the measurement. A successful measurement is not a declaration
that the measured loading time is acceptable.
