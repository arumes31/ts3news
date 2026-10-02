# Brawl action retries and recovery

This contract describes production POST behavior and browser recovery. Use it
with the [control fields](brawl-control-contract.md) and
[practice/reward boundaries](brawl-practice-and-rewards.md).

## Shared revision rules

Authentication, request validation and action/mode validation still apply to
replays. The server locks the account row and reads the saved mode-specific run
and current economy epoch within a transaction. Except for `start`, a request
must name the saved run and match its economy epoch before revision handling:

- A revision at or below the saved revision returns the current saved snapshot
  without executing the action or writing rewards/state.
- Exactly saved revision plus one executes the action, subject to its state
  checks, then saves that revision.
- A skipped revision conflicts. A missing run, different run ID or expired epoch
  also conflicts, even if the revision is old.

An old-revision response is not an acknowledgment that this particular action
ran. Another tab may have used that revision for a different action. Its response
may contain a newer room, mission, status or receipt. Adopt the returned state;
never reconstruct success from the submitted action name.

`request_id` is the replay identity for `start`; non-start replay suppression uses
run identity, epoch and revision, not request ID. Reusing a request ID with a new
revision does not suppress another non-start action.

## Every action

For every non-start row below, replay uses the shared revision rules above and
skips the listed effect entirely. The final column explains why manufacturing a
new revision after a lost reply is unsafe or has different meaning.

| Action | Effect at a fresh revision | Meaning of another fresh request |
| --- | --- | --- |
| `start` | Creates a campaign or practice run. Same saved StartKey/request ID and current epoch returns the saved run first, even after it progressed. Otherwise an active fighting/cleared run conflicts; replacing a saved run requires its ID. | A new request ID is a new start attempt. The special StartKey replay bypasses ordinary revision sequencing, but is not a permanent history of old starts after replacement. |
| `step` | Applies current input and bounded server elapsed time. Paused/terminal runs skip simulation, though timestamps and saved revision can change. | Another input step can move, attack or cast again. Never replay historical input with an invented revision. |
| `pause` | Pauses fighting/cleared runs and tracks the pause interval. Terminal statuses do not change. | Reasserts pause; still consumes a revision and saves. |
| `resume` | Resumes fighting/cleared runs and settles connection/pause timing. Terminal statuses do not change. | A new intent to resume; can undo another tab's pause. |
| `bank` | Requires campaign cleared status; banks pending rewards. Keeps a nonfinal checkpoint cleared, or completes the final room. | Can revisit a still-cleared checkpoint without paying already banked rewards; conflicts after terminal completion. |
| `exit` | Requires campaign cleared status; banks rewards and ends the run as banked. | Conflicts on the terminal receipt; use its saved revision to recognize a replay. |
| `next` | Requires campaign cleared status; banks rewards and enters the next room, or completes at the final room. | Conflicts while fighting; if another checkpoint is cleared later, would bank/advance that different checkpoint. |
| `advance` | Requires campaign cleared status; banks rewards, enters the next room or seamlessly starts the next mission. Completes at the campaign end. | Could advance a later cleared checkpoint. Does not identify the original checkpoint independently of revision. |
| `retry_boss` | Requires a defeated campaign boss encounter. Restores resources, rebuilds its frozen enemy plan and pauses the restarted fight; clears unbanked losses. | Conflicts unless defeated again; then restarts that new defeat. Does not restore lost rewards. |
| `practice_spawn` | Requires active skills practice and an exact current Abyss monster name. Replaces enemies/projectiles/mark state with the selected target. | Replaces the arena again; could destroy progress made after the first spawn. |
| `practice_clear` | Requires active skills practice. Removes enemies/projectiles/marks without restoring resources. | Could remove targets spawned since the first clear. |
| `practice_reset` | Rebuilds the practice drill with the same run/build identity; applies supported boss/hazard setup options. | Resets elapsed progress again. Recover before deciding to restart a drill. |
| `practice_health` | Requires active practice; restores HP to maximum. | Restores any damage incurred since the first request. |
| `practice_mana` | Requires active practice; restores mana to 100. | Restores mana spent since the first request. |
| `practice_cooldowns` | Requires active practice; clears equipped skill, signature and ultimate timers. | Clears cooldowns started since the first request. |
| `practice_freeze` | Requires active practice; toggles enemy movement freeze and records assistance when enabled. | Toggles back: especially unsafe to retry at a new revision. |
| `practice_bank` | Requires banking practice with all three tokens collected and checkpoint reached. Marks tokens secured and completes the drill without real rewards. | Conflicts after completion; a valid old revision returns the completed practice receipt. |

Action availability and field restrictions are checked before these transaction
rules. For example, campaign banking actions are unavailable in practice, and
spawn/clear are available only in skills practice.

## Interrupted requests

A timeout, broken connection, server failure or truncated/invalid JSON response
does not establish whether the transaction committed. The browser stops local
play, clears input/timers/audio and offers **Recover expedition**. Campaign
banking actions show delivery as unconfirmed. Recovery reads the authoritative
saved run with GET; it does not automatically repeat the failed POST. Initial
loading failures instead offer **Retry loading**.

After recovery, display the saved status and receipt, then let the next deliberate
action use that snapshot's revision. Do not add one to an uncertain local revision
and resend the old intent. This matters even for actions that appear harmless:
`practice_freeze` toggles, and `advance` may apply to a different checkpoint.

A 409 means state conflict and requires recovery. Malformed requests receive 400;
authentication failures require signing in. Other failures are not commit receipts.
A GET after an economy reset exposes the old run as expired with spendable rewards
cleared; the old run cannot continue mutating under its obsolete epoch.

## Atomic rewards and evidence

Campaign reward writes, objective rewards, inventory changes, checkpoint changes
and the serialized snapshot share the same database transaction. A failure before
commit rolls them back together. A commit error or lost response still requires
reading saved state to resolve uncertainty. Practice actions never use campaign
banking; their tokens are valueless.

Implementation: [HTTP and transaction logic](../internal/bot/web_rift.go),
[checkpoint transitions](../internal/rift/levels.go),
[practice tools](../internal/rift/practice.go),
[practice enemy changes](../internal/rift/practice_enemy.go),
[boss retry](../internal/rift/boss_retry.go), and
[browser recovery](../internal/bot/webassets/rift.js).

Regression evidence includes [advance replay](../internal/bot/web_rift_advance_replay_test.go),
[terminal receipt replay](../internal/bot/web_rift_finish_replay_test.go),
[practice banking](../internal/bot/web_rift_banking_practice_test.go), and
[committed but truncated replies](../tests/e2e/rift-truncated-response.spec.js).
Keep this table synchronized with every action accepted by `validRiftRequest`.
