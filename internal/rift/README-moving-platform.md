# Moving platform challenge (0406 verified)

The engine now supports the isolated moving_platform drill. A wooden raised
platform travels from X=300 to X=900 and back every 16 combat seconds. Grounded
riders move with it; jumping detaches them. Its existing sloped surface continues
to determine actor elevation and footsteps. Pausing freezes the deck. Saving
preserves its position and accumulated ride distance, and reset clears both.

Riders use short collision sweeps, so a wall can stop an actor while the deck
continues. Only actual player displacement earns credit, capped at 250 units.
Completion requires that credit and reaching X=1250. Nearby actors are not
carried, and the drill cannot bank loot or change campaign progress.

Focused engine tests cover boarding through normal input, riding to the goal,
continuous reversal, jumps, nearby bystanders, obstruction, paused steps,
save/recovery equivalence, reset and reward isolation. The full Rift test suite
passed after this foundation change (go test ./internal/rift -count=1).

The browser entry is /abyss/rift?practice=moving_platform, also linked from the
practice menu. Instructions follow remapped controls. Numeric ride progress is
paired with milestone announcements, and the finish remains gated by actual
travel. Client validation rejects nonnumeric, negative and over-cap progress.

The moving deck uses original procedural wooden planks, sloped faces and an
always-visible dashed route. Its position is also shown on the minimap. Reduced
motion retains essential gameplay travel. Boarding and reaching 250 units each
emit one synthesized audio cue per drill; wood footsteps use the shared surface
sounds. All cues use the existing mute, volume, voice and pause controls.

The keyboard journeys passed for normal and reduced motion: board, ride, pause,
reload/recover, finish, reset, reject malformed progress and play both new cues.
Desktop/mobile captures were inspected after fixing the plank outline path.
Evidence is local under test-results/brawl-ferry-final. The combined run passed
both ferry journeys and all practice guidance/remapping checks. Its remaining
raised-platform footstep assertion exposed a timing-dependent test driver; the
driver now waits for a real stone step while strafing on the flat deck. All four
raised-platform cases then passed twice under
test-results/brawl-platform-footstep-confirmed (eight passes).

The regression tests also now use an unassigned guard key (C belongs to dodge)
and recognize cached atlas canvases when checking the injected hit's height.
Full Rift engine tests pass after the cue changes. This closes ledger 0406;
none of this changes campaign terrain or grants practice rewards.
