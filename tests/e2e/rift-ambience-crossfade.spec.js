const { test, expect } = require('@playwright/test');

test.describe('Crossfade ambience during region transitions (Proposal 0169)', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/abyss/rift');
    await expect(page.locator('#rift-start')).toBeEnabled();
  });

  test('audio system smoothly crossfades ambience and music during region transitions', async ({ page }) => {
    // Unlock and start audio in Region 0 (Area index 0)
    const initialState = await page.evaluate(async () => {
      const audio = window.RiftAudio;
      await audio.setActive(true, 0); // Region 0, Tier 0
      return {
        active: Boolean(audio.context && audio.context.state === 'running'),
        region: audio.currentRegion,
        crossfading: audio.crossfading,
        nodes: audio.context ? 4 : 0 // 1 wind + 3 chord oscillators
      };
    });

    expect(initialState.active).toBe(true);
    expect(initialState.region).toBe(0);
    expect(initialState.crossfading).toBe(false);

    // Transition to Region 1 (Area index 3)
    const transitionState = await page.evaluate(async () => {
      const audio = window.RiftAudio;
      // Trigger region transition from region 0 to region 1
      audio.area(3);
      return {
        region: audio.currentRegion,
        crossfading: audio.crossfading
      };
    });

    expect(transitionState.region).toBe(1);
    expect(transitionState.crossfading).toBe(true);

    // Wait for the 1.5s crossfade duration to complete
    await page.waitForTimeout(1700);

    const postTransitionState = await page.evaluate(() => {
      const audio = window.RiftAudio;
      return {
        region: audio.currentRegion,
        crossfading: audio.crossfading
      };
    });

    expect(postTransitionState.region).toBe(1);
    expect(postTransitionState.crossfading).toBe(false);

    await page.evaluate(async () => {
      await window.RiftAudio.setActive(false);
    });
  });

  test('crossfading cleanly stops and suspends audio context on pause without lingering sound', async ({ page }) => {
    const pauseState = await page.evaluate(async () => {
      const audio = window.RiftAudio;
      await audio.setActive(true, 0);
      // Trigger crossfade to region 2 (Area index 6)
      audio.area(6);
      const isCrossfading = audio.crossfading;

      // Pause immediately while crossfade is active
      await audio.setActive(false);
      return {
        crossfadingBeforePause: isCrossfading,
        crossfadingAfterPause: audio.crossfading,
        contextState: audio.context.state
      };
    });

    expect(pauseState.crossfadingBeforePause).toBe(true);
    expect(pauseState.crossfadingAfterPause).toBe(false);
    expect(pauseState.contextState).toBe('suspended');
  });

  test('same-region tier transitions execute shorter crossfades', async ({ page }) => {
    const tierFadeState = await page.evaluate(async () => {
      const audio = window.RiftAudio;
      await audio.setActive(true, 0); // Region 0, Tier 0
      audio.area(1); // Region 0, Tier 1 (within same region)
      return {
        region: audio.currentRegion,
        crossfading: audio.crossfading
      };
    });

    expect(tierFadeState.region).toBe(0);
    expect(tierFadeState.crossfading).toBe(true);

    // Same region fade is 0.6s, so at 850ms it should already be finished
    await page.waitForTimeout(850);

    const finished = await page.evaluate(() => window.RiftAudio.crossfading);
    expect(finished).toBe(false);

    await page.evaluate(async () => {
      await window.RiftAudio.setActive(false);
    });
  });
});
