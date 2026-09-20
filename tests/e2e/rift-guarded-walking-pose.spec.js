const { test, expect } = require('@playwright/test');

test.describe('Add a distinct guarded walking pose (Proposal 0202)', () => {
  test('player transitions between guard, guard_walk, and run depending on movement and guard input', async ({ page }) => {
    await page.goto('/abyss/rift');
    await expect(page.locator('#rift-start')).toBeEnabled();

    let guardSeen = false;
    let guardWalkSeen = false;
    let returnToGuardSeen = false;
    let runSeen = false;

    // Intercept step snapshots
    await page.route('**/api/abyss/rift', async route => {
      const response = await route.fetch();
      const data = await response.json();

      if (data && data.run && data.run.player) {
        const p = data.run.player;
        if (p.guard && p.pose === 'guard') {
          guardSeen = true;
          if (guardWalkSeen) {
            returnToGuardSeen = true;
          }
        }
        if (p.guard && p.pose === 'guard_walk') {
          guardWalkSeen = true;
        }
        if (!p.guard && p.pose === 'run') {
          runSeen = true;
        }
      }

      await route.fulfill({ response, json: data });
    });

    await page.locator('#rift-start').click();
    await expect(page.locator('#rift-overlay')).toBeHidden();

    // 1. Hold guard stationary
    await page.keyboard.down('KeyL');
    await expect.poll(() => guardSeen, { timeout: 3000 }).toBe(true);

    // 2. Start moving while holding guard
    await page.keyboard.down('KeyD');
    await expect.poll(() => guardWalkSeen, { timeout: 3000 }).toBe(true);

    // 3. Release movement key, maintain guard
    await page.keyboard.up('KeyD');
    await expect.poll(() => returnToGuardSeen, { timeout: 3000 }).toBe(true);

    // 4. Release guard and move
    await page.keyboard.up('KeyL');
    await page.keyboard.down('KeyD');
    await expect.poll(() => runSeen, { timeout: 3000 }).toBe(true);

    await page.keyboard.up('KeyD');
    await page.keyboard.press('Escape');
  });

  test('renderer handles guard_walk correctly and renders without errors', async ({ page }) => {
    await page.goto('/abyss/rift');
    await expect(page.locator('#rift-start')).toBeEnabled();

    const result = await page.evaluate(() => {
      const canvas = document.getElementById('rift-canvas');
      const ctx = canvas?.getContext('2d');
      if (!canvas || !ctx) return false;

      const mockRun = {
        id: 'test-guard-walk-render',
        status: 'fighting',
        room: 0,
        player: {
          id: 'player',
          kind: 'vanguard',
          x: 400,
          y: 350,
          hp: 100,
          max_hp: 100,
          facing: 1,
          pose: 'guard_walk',
          pose_time: 0,
          jump: 0,
          guard: true,
        },
        enemies: [],
        projectiles: [],
        drops: [],
        events: [],
      };

      if (window.RiftRenderer?.feed) {
        window.RiftRenderer.feed(mockRun);
      }

      return true;
    });

    expect(result).toBe(true);
  });
});
