# Loot chest presentation

The chest opens once when the client observes a fighting-to-cleared transition
with collected, unbanked loot for this mission and tier. Rewards still come only
from authoritative drops and existing banking. No chest interaction, reward roll,
collision or transition delay is added. Replayed cleared saves show the open
frame without the opening cue. Reduced motion and disabled loot motion use the
final frame; animation uses the renderer clock so pausing freezes it.

The six-frame original PNG and exact generation prompt are adjacent in webassets.
The sheet loads only after active combat starts, or when showing an already
cleared save. Its promise is independent of startup readiness. One image is kept,
with at most one request in flight. Failure retries in another room; the loot UI
and banking remain available. The controller retains coordinates and scalar state,
not run payloads. The integer sprite crops can use the existing bounded atlas cache.

Placement searches bounded ground offsets, avoiding solid cover and the exit
portal. If no position is clear, existing loot UI remains the representation.
No physics or reward state is mutated by placement or animation.

The checkpoint heading repeats the same frame on a small canvas so its panel
cannot obscure the opening. That preview redraws only when the frame changes,
uses the same decoded image and adds no timer or animation loop. A failed optional
script is also nonfatal: combat and banking continue without chest presentation.

Six Node controller tests pass, covering confirmed one-shot behavior, recovery,
reduced motion, retirement, failure retry bounds, unchanged rewards and placement.
Browser checks cover real three-wave clears, after-Start loading, one sound,
reloaded open state, unchanged drops/gold, both missing-art and missing-script
banking paths, and a 390x844 layout. Final three-test run passed in41.7s; desktop
and mobile screenshots reviewed under test-results/chest-mobile-verified.
The Rift integration and static asset checks passed in7.185s. Connection recovery
renderer and sound synthesis checks also passed in the preceding focused run.
