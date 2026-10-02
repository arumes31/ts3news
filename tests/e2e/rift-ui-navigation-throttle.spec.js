const { test, expect } = require('@playwright/test');

test.describe('Limit repeated UI sounds during held navigation (Proposal 0190)', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/abyss/rift');
    await expect(page.locator('#rift-start')).toBeEnabled();
  });

  test('canPlayUINavigation enforces held and discrete navigation rate limits', async ({ page }) => {
    const results = await page.evaluate(() => {
      const audio = window.RiftAudio;
      audio.resetUINavLimits();

      // At t=1000, initial discrete call should pass
      const discrete1 = audio.canPlayUINavigation(false, 1000);

      // Record navigation timestamp at t=1000
      audio.recordUINavigation(1000);

      // At t=1020 (20ms later), discrete navigation should be throttled (< 35ms)
      const discreteTooSoon = audio.canPlayUINavigation(false, 1020);

      // At t=1040 (40ms later), discrete navigation should be permitted (>= 35ms)
      const discreteOk = audio.canPlayUINavigation(false, 1040);

      // Held navigation requires >= 110ms
      const heldTooSoonAt40 = audio.canPlayUINavigation(true, 1040);
      const heldTooSoonAt100 = audio.canPlayUINavigation(true, 1100);
      const heldOkAt120 = audio.canPlayUINavigation(true, 1120);

      return {
        discrete1,
        discreteTooSoon,
        discreteOk,
        heldTooSoonAt40,
        heldTooSoonAt100,
        heldOkAt120,
      };
    });

    expect(results.discrete1).toBe(true);
    expect(results.discreteTooSoon).toBe(false);
    expect(results.discreteOk).toBe(true);
    expect(results.heldTooSoonAt40).toBe(false);
    expect(results.heldTooSoonAt100).toBe(false);
    expect(results.heldOkAt120).toBe(true);
  });

  test('playUINav throttles rapid repeated held calls and plays again after delay', async ({ page }) => {
    const results = await page.evaluate(async () => {
      const audio = window.RiftAudio;
      audio.resetUINavLimits();
      await audio.setActive(true);

      const before = audio.played;

      // Burst of 10 rapid held navigation calls
      const burstResults = [];
      for (let i = 0; i < 10; i++) {
        burstResults.push(audio.playUINav(0, true));
      }

      const playedAfterBurst = audio.played - before;

      // Wait 125ms for held rate limit interval to pass
      await new Promise(r => setTimeout(r, 125));

      const afterWaitCall = audio.playUINav(0, true);
      const playedTotal = audio.played - before;

      await audio.setActive(false);

      return {
        burstResults,
        playedAfterBurst,
        afterWaitCall,
        playedTotal,
      };
    });

    // Only the first call in the rapid burst should have succeeded
    expect(results.burstResults[0]).toBe(true);
    expect(results.burstResults.slice(1).every(v => v === false)).toBe(true);
    expect(results.playedAfterBurst).toBe(1);

    // Call after waiting >= 110ms should succeed
    expect(results.afterWaitCall).toBe(true);
    expect(results.playedTotal).toBe(2);
  });

  test('audio.play ui cue respects isHeld rate limiting parameter', async ({ page }) => {
    const results = await page.evaluate(async () => {
      const audio = window.RiftAudio;
      audio.resetUINavLimits();
      await audio.setActive(true);

      const before = audio.played;

      // First held UI cue
      const res1 = audio.play('ui', 0, true);
      // Immediately second held UI cue
      const res2 = audio.play('ui', 0, true);
      // Immediately third held UI cue
      const res3 = audio.play('ui', 0, true);

      const count = audio.played - before;
      await audio.setActive(false);

      return { res1, res2, res3, count };
    });

    expect(results.res1).toBe(true);
    expect(results.res2).toBe(false);
    expect(results.res3).toBe(false);
    expect(results.count).toBe(1);
  });

  test('campaign mission card keydown triggers rate-limited navigation sound', async ({ page }) => {
    // Open campaign accordion
    await page.evaluate(() => {
      const campaign = document.getElementById('rift-campaign');
      if (campaign) campaign.open = true;
    });

    const results = await page.evaluate(async () => {
      const audio = window.RiftAudio;
      audio.resetUINavLimits();
      await audio.setActive(true);

      const grid = document.getElementById('rift-levels');
      const cards = Array.from(grid.querySelectorAll('[data-level]')).filter(b => !b.hidden && !b.disabled);
      if (cards.length < 2) return { skipped: true };

      const card1 = cards[0];
      card1.focus();

      const before = audio.played;

      // Dispatch ArrowRight on card1 (first press, not held: repeat=false)
      card1.dispatchEvent(new KeyboardEvent('keydown', {
        key: 'ArrowRight',
        bubbles: true,
        cancelable: true,
        repeat: false,
      }));

      const afterFirstKey = audio.played - before;

      // Dispatch 5 rapid held ArrowRight keydowns (repeat=true)
      for (let i = 0; i < 5; i++) {
        document.activeElement.dispatchEvent(new KeyboardEvent('keydown', {
          key: 'ArrowRight',
          bubbles: true,
          cancelable: true,
          repeat: true,
        }));
      }

      const afterHeldKeys = audio.played - before;

      // Wait 130ms for held delay
      await new Promise(r => setTimeout(r, 130));

      document.activeElement.dispatchEvent(new KeyboardEvent('keydown', {
        key: 'ArrowRight',
        bubbles: true,
        cancelable: true,
        repeat: true,
      }));

      const afterDelayedHeldKey = audio.played - before;

      await audio.setActive(false);

      return {
        skipped: false,
        afterFirstKey,
        afterHeldKeys,
        afterDelayedHeldKey,
      };
    });

    expect(results.skipped).toBe(false);
    expect(results.afterFirstKey).toBe(1);
    // Rapid held presses should be throttled
    expect(results.afterHeldKeys).toBeLessThanOrEqual(2);
    // After interval elapsed, another keydown played
    expect(results.afterDelayedHeldKey).toBe(results.afterHeldKeys + 1);
  });
});
