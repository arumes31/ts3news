# Hazard shutdown switches

Ten Moonlit Necropolis first-tier arenas have a lever near the entrance lane.
Stand within 28 units, on the same elevation and with an unobstructed interaction
path, then hold the existing guard control for 0.6 seconds. Attacking, requesting
a skill, jumping, dodging, recovery/hit/attack poses, attack cooldown or leaving
range resets progress. Keyboard, touch and gamepad share the guard input.

Activation permanently disables every hazard in that room and emits one shutdown
sound. It grants no loot or kill credit and does not replace the room objective.
The lever and its partial charge/used state are saved with the arena; pause does
not advance them. Old saved arenas have no newly added lever. Campaign templates
copy the switch independently, and backend/browser validate position, charge and
used-state consistency with disabled hazards.

The lever has a progress bar and guard prompt, changes position/colour when used,
and then reads HAZARDS OFF. The minimap marks it in gold. Existing disabled-hazard
residue replaces the live danger. Tests cover deliberate activation, interrupted
charge, blocked interaction, saved progress, one-shot shutdown, all ten placement
routes, detached templates and persisted validation. Browser checks cover normal/
reduced feedback and map shutdown; audio checks cover sources, mute and cleanup.
No performance gate is established here.
