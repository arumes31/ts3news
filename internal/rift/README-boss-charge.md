# Ancient Dragon aimed charge

Canonical Ancient Dragon actors in new encounters gain a charge option on every
third attack. It starts only at 140-480 units and within 100 units of the player's
lane; outside that range the existing slam/volley sequence proceeds. Other bosses
and previously frozen actors without Charging retain their existing behavior.
Committed attacks are not replaced.

The boss fixes its destination at warning time and gives 1.15 seconds of warning
in every phase (2.3 with slow practice telegraphs). It rushes at 320 units/second
using collision substeps no larger than six units and the boss collision radius.
Player contact, reaching the locked destination or hitting cover/bounds ends the
rush. It hits at most once; sidestepping, jumping or dodge invulnerability avoid
contact damage. Damaging hits and control effects interrupt warning or movement.

Every ending leaves 1.2 seconds of stationary, vulnerable recovery and a minimum
2.3-second attack cooldown. Saved warnings, rushes and recovery retain their
state. Long charges keep their attack pose/attacker slot for the full travel time.
The normal arena attacker budget applies before beginning a charge.

Existing charge warning/rush/recovery sounds, sprites and intent labels are
reused. The fixed aim path and SIDESTEP label replace slam and volley guidance;
RECOVERING shows the remaining stationary window. Text uses a dark outline for
contrast and remains static under reduced motion. No new economic rewards or
boss damage multiplier is introduced.

Tests cover canonical assignment, cadence, committed attacks, planning, slow
practice, aim lock, collision/recovery, punishability, save/reload, jump and hit
interruptions, single contact damage and attacker limits during long rushes.
Browser fixtures cover boss and ordinary charge/recovery cues and shared audio.
This implements improvement 0330.
