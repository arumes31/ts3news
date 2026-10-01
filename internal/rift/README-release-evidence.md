# Brawl Candidate Release Evidence

This document records candidate verification evidence for the Abyss Rift Brawl
game mode in accordance with [Brawl release checklist](README-release.md) and
the 1,000-item improvement ledger in [tasks/brawl-improvements.md](../../tasks/brawl-improvements.md).
All measurements and tests use synthetic fixtures; no private player data,
account tokens, session cookies, or production credentials are included.

## Candidate Environment and Metadata

- **Candidate Revision**: Target commit following candidate verification
- **Base Commit**: `1008b7046dfb4bba4b64b6912a16c72002dd08b6` ("test: measure live crowded Brawl boss performance")
- **Go Version**: `go version go1.27.1 windows/amd64`
- **Node.js Version**: `v24.19.0`
- **Playwright / Chromium Version**: `153.0.8010.12`
- **Operating System**: `Microsoft Windows NT 10.0.26200.0`
- **Database / Backend**: SQLite in-memory / SQL-mock synthetic fixtures for automated tests
- **Migrations & Assets**:
  - Zero SQL schema migrations required.
  - Sectioned and optimized regional/character/mob/prop/creature atlases verified lossless with SHA-256 integrity checks.

## Automated Verification Gates

All automated candidate gates defined in [README-release.md](README-release.md) were executed:

### 1. Campaign Content Validator (`cmd/brawl-validate`)
Command:
```powershell
go run ./cmd/brawl-validate -out-dir ".tmp/brawl-release-20261001"
```
Outcome: **PASS** (Exit Code 0 across all four gates)
- **Combat engine regressions** (`engine-tests.txt`): PASSED. All unit and integration tests in `internal/rift` pass cleanly.
- **Population and reward budgets** (`population.json`): PASSED. Evaluated across 300 campaign rooms (100 missions x 3 tiers):
  - 0 rooms exceed the 8-enemy active budget.
  - 0 rooms exceed the 1-boss budget.
  - Conservative reward ceilings (gold and gear drop caps) strictly enforced.
- **Walking reachability** (`reachability.json`): PASSED. 300 rooms evaluated with 10-unit pathfinding grid: 0 unreachable targets or spawn reachability failures.
- **Hazard route intervals** (`hazards.json`): PASSED. 300 rooms evaluated over a 30.00-second horizon: 0 rooms with unsafe intervals or impassable cycles.

### 2. Campaign Validator & Report CLI Tests
Command:
```powershell
go test ./cmd/brawl-validate ./cmd/brawl-population-report ./cmd/brawl-reachability-report ./cmd/brawl-hazard-report -count=1
```
Outcome: **PASS** (Exit Code 0)
- `ts3news/cmd/brawl-validate`: ok (0.558s)
- `ts3news/cmd/brawl-population-report`: ok (0.896s)
- `ts3news/cmd/brawl-reachability-report`: ok (0.920s)
- `ts3news/cmd/brawl-hazard-report`: ok (2.083s)

### 3. Server, Static References, and Production Asset Tests
Command:
```powershell
go test ./internal/bot -run 'TestRift|TestAbyssStaticAssetReferencesResolve|TestProductionStaticAssets' -count=1
```
Outcome: **PASS** (Exit Code 0)
- `ts3news/internal/bot`: ok (8.098s)
- All 12 selectable subclasses verified compatible with all offered mission objectives, including "Potions in reserve".
- Gear provenance tracking verified across all 100 missions and tiers through database banking transactions and receipt decoding.
- Regional background atlas panel contracts verified against `rift_region_sections.js` and renderer geometry.
- Production static assets and references verified resolving locally without external CDN dependencies.

### 4. Full Combat Engine Regression Suite
Command:
```powershell
go test ./internal/rift -count=1
```
Outcome: **PASS** (Exit Code 0, 38.258s)

### 5. Transport Section Asset Verification
Command:
```powershell
python scripts/build-brawl-creature-sections.py --check
python scripts/build-brawl-hero-sections.py --check
python scripts/build-brawl-mob-sections.py --check
python scripts/build-brawl-prop-sections.py --check
python scripts/build-brawl-region-sections.py --check
```
Outcome: **PASS**
- 33 creature transport assets verified
- 13 hero section assets verified
- 7 local mob row assets verified
- 9 prop transport assets verified
- 11 region transport assets verified

## Performance and Reliability Status

Measurements recorded across the development cycle against the [minimum-device performance budget](README-performance-budget.md):

1. **Cold-Start Readiness**:
   - Idle development cold-start achieved: median 14.716s, 2.766 MB transfer payload (under the 25s development gate).
   - First-playable with complete combat encounter: median 41.042s.
   - Status: Development idle gate **PASS**; constrained connection first-fight gate remains **OPEN/FAILED**.

2. **Frame Stability (Crowded Boss Encounter)**:
   - Measured live at Mission 100 Room 2 under full load (120 enemies, 14 bosses, player controls):
     - Sample 1: interval p95 600.0 ms, render p95 281.3 ms
     - Sample 2: interval p95 666.6 ms, render p95 362.0 ms
     - Sample 3: interval p95 916.6 ms, render p95 432.7 ms
   - Zero runtime exceptions or WebGL context losses observed.
   - Status: **FAILED** 50ms interval / 16ms render budget. Preserved as documented candidate limitation.

3. **Memory and Replay Durability**:
   - 60 sequential three-tier replays over 39.6 measured minutes completed without memory leaks or runaway retention.
   - Safe exporter confirmed 12/12 snapshots contained current-run identifiers only; attempt ownership cleanly separated.
   - Status: Verified stable for development profiles; native memory and long-term physical hardware profiling remain **OPEN**.

4. **Input Dispatch**:
   - Input wake responsiveness candidate reduced median latency by 12.1% (queue delay 317.5ms -> 248.9ms).
   - Strict 300ms p95 end-to-end target under heavy load remains an open engineering item.

## Improvement Ledger Completion Audit

- Total proposed backlog items: **1,000**
- Previously addressed and checked items: **999** (Batches 1–47, through commit `1008b704`)
- Final item addressed:
  - **1000**: *Record release evidence without including private player data* — **COMPLETE** (recorded in this evidence document, adhering to synthetic test fixtures and zero private data disclosure).
- Current Ledger Status: **1000/1000 (100% of backlog items addressed)**.

## Decision and Next Steps

- **Candidate Quality Decision**: Automated content validation, game logic, mission structure, and authoring budgets are fully verified and passing. Known frame stability constraints under extreme crowd stress are formally cataloged.
- **Rollback Compatibility**: No database migrations are required; reverting the application binary restores previous state cleanly. Snapshot compatibility preserves backwards-compatible JSON decoding alongside optional gzip envelope support.
