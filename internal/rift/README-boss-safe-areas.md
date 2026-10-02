# Boss hazard shelters

Item 0326 reserves hazard-free escape areas for the committed floor patterns.
The shared `reservedBossArea` predicate is used by arena hazard contact handling.
It does not stop hazard clocks, block enemy attacks, or guarantee immunity where
another committed pattern overlaps the marked area.

| Pattern | Reserved area | Warning / saved impact |
| --- | --- | --- |
| Lane Slam | The two other floor bands | 1.6s / 0.4s |
| Void Ring | Center and open quadrant within the 200/90 ellipse | 2s / 0.45s |
| Time Pulse | Outside the 125/62 blast and inside the 200/90 ellipse | 2s / 0.45s |

Any simultaneous lane, ring or channel danger takes precedence over a reservation.
Cancellation removes a pending warning. Completed impacts retain their saved
reservation through boss defeat; pause freezes timers and room entry clears them.
Other enemies and projectiles remain dangerous. Ground Surge deliberately covers
the arena floor and retains its separately verified jump/dodge counterplay under
0325/0360. Projectile volleys, Rotating Fan and charges retain their existing
movement/cover/guard counterplay; they do not claim floor-hazard shelter.

Visible geometry, interruption, expiry, reduced motion, clean screenshots and
mobile Time Pulse label bounds are covered by the browser checks. Four lane/pulse
checks passed in test-results/boss-shelter-final-20260928; ring checks and visual
review are recorded in README-boss-ring.md. Labels identify shelter while retaining
warnings about enemy attacks and overlapping patterns.

The campaign audits use actual collision geometry and 300ms reaction delay, with
normal and slowed movement. Lane coverage includes 147794 legal starts per speed
across all 300 rooms, plus 37660 per speed across 80 wave-floor/gate states. Ring
coverage includes 16307 threatened starts per speed across 1408 legal boss centers
in all 100 final arenas. Pulse coverage includes 180566 starts per speed across
370 room/floor states. Every sampled start has a tested route. Focused combat-tick
regressions preserve active slow and authored hazard processing, including pulse
escapes from all 300 room entrances.

These are finite sampled routes, not a proof for arbitrary coordinates, every
combination of status effects, or simultaneous enemies. Consult the linked pattern
files and tests for exact boundaries and lifecycle assertions:

- README-boss-lane-slam.md and boss_safe_lanes_test.go
- README-boss-ring.md and boss_ring_reservation_test.go
- README-boss-channel.md and boss_channel_reservation_test.go
- boss_safe_areas.go for the shared precedence rules

The combined reservation, combat-tick and four terrain-grid audit run passed in
37.144s on September 28. This completes the reserved-area implementation for0326;
performance and release evidence remain independent open requirements.
