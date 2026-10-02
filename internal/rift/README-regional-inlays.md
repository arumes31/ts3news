# Regional ground inlays

Entrance, exit and hazard-switch ground marks carry one of ten static symbols:
Mossbound leaf, Ember flame, Glacial snowflake, Storm lightning, Venom thorns,
Drowned waves, Barracks battlement, Necropolis skull, Starless star and Citadel
crown. Geometry distinguishes the regions without relying on a color change.

Marks use small perspective-flattened canvas paths. The paths are cached once
at renderer startup and stroked inside existing interaction marks; no texture
requests, collision geometry or simulation state are added. Interaction labels
and open/closed exit styling remain. Practice arenas omit campaign symbols.
Both motion modes use the same static art. Unknown regions omit the inlay.

The browser visual check captures a ten-region sheet in each motion mode.
Existing entrance/exit checks cover marker placement and legacy absence.
This change does not establish a performance gate.
