# Chronos alternating lane slams

Canonical Chronos actors in new encounters replace slam turns with Lane Slam.
The arena floor is divided into three equal horizontal bands from Y315 to Y490.
Each band spans the full walkable width. The first warning selects the player's
current band; subsequent slam warnings advance one band, wrapping after the
third. Ordinary projectile turns remain between slams. Saved lane and warning
state preserve the selected band; player movement cannot retarget it.

Warnings last 1.4 seconds in every phase, or 2.8 in slow practice. Chronos can begin
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
