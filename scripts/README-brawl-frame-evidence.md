# Public Brawl frame evidence

Run `node scripts/brawl-frame-evidence.cjs crowd <capture-directory>` or use
`boss` for authored boss captures. The command reads direct Playwright output
subdirectories and writes `public-frame-evidence.json` beside them. Raw reports,
CPU profiles, patches and timings stay local; do not publish those by copying
an entire capture directory.

The export contains only fixed labels, validated source hashes and numeric
aggregates. It recalculates nearest-rank percentiles from raw timings against
50ms interval p95, 100ms interval p99 and 16ms render p95. Reported summaries,
threshold overrides, host names, commands, paths and error messages are not
copied. Runtime errors are counts only. It requires the recorded development
profile, complete timing-window coverage within observed frame boundary costs,
no hidden/context-lost interval, explicit non-smoke/non-profiled metadata and
three matching source identities for an aggregate development numeric pass.
Boss windows under 60 seconds require a terminal encounter outcome.

`numericStatus` describes the raw timing thresholds independently of capture
metadata completeness. Historical captures missing the profiling flag remain
incomplete even when numeric failures are visible. Smoke/profiled captures are
diagnostic only. `releaseReady` and `physicalDeviceVerified` are always false.
The exporter does not prove scenario execution or actual server-build identity;
review the accompanying source capture and scenario tests for that evidence.
It cannot independently establish that a collector omitted no frames.

Validation: `node --test tests/performance/frame-evidence.test.cjs`.
Seven tests cover private-field exclusion, recomputation, failures, malformed or
truncated samples, profile limits, short boss outcomes and source mismatches.
Re-exporting the retained crowd baseline preserves its numeric failures and
marks missing legacy profiling metadata incomplete. The atlas-cache boss
baseline reproduces all three failed frame gates despite passing render p95.
Ledger 1000 still requires the broader candidate release evidence and review.
