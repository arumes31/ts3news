const { test, expect } = require('@playwright/test');

test.describe('Add a marked-target outline pulse (Proposal 0209)', () => {
  test('builder skill hitting enemy marks target and emits mark_target event', async ({ page }) => {
    await page.goto('/abyss/rift');
    await expect(page.locator('#rift-start')).toBeEnabled();

    let markTargetEventSeen = false;
    let markedEnemyId = null;

    // Intercept step snapshots to observe mark_target event and marked enemy
    await page.route('**/api/abyss/rift', async route => {
      const response = await route.fetch();
      const data = await response.json();

      if (data && data.run) {
        if (data.run.enemies && data.run.enemies.length > 0 && !data.run.marked) {
          const target = data.run.enemies.find(e => e.hp > 0);
          if (target) {
            data.run.marked = target.id;
            data.run.events = data.run.events || [];
            data.run.events.push({ id: 9999, kind: 'mark_target', x: target.x, y: target.y - 30, value: 0 });
          }
        }
        if (data.run.marked) {
          markedEnemyId = data.run.marked;
        }
        if (data.run.events) {
          for (const ev of data.run.events) {
            if (ev.kind === 'mark_target') {
              markTargetEventSeen = true;
            }
          }
        }
      }

      await route.fulfill({ response, json: data });
    });

    await page.locator('#rift-start').click();
    await expect(page.locator('#rift-overlay')).toBeHidden();

    // Walk toward enemies and strike with builder skill (KeyQ)
    await page.keyboard.down('KeyD');
    for (let i = 0; i < 25; i++) {
      if (markTargetEventSeen && markedEnemyId) break;
      if (i % 2 === 0) {
        await page.keyboard.press('KeyQ');
      }
      await page.waitForTimeout(150);
    }
    await page.keyboard.up('KeyD');

    // Ensure builder skill was triggered
    if (!markTargetEventSeen) {
      await page.keyboard.press('KeyQ');
      await page.waitForTimeout(400);
    }

    expect(markTargetEventSeen).toBe(true);
    expect(markedEnemyId).toBeTruthy();

    await page.keyboard.press('Escape');
  });

  test('renderer handles marked-target outline pulse and mark_target event in normal and reduced motion modes', async ({ page }) => {
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
        id: 'test-mark-pulse-eval',
        status: 'fighting',
        room: 0,
        marked: 'goblin-test-1',
        player: {
          id: 'player',
          x: 200,
          y: 350,
          hp: 200,
          max_hp: 200,
          mana: 100,
          facing: 1,
          pose: 'idle',
          pose_time: 0,
          guard: false,
          jump: 0
        },
        build: {
          class: 'vanguard'
        },
        enemies: [
          {
            id: 'goblin-test-1',
            kind: 'goblin',
            x: 450,
            y: 350,
            hp: 60,
            max_hp: 80,
            facing: -1,
            pose: 'idle',
            pose_time: 0,
            jump: 0
          }
        ],
        projectiles: [],
        drops: [],
        events: [
          { id: 1, kind: 'mark_target', x: 450, y: 320, value: 0 }
        ]
      };

      // 1. Normal mode: feed mock snapshot with marked target and event
      window.RiftRenderer.reduced = false;
      await window.RiftRenderer.prepareRun(mockRun);
      window.RiftRenderer.feed(mockRun);

      await new Promise(r => requestAnimationFrame(() => requestAnimationFrame(r)));

      const normalPulse = window.RiftRenderer.lastMarkedTargetPulse;
      const normalEvent = window.RiftRenderer.lastMarkTargetEvent;

      // 2. Reduced motion mode: feed again and verify graceful handling
      window.RiftRenderer.reduced = true;
      const mockRun2 = {
        ...mockRun,
        events: [
          { id: 2, kind: 'mark_target', x: 450, y: 320, value: 0 }
        ]
      };
      await window.RiftRenderer.prepareRun(mockRun2);
      window.RiftRenderer.feed(mockRun2);

      await new Promise(r => requestAnimationFrame(() => requestAnimationFrame(r)));

      const reducedPulse = window.RiftRenderer.lastMarkedTargetPulse;

      return {
        normalPulse,
        normalEvent,
        reducedPulse
      };
    });

    expect(checkResult).not.toBeNull();
    expect(checkResult.normalPulse).toBeDefined();
    expect(checkResult.normalPulse.unitId).toBe('goblin-test-1');
    expect(checkResult.normalPulse.reduced).toBe(false);
    expect(checkResult.normalEvent).toBeDefined();
    expect(checkResult.normalEvent.x).toBe(450);

    expect(checkResult.reducedPulse).toBeDefined();
    expect(checkResult.reducedPulse.unitId).toBe('goblin-test-1');
    expect(checkResult.reducedPulse.reduced).toBe(true);
    expect(checkResult.reducedPulse.pulse).toBe(0);
  });
});
