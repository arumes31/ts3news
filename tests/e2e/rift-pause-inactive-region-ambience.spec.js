const { test, expect } = require('@playwright/test');

test.describe('Pause ambient emitters when their region is inactive (Proposal 0195)', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/abyss/rift');
    await expect(page.locator('#rift-start')).toBeEnabled();
  });

  test('ambient emitters are active by default and pause when region is deactivated', async ({ page }) => {
    const results = await page.evaluate(async () => {
      const audio = window.RiftAudio;
      await audio.setActive(true, 0); // Region 0

      const beforePause = {
        isRegionActive: audio.isRegionActive(0),
        isAmbiencePaused: audio.isAmbiencePaused(),
        gains: window.RiftAudio.ambientNodes ? [0.14, 0.017] : [],
      };

      // Pause region 0 ambience
      audio.setRegionActive(0, false, 0.02);

      // Wait 30ms for fade ramp
      await new Promise(r => setTimeout(r, 35));

      const afterPause = {
        isRegionActive: audio.isRegionActive(0),
        isAmbiencePaused: audio.isAmbiencePaused(),
      };

      // Unpause region 0 ambience
      audio.setRegionActive(0, true, 0.02);
      await new Promise(r => setTimeout(r, 35));

      const afterResume = {
        isRegionActive: audio.isRegionActive(0),
        isAmbiencePaused: audio.isAmbiencePaused(),
      };

      await audio.setActive(false);

      return { beforePause, afterPause, afterResume };
    });

    expect(results.beforePause.isRegionActive).toBe(true);
    expect(results.beforePause.isAmbiencePaused).toBe(false);

    expect(results.afterPause.isRegionActive).toBe(false);
    expect(results.afterPause.isAmbiencePaused).toBe(true);

    expect(results.afterResume.isRegionActive).toBe(true);
    expect(results.afterResume.isAmbiencePaused).toBe(false);
  });

  test('procedural ticks are suppressed while ambient emitters are paused', async ({ page }) => {
    const results = await page.evaluate(async () => {
      const audio = window.RiftAudio;
      await audio.setActive(true, 0);

      // Pause region 0
      audio.pauseRegionAmbience(0, 0.01);
      await new Promise(r => setTimeout(r, 20));

      const playedBefore = audio.played;

      // Force context time past next bird and run tick
      if (audio.context) {
        audio.tick();
      }

      const playedAfter = audio.played;
      const isPaused = audio.isAmbiencePaused();

      await audio.setActive(false);

      return {
        isPaused,
        playedDelta: playedAfter - playedBefore,
      };
    });

    expect(results.isPaused).toBe(true);
    expect(results.playedDelta).toBe(0);
  });

  test('entering an inactive region initializes its ambient emitters in a paused state', async ({ page }) => {
    const results = await page.evaluate(async () => {
      const audio = window.RiftAudio;
      // Mark region 2 (index 6..8) as inactive beforehand
      audio.setRegionActive(2, false);

      await audio.setActive(true, 6); // Area 6 = Region 2, Room 0

      const inactiveAreaState = {
        region: 2,
        isRegionActive: audio.isRegionActive(2),
        isAmbiencePaused: audio.isAmbiencePaused(),
      };

      // Now reactivate region 2
      audio.resumeRegionAmbience(2, 0.02);
      await new Promise(r => setTimeout(r, 35));

      const activeAreaState = {
        isRegionActive: audio.isRegionActive(2),
        isAmbiencePaused: audio.isAmbiencePaused(),
      };

      await audio.setActive(false);

      return { inactiveAreaState, activeAreaState };
    });

    expect(results.inactiveAreaState.isRegionActive).toBe(false);
    expect(results.inactiveAreaState.isAmbiencePaused).toBe(true);

    expect(results.activeAreaState.isRegionActive).toBe(true);
    expect(results.activeAreaState.isAmbiencePaused).toBe(false);
  });

  test('deactivating an unrelated region does not affect active emitters in current region', async ({ page }) => {
    const results = await page.evaluate(async () => {
      const audio = window.RiftAudio;
      await audio.setActive(true, 3); // Region 1

      // Deactivate region 0 and region 3
      audio.setRegionActive(0, false);
      audio.setRegionActive(3, false);

      const state = {
        currentRegion: 1,
        isCurrentActive: audio.isRegionActive(1),
        isOther0Active: audio.isRegionActive(0),
        isOther3Active: audio.isRegionActive(3),
        isAmbiencePaused: audio.isAmbiencePaused(),
      };

      await audio.setActive(false);

      return state;
    });

    expect(results.isCurrentActive).toBe(true);
    expect(results.isOther0Active).toBe(false);
    expect(results.isOther3Active).toBe(false);
    expect(results.isAmbiencePaused).toBe(false);
  });
});
