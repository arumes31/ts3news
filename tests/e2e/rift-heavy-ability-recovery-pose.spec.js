const { test, expect } = require('@playwright/test');

test.describe('Add a recovery pose after heavy abilities (Proposal 0207)', () => {
  test('casting heavy ability transitions into recovery pose and emits heavy_recovery event in combat', async ({ page }) => {
    await page.goto('/abyss/rift');
    await expect(page.locator('#rift-start')).toBeEnabled();

    let recoveryPoseSeen = false;
    let heavyRecoveryEventSeen = false;

    // Intercept step snapshots to observe recovery pose and heavy_recovery event
    await page.route('**/api/abyss/rift', async route => {
      const response = await route.fetch();
      const data = await response.json();

      if (data && data.run) {
        if (data.run.player && data.run.player.pose === 'recovery') {
          recoveryPoseSeen = true;
        }
        if (data.run.events) {
          for (const ev of data.run.events) {
            if (ev.kind === 'heavy_recovery') {
              heavyRecoveryEventSeen = true;
            }
          }
        }
      }

      await route.fulfill({ response, json: data });
    });

    await page.locator('#rift-start').click();
    await expect(page.locator('#rift-overlay')).toBeHidden();

    // Cast ultimate with KeyR (heavy ability) or build with KeyQ and finish with KeyE
    await page.keyboard.press('KeyR');
    await page.waitForTimeout(600);

    for (let i = 0; i < 6; i++) {
      if (recoveryPoseSeen || heavyRecoveryEventSeen) break;
      await page.keyboard.press('KeyQ');
      await page.waitForTimeout(100);
      await page.keyboard.press('KeyE');
      await page.waitForTimeout(450);
    }

    expect(recoveryPoseSeen || heavyRecoveryEventSeen).toBe(true);

    await page.keyboard.press('Escape');
  });

  test('renderer handles recovery pose and heavy_recovery visual accent in normal and reduced motion modes', async ({ page }) => {
    await page.goto('/abyss/rift');
    await expect(page.locator('#rift-start')).toBeEnabled();

    const checkResult = await page.evaluate(async () => {
      const canvas = document.getElementById('rift-canvas');
      const ctx = canvas?.getContext('2d');
      if (!canvas || !ctx || !window.RiftRenderer) return null;

      if (window.RiftRenderer.ready) {
        await window.RiftRenderer.ready;
      }

      const mockRun = {
        id: 'test-recovery-eval',
        status: 'fighting',
        room: 0,
        player: {
          id: 'player',
          x: 400,
          y: 350,
          hp: 200,
          max_hp: 200,
          mana: 100,
          facing: 1,
          pose: 'recovery',
          pose_time: 0.20,
          guard: false,
          jump: 0
        },
        build: {
          class: 'vanguard'
        },
        enemies: [],
        projectiles: [],
        drops: [],
        events: [
          { id: 1, kind: 'heavy_recovery', x: 400, y: 315, value: 0 }
        ]
      };

      // 1. Normal mode: feed mock snapshot with recovery pose and heavy_recovery event
      window.RiftRenderer.reduced = false;
      window.RiftRenderer.feed(mockRun);

      await new Promise(r => requestAnimationFrame(() => requestAnimationFrame(r)));

      const normalAccent = window.RiftRenderer.lastHeavyRecovery;

      // 2. Reduced motion mode: feed again and verify graceful handling
      window.RiftRenderer.reduced = true;
      const mockRun2 = {
        ...mockRun,
        events: [
          { id: 2, kind: 'heavy_recovery', x: 410, y: 315, value: 0 }
        ]
      };
      window.RiftRenderer.feed(mockRun2);

      await new Promise(r => requestAnimationFrame(() => requestAnimationFrame(r)));

      const reducedAccent = window.RiftRenderer.lastHeavyRecovery;

      return {
        normalAccent,
        reducedAccent
      };
    });

    expect(checkResult).not.toBeNull();
    expect(checkResult.normalAccent).toBeDefined();
    expect(checkResult.normalAccent.x).toBe(400);
    expect(checkResult.reducedAccent).toBeDefined();
    expect(checkResult.reducedAccent.x).toBe(410);
  });
});
