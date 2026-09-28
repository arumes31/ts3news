# Chronos alternating lane slams

Canonical Chronos actors in new encounters replace slam turns with Lane Slam.
The arena floor is divided into three equal horizontal bands from Y315 to Y490.
Each band spans the full walkable width. The first warning selects the player's
current band; subsequent slam warnings advance one band, wrapping after the
third. Ordinary projectile turns remain between slams. Saved lane and warning
state preserve the selected band; player movement cannot retarget it.

Warnings last 1.6 seconds in every phase, or 3.2 in slow practice. Chronos can begin
this full-width pattern without approaching into melee range. One impact damages
only players in the selected band, with normal slam power and guard rules.
Jumping or dodge invulnerability avoids it; other bands avoid slam damage and
open the usual weak-point opportunity. Recovery remains 2.3 seconds. Guard breaks
and phase transitions can cancel the pending warning through existing logic.

The display marks the dangerous band with dashed amber borders, a countdown and
MOVE OR JUMP guidance. The other two bands say HAZARDS PAUSED / ENEMIES ACTIVE. Arena hazard
contact is suppressed there during the warning and for 0.4 seconds after impact;
other enemies and their projectiles remain dangerous. Hazard clocks and cues
continue normally. Simultaneous danger bands take precedence over reservations.
Interrupted or dead bosses lose pending reservations, while completed impacts
retain their brief saved reservation. Room transitions clear impact reservations. Borders and text render above sprites;
static reduced-motion cues remain, and clean screenshots hide warnings. A short
full-band impact outline/fill uses the existing slam sound. No full-screen flash
or new external artwork is required.

Tests cover lane cycle, saves, full-width damage, jump/dodge/movement, interrupted
attacks, ordinary volleys and slow practice. Exact boundary regression prevents
floating-point division from assigning a displayed edge to the wrong band.
All 900 legal campaign entry-lane positions, covering 100 missions and three tiers,
have a tested vertical walking escape after 300 ms reaction with actual walls and
hazards present. This does not prove escapes from every coordinate or crowded
state. Existing all-catalog opening and all-campaign boss audits also pass.
Browser tests check all three band positions, two clear labels, removed circular
slam guidance, reduced motion, clean suppression and audio lifecycle.
Implements 0328; global safe-lane guarantees and whole-arena attacks remain separate.

Safe-lane hazard reservations are an incremental part of 0326. Other large
patterns and their terrain escape guarantees still need separate verification.

## Slowed escape timing audit

The warning is now 1.6 seconds (3.2 in slow practice). The former 1.4-second
warning left two sampled slowed positions without a tested escape after a
300ms reaction delay: mission13/final tier at (695,315), and mission53/final
tier at (725,315), above the offset bastion pillar. The new warning allows a
left turn around the pillar followed by downward movement. Normal combat tests
exercise that route with an active slow and actual hazard processing. Existing
saved warnings retain their remaining time; newly started warnings use 1.6s.

Run the wider geometry audit with:

    go test -tags=brawl_audit ./internal/rift -run TestCampaignLaneEscapeGridAudit -count=1 -v

It samples all300 campaign rooms at X35..1565 in30-unit steps and ten Y values
including both arena edges. It skips7132 starts inside intact obstacles and
checks148868 legal positions per speed:235 horizontal/141 vertical normally,
and141 horizontal/84.6 vertical while slowed. Routes use real moveActor collision,
20ms steps, eight held directions, then every single-turn combination if needed.
The walking budget is the canonical warning minus300ms, rather than a separate
hard-coded test allowance. At1.4s,107 slowed positions needed a turn and2 had no
tested route. At1.6s,8 need a turn and all sampled positions have a route. All
normal-speed sampled positions pass in both versions.

This is finite geometric coverage, not proof for arbitrary coordinates, compound
status effects, carried relics, guard-walking, traction, concurrent attacks or
hazard timing. The two focused regressions additionally run full combat ticks;
the existing900 entry tests cover actual hazards. Item0326 remains open for the
broader large-pattern escape guarantees.
