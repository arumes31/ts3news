const { test, expect } = require('@playwright/test');

test.describe('Add a defeated-player fade that preserves silhouette (Proposal 0215)', () => {
  test('player defeat enters defeat pose and emits defeat event in combat', async ({ page }) => {
    await page.goto('/abyss/rift');
    await expect(page.locator('#rift-start')).toBeEnabled({ timeout: 15000 });

    let playerDefeatSeen = false;
    let allowDefeat = false;

    // Keep combat active until its initial UI has been observed, then defeat
    // the player on a step response. Request timing must not end it early.
    await page.route('**/api/abyss/rift', async route => {
      const response = await route.fetch();
      const data = await response.json();

      if (allowDefeat && route.request().method() === 'POST' &&
          route.request().postDataJSON().kind === 'step' && data.run?.status === 'fighting') {
        data.run.player.hp = 0;
        // The protocol caps the damage allowance at remaining health.
        data.hazard_hit_damage = 0;
        data.run.player.pose = 'defeat';
        data.run.status = 'defeated';
        data.run.events = data.run.events || [];
        data.run.events.push({
          id: 9982,
          kind: 'defeat',
          x: data.run.player.x,
          y: data.run.player.y,
          value: 0
        });
        playerDefeatSeen = true;
      }

      await route.fulfill({ response, json: data });
    });

    await page.locator('#rift-start').click();
    await expect(page.locator('#rift-overlay')).toBeHidden();

    allowDefeat = true;
    await expect.poll(() => playerDefeatSeen).toBe(true);
    await expect(page.locator('#rift-overlay')).toBeVisible();
    await expect(page.locator('#rift-result-heading')).toHaveText('Expedition Defeat');
    await expect.poll(() => page.evaluate(() =>
      window.RiftRenderer.lastDefeatedPlayerSilhouette?.preserved
    )).toBe(true);
  });

  test('renderer handles defeated player silhouette fade in normal and reduced motion modes', async ({ page }) => {
    await page.goto('/abyss/rift');
    await expect(page.locator('#rift-start')).toBeEnabled({ timeout: 15000 });

    const checkResult = await page.evaluate(async () => {
      const canvas = document.getElementById('rift-canvas');
      const ctx = canvas?.getContext('2d');
      if (!canvas || !ctx || !window.RiftRenderer) return null;

      if (window.RiftRenderer.ready) {
        await window.RiftRenderer.ready;
      }

      const mockDefeatedRun = {
        id: 'test-defeated-player-eval',
        status: 'defeated',
        room: 1,
        player: {
          id: 'player',
          x: 280,
          y: 350,
          hp: 0,
          max_hp: 200,
          facing: 1,
          pose: 'defeat',
          jump: 0
        },
        build: { class: 'vanguard' },
        enemies: [],
        projectiles: [],
        drops: [],
        events: []
      };

      // 1. Normal motion mode: verify silhouette preservation telemetry and rendering
      window.RiftRenderer.reduced = false;
      await window.RiftRenderer.prepareRun(mockDefeatedRun);
      window.RiftRenderer.feed(mockDefeatedRun);
      await new Promise(r => requestAnimationFrame(() => requestAnimationFrame(r)));
      const normalSilhouette = window.RiftRenderer.lastDefeatedPlayerSilhouette;

      // 2. Reduced motion mode: verify graceful preserved silhouette
      window.RiftRenderer.reduced = true;
      await window.RiftRenderer.prepareRun(mockDefeatedRun);
      window.RiftRenderer.feed(mockDefeatedRun);
      await new Promise(r => requestAnimationFrame(() => requestAnimationFrame(r)));
      const reducedSilhouette = window.RiftRenderer.lastDefeatedPlayerSilhouette;

      return {
        normalSilhouette,
        reducedSilhouette
      };
    });

    expect(checkResult).not.toBeNull();

    // Normal mode assertions
    expect(checkResult.normalSilhouette).toBeDefined();
    expect(checkResult.normalSilhouette.preserved).toBe(true);
    expect(checkResult.normalSilhouette.silhouetteAlpha).toBeGreaterThan(0.5);
    expect(checkResult.normalSilhouette.reduced).toBe(false);

    // Reduced motion assertions
    expect(checkResult.reducedSilhouette).toBeDefined();
    expect(checkResult.reducedSilhouette.preserved).toBe(true);
    expect(checkResult.reducedSilhouette.silhouetteAlpha).toBeGreaterThan(0.5);
    expect(checkResult.reducedSilhouette.reduced).toBe(true);
  });
});
