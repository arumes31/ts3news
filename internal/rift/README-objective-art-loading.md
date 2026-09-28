# Objective artwork readiness

Brawl loads objective artwork when preparing a mission or saved expedition.
Idle startup no longer downloads all seven objective PNGs. The base atlas counter
now covers eight images, plus four shared creature atlases and the selected hero
sheet loaded through their existing readiness promises.

`RiftRenderer.prepareRun(run)` prepares the union of objectives in every frozen
`run.level.rooms` tier, the current `room_objective`, and saved objective actors.
The client already awaits this promise before applying initial state or mutation
responses. All future tiers in that mission are therefore decoded before play;
advancing a tier does not discover a new objective image. New missions can require
additional artwork before their first scene. Existing decoded images are reused
for the lifetime of the page.

| Objective | Artwork |
| --- | --- |
| Sigils, rune gate, moving beacons, hold circle | Sigil |
| Destroy totems | Totem |
| Disable generators | Generator |
| Carry relic | Relic |
| Escort spirit | Spirit |
| Rescue companions | Cage and spirit |
| Protect lantern, split defense | Lantern |

Current actors and nested objective data also cover older saves without campaign
level metadata. Cleared cages retain their spirit artwork for the rescue animation.
Concurrent preparations share a pending request per image. Decode completes before
preparation resolves; a failed promise is removed so the existing recovery flow
can retry. No image placeholder or blank-objective fallback is introduced.

Direct renderer diagnostics that construct synthetic objective scenes must await
`prepareRun` before calling synchronous `snapshot` or `renderActor`. Production
state application follows this contract through the client readiness boundary.

## Transfer scope

The seven files total 8240354 bytes. Removing them from idle startup reduces its
required PNG artwork from 38355349–38450706 to 30114995–30210352 bytes, depending on
the hero sheet. A resumed expedition additionally loads its required objectives.
This is a dependency and file-size reduction, not a new constrained-network timing
result. It still exceeds the 3000000-byte cold-start target by a wide margin.
The earlier cold-start gate remains failed. Cached startup was measured separately
for this candidate as described below.

The loader does not change combat, rewards, mission content, image quality, audio,
image dimensions, atlas cache policy or authoritative state.

## Verification

Fourteen readiness/rendering checks passed, including base decode, objective decode,
legacy boss art, concurrent retry and all-tier preparation. The 25 objective
journey checks passed across their initial run and focused rerun: two stale test
assumptions were corrected after inspecting failure traces. The defeated-circle
fixture now sets its hazard preview to zero alongside zero player HP. The rune
journey asserts observed authoritative contact, then persistent progress and cues,
instead of expecting transient contact to survive a queued movement step.
The final focused rerun passed all eight checks; embedded bot tests passed in 7.013s.
The rescue screenshot was visually checked for its loaded cage and spirit art.

Three independent cached-startup captures on the existing development profile
passed in 1709.7, 1631.2 and 1698.5ms. Each reused 13 PNG images, transferred 98629
bytes for the other completed resources, and recorded no browser/request errors.
Raw evidence: [warm-start-objective-art-2026-09-28.json](../../tests/performance/baselines/warm-start-objective-art-2026-09-28.json).
This verifies the warm-cache gate only; it does not establish cold-load timing or
physical-device performance. Local artifacts are in `objective-art-loading`,
`objective-art-gameplay`, `objective-art-verified`, and `warm-start-objective-art`
under `test-results`. Initial failures remain preserved in the gameplay directory.
