const { test, expect } = require('@playwright/test');

test.describe('Add consistent enemy jump shadows (Proposal 0211)', () => {
  test('enemy in combat renders jump shadow pinned to ground floor', async ({ page }) => {
    await page.goto('/abyss/rift');
    await expect(page.locator('#rift-start')).toBeEnabled();

    let enemyShadowObserved = false;

    // Intercept step snapshots to supply an enemy with jump altitude
    await page.route('**/api/abyss/rift', async route => {
      const response = await route.fetch();
      const data = await response.json();

      if (data && data.run && data.run.status === 'fighting') {
        if (data.run.enemies && data.run.enemies.length > 0) {
          // Give first enemy a jump altitude to observe jump shadow
          data.run.enemies[0].jump = 0.35;
          enemyShadowObserved = true;
        }
      }

      await route.fulfill({ response, json: data });
    });

    await page.locator('#rift-start').click();
    await expect(page.locator('#rift-overlay')).toBeHidden();

    for (let i = 0; i < 15; i++) {
      if (enemyShadowObserved) break;
      await page.waitForTimeout(150);
    }

    const shadowState = await page.evaluate(() => {
      return window.RiftRenderer?.lastEnemyJumpShadow || null;
    });

    expect(shadowState).not.toBeNull();
    expect(shadowState.unitId).toBeTruthy();

    await page.keyboard.press('Escape');
  });

  test('renderer handles consistent enemy jump shadow scaling and diffusion in normal and reduced motion modes', async ({ page }) => {
    await page.goto('/abyss/rift');
    await expect(page.locator('#rift-start')).toBeEnabled();

    const checkResult = await page.evaluate(async () => {
      const canvas = document.getElementById('rift-canvas');
      const ctx = canvas?.getContext('2d');
      if (!canvas || !ctx || !window.RiftRenderer) return null;

      if (window.RiftRenderer.ready) {
        await window.RiftRenderer.ready;
      }

      const groundY = 350;

      // 1. High-altitude jumping enemy: jump = 0.325 (peak altitude: sin(0.5*PI)*52 = 52px)
      const mockRunAir = {
        id: 'test-enemy-shadow-air',
        status: 'fighting',
        room: 0,
        player: {
          id: 'player',
          x: 200,
          y: groundY,
          hp: 200,
          max_hp: 200,
          facing: 1,
          pose: 'idle',
          jump: 0
        },
        build: { class: 'vanguard' },
        enemies: [
          {
            id: 'goblin-jumper',
            kind: 'goblin',
            x: 450,
            y: groundY,
            hp: 80,
            max_hp: 80,
            facing: -1,
            pose: 'idle',
            jump: 0.325 // exactly peak of 0.65s jump
          }
        ],
        projectiles: [],
        drops: [],
        events: []
      };

      window.RiftRenderer.reduced = false;
      await window.RiftRenderer.prepareRun(mockRunAir);
      window.RiftRenderer.feed(mockRunAir);
      await new Promise(r => requestAnimationFrame(() => requestAnimationFrame(r)));
      const airShadowNormal = window.RiftRenderer.lastEnemyJumpShadow;

      // 2. Grounded enemy: jump = 0
      const mockRunGround = {
        ...mockRunAir,
        enemies: [
          {
            ...mockRunAir.enemies[0],
            jump: 0
          }
        ]
      };
      await window.RiftRenderer.prepareRun(mockRunGround);
      window.RiftRenderer.feed(mockRunGround);
      await new Promise(r => requestAnimationFrame(() => requestAnimationFrame(r)));
      const groundShadowNormal = window.RiftRenderer.lastEnemyJumpShadow;

      // 3. Reduced motion mode with jumping enemy
      window.RiftRenderer.reduced = true;
      await window.RiftRenderer.prepareRun(mockRunAir);
      window.RiftRenderer.feed(mockRunAir);
      await new Promise(r => requestAnimationFrame(() => requestAnimationFrame(r)));
      const airShadowReduced = window.RiftRenderer.lastEnemyJumpShadow;

      return {
        groundY,
        airShadowNormal,
        groundShadowNormal,
        airShadowReduced
      };
    });

    expect(checkResult).not.toBeNull();
    // Shadow remains firmly anchored at floor Y (y + 2), NOT elevated into air
    expect(checkResult.airShadowNormal).toBeDefined();
    expect(checkResult.airShadowNormal.y).toBe(checkResult.groundY + 2);
    expect(checkResult.airShadowNormal.jump).toBe(52); // peak 52px
    expect(checkResult.airShadowNormal.scale).toBeLessThan(checkResult.groundShadowNormal.scale);
    expect(checkResult.airShadowNormal.alpha).toBeLessThan(checkResult.groundShadowNormal.alpha);

    // Grounded shadow has full scale 1.0
    expect(checkResult.groundShadowNormal).toBeDefined();
    expect(checkResult.groundShadowNormal.scale).toBe(1.0);
    expect(checkResult.groundShadowNormal.jump).toBe(0);

    // Reduced motion mode keeps stable scale 1.0
    expect(checkResult.airShadowReduced).toBeDefined();
    expect(checkResult.airShadowReduced.reduced).toBe(true);
    expect(checkResult.airShadowReduced.scale).toBe(1.0);
  });
});
