# Bounded atlas frame cache

Small drawAtlas calls reuse original-resolution integer crop rectangles, with a
one-pixel source border. Destination coordinates, alpha, transforms and filters
remain unchanged. Prop crops instead copy a bounded prefix from the atlas origin,
keeping fractional source coordinates unchanged. Other fractional rectangles stay
native because rebasing them changed edge pixels under transformed drawing. No
quality tolerance was weakened. Oversized prefixes and large/background draws
remain native. The cache adds no network assets.

An LRU map holds at most64 source canvases and8MiB of calculated RGBA pixel backing.
Each entry is at most2MiB. Eviction zeros both canvas dimensions and releases the
map reference. This is a bounded backing-size invariant, not a measured total
browser/GPU memory limit. Original images remain loaded; the long-session memory
gate is still outstanding. atlasCacheStats exposes read-only counter snapshots.

The visual comparison uses an uncached drawAtlas oracle with actual loaded art,
checks exact PNG output and parent state under opacity/rotation/clipping/filtering/
flips, and covers all hero/effect frames. It exercises both count and byte eviction.
Atlas bounds, diagnostic overlays and effect-retirement checks guard integration.

Three complete mission100 boss captures reduced synchronous render p95 to14.2,
12.4 and13.0ms, below16ms in every run; original values were93.3,100.7 and102.0ms.
Frame p95 remained133.3,133.4 and149.9ms; p99 remained233.3,183.3 and216.7ms.
The overall frame gate still FAILS, and this software-rendered development profile
does not establish physical-device performance. See the boss baseline report for
raw captures and scope. Preserve the failed interval evidence when reporting the
submission-cost improvement.

The prop-origin change passed exact pixels, state, eviction and oversized fallback
checks plus nine foreground-fade/culling regressions (normal/reduced motion).
Three full crowd samples reduced median render p95 from94.0 to48.6ms against
three fresh controls, at an additional792100 calculated cache bytes in that scene.
Median frame intervals did not materially improve; all six crowd samples still
fail the frame budget. See README-crowd-frame-baseline.md for exact evidence.
