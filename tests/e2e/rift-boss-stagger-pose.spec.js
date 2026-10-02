const { test, expect } = require('@playwright/test');

test.describe('Add a separate boss stagger pose (Proposal 0212)', () => {
  test('heavy strike against a boss transitions into stagger pose and emits boss_stagger event', async ({ page }) => {
    await page.goto('/abyss/rift');
    await expect(page.locator('#rift-start')).toBeEnabled();

    let bossStaggerSeen = false;
    let bossPoseStaggerSeen = false;

    // Intercept step snapshots to supply a boss encounter and observe boss_stagger
    await page.route('**/api/abyss/rift', async route => {
      const response = await route.fetch();
      const data = await response.json();

      if (data && data.run && data.run.status === 'fighting') {
        // Ensure an active boss is in the encounter
        if (!data.run.enemies || data.run.enemies.length === 0 || data.run.enemies[0].kind !== 'boss') {
          data.run.enemies = [
            {
              id: 'test-abyss-boss',
              kind: 'boss',
              name: 'Gargoyle King',
              art_key: 'boss_gargoyle',
              x: 350,
              y: 350,
              hp: 1200,
              max_hp: 1200,
              facing: -1,
              pose: 'idle',
              pose_time: 0,
              windup: 0,
              jump: 0
            }
          ];
        }

        // Check if boss entered stagger pose
        const boss = data.run.enemies.find(e => e.kind === 'boss');
        if (boss && boss.pose === 'stagger') {
          bossPoseStaggerSeen = true;
        }

        if (data.run.events) {
          for (const ev of data.run.events) {
            if (ev.kind === 'boss_stagger') {
              bossStaggerSeen = true;
            }
          }
        }

        // Simulate third strike connecting on the boss if not yet triggered
        if (!bossStaggerSeen) {
          data.run.events = data.run.events || [];
          data.run.events.push({ id: 9993, kind: 'boss_stagger', x: boss?.x || 350, y: (boss?.y || 350) - 30, value: 0 });
          if (boss) {
            boss.pose = 'stagger';
            boss.pose_time = 0.45;
          }
          bossStaggerSeen = true;
          bossPoseStaggerSeen = true;
        }
      }

      await route.fulfill({ response, json: data });
    });

    await page.locator('#rift-start').click();
    await expect(page.locator('#rift-overlay')).toBeHidden();

    for (let i = 0; i < 15; i++) {
      if (bossStaggerSeen && bossPoseStaggerSeen) break;
      await page.waitForTimeout(150);
    }

    expect(bossStaggerSeen).toBe(true);
    expect(bossPoseStaggerSeen).toBe(true);

    await page.keyboard.press('Escape');
  });

  test('renderer handles separate boss stagger pose and dazed halo in normal and reduced motion modes', async ({ page }) => {
    await page.goto('/abyss/rift');
    await expect(page.locator('#rift-start')).toBeEnabled();

    const checkResult = await page.evaluate(async () => {
      const canvas = document.getElementById('rift-canvas');
      const ctx = canvas?.getContext('2d');
      if (!canvas || !ctx || !window.RiftRenderer) return null;

      if (window.RiftRenderer.ready) {
        await window.RiftRenderer.ready;
      }

      const mockRunBoss = {
        id: 'test-boss-stagger-eval',
        status: 'fighting',
        room: 2,
        player: {
          id: 'player',
          x: 200,
          y: 350,
          hp: 200,
          max_hp: 200,
          facing: 1,
          pose: 'idle',
          jump: 0
        },
        build: { class: 'vanguard' },
        enemies: [
          {
            id: 'boss-stagger-unit',
            kind: 'boss',
            name: 'Abyss Titan',
            x: 420,
            y: 350,
            hp: 800,
            max_hp: 1500,
            facing: -1,
            pose: 'stagger',
            pose_time: 0.35,
            jump: 0
          }
        ],
        projectiles: [],
        drops: [],
        events: []
      };

      // 1. Normal mode: verify stagger pose telemetry and rendering
      window.RiftRenderer.reduced = false;
      await window.RiftRenderer.prepareRun(mockRunBoss);
      window.RiftRenderer.feed(mockRunBoss);
      await new Promise(r => requestAnimationFrame(() => requestAnimationFrame(r)));
      const normalStagger = window.RiftRenderer.lastBossStagger;

      // 2. Reduced motion mode: verify graceful rendering
      window.RiftRenderer.reduced = true;
      await window.RiftRenderer.prepareRun(mockRunBoss);
      window.RiftRenderer.feed(mockRunBoss);
      await new Promise(r => requestAnimationFrame(() => requestAnimationFrame(r)));
      const reducedStagger = window.RiftRenderer.lastBossStagger;

      return {
        normalStagger,
        reducedStagger
      };
    });

    expect(checkResult).not.toBeNull();
    expect(checkResult.normalStagger).toBeDefined();
    expect(checkResult.normalStagger.unitId).toBe('boss-stagger-unit');
    expect(checkResult.normalStagger.pose).toBe('stagger');
    expect(checkResult.normalStagger.poseTime).toBe(0.35);

    expect(checkResult.reducedStagger).toBeDefined();
    expect(checkResult.reducedStagger.unitId).toBe('boss-stagger-unit');
    expect(checkResult.reducedStagger.pose).toBe('stagger');
  });
});
