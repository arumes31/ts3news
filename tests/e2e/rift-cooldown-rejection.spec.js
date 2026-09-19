const { test, expect } = require('@playwright/test');

test.describe('Distinct cooldown-rejection cue with rate limiting (Proposal 0176)', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/abyss/rift');
    await expect(page.locator('#rift-start')).toBeEnabled();
  });

  test('cooldown_rejection cue plays through the interface bus and increments played count', async ({ page }) => {
    const result = await page.evaluate(async () => {
      const audio = window.RiftAudio;
      await audio.setActive(true);
      const before = audio.played;
      const played = audio.playCooldownRejection(0);
      const after = audio.played;
      return { before, after, played };
    });

    expect(result.played).toBe(true);
    expect(result.after).toBe(result.before + 1);

    await page.evaluate(async () => {
      await window.RiftAudio.setActive(false);
    });
  });

  test('rate limits cooldown_rejection cue to once per 500ms', async ({ page }) => {
    const result = await page.evaluate(async () => {
      const audio = window.RiftAudio;
      await audio.setActive(true);
      const playedCountStart = audio.played;

      // First call should succeed
      const first = audio.playCooldownRejection(0);

      // Rapid successive calls should be blocked by the 500ms rate limiter
      const rapid1 = audio.playCooldownRejection(0);
      const rapid2 = audio.playCooldownRejection(0);
      const rapid3 = audio.playCooldownRejection(0);

      const countAfterRapid = audio.played;

      // Wait 550ms to exceed cooldown
      await new Promise(r => setTimeout(r, 550));
      const afterCooldown = audio.playCooldownRejection(0);
      const countFinal = audio.played;

      return {
        first,
        rapid1,
        rapid2,
        rapid3,
        afterCooldown,
        playedCountStart,
        countAfterRapid,
        countFinal
      };
    });

    expect(result.first).toBe(true);
    expect(result.rapid1).toBe(false);
    expect(result.rapid2).toBe(false);
    expect(result.rapid3).toBe(false);
    expect(result.countAfterRapid).toBe(result.playedCountStart + 1);
    expect(result.afterCooldown).toBe(true);
    expect(result.countFinal).toBe(result.playedCountStart + 2);

    await page.evaluate(async () => {
      await window.RiftAudio.setActive(false);
    });
  });

  test('active ability cooldown triggers cooldown-rejection cue when mana is sufficient', async ({ page }) => {
    await page.locator('#rift-start').click();
    await expect(page.locator('#rift-overlay')).toBeHidden();
    await page.keyboard.press('Escape');

    const testResult = await page.evaluate(async () => {
      const audio = window.RiftAudio;
      await audio.setActive(true);
      const playedBefore = audio.played;

      // Fetch active run
      const res = await fetch('/api/abyss/rift');
      const data = await res.json();
      const run = data.run;

      const skill = run.build.signatures[0];
      run.player.mana = 100; // full mana
      run.player.cooldown = 0;
      run.skill_timers = { [skill.id]: 2.5 }; // on cooldown

      const intents = window.RiftIntents;
      intents.sync(run, true);
      intents.press('signature0');

      // take intent with active cooldown should reject and trigger playCooldownRejection
      const outcome = intents.take(run, () => false, false);
      const playedAfter = audio.played;

      return {
        skill: outcome.skill,
        playedBefore,
        playedAfter
      };
    });

    expect(testResult.skill).toBe('');
    expect(testResult.playedAfter).toBe(testResult.playedBefore + 1);

    await page.evaluate(async () => {
      await window.RiftAudio.setActive(false);
    });
  });

  test('empty-mana cue takes precedence when mana is insufficient even if cooldown is active', async ({ page }) => {
    await page.locator('#rift-start').click();
    await expect(page.locator('#rift-overlay')).toBeHidden();
    await page.keyboard.press('Escape');

    const testResult = await page.evaluate(async () => {
      const audio = window.RiftAudio;
      await audio.setActive(true);

      let emptyManaCalls = 0;
      let cooldownRejectionCalls = 0;
      const origEmptyMana = audio.playEmptyMana;
      const origCooldown = audio.playCooldownRejection;
      audio.playEmptyMana = function(pan) { emptyManaCalls++; return origEmptyMana.call(audio, pan); };
      audio.playCooldownRejection = function(pan) { cooldownRejectionCalls++; return origCooldown.call(audio, pan); };

      const res = await fetch('/api/abyss/rift');
      const data = await res.json();
      const run = data.run;

      const skill = run.build.signatures[0];
      run.player.mana = 0; // insufficient mana
      run.player.cooldown = 0;
      run.skill_timers = { [skill.id]: 2.5 }; // also on cooldown

      const intents = window.RiftIntents;
      intents.sync(run, true);
      intents.press('signature0');

      intents.take(run, () => false, false);

      audio.playEmptyMana = origEmptyMana;
      audio.playCooldownRejection = origCooldown;

      return {
        emptyManaCalls,
        cooldownRejectionCalls
      };
    });

    expect(testResult.emptyManaCalls).toBe(1);
    expect(testResult.cooldownRejectionCalls).toBe(0);

    await page.evaluate(async () => {
      await window.RiftAudio.setActive(false);
    });
  });
});
