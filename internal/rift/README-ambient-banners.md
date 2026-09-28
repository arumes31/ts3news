# Regional rear-wall banners

Each campaign region has three muted hanging banners at worldX300/780/1260.
They use distinct cloth colors and the same regional emblem as entry/exit marks:
leaf, flame, frost, lightning, thorns, water, battlement, skull, star and crown.
Mounting brackets and pointed cloth distinguish them from rectangular solid
cover. All drawing is clipped aboveY300, separated from the playable floor that
starts atY315. They have no collision, interaction, pickup or combat data.

The renderer draws banners behind actors, projectiles, pickups and queued
warnings/prompts. Only visible banners are drawn; the cap is three per region.
Practice arenas omit them. Clean screenshots retain this scenery. Small cloth
sway follows the existing decoration clock and motion intensity; pause freezes
it, while reduced motion and zero motion intensity keep it static. Existing
regional ambience supplies the sound bed for these distant props.

Graphics are bounded procedural cloth/bracket shapes with cached regional paths,
so this adds no image download, new atlas, persistent state, server geometry or
per-frame resource allocation for emblem paths. This is not a performance-gate
claim: overall frame and loading budgets remain separate work.

Browser checks cover all ten regional designs, unchanged arena data, positions
above the combat floor, culling at both camera ends, pause/reduced/zero motion,
practice omission, and existing regional ground marks. Contact sheets wait for
a new render frame for each region and are reviewed alongside a full scene.
Implements0437.
