# Brawl live-announcement audit

Proposal 0995: verify that live announcements avoid repeated combat spam.

## Contract

Continuous visual values (cooldowns, percentages, latency, timestamps and exact
health) remain readable without being spoken on every snapshot. Live status text
changes for meaningful states: readiness, danger, recovery, objective milestones,
confirmed pickups, boss identity, phase changes and completion. Repeated optional
combat captions retain a four-second suppression window even when their visible
three-item history has evicted the cue. Distinct boss victories remain distinct.

## Evidence

- `tests/e2e/rift-live-combat-churn.spec.js`: all 12 actual subclasses, 100 HUD
  snapshots each, charged/relic/marked builds; changing movement, guard stamina,
  guard recovery, connection protection, slowing, all ability timers, save times
  and latency. No active status region exceeds five meaningful updates. 12 passed
  (40.6 seconds).
- `rift-precision-announcement.spec.js`, `rift-guard-announcement.spec.js`, and
  `rift-area-announcement.spec.js`: exact transition counts for readiness, mana,
  pause, guard break/recovery, protection and slowing; visible countdowns retained.
- `rift-boss-announcement-identity.spec.js` and
  `rift-countdown-announcements.spec.js`: boss identity and bounded windup/transition
  announcements. The boss-identity batch passed four checks (41.5 seconds).
- `rift-objective-announcement.spec.js`: real ritual and collapse encounters keep
  changing visual countdowns, announce actual imminent danger and pause, and stay
  within five status changes. Two strengthened checks passed (28.7 seconds).
  Related beacon, collapse, lantern, ritual, escort and split-defense regression
  batch passed 16 checks (2.6 minutes).
- `rift-practice-announcement.spec.js`: real movement drill retains more than ten
  visible updates but at most five announcements, including 50% and completion.
  Boss practice asserts phase and warning-setting announcements after start/reset.
  Practice regression batch passed 12 checks (1.6 minutes).
- `rift-free-practice-announcement.spec.js`: at least six real basic hits update
  the readable hit counter without mutating the free-practice live message.
- `rift-caption-repeat.spec.js`: 100 events rotating four cue kinds produce four
  initial messages despite the three-item display; the same cue becomes eligible
  again after four seconds. Combined feedback, directional captions, distinct boss
  victories and skill-practice batch passed 14 checks (1.2 minutes).

Test paths above are relative to `tests/e2e/` unless fully specified. Run with the
repository Playwright configuration and `ABYSS_E2E_PORT=18098`, preserving the user
server on 18089. These results are cumulative across the individual fixes.

## Source review

`rift_hud.js` uses equality-checked writes and separates continuous displays from
stable live text. `rift.js` separates room/practice progress and only changes its
main status on actions, recovery or loading outcomes. `rift_loot.js` returns when
there are no newly confirmed pickups. `rift_gamepad.js` equality-checks connection
status and marks continuous controller readings non-live. Intent confirmation
messages are action-driven. `rift_feedback.js` deduplicates confirmed event IDs,
limits repeated low-health warnings, and retains cue suppression independently
of display truncation.

This verifies DOM announcement behavior and the listed gameplay transitions.
It does not certify every screen reader/browser combination or claim that the
canvas game is fully playable without vision; those are separate accessibility
requirements.
