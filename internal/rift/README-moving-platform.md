# Moving platform challenge (0406 in progress)

The engine now supports the isolated moving_platform drill. A wooden raised
platform travels from X=300 to X=900 and back every 16 combat seconds. Grounded
riders move with it; jumping detaches them. Its existing sloped surface continues
to determine actor elevation and footsteps. Pausing freezes the deck. Saving
preserves its position and accumulated ride distance, and reset clears both.

Riders use short collision sweeps, so a wall can stop an actor while the deck
continues. Only actual player displacement earns credit, capped at 250 units.
Completion requires that credit and reaching X=1250. Nearby actors are not
carried, and the drill cannot bank loot or change campaign progress.

Focused engine tests cover boarding through normal input, riding to the goal,
continuous reversal, jumps, nearby bystanders, obstruction, paused steps,
save/recovery equivalence, reset and reward isolation. The full Rift test suite
passed after this foundation change (go test ./internal/rift -count=1).

Still required: client protocol validation, visible challenge entry, progress and
control guidance, route/deck rendering and audio, real keyboard/mobile browser
verification and screenshot inspection. The engine foundation alone does not
complete ledger 0406.
