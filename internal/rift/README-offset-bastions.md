# Offset bastions and flank corridors

The final tiers of Pillar Watch missions (3, 13, ..., 93) now contain a broad
western bastion and two smaller staggered defenses to the east. Regional offsets
and widths distinguish the ten layouts. The first two tiers retain their original
diagonal pillars. The campaign tactic advertises both flanks, and each final room
keeps its regional landmark under the Offset Bastion layout name.

The central approach is blocked. Two straight walking corridors flank the complete
defense footprint: at y328 above and y470 below for the player's normal footprint
plus two units of clearance. From a staging point at y410 to the far side at the
same y, these routes differ by 44 world units. This compares the two authored
straight flank routes, not globally shortest paths through the gaps between posts.
Players can also use the gaps for closer firing angles. Hazards keep their normal
regional schedules; a walkable corridor is not an invulnerability zone.

Existing tall-cover geometry supplies collision, shot blocking, fading, artwork,
minimap footprints and planning previews. No new assets or save fields are needed.
Fresh runs freeze the new geometry; resumed expeditions retain their saved layout.
The change does not alter loot, enemies, class skills or the seamless tier flow.

Tests move the real player collision body around both footprint-derived routes,
reject direct crossing, compare route lengths, audit every defender's reachability
from the regional entrance and round-trip saved geometry. Existing full-suite
checks still cover every prop perimeter, diagonal firing lane, archer spawn,
boss space, hazard escape and pursuit. Browser checks cover all ten regions in
normal/reduced motion, actual minimap widths/positions, and a saved final-tier
reload. Existing first-tier diagonal and mobile checks also pass. These gameplay
checks do not establish the open frame, startup or memory performance gates.
