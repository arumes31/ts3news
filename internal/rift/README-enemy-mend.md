# Healing enemy priorities

Frost Lich receives an interruptible Mend ally support move in Brawl, identified
through its canonical localized name during adaptation. Explicit healer roles can
also use the same behavior. Frozen old actors without the healer flag retain their
previous moves. This is a Brawl move; it does not change the shared Abyss template.

An idle, ready healer selects the living ally with the lowest health percentage
within 320 units and a clear projectile path. It excludes itself, treasure goblins
and objective props. A selected target is locked for a visible 0.8-second cast;
new injuries cannot silently redirect it. Healing is 12% of the target's maximum
health, capped at 30 HP and at missing health. A five-second cooldown begins when
the cast starts, including casts that are interrupted or lose their target.

Damage, stagger or knockdown cancels the cast. A dead, missing, fully healed,
out-of-range or obscured target cancels it without resurrection or healing through
cover. Normal attacks remain available when no eligible ally needs healing, and
already committed attacks are not replaced. Existing attacker limits apply.
Successful healing adds a short cast recovery and never increases player healing
statistics, kill credit, gold or loot. Saved target and cooldown fields preserve
an in-progress cast through reload.

The cast sprite, green ally link/ground ring and MEND / INTERRUPT prompt identify
the target and counterplay. Ordinary hostile aiming lines are suppressed for this
cast. Optional intent labels say Mending ally. A synthesized start cue and existing
heal effects mark cast start/completion; reduced motion keeps static guidance.

Simulation tests cover priority, delayed completion, save/reload, cooldown, cap,
overheal prevention, exclusions, interruption/lost targets, attacker budget,
locked targets and ordinary-attack fallback. Browser snapshots verify ally cues,
intent and removal after interruption; audio tests cover mute and pause cleanup.
This implements 0301. Defender-aware support retreat (0302) remains separate.
