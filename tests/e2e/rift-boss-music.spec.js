const { test, expect } = require('@playwright/test');

test.describe('Crossfade boss music at encounter start (Proposal 0170)', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/abyss/rift');
    await expect(page.locator('#rift-start')).toBeEnabled();
  });

  test('boss music crossfades in smoothly at encounter start over 1.5 seconds', async ({ page }) => {
    const initialState = await page.evaluate(async () => {
      const audio = window.RiftAudio;
      await audio.setActive(true, 0);
      return {
        active: Boolean(audio.context && audio.context.state === 'running'),
        bossActive: audio.bossMusicActive,
        bossCrossfading: audio.bossCrossfading
      };
    });

    expect(initialState.active).toBe(true);
    expect(initialState.bossActive).toBe(false);
    expect(initialState.bossCrossfading).toBe(false);

    // Trigger boss music crossfade
    const startResult = await page.evaluate(() => {
      const audio = window.RiftAudio;
      const started = audio.startBossMusic();
      return {
        started,
        bossActive: audio.bossMusicActive,
        bossCrossfading: audio.bossCrossfading
      };
    });

    expect(startResult.started).toBe(true);
    expect(startResult.bossActive).toBe(true);
    expect(startResult.bossCrossfading).toBe(true);

    // Wait for the 1.5s crossfade duration to complete
    await page.waitForTimeout(1700);

    const settledResult = await page.evaluate(() => {
      const audio = window.RiftAudio;
      return {
        bossActive: audio.bossMusicActive,
        bossCrossfading: audio.bossCrossfading
      };
    });

    expect(settledResult.bossActive).toBe(true);
    expect(settledResult.bossCrossfading).toBe(false);

    await page.evaluate(async () => {
      await window.RiftAudio.setActive(false);
    });
  });

  test('boss music nodes route through the dedicated music bus', async ({ page }) => {
    await page.addInitScript(() => {
      window.audioEdges = [];
      const original = AudioNode.prototype.connect;
      AudioNode.prototype.connect = function (target, ...args) {
        window.audioEdges.push([this, target]);
        return original.call(this, target, ...args);
      };
    });

    await page.goto('/abyss/rift');

    const routing = await page.evaluate(async () => {
      const audio = window.RiftAudio;
      audio.set('music', 0.45);
      await audio.setActive(true, 0);

      window.audioEdges = [];
      audio.startBossMusic(0.05);

      // Find all gain nodes connecting to a bus with gain value 0.45 (music bus)
      const musicInputs = window.audioEdges.filter(([source, target]) =>
        source instanceof GainNode && target instanceof GainNode && target.gain.value === Math.fround(0.45)
      );

      await audio.setActive(false);
      return {
        bossMusicCount: musicInputs.length
      };
    });

    // 1 bassline + 4 harmonic tension oscillators = 5 nodes connected to music bus
    expect(routing.bossMusicCount).toBeGreaterThanOrEqual(5);
  });

  test('boss music cleanly stops and suspends audio context on pause', async ({ page }) => {
    const pauseState = await page.evaluate(async () => {
      const audio = window.RiftAudio;
      await audio.setActive(true, 0);
      audio.startBossMusic();
      const beforePause = {
        bossActive: audio.bossMusicActive,
        bossCrossfading: audio.bossCrossfading
      };

      await audio.setActive(false);
      return {
        beforePause,
        afterPauseBossActive: audio.bossMusicActive,
        afterPauseCrossfading: audio.bossCrossfading,
        contextState: audio.context.state
      };
    });

    expect(pauseState.beforePause.bossActive).toBe(true);
    expect(pauseState.beforePause.bossCrossfading).toBe(true);
    expect(pauseState.afterPauseBossActive).toBe(false);
    expect(pauseState.afterPauseCrossfading).toBe(false);
    expect(pauseState.contextState).toBe('suspended');
  });
});
