# Authored spawn pursuit audit

Improvement 0981 is covered by two deterministic simulation audits across all
100 campaign missions and all three tiers. The fixed pursuit-audit seed produces
1,317 planned spawn entries, including entries held back for later waves. The
native encounter pass includes all 113 current canonical monster templates; an
explicit assertion fails if a catalog addition is not represented.

Each native actor starts at its planned position with its actual adapted role,
speed, footprint, cooldown and attacks. A second pass tests goblin, knight, archer,
boss, treasure, wolf and spore roles at every position: 9,219 combinations. This
pass uses the production spawn settlement for each footprint and a standard speed
of 90, so it covers route geometry separately from native stat adaptation.

Each probe retains the arena's walls, cover and drop geometry. It must reach the
stationary entry player and deal contact/slam damage, or release a projectile on
a clear firing line, within 60 simulated seconds. Treasure goblins must escape.
A winding-up pose alone does not count as success. Failures report the mission,
tier, slot, role, starting coordinates, stopped coordinates and route target.

Scope: isolated pursuit/escape after awareness, with one actor and no active
objective assignment or initial pack lockout. This deliberately separates route
geometry from patrol detection, crowd scheduling, ward defense and ritual work.
Those behaviors retain their own tests. It does not claim every monster stat
combination at every position, dynamic objective/gate transitions, hazard-safe
player routes (0982), or frame-time performance.

Verification: both audits passed in 7.275s. The complete internal/rift suite,
including existing boss, wave-arrival, spawn and drop tests, passed in 18.727s.
The full local output is test-results/pursuit-audit-full.txt. No production route
change was needed for the current authored campaign.
