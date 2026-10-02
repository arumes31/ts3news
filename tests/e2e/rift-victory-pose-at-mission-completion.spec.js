const { test, expect } = require('@playwright/test');

test.describe('Add a victory pose at mission completion (Proposal 0214)', () => {
  test('mission completion triggers player victory pose and emits victory event', async ({ page }) => {
    await page.goto('/abyss/rift?scenario=victory-final');
    await expect(page.locator('#rift-start')).toBeEnabled();
    await page.locator('#rift-auto').uncheck();
    await page.locator('#rift-start').click();
    await expect(page.locator('#rift-overlay')).toBeHidden();
    const victory = page.waitForResponse(async response => {
      if (!response.url().endsWith('/api/abyss/rift') || response.request().method() !== 'POST') return false;
      const {run} = await response.json();
      return run?.events?.some(event => event.kind === 'victory');
    });
    await page.keyboard.down('KeyJ');
    let cleared;
    try { cleared = (await (await victory).json()).run; }
    finally { await page.keyboard.up('KeyJ'); }
    expect(cleared.status).toBe('cleared');
    expect(cleared.player.pose).toBe('victory');
    expect(cleared.player.pose_time).toBeGreaterThan(0);
    const completion = page.waitForResponse(response =>
      response.url().endsWith('/api/abyss/rift') && response.request().postDataJSON()?.kind === 'next');
    await page.locator('#rift-next').click();
    const {run} = await (await completion).json();
    expect(run.status).toBe('complete');
    expect(run.player.pose).toBe('victory');
    expect(run.player.pose_time).toBeGreaterThan(0);
    await expect(page.locator('#rift-result-heading')).toContainText('Mission 1 Cleared:');

  });

  test('renderer handles victory pose aura and victory herald in normal and reduced motion modes', async ({ page }) => {
    await page.goto('/abyss/rift');
    await expect(page.locator('#rift-start')).toBeEnabled();

    const checkResult = await page.evaluate(async () => {
      const canvas = document.getElementById('rift-canvas');
      const ctx = canvas?.getContext('2d');
      if (!canvas || !ctx || !window.RiftRenderer) return null;

      if (window.RiftRenderer.ready) {
        await window.RiftRenderer.ready;
      }

      const mockVictoryRun = {
        id: 'test-victory-eval',
        status: 'cleared',
        room: 3,
        player: {
          id: 'player',
          x: 250,
          y: 350,
          hp: 200,
          max_hp: 200,
          facing: 1,
          pose: 'victory',
          pose_time: 3.5,
          jump: 0
        },
        build: { class: 'vanguard' },
        enemies: [],
        projectiles: [],
        drops: [],
        events: [
          { id: 9911, kind: 'victory', x: 250, y: 320, value: 0 }
        ]
      };

      // 1. Normal motion mode: verify victory pose telemetry and effect rendering
      window.RiftRenderer.reduced = false;
      await window.RiftRenderer.prepareRun(mockVictoryRun);
      window.RiftRenderer.feed(mockVictoryRun);
      await new Promise(r => requestAnimationFrame(() => requestAnimationFrame(r)));
      const normalPose = window.RiftRenderer.lastVictoryPose;
      const normalEvent = window.RiftRenderer.lastVictoryEvent;

      // 2. Reduced motion mode: verify graceful non-animated aura
      window.RiftRenderer.reduced = true;
      await window.RiftRenderer.prepareRun(mockVictoryRun);
      window.RiftRenderer.feed(mockVictoryRun);
      await new Promise(r => requestAnimationFrame(() => requestAnimationFrame(r)));
      const reducedPose = window.RiftRenderer.lastVictoryPose;
      const reducedEvent = window.RiftRenderer.lastVictoryEvent;

      return {
        normalPose,
        normalEvent,
        reducedPose,
        reducedEvent
      };
    });

    expect(checkResult).not.toBeNull();

    // Normal mode assertions
    expect(checkResult.normalPose).toBeDefined();
    expect(checkResult.normalPose.pose).toBe('victory');
    expect(checkResult.normalPose.reduced).toBe(false);
    expect(checkResult.normalEvent).toBeDefined();
    expect(checkResult.normalEvent.reduced).toBe(false);

    // Reduced motion assertions
    expect(checkResult.reducedPose).toBeDefined();
    expect(checkResult.reducedPose.pose).toBe('victory');
    expect(checkResult.reducedPose.reduced).toBe(true);
    expect(checkResult.reducedEvent).toBeDefined();
    expect(checkResult.reducedEvent.reduced).toBe(true);
  });
});
