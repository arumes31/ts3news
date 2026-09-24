# Brawl run status transitions

The wire protocol accepts six run statuses. Pause, connection recovery, practice
completion flags and objective statuses are separate fields. Use this map with
[action retry semantics](brawl-action-retries.md); a stale revision returns saved
state instead of performing the transition described here.

## Status meanings

| Status | Meaning |
| --- | --- |
| `fighting` | Active campaign room or practice drill; simulation runs only while unpaused. |
| `cleared` | Campaign enemies and required room objective are finished. Loot is collected into the bag but may still need banking. |
| `defeated` | The player died or a failure objective ended the attempt. Unbanked gold/drops are lost. |
| `complete` | Finished mission/campaign receipt, or completed practice drill. Practice completion grants no real rewards. |
| `banked` | Campaign exit receipt after securing rewards at a cleared checkpoint. |
| `expired` | GET response for a saved run belonging to an obsolete economy epoch. It cannot continue under that epoch. |

## Campaign transitions

| From | Trigger and condition | To and consequences |
| --- | --- | --- |
| No saved run | Accepted `start` | New `fighting` run with a new ID and selected mission/build. |
| `defeated`, `complete`, `banked` | Accepted replacement `start` naming the previous run | New `fighting` run/ID; supported campaign history is inherited. |
| Any saved status with an old economy epoch | Accepted replacement `start` naming the previous run | New `fighting` run/ID in the current epoch. The active-run restriction applies only within the same epoch. |
| `fighting` | Combat tick finds no living enemies and no unfinished required room objective | `cleared`; sweeps remaining drops into the bag, records clear/split/encounter and removes projectiles. |
| Active simulation | HP reaches zero, lantern is extinguished, or defense is lost | `defeated`; records defeat and clears unbanked gold, drops and projectiles. Previously banked rewards remain secured. |
| `cleared`, nonfinal room | `bank` | Stays `cleared`; secures pending rewards and settles pause. Does not enter another room. |
| `cleared`, any room | `exit` | `banked`; secures pending rewards and ends the expedition. Final-room history is completed; earlier exit history is exited. |
| `cleared`, nonfinal room | `next` or `advance` | `fighting` in the next room, same run ID/build/receipt; restores 25% maximum HP up to the cap and mana to 100. |
| `cleared`, final room | `bank` or `next` | `complete`; records mission completion and victory. |
| `cleared`, final room before mission 100 | `advance` with a mission definition | `fighting` in the next mission, same run ID/build/receipt; restores resources and removes already-banked active drops. |
| `cleared`, final room of mission 100, or no next mission definition | `advance` | `complete`; no mission 101 is created. |
| `defeated` | `retry_boss` with a valid frozen campaign boss lineup | `fighting` and paused in the same room/run; restores resources and enemies without restoring lost rewards. |
| Any saved status | GET detects an economy epoch mismatch | Returned view is `expired`; clears spendable/active rewards and moves banked totals into historical totals in that view. GET does not persist this projection. |

The shared tick's defeat guard is evaluated before room/drill completion. `Step`
accepts fighting and cleared statuses, but enemy AI and floor hazards execute
only while fighting. A cleared snapshot with an already-failed player/objective
can therefore hit the shared defeat guard; a secured checkpoint is not a separate
simulation mode with a different status enum.

All campaign checkpoint actions require `cleared` for a fresh revision and bank
rewards atomically with the transition. Final-room completion is recorded before
choosing exit/complete/next mission. Consequently `banked` describes how the run
ended, not whether its final mission was successfully completed.

## Practice transitions

Practice starts in `fighting` using separate mode-specific storage. It never uses
campaign `cleared`, checkpoint advance or campaign banking. A completed drill sets
both `Practice.Completed` and run status `complete`. Its completion conditions are:

| Mode | Completion condition |
| --- | --- |
| `movement` | Reach the goal X coordinate. |
| `jump` | Reach the goal after at least one jump. |
| `combo` | At least three practice hits with combo equal to three. |
| `guard` | At least three recorded guards. |
| `perfect_guard` | At least three perfect guards. |
| `hazard` | Three consecutive resolved hazard pulses without additional hits. |
| `class` | Record a qualifying class practice hit. |
| `ranged` | Record three ranged practice hits. |
| `ultimate` | Record a qualifying ultimate during the timing window. |
| `resource` | Complete two qualifying full-charge resource cycles. |
| `boss` | The single practice boss has no remaining HP. |
| `pickup` | Collect all three valueless tokens. |
| `banking` | Collect three tokens, reach the checkpoint, then secure them with `practice_bank`. Reaching the checkpoint alone remains fighting. |
| `skills` | No automatic completion; leaving the page does not create a terminal status. |

Practice can also become `defeated` through the shared failure guard. An accepted
`practice_reset` reconstructs the drill as `fighting` with the same run ID/build,
clears drill progress and retains supported configuration. It can reset an active,
completed or defeated drill; it cannot bypass the transaction's epoch check.
Alternatively a terminal drill may be replaced by an accepted `start`, producing
a new ID. Recovery tools, enemy selection and freeze do not themselves change
status; `practice_bank` is the exception that completes its drill.

## State changes that are not status transitions

- Pause/resume changes `Paused` for fighting/cleared runs. There is no `paused`
  status. Terminal statuses do not resume simulation.
- Network errors stop client play and offer recovery; they do not establish a
  server status or undo an already committed transition.
- Repeated start identity or stale revisions return the current saved state.
- A fresh action can consume a revision without changing status, such as pause,
  resource restoration or a step on a paused run.
- `expired` is a read-time view; old-epoch mutations conflict. Resetting a practice
  drill is not a way to renew its economy epoch.

## Source and verification map

- [Constructors](../internal/rift/monsters.go): initial fighting status.
- [Combat and NextRoom](../internal/rift/combat.go): defeat, clear and room entry.
- [FinishCheckpoint](../internal/rift/levels.go): bank/exit/next/advance branches.
- [Boss retry](../internal/rift/boss_retry.go): defeated boss restart.
- [Practice](../internal/rift/practice.go): reset and drill completion conditions.
- [HTTP transactions and load projection](../internal/bot/web_rift.go): start replacement, epoch expiry and atomic persistence.
- [Protocol validation](../internal/bot/webassets/rift_protocol.js): six wire statuses.
- [Campaign end tests](../internal/rift/campaign_end_test.go),
  [checkpoint clearance tests](../internal/rift/advance_clearance_test.go), and
  [terminal receipt replay tests](../internal/bot/web_rift_finish_replay_test.go).

When adding a status or a status assignment, update this map, client validation
and affected transition/replay tests together.
