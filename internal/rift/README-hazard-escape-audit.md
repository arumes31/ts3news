# Hazard escape and cycle safety audit

Improvement 0982 combines two checks over the current 100-mission, 300-arena
campaign. The permanent safe-ground audit finds an entrance-to-exit walking route
outside the union of every enabled hazard footprint, with a conservative margin.
It uses production collision and drop traversal, checks intermediate steps, and
therefore remains safe for every phase combination throughout repeating cycles.
Its existing negative controls reject full coverage, disconnected refuges and
thin hazards between grid samples.

The new timed escape audit samples quarter, center and three-quarter positions
in both axes of every authored hazard at every difficulty. Positions outside
world bounds or inside solid cover are not legal starting positions. All 5,308
legal samples pass: after a 300 ms reaction delay, normal walking reaches ground
outside every hazard footprint (with a two-unit margin) by the end of the full
1.2-second warning, without damage, contacts, jump or dodge.

To avoid relying on favorable phase alignment, all neighboring hazards remain
active throughout each escape probe. Only the focused hazard begins in warning.
This conservative test configuration changes a detached arena copy, never the
campaign. Walls, cover, ledges, elevations and movement rules remain intact.
The player has zero armor and no grace timers. The probe tries eight movement
directions using actual simulation ticks. Negative controls verify that full-floor
hazards, a closed wall cage and an already-active overlapping neighbor fail.

Scope: static authored hazard footprints and ordinary walking without enemy
pressure. Timed escape is a finite spatial sample, not a proof for every possible
player coordinate or imposed control effect. The permanent route is a separate
phase-independent geometry check. New moving hazards or changing arena geometry
will require extending this audit when implemented.

Verification: negative controls passed (0.688s). Full internal/rift suite passed
(25.449s), including all 5,308 timed samples and permanent safe ground in all 300
arenas. Full local output: test-results/hazard-escape-audit-full.txt. No production
arena change was needed.
