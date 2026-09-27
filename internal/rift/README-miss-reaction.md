# Enemy reaction to missed basic strikes

A basic strike that hits neither an enemy nor destructible cover can prompt one
nearby idle melee fighter to advance for 0.24 seconds. The nearest eligible
fighter within 180 units and 60 lane units reacts. It must have a clear melee
path, and respect the arena's attack budget and pack lockout. Unalerted patrols,
archers, bosses, treasure goblins, dead actors, airborne actors, active attacks,
hit poses, cooldowns and charge recovery are excluded.

The approach uses collision-aware movement at 1.25 times movement speed (base
speed bounded to 80-160), stopping at melee spacing. It deals no damage and never
reduces attack windup. An enemy already reacting cannot extend its own timer.
Any damaging hit, knockdown or stagger cancels it. Losing the clear path also
ends the reaction. Paused simulation leaves the saved timer unchanged.

The existing run sprite animation, a static PRESSING / MISSED STRIKE cue and a
short synthesized effects sound communicate the response. Reduced motion keeps
the label. Clean screenshot mode hides it. No new loot or score reward is added.

Tests exercise actual basic strikes and enemy ticks, nearest-only selection,
normal warning time, hits on enemies/wood, rejected guard/cooldown inputs,
visibility and busy-role exclusions, damage interruption, and saved expiry.
Browser snapshots inspect cue rendering and disappearance on interruption;
audio tests check source creation, mute suppression and pause cleanup.
Implements 0318; the optional intent training overlay (0320) remains separate.
