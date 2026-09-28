# Shared elemental rules and boss phase wards

Status: engine, metadata, protocol and HUD implemented. Real-browser phase-hint
verification is pending; ledger 0349 remains open. The ongoing memory fixture
uses an older build and does not cover this feature.

## Shared Abyss rules

`internal/content/element_matchups.go` is authoritative for Abyss combat,
previews and Brawl ward damage. Fire > Air > Earth > Water > Fire: advantage
is 2, resistance is 0.5, otherwise 1. Physical, absent, unknown and differently
cased attack elements remain neutral. `ElementWeakness` returns the canonical
advantage element, or an empty value when no weakness is defined. Content tests
cover all 64 pairs of canonical and unknown values; existing Abyss caller tests
passed after extraction.

New Brawl builds snapshot `weapon_element` through the same equipped-weapon
resolver as Abyss. Learned skills and class signatures retain their canonical
`element`. Projectiles embed their skill, preserving its element through flight
and save/reload. A Fire skill with ice artwork is tested explicitly. Old saves
without element metadata remain neutral; no animation-based inference occurs.
Ultimates have no canonical element in their source definition and remain
unspecified rather than borrowing an element from their visual effect.

## Saved boss wards

New elemental bosses freeze three wards starting at their canonical Abyss
element and advancing around Fire, Air, Earth, Water. For a Fire boss, phases
one/two/three defend as Fire/Air/Earth and are weak to Water/Fire/Air. The source
monster identity and base element are unchanged. Each phase stores its canonical
weakness for the HUD. A fixed three-entry array preserves Actor comparability;
the zero value is omitted from JSON. Physical/unknown bosses and older saved
actors without wards preserve their original neutral damage behavior.

Direct basic hits use the saved weapon element. Direct skills and projectiles
use their canonical skill element. The shared multiplier applies before armor,
shield absorption and final damage accounting, alongside existing class and
weak-point modifiers. Environmental and effect-only legacy damage remain
unchanged. The hit crossing a phase threshold uses the pre-impact ward; subsequent
hits use the new one. Projectiles use the phase at impact, not at launch.

The boss HUD displays the saved current ward and its weakness as direct hits
with a ×2 multiplier before defenses. Its text updates with the boss phase and
clears for legacy bosses or when no boss remains. It is a 12px wrapping status
line, not a canvas label. The pending browser test covers practice phase reset,
save/reload, mobile layout, and clearing legacy/missing-boss hints.

## Verification

- Full `go test ./internal/rift -count=1` passed in 39.129s (GOMAXPROCS=2).
- `TestBossElementPhases*` passed again after adding impact-time projectile and
  neutral JSON omission tests. Coverage includes four starting elements, three
  phases, six attack elements, saves, legacy/environment neutrality, all direct
  attack paths and phase-transition ordering.
- Six public protocol tests passed across `element-protocol.test.cjs` and
  `terrain-protocol.test.cjs`. Legacy absent attack fields and future string
  elements are accepted; invalid scalar types are rejected. Ward data must have
  exactly three typed elemental entries on a boss in phase 1–3. Canonical
  weakness-to-damage parity is established by the server's exhaustive tests.
- HUD and browser-test JavaScript syntax checks passed.
- `tests/e2e/rift-boss-elements.spec.js` is prepared but has not run. Real-browser
  verification must finish before marking 0349 complete.
