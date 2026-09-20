const { test, expect } = require('@playwright/test');

test.describe('Recover sound after returning through browser history (Proposal 0194)', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/abyss/rift');
    await expect(page.locator('#rift-start')).toBeEnabled();
  });

  test('audio context and sound recover when resuming expedition after pagehide and pageshow', async ({ page }) => {
    // 1. Enter expedition
    await page.locator('#rift-start').click();
    await expect(page.locator('#rift-overlay')).toBeHidden();

    // Verify audio is actively running
    const initialAudio = await page.evaluate(() => {
      const audio = window.RiftAudio;
      return {
        active: audio.active,
        contextState: audio.context?.state,
      };
    });
    expect(initialAudio.active).toBe(true);
    expect(initialAudio.contextState).toBe('running');

    // 2. Simulate navigating away (pagehide)
    await page.evaluate(() => {
      window.dispatchEvent(new PageTransitionEvent('pagehide', { persisted: true }));
    });

    const hiddenAudio = await page.evaluate(() => {
      const audio = window.RiftAudio;
      return {
        active: audio.active,
        isSilent: audio.isSilent(),
        contextClosed: audio.context === null || audio.context?.state === 'closed',
      };
    });
    expect(hiddenAudio.active).toBe(false);
    expect(hiddenAudio.isSilent).toBe(true);
    expect(hiddenAudio.contextClosed).toBe(true);

    // 3. Simulate returning through history (pageshow)
    await page.evaluate(() => {
      window.dispatchEvent(new PageTransitionEvent('pageshow', { persisted: true }));
    });

    // Expedition should be paused awaiting user gesture, audio remains silent
    await expect(page.locator('#rift-overlay-title')).toHaveText('Your expedition awaits.');
    await expect(page.locator('#rift-start')).toHaveText('Resume expedition');

    const returnedAudio = await page.evaluate(() => {
      const audio = window.RiftAudio;
      return {
        active: audio.active,
        isSilent: audio.isSilent(),
      };
    });
    expect(returnedAudio.active).toBe(false);
    expect(returnedAudio.isSilent).toBe(true);

    // 4. Click "Resume expedition" (user gesture)
    await page.locator('#rift-start').click();
    await expect(page.locator('#rift-overlay')).toBeHidden();

    // Verify sound has completely recovered
    const recoveredAudio = await page.evaluate(() => {
      const audio = window.RiftAudio;
      const playedBefore = audio.played;
      const uiPlayed = audio.play('ui', 0);
      const playedAfter = audio.played;
      return {
        active: audio.active,
        contextState: audio.context?.state,
        isSilent: audio.isSilent(),
        uiPlayed,
        playedDelta: playedAfter - playedBefore,
      };
    });

    expect(recoveredAudio.active).toBe(true);
    expect(recoveredAudio.contextState).toBe('running');
    expect(recoveredAudio.isSilent).toBe(false);
    expect(recoveredAudio.uiPlayed).toBe(true);
    expect(recoveredAudio.playedDelta).toBe(1);
  });

  test('audio.unlock recreates fresh context if resume on stale context fails after history navigation', async ({ page }) => {
    const results = await page.evaluate(async () => {
      const audio = window.RiftAudio;
      // Start audio context initially
      await audio.setActive(true, 0);
      const firstContext = audio.context;

      // Put context into suspended state like browser does on history navigation
      await firstContext.suspend();
      firstContext.resume = async () => { throw new Error('Cannot resume in bfcache'); };

      // Dispatch popstate (history traversal)
      window.dispatchEvent(new PopStateEvent('popstate'));

      // Call unlock: should catch resume rejection, tear down the broken context, and create a fresh running one
      const unlocked = await audio.unlock();
      const secondContext = audio.context;

      return {
        unlocked,
        isDifferentContext: secondContext !== firstContext,
        state: secondContext?.state,
      };
    });

    expect(results.unlocked).toBe(true);
    expect(results.isDifferentContext).toBe(true);
    expect(results.state).toBe('running');
  });

  test('RiftAudio.recover successfully restores audio state after history navigation', async ({ page }) => {
    const results = await page.evaluate(async () => {
      const audio = window.RiftAudio;
      await audio.setActive(true, 1);

      // Simulate pagehide
      window.dispatchEvent(new PageTransitionEvent('pagehide', { persisted: true }));
      // Simulate pageshow
      window.dispatchEvent(new PageTransitionEvent('pageshow', { persisted: true }));

      const beforeRecovery = {
        active: audio.active,
        isSilent: audio.isSilent(),
      };

      // Call recover
      const recoverResult = await audio.recover(1);

      const afterRecovery = {
        active: audio.active,
        isSilent: audio.isSilent(),
        contextState: audio.context?.state,
      };

      return {
        beforeRecovery,
        recoverResult,
        afterRecovery,
      };
    });

    expect(results.beforeRecovery.active).toBe(false);
    expect(results.beforeRecovery.isSilent).toBe(true);
    expect(results.recoverResult).toBe(true);
    expect(results.afterRecovery.active).toBe(true);
    expect(results.afterRecovery.isSilent).toBe(false);
    expect(results.afterRecovery.contextState).toBe('running');
  });
});
