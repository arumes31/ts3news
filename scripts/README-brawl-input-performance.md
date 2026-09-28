# Brawl isolated input response measurement

Run a fresh local synthetic fixture and three independent browser samples:

```powershell
$env:ABYSS_E2E_PORT='18102'
$env:GOMAXPROCS='2'
npx playwright test --config=playwright.input-performance.config.js --output=test-results/input-full
```

Set BRAWL_INPUT_SMOKE=1 for one six-input harness check. Remove it for the full
three-sample measurement. Each full sample sends60 isolated keyboard inputs:
40 WASD presses,10 jumps and10 dodges. Alternating horizontal order keeps the
character away from arena boundaries. Before a press, normal simulation must
finish jump/dodge cooldown and recovery; the harness never injects game state.
A fixed five-value delay sequence varies arrival relative to the polling loop.

The existing free-practice touch arena accepts all controls. The movement-only
practice arena intentionally removes dodge, so it cannot verify action acceptance.
This is a keyboard measurement in that arena, not a touch, crowded-combat,
rapid-input or physical-device performance claim. Those checks remain separate.

Warm artwork first, set Lower power30FPS,1280x900/DPR1 and CPU slowdown4, then
apply150ms latency,200000 B/s down and93750 B/s up. Three fresh contexts use the
same managed fixture. Direct API reads between attempts establish ready state;
no readiness read is issued while waiting for the measured action response.

Each press is matched to its POST input field and validated response run/revision.
Movement must change the corresponding coordinate in the intended direction;
jump must increment the authoritative count; dodge must start its cooldown.
The timing diagnostic starts at recognized input and ends after validation,
asset preparation and UI snapshot application. Queue and request times are
reported separately. HTTP success alone does not establish action acceptance.

Reports retain every attempted input, including missing timing or rejected
execution, plus runtime failures and visibility/context-loss state. Numeric
thresholds are p95<=300ms,maximum<=1000ms and at least50 accepted inputs with
none rejected or unconfirmed. Test completion means collection succeeded;
inspect report.status for threshold_failure. Smoke never qualifies for the gate.
The fixed sequence checks isolated discrete inputs only; it cannot establish
that bursts, simultaneous actions or repeated cooldown-limited presses are kept.

Raw input-report.json stays under the chosen ignored output directory. It uses
synthetic fixture data and stores action labels,timings,acceptance,source hash,
host/browser/profile and aggregate errors; run IDs and server response contents
are checked in memory rather than exported. Preserve source throughout a full
capture. Source changes after a result invalidate its claim for a later candidate.
