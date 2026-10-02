# Optional enemy intent training overlay

Battlefield display includes Enemy intent labels (training), disabled by default.
Its boolean preference is saved locally with the other version-1 display options.
All display presets and reset restore it to off. Preset descriptions disclose this.
It changes presentation only and makes no combat request or simulation mutation.

Labels describe the latest confirmed snapshot, rather than predict future AI
choices. Priority follows control effects, arrival/awareness, patrol, charge or
miss-response states, attack windup/pose, fleeing, guarding, archer repositioning,
movement, cooldown recovery, then holding. Ordinary hit reactions are distinct
from stagger stun. Boss windups say Preparing attack; existing boss warnings
continue to carry their precise attack name and target area.

Known combat roles receive labels. Dead actors, the passive practice target and
objective props do not. Text uses a pale cyan fill, dark outline and backplate,
respects HUD text scaling, and stays static under reduced motion. Clean screenshot
mode suppresses it, while hidden enemy names do not disable this independent aid.
Label drawing uses existing viewport rejection and interaction-prompt rendering.

Browser tests cover default off, enable/disable persistence, all state branches,
ordinary-hit versus stun distinction, dead/prop/target exclusions, reduced motion,
explicit text color, clean screenshots and preset reset. Existing settings tests
cover paused, defeated and failed-request views. No unused server Intent field is
consumed or committed. This implements 0320.
