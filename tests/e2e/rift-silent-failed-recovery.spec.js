const { test, expect } = require('@playwright/test');

test.describe('Keep audio silent after a failed recovery request (Proposal 0192)', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/abyss/rift');
    await expect(page.locator('#rift-start')).toBeEnabled();
  });

  test('RiftAudio.recover returns false and keeps audio silent when unlock fails', async ({ page }) => {
    const results = await page.evaluate(async () => {
      const audio = window.RiftAudio;
      // Force unlock to fail by mocking a failed context resume
      const originalUnlock = audio.unlock;
      audio.unlock = async () => false;

      const before = audio.played;
      const recovered = await audio.recover(0);
      const isSilent = audio.isSilent();
      const playAttempt = audio.play('ui', 0);
      const stepAttempt = audio.step?.('stone', 0);
      const after = audio.played;

      // Restore original unlock
      audio.unlock = originalUnlock;

      return {
        recovered,
        isSilent,
        active: audio.active,
        playAttempt,
        playedDelta: after - before,
        voices: audio.voices,
      };
    });

    expect(results.recovered).toBe(false);
    expect(results.isSilent).toBe(true);
    expect(results.active).toBe(false);
    expect(results.playAttempt).toBe(false);
    expect(results.playedDelta).toBe(0);
    expect(results.voices).toBe(0);
  });

  test('RiftAudio.setActive(true) remains silent when audio unlock fails', async ({ page }) => {
    const results = await page.evaluate(async () => {
      const audio = window.RiftAudio;
      const originalUnlock = audio.unlock;
      audio.unlock = async () => false;

      const activated = await audio.setActive(true, 1);
      const isSilent = audio.isSilent();
      const bossMusic = audio.bossMusicActive;

      audio.unlock = originalUnlock;

      return {
        activated,
        isSilent,
        active: audio.active,
        bossMusic,
        voices: audio.voices,
      };
    });

    expect(results.activated).toBe(false);
    expect(results.isSilent).toBe(true);
    expect(results.active).toBe(false);
    expect(results.bossMusic).toBe(false);
    expect(results.voices).toBe(0);
  });

  test('audio remains completely silent when an expedition recovery request fails', async ({ page }) => {
    // 1. Enter expedition and ensure audio starts
    await page.locator('#rift-start').click();
    await expect(page.locator('#rift-overlay')).toBeHidden();

    // 2. Intercept subsequent POST to fail, forcing recovery state
    await page.route('**/api/abyss/rift', async route => {
      if (route.request().method() === 'POST') {
        await route.abort('failed');
      } else {
        await route.continue();
      }
    });

    // Press attack key to trigger a post
    await page.keyboard.press('KeyJ');

    // Overlay should appear with "Recover expedition"
    await expect(page.locator('#rift-overlay-title')).toHaveText('Your expedition is saved.');
    await expect(page.locator('#rift-start')).toHaveText('Recover expedition');

    // 3. Now intercept GET recovery request to fail as well
    await page.unroute('**/api/abyss/rift');
    await page.route('**/api/abyss/rift', async route => {
      await route.abort('failed');
    });

    // Click "Recover expedition"
    await page.locator('#rift-start').click();

    // Recovery failed: button transitions to "Retry loading"
    await expect(page.locator('#rift-start')).toHaveText('Retry loading');

    // Verify audio state remains completely silent
    const audioState = await page.evaluate(() => {
      const audio = window.RiftAudio;
      return {
        isSilent: audio.isSilent(),
        active: audio.active,
        bossMusic: audio.bossMusicActive,
        voices: audio.voices,
        contextState: audio.context?.state,
      };
    });

    expect(audioState.isSilent).toBe(true);
    expect(audioState.active).toBe(false);
    expect(audioState.bossMusic).toBe(false);
    expect(audioState.voices).toBe(0);
  });
});
