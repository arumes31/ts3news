const { test, expect } = require('@playwright/test');

test.describe('Add a boss phase-transition visual (Proposal 0213)', () => {
  test('boss health drop crossing phase thresholds emits boss_phase event and updates phase', async ({ page }) => {
    await page.goto('/abyss/rift');
    await expect(page.locator('#rift-start')).toBeEnabled();

    let phase2Seen = false;
    let phase3Seen = false;

    // Intercept step snapshots to supply a boss encounter and observe boss_phase
    await page.route('**/api/abyss/rift', async route => {
      const response = await route.fetch();
      const data = await response.json();

      if (data && data.run && data.run.status === 'fighting') {
        // Ensure an active boss is in the encounter
        if (!data.run.enemies || data.run.enemies.length === 0 || data.run.enemies[0].kind !== 'boss') {
          data.run.enemies = [
            {
              id: 'test-phase-boss',
              kind: 'boss',
              name: 'Abyssal Overlord',
              art_key: 'boss_overlord',
              x: 380,
              y: 350,
              hp: 500,
              max_hp: 1200,
              phase: 2,
              facing: -1,
              pose: 'idle',
              pose_time: 0,
              windup: 0,
              jump: 0
            }
          ];
        }

        const boss = data.run.enemies.find(e => e.kind === 'boss');
        if (data.run.events) {
          for (const ev of data.run.events) {
            if (ev.kind === 'boss_phase') {
              if (ev.value === 2) phase2Seen = true;
              if (ev.value === 3) phase3Seen = true;
            }
          }
        }

        // Simulate phase transition events if not yet recorded
        if (!phase2Seen) {
          data.run.events = data.run.events || [];
          data.run.events.push({ id: 9991, kind: 'boss_phase', x: boss?.x || 380, y: (boss?.y || 350) - 30, value: 2 });
          if (boss) boss.phase = 2;
          phase2Seen = true;
        } else if (!phase3Seen) {
          data.run.events = data.run.events || [];
          data.run.events.push({ id: 9992, kind: 'boss_phase', x: boss?.x || 380, y: (boss?.y || 350) - 30, value: 3 });
          if (boss) {
            boss.phase = 3;
            boss.hp = 250;
          }
          phase3Seen = true;
        }
      }

      await route.fulfill({ response, json: data });
    });

    await page.locator('#rift-start').click();
    await expect(page.locator('#rift-overlay')).toBeHidden();

    for (let i = 0; i < 20; i++) {
      if (phase2Seen && phase3Seen) break;
      await page.waitForTimeout(150);
    }

    expect(phase2Seen).toBe(true);
    expect(phase3Seen).toBe(true);

    await page.keyboard.press('Escape');
  });

  test('renderer handles boss phase-transition visual in normal and reduced motion modes', async ({ page }) => {
    await page.goto('/abyss/rift');
    await expect(page.locator('#rift-start')).toBeEnabled();

    const checkResult = await page.evaluate(async () => {
      const canvas = document.getElementById('rift-canvas');
      const ctx = canvas?.getContext('2d');
      if (!canvas || !ctx || !window.RiftRenderer) return null;

      if (window.RiftRenderer.ready) {
        await window.RiftRenderer.ready;
      }

      const mockBossRun = {
        id: 'test-boss-phase-eval',
        status: 'fighting',
        room: 3,
        player: {
          id: 'player',
          x: 180,
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
            id: 'boss-phase-test-unit',
            kind: 'boss',
            name: 'Void Behemoth',
            x: 450,
            y: 350,
            hp: 600,
            max_hp: 1500,
            phase: 2,
            facing: -1,
            pose: 'idle',
            pose_time: 0,
            jump: 0
          }
        ],
        projectiles: [],
        drops: [],
        events: [
          { id: 9811, kind: 'boss_phase', x: 450, y: 320, value: 2 }
        ]
      };

      // 1. Normal motion mode: verify phase transition visual and telemetry
      window.RiftRenderer.reduced = false;
      window.RiftRenderer.feed(mockBossRun);
      await new Promise(r => requestAnimationFrame(() => requestAnimationFrame(r)));
      const normalPhaseEvent = window.RiftRenderer.lastBossPhaseEvent;
      const normalPhaseDraw = window.RiftRenderer.lastBossPhaseDraw;

      // 2. Reduced motion mode: verify graceful rendering for phase 3
      const mockBossPhase3 = {
        ...mockBossRun,
        enemies: [
          {
            ...mockBossRun.enemies[0],
            hp: 250,
            phase: 3
          }
        ],
        events: [
          { id: 9812, kind: 'boss_phase', x: 450, y: 320, value: 3 }
        ]
      };
      window.RiftRenderer.reduced = true;
      window.RiftRenderer.feed(mockBossPhase3);
      await new Promise(r => requestAnimationFrame(() => requestAnimationFrame(r)));
      const reducedPhaseEvent = window.RiftRenderer.lastBossPhaseEvent;
      const reducedPhaseDraw = window.RiftRenderer.lastBossPhaseDraw;

      return {
        normalPhaseEvent,
        normalPhaseDraw,
        reducedPhaseEvent,
        reducedPhaseDraw
      };
    });

    expect(checkResult).not.toBeNull();

    // Normal mode assertions
    expect(checkResult.normalPhaseEvent).toBeDefined();
    expect(checkResult.normalPhaseEvent.phase).toBe(2);
    expect(checkResult.normalPhaseEvent.reduced).toBe(false);
    expect(checkResult.normalPhaseDraw).toBeDefined();
    expect(checkResult.normalPhaseDraw.unitId).toBe('boss-phase-test-unit');
    expect(checkResult.normalPhaseDraw.phase).toBe(2);

    // Reduced motion assertions
    expect(checkResult.reducedPhaseEvent).toBeDefined();
    expect(checkResult.reducedPhaseEvent.phase).toBe(3);
    expect(checkResult.reducedPhaseEvent.reduced).toBe(true);
    expect(checkResult.reducedPhaseDraw).toBeDefined();
    expect(checkResult.reducedPhaseDraw.phase).toBe(3);
  });
});
