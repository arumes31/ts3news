# Burrowing approach

Canonical common rats gain Burrowing in Brawl through the shared localized noun
pool. They keep their Abyss identity, art, stats, rewards and other roles. Existing
frozen actors without the flag retain their saved behavior. Bosses, ranged actors,
treasure creatures and objective props do not use this approach.

A ready rat100–240 units away locks the player's ground position, provided both
actors are on ground elevation0 and the route clears solid cover with the actor's
footprint. It spends the normal attack slot and pack lockout, travels at240 units/s
in steps no larger than6 units, then waits stationary for a full second before
emerging. Total warning is travel time plus1s. The target never follows later
player movement. Terrain obstruction interrupts the approach instead of allowing
wall/cliff tunneling or teleporting to the target.

The visible mound remains hittable. Positive damage, stagger, knockdown and death
cancel the burrow and impose a2s retry cooldown. A successful emergence deals one
normal enemy hit inside an ellipse of radii75/40, respecting cover, guard/armor,
jump >=0.1 and dodge invulnerability. It then recovers stationary for0.9s, with a
2s attack cooldown. No extra kill credit, damage credit or loot is fabricated.
Burrow flag, locked target, warning and recovery survive ordinary saves.

The renderer draws a moving dirt mound, dotted approach path, marked impact area
and distinct approach/emergence text. Dark-backed outlines remain readable on
textured floors. Reduced motion keeps these cues static; screenshot-clean mode
hides warning overlays consistently with other enemy warnings. Optional intent
labels distinguish approach, emergence and recovery. Start, interruption and
emergence have distinct sound cues using the existing sound mix/mute/pause rules.

Tests cover the live rat roster, locked save/resume target, full warning, single
hit/recovery, walking/jump/dodge escape, damage/control/death interruption, cover,
committed attacks, role exclusions and attacker/pack budgets. Browser coverage
checks the mound, both stages, cancellation and clean mode with normal/reduced
motion; shared sound tests exercise source creation, mute and pause cleanup.

The authored-spawn audit now advances the pack lockout as Run.tick does. A new
integration regression preserves the mission71 terrain cancellation that exposed
the fixture's frozen timer: the real world tick resumes attacking at5.22s. The
full1,317-spawn/113-template and9,219-role-position audits retain their geometry,
60s limits and damage/projectile requirements.
