const { test, expect } = require('@playwright/test');

test.describe('Add directional hit-recoil offsets (Proposal 0203)', () => {
  test('enemy hit by player receives directional recoil_x offset away from player', async ({ page }) => {
    await page.goto('/abyss/rift');
    await expect(page.locator('#rift-start')).toBeEnabled();

    let enemyHitRecoilSeen = false;
    let enemyRecoilSignCorrect = false;

    // Intercept step snapshots to observe recoil
    await page.route('**/api/abyss/rift', async route => {
      const response = await route.fetch();
      const data = await response.json();

      if (data && data.run && data.run.player && data.run.enemies) {
        const player = data.run.player;
        for (const enemy of data.run.enemies) {
          if (enemy.recoil_x !== undefined && enemy.recoil_x !== 0) {
            enemyHitRecoilSeen = true;
            // Enemy should recoil away from player
            if (enemy.x > player.x && enemy.recoil_x > 0) {
              enemyRecoilSignCorrect = true;
            } else if (enemy.x < player.x && enemy.recoil_x < 0) {
              enemyRecoilSignCorrect = true;
            }
          }
        }
      }

      await route.fulfill({ response, json: data });
    });

    await page.locator('#rift-start').click();
    await expect(page.locator('#rift-overlay')).toBeHidden();

    // Approach enemies and attack
    await page.keyboard.down('KeyD');
    await page.waitForTimeout(400);
    await page.keyboard.up('KeyD');

    for (let i = 0; i < 15; i++) {
      await page.keyboard.press('KeyJ');
      await page.waitForTimeout(100);
      if (enemyHitRecoilSeen && enemyRecoilSignCorrect) break;
    }

    expect(enemyHitRecoilSeen).toBe(true);
    expect(enemyRecoilSignCorrect).toBe(true);

    await page.keyboard.press('Escape');
  });

  test('renderer handles directional hit recoil correctly and respects reduced motion settings', async ({ page }) => {
    await page.goto('/abyss/rift');
    await expect(page.locator('#rift-start')).toBeEnabled();

    const checkResult = await page.evaluate(async () => {
      const canvas = document.getElementById('rift-canvas');
      const ctx = canvas?.getContext('2d');
      if (!canvas || !ctx || !window.RiftRenderer) return null;

      const mockRun = {
        id: 'test-recoil-render',
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
          pose: 'hit',
          pose_time: 0.15,
          jump: 0,
          guard: false,
          recoil_x: -9.0,
        },
        enemies: [
          {
            id: 'mob-1',
            kind: 'goblin',
            x: 450,
            y: 350,
            hp: 80,
            max_hp: 100,
            facing: -1,
            pose: 'hit',
            pose_time: 0.18,
            jump: 0,
            recoil_x: 10.0,
          }
        ],
        projectiles: [],
        drops: [],
        events: [],
      };

      // Normal mode: recoil offset applied to renderer.lastPlayerRecoil
      await window.RiftRenderer.prepareRun(mockRun);
      window.RiftRenderer.feed(mockRun);
      window.RiftRenderer.renderActor(mockRun.player);
      const normalRecoil = window.RiftRenderer.lastPlayerRecoil;

      // Reduced motion mode: recoil offset should be 0
      window.RiftRenderer.reduced = true;
      await window.RiftRenderer.prepareRun(mockRun);
      window.RiftRenderer.feed(mockRun);
      window.RiftRenderer.renderActor(mockRun.player);
      const reducedRecoil = window.RiftRenderer.lastPlayerRecoil;

      // Restore
      window.RiftRenderer.reduced = false;

      return { normalRecoil, reducedRecoil };
    });

    expect(checkResult).not.toBeNull();
    expect(checkResult.normalRecoil).toBe(-9.0);
    expect(checkResult.reducedRecoil).toBe(0);
  });
});
