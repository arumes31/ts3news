# The Abyss — What's New

## 2026-09-06 — Classes and empowered subclasses
- Choose six freely switchable classes and twelve subclasses in My Build, each with its own empowered animated character and signature spell effects.
- Build and spend a visible class resource with two signature actions. Costs, scaling, target setup and payoff are shown before you cast.
- Save acquired skills and pins per subclass, compare gear stats, and see an affordable Skill Web upgrade. Class changes unlock after banking your run.
- Existing gear, levels, purchases, Skill Web allocations, presets and legacy specializations remain available. Acquired skills stay in your learned collection.

## 2026-09-06 — Combat in motion

- Combat plays confirmed attacks and spells in order, with distinct critical, healing, shield, dodge, block and status feedback.
- Delvers, creatures, pets and named bosses gain idle, attack, cast, hurt and defeat poses. Every catalog skill and ultimate has a distinct effect profile.
- Combatants keep their formation and target identity through updates. The spell bar stays below the battlefield and keeps keyboard focus.
- Combat settings offer saved normal or fast animation and full or reduced effects. Reconnecting skips old animations, and device motion preferences are respected.

## 2026-09-05 — Clearer preparation and navigation

- Run preparation uses larger labels, aligned tier recommendations and a full-width tier selector on mobile.
- Loot filters, utility controls and empty states share clearer spacing, selected states and keyboard focus.
- Workspace destinations stay below the navigation bar, and the mobile back-to-top button clears the run-action dock.

## 2026-09-05 — Combat decisions within reach

- Tactical details and party controls remain available in an expandable panel at every screen size, with a live round timer while you inspect them.
- Mobile combat collapses workspace navigation into a menu and places target selection beside the action deck.
- Larger action names, queue text and controls make timed choices easier to read. Combat settings and audio controls have their own panel.
- Rejected tactic, pause and targeting settings return to the last confirmed values. Pending saves, action failures and reconnects stay explicit, and timeout recommendations name the ability.

## 2026-09-05 — Economy and progression balance

- Gold-to-XP now starts at 10,000 gold per XP. Each successful purchase adds 1,000 gold per XP until Monday at 00:00 UTC, when your rate resets.
- XP purchases stop at the next prestige threshold. Only usable XP is charged; rounding and overflow gold remain in your wallet.
- Cache interest only applies to gold up to the floor's soft cap, preventing exponential payouts on long runs. Existing balances remain intact.
- Deeper combat victories award more XP, with a bounded depth bonus and extra rewards for harder tiers. Defeat XP stays small.
- High loot quality preserves gear and consumable drops. Loot-find bonuses improve the shared drop table instead of disproportionately granting ultimates.

## 2026-08-26 — A distinct Skill Web

- Every live Skill Web node now has a unique pixel-art cell drawn from one of six discipline atlases, with recognizable silhouettes, high-contrast support, and crisp canvas rendering at every zoom level.
- Atlas generation is deterministic and validated against the complete node catalog, so new nodes cannot silently ship with missing or duplicate artwork.
- The seven Abyss workspaces remain on one keyboard-accessible navigation rail that scrolls cleanly on narrow screens and restores the selected workspace.
- Identified rings can reroll a one-to-three-socket layout for five Void Shards without losing or downgrading fitted gems.

## 2026-08-25 — Combat clarity and reliability

- Live-combat reconnect now resets its event cursor for every new session, preventing stale reconnect loops after descending.
- Combat feedback adds opt-in synthesized hit, cast, heal, ultimate, and defeat cues with saved mute and volume settings.
- Visual impact flashes can be disabled independently and respect the operating system's reduced-motion preference.
- Pixel combatants now use overhead health bars, animated actions, stable monster art, boss variants, and readable hit/heal numbers.
- Forge and shop tools live in dedicated tabs, keeping the main descent flow focused.
- Community tools can read versioned, anonymous run and tier aggregates from `/api/abyss/public/stats`.
- Finished live combats retain their deterministic seed and bounded event history in an owner-authorized replay viewer.
- Live combat now shows the exact action changes remaining each round and preserves the queued or timeout action when that safety budget is exhausted.
