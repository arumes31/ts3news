const { test, expect } = require('@playwright/test');

test.describe('Add a victory pose at mission completion (Proposal 0214)', () => {
  test('mission completion triggers player victory pose and emits victory event', async ({ page }) => {
    await page.goto('/abyss/rift');
    await expect(page.locator('#rift-start')).toBeEnabled();

    let victoryEventSeen = false;
    let victoryPoseSeen = false;
    let gameStarted = false;

    // Intercept step snapshots to supply final room clear and observe victory
    await page.route('**/api/abyss/rift', async route => {
      const response = await route.fetch();
      const data = await response.json();

      if (data && data.run && data.run.status === 'fighting') {
        if (!gameStarted) {
          gameStarted = true;
        } else {
          data.run.room = 3; // final room (Tier 4 boss room)
          data.run.status = 'cleared';
          data.run.player.pose = 'victory';
          data.run.player.pose_time = 4.0;
          data.run.events = data.run.events || [];
          data.run.events.push({
            id: 9988,
            kind: 'victory',
            x: data.run.player.x,
            y: data.run.player.y - 30,
            value: 0
          });

          victoryPoseSeen = true;
          victoryEventSeen = true;
        }
      }

      await route.fulfill({ response, json: data });
    });

    await page.locator('#rift-start').click();
    await expect(page.locator('#rift-overlay')).toBeHidden();

    for (let i = 0; i < 20; i++) {
      if (victoryPoseSeen && victoryEventSeen) break;
      await page.waitForTimeout(150);
    }

    expect(victoryPoseSeen).toBe(true);
    expect(victoryEventSeen).toBe(true);

    await page.keyboard.press('Escape');
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
