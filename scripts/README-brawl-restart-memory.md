# Repeated expedition memory capture

Run the opt-in benchmark against a fresh isolated synthetic fixture:

```powershell
$env:ABYSS_E2E_PORT='18098'
Remove-Item Env:BRAWL_MEMORY_SMOKE -ErrorAction SilentlyContinue
node node_modules/@playwright/test/cli.js test --config=playwright.memory.config.js
```

This runs three independent browser contexts, with 20 in-page expedition starts,
actual first-room fights and checkpoint exits each. It warms one full expedition
before baseline. No page reload or fixture reset substitutes for a restart.
The player is bloodblade, mission 1, with ordinary generated encounter seeds.
The controller uses WASD, Space and the normal class/basic attack keys. A death,
stalled fight, duplicated run ID or browser error leaves the run incomplete.

Set `BRAWL_MEMORY_SMOKE=1` for one measured cycle after warmup. Smoke evidence
never satisfies the 20-cycle/three-run gate. The managed server refuses reuse
of an existing server; keep the user app on its own port. Allow up to 130 minutes
for the complete three-run command. Do not run alongside another fixture process
on the same port. Neither the fixture nor browser touches production saves.

Reports and full heap snapshots stay under ignored `test-results/restart-memory`.
Reports record host CPU/RAM, browser, checkout revision and tracked diff hash,
settings, run IDs, timings, errors, every-cycle GC heap/DOM counters and reachable
node/type counts from snapshots at baseline and every fifth restart. The server
is freshly built from this checkout; a dirty hash identifies changes beyond the
commit. Keep source unchanged during measurement and retain the exact source diff
locally when comparing builds. Traces are disabled to avoid measurement overhead.

Each checkpoint settles 2 seconds, forces GC, settles 1 second and forces GC
again before sampling. Snapshot collection follows sampling. The 10 MiB gate
checks final versus warmed baseline. Passing that numeric check is not a complete
memory pass: inspect the final five checkpoints and snapshot retaining paths for
actor, effect, DOM, listener, timer and audio-node growth. Reachable node counts
are not dominator retained sizes and constructor names alone cannot identify
actor/effect objects. Record that review before marking the gate passed.

This uses headless Chromium at 1280x900/DPR1, Lower power and CPU slowdown4. It is
a development comparison, not physical minimum-device evidence. Browser process
memory and the separate 30-minute session gate are not measured by this command.

Analyze saved captures with `python scripts/analyze-brawl-restart-memory.py <output-directory>`.
This writes reachable payload counts and direct run retainers, checks retained run
IDs against the current expedition, and leaves the leak verdict to review.
See [the initial measurement](../internal/rift/README-restart-memory-baseline.md)
for results and the still-unmet object plateau criterion.
