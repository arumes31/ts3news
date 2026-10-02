# Between-wave rest alcoves

The ten wave tiers have an entrance alcove, saved with the arena. After all
current enemies fall, standing inside holds the reinforcement countdown. Guard
also holds the countdown outside, allowing a return trip, but grants no refuge
protection there. Leave the alcove and release Guard to resume the remaining time.
The last wave completes normally without a rest interval.

Inside the alcove during an intermission, damage does not consume health,
barriers, first-hit grace, or award guard credit. Floor hazards also cannot slow
or pull the resting player. Warning clocks still advance. No living wave grants
this protection. Rest does not award healing or resources beyond normal combat
clock regeneration; that time remains part of the run duration.

Eligibility is derived from saved geometry, player position, current enemies and
wave countdown. There is no cached immunity flag to become stale. Old saved
arenas without an alcove retain their original behavior. Geometry is cloned with
the campaign and does not block movement. All ten alcoves are checked against
solid terrain and hazards and contain their entrance.

A static floor outline, objective instructions, minimap and preview explain the
refuge. Short synthesized entry/exit tones play on confirmed client transitions,
excluding replay. No image downloads or continuous effects are added.

Verification: the full Rift engine suite, targeted bot integration suite, and
browser journeys cover holding, save/reload, departure, wave gates, geometry
validation, synthesized sound and completion of all three waves.
