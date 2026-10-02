const { test, expect } = require('@playwright/test');

test.describe('Add a brief shield-absorption shimmer (Proposal 0208)', () => {
  test('barrier absorbing damage emits shield_absorb event during combat', async ({ page }) => {
    await page.goto('/abyss/rift');
    await expect(page.locator('#rift-start')).toBeEnabled();

    let shieldAbsorbEventSeen = false;

    // Intercept step snapshots to observe shield_absorb event
    await page.route('**/api/abyss/rift', async route => {
      const response = await route.fetch();
      const data = await response.json();

      if (data && data.run && data.run.events) {
        for (const ev of data.run.events) {
          if (ev.kind === 'shield_absorb') {
            shieldAbsorbEventSeen = true;
          }
        }
      }

      await route.fulfill({ response, json: data });
    });

    await page.locator('#rift-start').click();
    await expect(page.locator('#rift-overlay')).toBeHidden();

    // Cast signature 0 (builder: shield/barrier for Vanguard) using KeyQ
    await page.keyboard.press('KeyQ');
    await page.waitForTimeout(100);

    // Walk toward enemies to receive an attack on the active barrier
    await page.keyboard.down('KeyD');
    for (let i = 0; i < 25; i++) {
      if (shieldAbsorbEventSeen) break;
      if (i % 3 === 0) {
        await page.keyboard.press('KeyQ');
      }
      await page.waitForTimeout(150);
    }
    await page.keyboard.up('KeyD');

    // If live combat timing varied across machines, ensure event was observed or feed snapshot
    if (!shieldAbsorbEventSeen) {
      await page.keyboard.press('KeyQ');
      await page.waitForTimeout(500);
    }

    expect(shieldAbsorbEventSeen).toBe(true);

    await page.keyboard.press('Escape');
  });

  test('renderer handles shield_absorb shimmer visual accent in normal and reduced motion modes', async ({ page }) => {
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
        id: 'test-shimmer-eval',
        status: 'fighting',
        room: 0,
        barrier: 35,
        player: {
          id: 'player',
          x: 400,
          y: 350,
          hp: 200,
          max_hp: 200,
          mana: 100,
          facing: 1,
          pose: 'hit',
          pose_time: 0.15,
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
          { id: 1, kind: 'shield_absorb', x: 400, y: 320, value: 15 }
        ]
      };

      // 1. Normal mode: feed mock snapshot with shield_absorb event
      window.RiftRenderer.reduced = false;
      await window.RiftRenderer.prepareRun(mockRun);
      window.RiftRenderer.feed(mockRun);

      await new Promise(r => requestAnimationFrame(() => requestAnimationFrame(r)));

      const normalShimmer = window.RiftRenderer.lastShieldShimmer;

      // 2. Reduced motion mode: feed again and verify graceful handling
      window.RiftRenderer.reduced = true;
      const mockRun2 = {
        ...mockRun,
        events: [
          { id: 2, kind: 'shield_absorb', x: 410, y: 320, value: 20 }
        ]
      };
      await window.RiftRenderer.prepareRun(mockRun2);
      window.RiftRenderer.feed(mockRun2);

      await new Promise(r => requestAnimationFrame(() => requestAnimationFrame(r)));

      const reducedShimmer = window.RiftRenderer.lastShieldShimmer;

      return {
        normalShimmer,
        reducedShimmer
      };
    });

    expect(checkResult).not.toBeNull();
    expect(checkResult.normalShimmer).toBeDefined();
    expect(checkResult.normalShimmer.x).toBe(400);
    expect(checkResult.normalShimmer.value).toBe(15);
    expect(checkResult.reducedShimmer).toBeDefined();
    expect(checkResult.reducedShimmer.x).toBe(410);
    expect(checkResult.reducedShimmer.value).toBe(20);
  });
});
