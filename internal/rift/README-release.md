# Brawl release checklist

Use this checklist for a specific candidate commit. An unchecked gate is pending,
not evidence of success. This document does not assert that the current branch
is ready to release. Record results against the exact built revision; changes
after verification invalidate the affected evidence.

## Candidate and automated gates

- [ ] Record commit, dirty-worktree status, build tool versions, operating system,
  browser version and intended environment. Identify migrations and asset changes.
- [ ] Run the campaign validator and retain its summary plus all four outputs:

```powershell
go run ./cmd/brawl-validate -out-dir ".tmp/brawl-release"
go test ./cmd/brawl-validate ./cmd/brawl-population-report ./cmd/brawl-reachability-report ./cmd/brawl-hazard-report -count=1
go test ./internal/bot -run 'TestRift|TestAbyssStaticAssetReferencesResolve|TestProductionStaticAssets' -count=1
```

- [ ] Inspect failures and report completeness, not only the command exit code.
  [Author gate scope](../../cmd/brawl-validate/README.md) excludes difficulty,
  frame stability and timed escape feasibility. Do not lower budgets to pass.
- [ ] Build the candidate's fixture on an unused local port and run the Brawl
  browser suite against that build. Do not reuse a binary from an older commit.
  [Fixture setup and reset behavior](../../tests/e2e/README-brawl-fixtures.md)
  explains external-server mode and isolated sessions.

```powershell
node node_modules/@playwright/test/cli.js test rift- --reporter=line --output=test-results/brawl-release
```

- [ ] Inspect retained traces/screenshots and browser console/network failures.
  Test-only state injection verifies a targeted rendering case; it does not prove
  that a player can reach the same state through normal controls.

## Gameplay and shared Abyss content

- [ ] Play a complete three-tier expedition using WASD, Space and attack/skill
  controls. Check collisions, airborne hits, guard, dodge, deaths and boss retry.
- [ ] Check all 100 mission definitions and three tiers per mission through the
  author reports. Exercise objective varieties, later-wave spawns, cover, hazards
  and authored escape routes; sampled geometry alone is insufficient for pursuit
  or a player's ability to escape a hazard cycle.
- [ ] Verify seamless tier and mission advancement, pause/background interruption,
  checkpoint banking, explicit exit and mission-100 completion without mission 101.
- [ ] Check all Abyss classes/subclasses and their ordered signature skills,
  resources, ultimate ownership and equipped-character snapshots.
- [ ] Check the live monster union, new-entry eligibility, bestiary and practice;
  retain removed-monster saved fights. See [monster sync](README-monster-sync.md).
- [ ] Verify spells, fighters, enemies, bosses, terrain and areas visually, with
  correctly bounded atlas frames and fallback art. Check audio for attacks,
  enemies, bosses, effects and areas, including rejected audio initialization.

## Input, accessibility and performance

- [ ] Complete keyboard-only campaign selection and touch-only checkpoint flows.
  Check focus restoration, visible focus, display zoom, narrow screens and touch
  targets without hidden essential controls.
- [ ] Complete an expedition with sound disabled and another with reduced motion;
  critical warnings and outcomes must remain understandable. Check live-region
  announcements for repeated combat spam and meters for authoritative values.
- [ ] Exercise delayed/truncated responses, conflicting tabs, rapid actions,
  interrupted banking, corrupt preferences and pause during advancement.
- [ ] Measure cold readiness on a constrained connection, crowded-boss frame
  stability and memory after repeated restarts. Record device, viewport, network
  conditions, run length and measurements against an agreed release budget.
  Bounded animation caches and population caps do not establish these metrics.
- [ ] Check every required asset on the candidate environment, including cold-cache
  loading and visible failure recovery; a warm browser cache can hide missing files.

## Saves and reward durability

- [ ] Run transaction/rollback/replay checks for banking, objectives and records.
  Verify unowned skills and malformed mission IDs are rejected before settlement.
- [ ] Verify representative legacy, active, defeated, checkpoint, practice and
  long-receipt saves reload with intended state. Banking must deliver loot once
  despite duplicate finish requests or a lost HTTP response.
- [ ] Review [snapshot codec compatibility](../bot/web_rift_snapshot.go): small
  saves remain JSON; larger writes can use `rift:gzip:v1:` envelopes, bounded to
  256,000 stored bytes and 2 MiB decoded JSON. An older binary without this decoder
  cannot read compressed saves. Test a compatible rollback binary or an explicit
  conversion strategy before allowing such writes in a release environment.
- [ ] Verify database-backed persistence and reward delivery in an appropriate
  test environment. The in-memory browser fixture cannot establish durability
  across server restarts or real inventory changes.

## Evidence and release decision

- [ ] Retain candidate-specific commands, outcomes, screenshots, measurements,
  limitations and outstanding defects. Use synthetic fixture characters; exclude
  account tokens, cookies, private player data and production save contents.
- [ ] Review every outstanding requirement in the improvement ledger. Link its
  direct evidence or record it as incomplete; this checklist does not waive gaps.
- [ ] Record reviewer decision, compatible rollback plan and post-release checks
  for loading, save recovery, asset availability and duplicate reward prevention.
  Perform publishing/deployment only under the task's deployment authorization.
