# Diagonal ranged cover

Pillar Watch missions (3,13,...,93) use three tall stone pillars across all three
tiers. Their ground footprints form a diagonal, reversed on alternating regional/
tier parity. The regional offsets and widths still distinguish mission layouts.
All three pillars block projectiles, unlike the previous one-tall/two-low pattern.
The mission tactic describes breaking ranged sightlines and flanking between them.

Pillar spacing leaves firing angles around each obstacle and straight upper/lower
bypasses for both player and boss footprints. No new collider, prop asset or custom
projectile rule is introduced: existing high_cover data drives movement, projectile
occlusion, arena art, minimap and planning previews. Fresh expeditions copy the new
geometry; existing saved levels retain their frozen geometry and version identity.

Simulation tests exercise all 30 modified arenas: diagonal ordering, blocked direct
shots, open offset shots, continuous movement along both outer lanes, and standard
archer pursuit/firing from every authored spawn location. Archer probes intentionally
exclude unrelated patrol/pack roles. They do not claim the broader all-monster
pursuit audit. Existing full-suite campaign spawn, route, boss-opening and saved
geometry checks remain required. Browser checks inspect opposite diagonal variants,
rendered minimap positions, saved reloads and mobile layout.
