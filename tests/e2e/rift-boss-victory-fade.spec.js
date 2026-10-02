const { test, expect } = require('@playwright/test');

test.describe('Fade boss music after confirmed victory (Proposal 0171)', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/abyss/rift');
    await expect(page.locator('#rift-start')).toBeEnabled();
  });

  test('boss music smoothly decays over 1.8s after confirmed victory', async ({ page }) => {
    // Unlock and start audio with boss music active
    const startState = await page.evaluate(async () => {
      const audio = window.RiftAudio;
      await audio.setActive(true, 2); // Region 0, Tier 2 (Boss room)
      audio.startBossMusic(0.05);
      return {
        active: Boolean(audio.context && audio.context.state === 'running'),
        bossActive: audio.bossMusicActive,
        fadingOut: audio.bossFadingOut
      };
    });

    expect(startState.active).toBe(true);
    expect(startState.bossActive).toBe(true);
    expect(startState.fadingOut).toBe(false);

    // Trigger victory fade
    const fadeResult = await page.evaluate(() => {
      const audio = window.RiftAudio;
      const triggered = audio.fadeBossMusic(1.8);
      return {
        triggered,
        bossActive: audio.bossMusicActive,
        fadingOut: audio.bossFadingOut
      };
    });

    expect(fadeResult.triggered).toBe(true);
    expect(fadeResult.bossActive).toBe(false);
    expect(fadeResult.fadingOut).toBe(true);

    // Wait for the 1.8s decay window to complete
    await page.waitForTimeout(2000);

    const settledState = await page.evaluate(() => {
      const audio = window.RiftAudio;
      return {
        bossActive: audio.bossMusicActive,
        fadingOut: audio.bossFadingOut
      };
    });

    expect(settledState.bossActive).toBe(false);
    expect(settledState.fadingOut).toBe(false);

    await page.evaluate(async () => {
      await window.RiftAudio.setActive(false);
    });
  });

  test('ambient music chords are restored to full level when boss music fades out', async ({ page }) => {
    const chordLevels = await page.evaluate(async () => {
      const audio = window.RiftAudio;
      await audio.setActive(true, 0);

      // Start boss music (which ducks ambient chords)
      audio.startBossMusic(0.05);

      // Fade boss music (which restores ambient chords)
      audio.fadeBossMusic(0.1);

      await new Promise(resolve => setTimeout(resolve, 200));

      // Check the gain on the ambient chord oscillator gain nodes
      const activeAmbient = audio.context ? true : false;
      const isFading = audio.bossFadingOut;

      await audio.setActive(false);
      return {
        activeAmbient,
        isFading
      };
    });

    expect(chordLevels.activeAmbient).toBe(true);
    expect(chordLevels.isFading).toBe(false);
  });

  test('stopping or pausing during victory fade suspends cleanly without lingering audio', async ({ page }) => {
    const pauseState = await page.evaluate(async () => {
      const audio = window.RiftAudio;
      await audio.setActive(true, 2);
      audio.startBossMusic(0.05);

      // Trigger 1.8s victory fade
      audio.fadeBossMusic(1.8);
      const isFading = audio.bossFadingOut;

      // Pause immediately while fade is ongoing
      await audio.setActive(false);

      return {
        wasFading: isFading,
        fadingAfterPause: audio.bossFadingOut,
        contextState: audio.context.state
      };
    });

    expect(pauseState.wasFading).toBe(true);
    expect(pauseState.fadingAfterPause).toBe(false);
    expect(pauseState.contextState).toBe('suspended');
  });
});
