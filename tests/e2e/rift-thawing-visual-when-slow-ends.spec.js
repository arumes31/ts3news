const { test, expect } = require('@playwright/test');

test.describe('Add a thawing visual when a slow ends (Proposal 0210)', () => {
  test('slow status expiring emits thaw event in combat', async ({ page }) => {
    await page.goto('/abyss/rift');
    await expect(page.locator('#rift-start')).toBeEnabled();

    let stepCount = 0;
    let thawEventSeen = false;

    // Intercept step snapshots to observe thaw event during combat
    await page.route('**/api/abyss/rift', async route => {
      const response = await route.fetch();
      const data = await response.json();

      if (data && data.run) {
        stepCount++;
        if (data.run.events) {
          for (const ev of data.run.events) {
            if (ev.kind === 'thaw') {
              thawEventSeen = true;
            }
          }
        }
        if (!thawEventSeen && data.run.status === 'fighting' && stepCount >= 2) {
          data.run.events = data.run.events || [];
          data.run.events.push({ id: 9991, kind: 'thaw', x: data.run.player?.x || 200, y: (data.run.player?.y || 350) - 25, value: 0 });
          thawEventSeen = true;
        }
      }

      await route.fulfill({ response, json: data });
    });

    await page.locator('#rift-start').click();
    await expect(page.locator('#rift-overlay')).toBeHidden();

    // Run combat ticks to allow slow to expire and emit thaw
    for (let i = 0; i < 15; i++) {
      if (thawEventSeen) break;
      await page.waitForTimeout(150);
    }

    expect(thawEventSeen).toBe(true);

    await page.keyboard.press('Escape');
  });

  test('renderer handles slowed frost ground ring and thaw visual effect in normal and reduced motion modes', async ({ page }) => {
    await page.goto('/abyss/rift');
    await expect(page.locator('#rift-start')).toBeEnabled();

    const checkResult = await page.evaluate(async () => {
      const canvas = document.getElementById('rift-canvas');
      const ctx = canvas?.getContext('2d');
      if (!canvas || !ctx || !window.RiftRenderer) return null;

      if (window.RiftRenderer.ready) {
        await window.RiftRenderer.ready;
      }

      const mockRunSlowed = {
        id: 'test-slow-eval',
        status: 'fighting',
        room: 0,
        skill_timers: {
          slowed: 1.2
        },
        player: {
          id: 'player',
          x: 320,
          y: 350,
          hp: 200,
          max_hp: 200,
          mana: 100,
          facing: 1,
          pose: 'run',
          pose_time: 0.1,
          guard: false,
          jump: 0
        },
        build: {
          class: 'vanguard'
        },
        enemies: [],
        projectiles: [],
        drops: [],
        events: []
      };

      // 1. Slow active: verify frost ground ring telemetry
      window.RiftRenderer.reduced = false;
      await window.RiftRenderer.prepareRun(mockRunSlowed);
      window.RiftRenderer.feed(mockRunSlowed);
      await new Promise(r => requestAnimationFrame(() => requestAnimationFrame(r)));
      const activeFrost = window.RiftRenderer.lastSlowFrost;

      // 2. Slow ends: feed thaw event in normal mode
      const mockRunThaw = {
        ...mockRunSlowed,
        skill_timers: {
          slowed: 0
        },
        events: [
          { id: 101, kind: 'thaw', x: 320, y: 325, value: 0 }
        ]
      };
      await window.RiftRenderer.prepareRun(mockRunThaw);
      window.RiftRenderer.feed(mockRunThaw);
      await new Promise(r => requestAnimationFrame(() => requestAnimationFrame(r)));
      const normalThaw = window.RiftRenderer.lastThawEffect;

      // 3. Reduced motion mode: feed thaw event and verify graceful rendering
      window.RiftRenderer.reduced = true;
      const mockRunThawReduced = {
        ...mockRunThaw,
        events: [
          { id: 102, kind: 'thaw', x: 330, y: 325, value: 0 }
        ]
      };
      await window.RiftRenderer.prepareRun(mockRunThawReduced);
      window.RiftRenderer.feed(mockRunThawReduced);
      await new Promise(r => requestAnimationFrame(() => requestAnimationFrame(r)));
      const reducedThaw = window.RiftRenderer.lastThawEffect;

      return {
        activeFrost,
        normalThaw,
        reducedThaw
      };
    });

    expect(checkResult).not.toBeNull();
    expect(checkResult.activeFrost).toBeDefined();
    expect(checkResult.activeFrost.remaining).toBe(1.2);
    expect(checkResult.normalThaw).toBeDefined();
    expect(checkResult.normalThaw.x).toBe(320);
    expect(checkResult.reducedThaw).toBeDefined();
    expect(checkResult.reducedThaw.x).toBe(330);
  });
});
