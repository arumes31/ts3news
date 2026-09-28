# Shared elemental rules and pending boss hints

Abyss elemental matchups now live in `internal/content/element_matchups.go`.
`ElementMultiplier` preserves the existing Fire > Air > Earth > Water > Fire
cycle: advantage is 2, resistance is 0.5, otherwise 1. Physical, empty, unknown,
and differently cased values remain neutral. `ElementWeakness` returns the
canonical opposing element or an empty value when none is defined. Existing
Abyss combat and preview helpers delegate to these functions.

The content tests cover all 64 ordered pairs of canonical and unknown values;
existing live weakness and elemental preview tests verify their callers.

This extraction does not add elemental multipliers to Brawl or complete ledger
0349. Brawl currently retains a monster's element but uses effect identifiers
for hit presentation; effects such as ice, rune, poison and void are not a
reliable substitute for canonical attack elements. Before adding phase-specific
weakness hints, propagate actual attack-element metadata through skills,
projectiles and weapon attacks, define the phase rule, and test its exact damage
and UI parity. Keep the shared elemental table authoritative, preserve saved
encounter behavior explicitly, and do not display a bonus that combat does not
apply.

## Canonical metadata retained

New Brawl builds now snapshot `weapon_element` through the same equipped-weapon
resolver as Abyss. Converted skills and class signatures retain their canonical
`element`; projectiles already embed the complete skill and therefore preserve
that value through flight and save/reload. Tests deliberately use a Fire skill
with ice artwork to ensure presentation does not overwrite combat metadata.
Old saves with no element remain empty; no animation-based migration is applied.
Ultimates have no canonical element field in their source definition and remain
unspecified here. No new multiplier or phase hint is active yet.
