# Regional entrances

All 300 authored campaign arenas now freeze an optional entrance coordinate.
Each region has a distinct arrival position, shared across its ten missions and
three tiers. Mossbound Ruins retains 160,410; other regions vary the approach lane
within the clear western arrival area. The Citadel entry leaves vertical space
for the existing downward volley escape.

Room spawning uses the frozen coordinate and computes its elevation/floor there.
Resuming a saved fight retains the player's actual position. Older frozen arenas
without an entrance keep 160,410. Campaign copies detach the entrance pointer;
current, future and practice arena metadata reject out-of-bounds entries.

A flat ground mark and ENTRY label identify arrival without introducing a new
collision object. The minimap marks the same point. New regions preserve normal
arrival sound and entry grace; no new input or portal is introduced.

Tests cover all 300 safe arrivals, distinct regions, collision/hazard clearance,
resume/legacy behavior, detached definitions and a frozen next-tier coordinate.
The full combat suite retains every authored-spawn pursuit and boss-opening
safety check. The existing burrow terrain-cancellation regression explicitly
uses its original player coordinate so changing region defaults does not remove
that scenario. Browser checks render all ten markers and remove absent legacy
markers. These checks do not establish performance gates.
