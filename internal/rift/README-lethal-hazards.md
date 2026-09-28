# Lethal single-contact hazard warnings

Arena hazards show a distinct `! LETHAL IF HIT` label when one accepted contact
would consume the player's remaining health. This warning is conditional, not a
new instant-kill rule: jumping, dodging, safe-area reservations, contact cooldowns
and temporary grace can prevent contact. It does not predict future enemy hits
or future defense changes. The latest authoritative defenses determine the cue.

Run.HazardContactHealthLoss uses the same first-hit reduction, armor floor,
frontal floor-contact guard reduction and barrier absorption as applied damage.
It caps the preview to current HP and returns zero for defeated players. Hazard
practice intensity is included. Previewing never consumes grace, stamina, barrier
sources, health, or statistics. The parity regression checks288 combinations of
region, health, armor, guard, barrier and first-hit protection against actual
hurtPlayerFromHazard damage, plus practice intensity and conditional evasion.

Full, legacy and lean HTTP responses include a top-level hazard_hit_damage.
The field is computed per response and does not enter canonical saved runs or
retained static baseline sections. Client validation rejects nonfinite, negative,
non-numeric or above-current-health values, then annotates the accepted render
snapshot. Missing legacy preview values remove stale render values rather than
reusing a previous packet. No damage formula is duplicated in JavaScript.

The high-contrast text and exclamation mark appear above every enabled hazard
warning/active footprint, before renderer branches for moving poison, lightning,
blades and sweeping flame. They remain static in reduced motion. Recovery,
disabled hazards, cleared rooms and dead players hide the cue. Hazard-label and
clean-screenshot settings are respected. Existing hazard warning/impact sounds
continue; no additional flashing, screen shake, audio layer or bitmap is needed.

Validation includes read-only full/lean API projection, geometric renderer
branches, exact lethal threshold, recovery/disabled/cleared/dead/settings hiding,
malformed preview rejection, and live lean baseline/reload recovery. This covers
item0465 as a warning for lethal single contacts; ordinary hazards are not made
inherently fatal.
