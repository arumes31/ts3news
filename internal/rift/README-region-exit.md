# Regional exits

All 300 arenas freeze an optional exit alongside the entrance. Each region uses
its own eastern destination, x1450 through1540, on the clear y320 corridor.
Collapse escapes, relic deliveries and spirit escorts use this saved location.
Legacy snapshots without an exit retain their original objective destinations.
Campaign copies detach exit pointers; server and browser validate coordinates.

The arena marks the exit with a dashed seal, then a solid open seal after clear.
The minimap shows the same destination. Existing checkpoint banking and seamless
advance remain available without requiring a post-combat walk to the seal.
Saved in-progress objectives retain their existing destination when resumed.

Tests check all 300 clear destinations, ten regional coordinates, objective
placement, detached copies, legacy behavior, saved collapse completion and the
full escort corridor. Browser checks cover each region, map position, cleared
rendering and legacy absence. These checks do not establish performance gates.
