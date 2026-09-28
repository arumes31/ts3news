# Cleared-exit rift portal

A cleared or completed campaign tier displays a small upright portal at its frozen
`arena.exit` coordinate. The original floor marker and regional inlay remain. The
open-exit label sits above the portal; closed exits retain their existing label.
Continue and Bank controls still advance or finish the expedition. The portal is
visual scenery and does not introduce a proximity trigger or modify movement.
Legacy scenes without exit coordinates and practice arenas do not draw a portal.

The ripple refracts already-drawn background pixels behind the portal with up to
seven horizontal strips, each at most 80 by 16 pixels. Each strip moves at most two
pixels horizontally, inside a 38 by 56 radius elliptical clip. Source bounds are
clamped to the 960 by 540 canvas. The effect is culled when outside the viewport.
It draws before enemies, player, hazards and labels, so these remain unwarped.
There is no fullscreen filter, image download, offscreen canvas or persistent
particle collection. Canvas implementations may still use internal copy buffers;
this is not a browser-process memory or performance-gate claim.

The existing decoration clock controls the slow ripple and freezes while paused.
Reduced motion and zero motion skip distortion, retaining the static oval portal.
Effect intensity scales the ripple opacity and portal brightness. There is no
flashing, new shake, ambient loop or repeated sound trigger. Existing room-clear
and area-transition audio remains the cue for the available passage.

The draw routine reads the saved exit without changing authoritative state. It
accounts for the renderer's current shake translation before sampling screen
pixels, and restores the canvas transform, clip, opacity and stroke state.


Verification covers bounded source rectangles and shifts, paused and active motion,
static reduced/zero-motion modes, offscreen culling, practice omission, absent
legacy exits and unchanged run JSON. Existing regional-marker checks cover all
ten regions. The final two browser checks passed in 36.8s; embedded Brawl bot tests
passed in 6.171s. The final screenshot in `test-results/portal-verified` was visually
reviewed after moving the label above the portal. The initial failing regression
is retained in `test-results/portal-red`.
