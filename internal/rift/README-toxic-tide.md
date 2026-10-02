# Toxic Tide challenge room

The optional Toxic Tide link opens an isolated challenge through the existing
practice system (`practice=toxic_tide`). Reach x=1350 beyond three moving toxic
puddles. It uses the equipped Abyss build, has no rewards, keeps the campaign save
separate and supports normal pause/resume/reset and recovery controls.

Each 240-unit warning lane contains an 80-unit active puddle. After a full
1.2-second warning it moves out and back over 3.8 seconds, then rests until the
seven-second cycle repeats. The three lanes have staggered phase offsets.
Ground contact uses poison damage/slow; jump, dodge, guard and movement work.
Only the moving footprint hurts. No attack or skill input is needed in this
traversal challenge. The finish line and progress text identify the goal.

Saved clock drives server collision, renderer, area-status and minimap. Reduced
motion retains essential translation. Poison contact uses existing poison audio.
The no-loot room hides unrelated pickup-training instructions and radius overlay.

Tests cover footprint endpoints, poison contact, empty-envelope safety, saved
continuation, reset and completion. Server tests include this mode in separate
storage and economic-action rejection checks. A real keyboard browser journey
completes/resets the challenge and proves the campaign save is unchanged; visual
checks verify both directions, HUD/map agreement and both motion settings.
No global performance gate is established by this feature.
