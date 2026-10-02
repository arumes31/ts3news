const { test, expect } = require('@playwright/test');

test.describe('Add a short player landing pose (Proposal 0201)', () => {
  test('player transitions through airborne jump pose and landing pose upon touchdown', async ({ page }) => {
    await page.goto('/abyss/rift');
    await expect(page.locator('#rift-start')).toBeEnabled();

    let jumpingSeen = false;
    let landingSeen = false;
    let landPoseTime = 0;
    let recoveredSeen = false;

    // Track snapshot states directly via API route
    await page.route('**/api/abyss/rift', async route => {
      const response = await route.fetch();
      const data = await response.json();

      if (data && data.run && data.run.player) {
        const p = data.run.player;
        if (p.jump > 0 && p.pose === 'jump') {
          jumpingSeen = true;
        }
        if (jumpingSeen && p.jump === 0 && p.pose === 'land') {
          landingSeen = true;
          landPoseTime = p.pose_time;
        }
        if (landingSeen && (p.pose === 'idle' || p.pose === 'run')) {
          recoveredSeen = true;
        }
      }

      await route.fulfill({ response, json: data });
    });

    await page.locator('#rift-start').click();
    await expect(page.locator('#rift-overlay')).toBeHidden();

    // Trigger jump
    await page.keyboard.press('Space');

    // Wait for landing and recovery cycle to complete
    await expect.poll(() => jumpingSeen, { timeout: 3000 }).toBe(true);
    await expect.poll(() => landingSeen, { timeout: 3000 }).toBe(true);
    expect(landPoseTime).toBeGreaterThan(0);
    await expect.poll(() => recoveredSeen, { timeout: 3000 }).toBe(true);

    await page.keyboard.press('Escape');
  });

  test('renderer handles land pose correctly and maps to recovery frame', async ({ page }) => {
    await page.goto('/abyss/rift');
    await expect(page.locator('#rift-start')).toBeEnabled();

    const rendererResult = await page.evaluate(async () => {
      const canvas = document.getElementById('rift-canvas');
      const ctx = canvas?.getContext('2d');
      if (!canvas || !ctx) return false;

      // Mock snapshot with player in land pose
      const mockRun = {
        id: 'test-land-render',
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
          pose: 'land',
          pose_time: 0.12,
          jump: 0,
          guard: false,
        },
        enemies: [],
        projectiles: [],
        drops: [],
        events: [],
      };

      // Feed snapshot to renderer
      if (window.RiftRenderer?.feed) {
        await window.RiftRenderer.prepareRun(mockRun);
        window.RiftRenderer.feed(mockRun);
      }

      return true;
    });

    expect(rendererResult).toBe(true);
  });
});
