# Void Lord minion intermissions

Void Lord's new canonical encounters enter a four-second intermission when
crossing a health-phase threshold. Boss practice retains its focused attack drill.
The boss remains hittable and stops attacking during this window. After one
second, it makes one attempt to summon up to two allies. At most two owned allies
and eight total living enemies may coexist. Phase state and the released flag
survive save/resume, preventing repeat waves; skipping a health phase does not
queue an extra wave. The two transitions permit at most four additions per boss.

Allies use canonical identities, artwork, roles and stats already frozen into the
expedition encounter plan. Bosses, treasure actors and objective props are excluded.
No eligible template or safe placement means no summon for that slot. Placement
avoids existing actors, the player's immediate area, solids, drops and enabled
hazard footprints. New allies have the existing 0.85-second arrival vulnerability.
Player-defeated allies use ordinary rewards; population and phase limits bound
those rewards. Killing the owner dismisses surviving allies without drops, kills
or bestiary defeat credit, and their projectiles retire in the ordinary ownership
checks before damage and after projectile processing. Other owners remain intact.

The display gives a summon countdown, living-allies count, intermission label,
arrival vulnerability and dismissal feedback. Static cues remain in reduced motion;
clean screenshots hide them. Warning and dismissal sounds use the existing mixer.

Verification: save/resume, one-second warning, two/all-eight population bounds,
two-wave lifetime limit, unique IDs, defeat before release, reward-free dismissal,
and boss death in either projectile order. Spawn checks cover all 100 final arenas
for clear placement outside cover, drops and hazards; they are not a proof of every
possible moving-player or crowded configuration. Full combat suite passed28.900s;
expanded focused tests passed0.772s. Browser cues and audio passed3 tests35.8s;
reduced-motion arrival screenshot reviewed. Implements0331 and0350.
