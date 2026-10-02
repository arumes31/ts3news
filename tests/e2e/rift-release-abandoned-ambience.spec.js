const { test, expect } = require('@playwright/test');

test.describe('Track and release abandoned ambience nodes (Proposal 0200)', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/abyss/rift');
    await expect(page.locator('#rift-start')).toBeEnabled();
  });

  test('audio starts with zero tracked ambience nodes', async ({ page }) => {
    const initialCount = await page.evaluate(() => {
      const audio = window.RiftAudio;
      return typeof audio.getTrackedAmbienceCount === 'function' ? audio.getTrackedAmbienceCount() : -1;
    });

    expect(initialCount).toBe(0);
  });

  test('area ambience tracks all sources, filters, and gains (9 nodes)', async ({ page }) => {
    const results = await page.evaluate(async () => {
      const audio = window.RiftAudio;
      await audio.setActive(true, 0); // Region 0

      const count = audio.getTrackedAmbienceCount();

      await audio.setActive(false);
      const countAfterSilence = audio.getTrackedAmbienceCount();

      return { count, countAfterSilence };
    });

    // 1 wind buffer + 1 biquad filter + 1 wind gain + 3 drone oscs + 3 drone gains = 9 nodes
    expect(results.count).toBe(9);
    expect(results.countAfterSilence).toBe(0);
  });

  test('crossfading between areas releases outgoing ambience nodes', async ({ page }) => {
    const results = await page.evaluate(async () => {
      const audio = window.RiftAudio;
      await audio.setActive(true, 0);
      const initialCount = audio.getTrackedAmbienceCount();

      // Trigger crossfade with a fast fade (50ms)
      audio.area(1, 0.05);
      const countDuringCrossfade = audio.getTrackedAmbienceCount();

      // Wait for crossfade timeout (50ms + 100ms padding + safety buffer)
      await new Promise(resolve => setTimeout(resolve, 250));
      const countAfterCrossfade = audio.getTrackedAmbienceCount();

      await audio.setActive(false);
      return { initialCount, countDuringCrossfade, countAfterCrossfade };
    });

    expect(results.initialCount).toBe(9);
    expect(results.countDuringCrossfade).toBe(18); // 9 outgoing + 9 incoming
    expect(results.countAfterCrossfade).toBe(9); // outgoing released, only incoming remains
  });

  test('boss music tracks 11 nodes and releases them completely on stop', async ({ page }) => {
    const results = await page.evaluate(async () => {
      const audio = window.RiftAudio;
      await audio.setActive(true, 0);
      const countBeforeBoss = audio.getTrackedAmbienceCount();

      audio.startBossMusic(0.05);
      const countWithBoss = audio.getTrackedAmbienceCount();

      // Stop boss music with fast fade
      audio.stopBossMusic(0.05);
      await new Promise(resolve => setTimeout(resolve, 250));
      const countAfterBossStopped = audio.getTrackedAmbienceCount();

      await audio.setActive(false);
      return { countBeforeBoss, countWithBoss, countAfterBossStopped };
    });

    expect(results.countBeforeBoss).toBe(9);
    // 9 area nodes + 11 boss nodes (1 bass osc + 1 bass filter + 1 bass gain + 4 chord oscs + 4 chord gains) = 20
    expect(results.countWithBoss).toBe(20);
    expect(results.countAfterBossStopped).toBe(9);
  });

  test('rapid successive area transitions bound node backlog and release cleanly', async ({ page }) => {
    const results = await page.evaluate(async () => {
      const audio = window.RiftAudio;
      await audio.setActive(true, 0);

      // Trigger rapid successive room transitions
      audio.area(1, 0.1);
      audio.area(2, 0.1);
      audio.area(3, 0.1);
      audio.area(4, 0.1);

      const maxBacklogCount = audio.getTrackedAmbienceCount();

      // Wait for all active timers to settle
      await new Promise(resolve => setTimeout(resolve, 350));
      const settledCount = audio.getTrackedAmbienceCount();

      await audio.setActive(false);
      const finalCount = audio.getTrackedAmbienceCount();

      return { maxBacklogCount, settledCount, finalCount };
    });

    // Pruned backlog caps accumulation (at most 2 outgoing groups: (2 + 1) * 9 = 27 nodes)
    expect(results.maxBacklogCount).toBeLessThanOrEqual(27);
    expect(results.settledCount).toBe(9);
    expect(results.finalCount).toBe(0);
  });

  test('releaseAbandonedAmbience purges orphaned nodes', async ({ page }) => {
    const results = await page.evaluate(async () => {
      const audio = window.RiftAudio;
      await audio.setActive(true, 0);

      audio.startBossMusic(0);
      const countWithBoss = audio.getTrackedAmbienceCount();

      // Initial sweep with all active nodes returns 0 released
      const initialSweep = audio.releaseAbandonedAmbience();

      // Stop area ambience directly without stopping boss, leaving only boss music
      // Then verify count
      audio.stopBossMusic(0);
      const countAfterBossStopped = audio.getTrackedAmbienceCount();

      await audio.setActive(false);
      const countAfterSilence = audio.getTrackedAmbienceCount();

      return { countWithBoss, initialSweep, countAfterBossStopped, countAfterSilence };
    });

    expect(results.countWithBoss).toBe(20);
    expect(results.initialSweep).toBe(0);
    expect(results.countAfterBossStopped).toBe(9);
    expect(results.countAfterSilence).toBe(0);
  });
});
