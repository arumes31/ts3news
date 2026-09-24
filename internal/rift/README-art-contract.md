# Brawl atlas, anchor and collision contract

Source of truth: [rift_renderer.js](../bot/webassets/rift_renderer.js),
[shared combat art](../bot/webassets/abyss_combat_art.js), and
[ground collision](levels.go). The measurements below describe current rendering
and physics, not instructions to infer collision from painted pixels.

## Atlas coordinates and frame selection

All row and column indices are zero-based. Uniform grids use
`sourceX = column * imageWidth / columns` and the corresponding row formula.
Do not divide irregularly spaced sheets into uniform rows.

| Asset family | Columns × rows | Selection |
| --- | --- | --- |
| Brawl heroes A/B | 16 × 6 | Class row, expanded Brawl pose column. |
| Brawl mobs | 16 × 6 | Rig row, expanded Brawl pose column. |
| Brawl effects | 6 × 6 | `effectRows` event mapping, animation frame column. |
| Brawl items | 4 × 4 | Slot icon index: column `index % 4`, row `floor(index / 4)`. |
| Regional props | 4 × 2 | Region mapping `[0,1,2,3,4,5,6,3,3,7]`. |
| Region backgrounds | 2 × 5 irregular rows | Region selects column `% 2` and row `floor(region / 2)`. |
| Shared combat roles/creatures/bestiary/bosses | 8 × 8 irregular rows | `actorFrame().source` supplies normalized crop bounds. |
| Shared class atlases | 8 × 6 | Shared class profile row and pose column. |
| Catalog portraits | 14 × 12 | Catalog identity row/column, not a Brawl animation sequence. |

Regions use normalized row boundaries `[0,.179,.363,.559,.755,1]`.
The renderer trims two source pixels from each region panel edge. Changing the
sheet's separators requires changing these boundaries and reviewing every panel.

Shared combat sheets use these measured row boundaries in the 1254-pixel
reference coordinate system, scaled proportionally to actual image height:

| Sheet | Row boundaries |
| --- | --- |
| roles | 0, 158, 318, 476, 638, 783, 924, 1086, 1254 |
| creatures | 0, 144, 298, 441, 617, 789, 947, 1076, 1254 |
| bestiary | 0, 143, 281, 435, 591, 758, 900, 1056, 1254 |
| bosses | 0, 155, 312, 466, 625, 786, 941, 1085, 1254 |

Shared pose columns are idle `[0,1]`, attack `[2,3]`, cast `[4,5]`, hurt `[6]`,
defeat `[7]`. Their row order follows the `rigs` array in the shared provider,
eight rigs per sheet. Consumers must use its returned source rectangle; CSS
portrait/class metadata is not interchangeable with a Brawl source rectangle.

Brawl player rows, in order, are A: vanguard, berserker, marksman, beastmaster,
elementalist, chronomancer; B: oracle, geomancer, bloodblade, voidwalker,
runesmith, alchemist. Foundation classes map to their corresponding subclass
row through `foundations`. Mob rows are goblin, archer, knight, boss, wolf, spore.
The expanded pose sequence uses idle 0–1, run 2–5, attack 8–10, cast 11, hit 12,
knockdown 13 and defeat 14. Guard, jump, recovery and victory select additional
columns or reuse these frames according to the renderer; do not replace the
16-column sheet with the shared eight-column layout.

Effect rows are 0 for physical impacts, 1 for fire/slam/quake, 2 for ice/pack,
3 for protection/healing/radiant/rune, 4 for void/poison/ultimate, and 5 for loot
and completion effects. These are themes, not an exhaustive event list;
`effectRows` contains the exact mappings and some effects are drawn procedurally.

Terrain cover uses four measured source rectangles rather than a uniform grid:
`[48,107,542,434]`, `[676,107,541,434]`, `[16,906,600,242]`,
`[657,733,574,399]` in `(x,y,width,height)` pixels. Preserve or remeasure these
when changing the terrain-cover sheet. Area/boss backgrounds, objective images
and the platform surface are whole-image drawings, not pose atlases.

## Intended foot and base anchors

Actor rendering places a square destination at `(-size/2, -size*.91)` around the
actor's draw origin. Thus its intended foot anchor is normalized **(0.5, 0.91)**
in both local Brawl cells and shared combat source rectangles. Leave transparent
space below the feet; aligning feet to the very bottom makes actors appear to
float relative to their ground position. Horizontal facing flips around this
center anchor. Changing animation pose should not move the painted ground contact
unless the animation deliberately lifts, recoils or falls.

Current display sizes are 168 for bosses; 80 for shared rat/bat/slime/spider/goblin
rigs; otherwise 63 for wolves and 101 for other actors. That ordered selection
matters: a small shared rig takes precedence over the wolf fallback. These are
painted sizes, not hitbox diameters.

The draw origin starts at ground `(actor.x - camera, actor.y - elevation)`.
Interpolation and recoil can adjust it; the jump animation subtracts a sine-arc
height up to 52 display units. Landing/recovery squash, guard stride, ultimate
hover and victory lift are visual adjustments around the same anchor. Shadows
remain associated with the ground rather than the airborne feet. Elevation is
separate from jump state and does not grant airborne hazard immunity.

Regional cover uses normalized base **(0.5, 0.90)**. Its destination width is
`obstacle.w + 14`; its height is `obstacle.h + 38` for low cover or
`obstacle.h + 100` for tall cover. Drawing begins at `obstacle.x - 7 - camera`
and `obstacle.y + obstacle.h - height*.9`, so the painted base aligns with the
lower edge of the footprint. The tall visual can extend well above that rectangle.

Terrain-cover crops instead bottom-align at `cover.y + cover.h`; destination
width is `cover.w + 12`, with six units of horizontal overhang on each side.
Intact art uses height `cover.h + 74`, broken art 30. Objective props (lantern,
cage, totem, generator, spirit) use individual offsets in the actor renderer;
they are whole images and do not inherit the 91% pose-atlas anchor.

Loot and effect cells are center-anchored at **(0.5, 0.5)**. Floor loot additionally
uses elevation, an eight-unit lift and optional visual bob. Neither that bob nor
an effect's painted radius changes server collision.

## Ground footprints, separate from artwork

The server's `actorClearance` expands obstacle rectangles on both axes around an
actor's ground point. It is an axis-aligned clearance test, not a circular pixel
mask. The strict inequalities in `contains` allow exact boundary contact.

| Actor | Obstacle clearance in world units |
| --- | --- |
| Player (regardless of class) | 10 |
| Boss | 18 |
| Treasure creature or wolf | 6 |
| Other actors | 10 |

Movement clamps actor centers to X 35–1565 in the 1600-wide world and Y 315–490.
Navigation adds two units when detecting the need to route around an obstacle;
this is distinct from the actual movement clearance. Both axes are checked.
Jump state above `.1` permits passing low cover, while tall cover remains solid.
Intact wooden cover is solid until destroyed; stone remains solid. Platforms
and one-way descents have their own height and traversal rules in the
[mission schema](README-mission-variants.md).

Attack reach, projectile contact and hazard contact are separate combat rules.
Do not use these clearance numbers as universal hurtboxes. Likewise, no painted
outline, sprite scale, glow, shadow or camera zoom is authoritative collision
geometry. Changing an asset must not silently change world-space footprints.

## Review after asset changes

Run the background/prop reference checks and inspect real browser crop bounds:

```sh
go test ./internal/bot -run 'TestRiftRegionBackgroundPanelsCoverCampaign|TestRiftCoverPropIndicesFitAtlas|TestRiftSkillEffectNamesHaveRendererSupport' -count=1
node node_modules/@playwright/test/cli.js test rift-actor-atlas-bounds.spec.js rift-scene-atlas-bounds.spec.js --reporter=line
```

Use the [isolated fixture instructions](../../tests/e2e/README-brawl-fixtures.md)
for the browser server. `?riftAtlasDebug=1` enables crop diagnostics. Bounds tests
prove source rectangles fit images; visually inspect feet, cover bases and pose
transitions as well. Mispainted anchors can stay inside valid crop bounds.
Compare collision using the [author validator](../../cmd/brawl-validate/README.md),
including walking and hazard reports. Neither crop validation nor a contact
sheet alone proves a playable route or correct attack reach.

The skill-effect contract reads emitted kind literals and signature mappings
from the Go build adapter, checks their effect-atlas rows, and requires both
procedural arrow drawing branches. Producer/mapping structure changes require
reviewing the validator as well. This catches missing kind registrations; actual
visual quality still requires the browser skill preview and combat checks.

For sound-name compatibility, run
`go test ./internal/bot -run '^TestRiftSoundEventNamesHaveAudioSupport$' -count=1`.
It scans literal combat event producers, direct UI calls, audible feedback cues,
class entry sounds and live monster hurt/death/projectile kinds against audio
cases. Projectile expiry is explicitly silent; directional captions with a false
sound flag are excluded. New dynamic event construction needs corresponding
coverage. The isolated browser test `rift-event-sound-coverage.spec.js` verifies
actual source creation and pause cleanup for arrival/barrier/resource/guard cues.

Cover drawing is culled horizontally before depth sorting. The renderer keeps a
96-world-unit margin around the viewport (or a larger margin for wide shadows),
so partial sprites, overhang, labels and camera shake remain visible. This changes
only draw work; full arena geometry still participates in server collision and
client interaction selection. Recheck the margin if future cover art extends
farther beyond its authored footprint.

Health bars use a conservative 40-unit horizontal margin; names and interaction
labels use their measured text width with six units for stroke and shake. This
keeps long edge-overlapping names visible while skipping fully off-screen text.
The health-label browser regression covers regular enemies, bosses, generators,
totems, cages and ward lanterns. Actor art and attack telegraphs remain independent.

The battlefield backing canvas is intentionally fixed at 960 by 540 pixels
(518,400 pixels). CSS scales its presentation at 16:9, using pixelated image
rendering; fullscreen contains the same canvas. Device-pixel-ratio changes do
not multiply backing resolution or require fresh atlases. Preserve this sprite
rendering contract unless a separately measured resolution policy replaces it.
`rift-canvas-resolution.spec.js` checks mobile-height changes, a 3840 by 2160
viewport and live Chromium DPR changes from 1 to 2 to 3 and back. It verifies
continued drawing, zero canvas-size writes and no additional image requests.
This is browser emulation evidence, not a physical-device battery benchmark.

Audio retirement is covered by `rift-audio-source-retirement.spec.js`. It wraps
real Chromium oscillator and buffer-source creation, then verifies stop and
disconnect calls for all started sources across repeated deactivation, silence
and pagehide cycles. Each cycle includes effects, region ambience, boss music
and an unfinished crossfade. Voice and ambience registries must return to zero.
Campaign completion permits its short result cue before the existing 1.5-second
silence callback; pause and page exit use immediate cleanup.

Effect-atlas sprites skip drawing only when their entire square plus an eight
unit rounding/shake margin is outside the 960x540 view. Steam vents use their
clipped plume footprint plus conservative margins. This preserves edge overlap
and keeps decorative drawing independent from authoritative hazard behavior.
The decoration-culling regression checks sparkles and vents at both camera
extremes; the scene-atlas suite still exercises all 36 effect cells.

Depth ordering caches only a permutation of current input indices. Each frame
checks that permutation against current depths, breaking equal-depth ties by
input index to retain stable ordering. It sorts again only after an inversion
or a length change; no actor, snapshot or cover object is retained by this cache.
`rift-depth-order-reuse.spec.js` verifies reuse while depths move without crossing,
immediate reordering on crossing, and stable tie restoration.

Required renderer atlases set `decoding="async"` and `fetchPriority="high"`
before assigning `src`; the props preload declares matching high priority.
All 22 required images still complete decoding before Start becomes available.
These are browser scheduling hints, not a reduction in asset bytes or proof of
a cold-start speedup. `rift-atlas-priority.spec.js` verifies the hints at request
initiation, alongside readiness, failure and legacy-fallback decode tests.
