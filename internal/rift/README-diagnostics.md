# Brawl diagnostic export

Battlefield settings offer a diagnostic preview followed by a JSON download.
Preparing is local and does not fetch or mutate a run. Download contains exactly
that preview, even if gameplay changes before the download. Re-prepare to refresh.
When downloading fails, the read-only preview remains available to select/copy.
No report is uploaded or automatically sent to anyone.

`rift_diagnostics.js` builds the versioned format from explicit scalar allowlists.
It never copies a run, build, error, browser storage, URL, log, user agent, account,
character, gear, rewards or request identifiers. Its update hook retains only
schema, known expedition status, numeric mission/tier and paused state. The report
labels these as the last confirmed snapshot, which can be stale after a failure.
Without a confirmed snapshot that section is null; export still works.

Viewport, renderer settings and audio mix values have bounds/type checks. Missing
or invalid fields are null. Frame/input debug sources contribute only sample count,
mean, p95 and maximum from at most 120 recent valid numeric samples; payload debug
contributes only maximum response bytes. Unknown properties and raw samples are
never serialized. Sampling remains opt-in through existing diagnostic flags.
These short summaries do not prove full-session performance or memory budgets.
No game state or storage is added by preparing/downloading a report.

Browser tests cover exact preview/download equality, private-string injection,
unknown values, bounded aggregation, live frame samples, unchanged server state,
no mutation requests, blocked storage/initial read failure, download failure and
mobile layout. Keep the allowlist boundary when adding future report fields.
