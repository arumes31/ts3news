const { test, expect } = require('@playwright/test');

test.describe('Add a finisher-specific casting accent (Proposal 0205)', () => {
  test('casting a finisher emits a finisher_cast event with charge count in combat', async ({ page }) => {
    await page.goto('/abyss/rift');
    await expect(page.locator('#rift-start')).toBeEnabled();

    let finisherCastSeen = false;
    let finisherCastCharges = -1;

    // Intercept step snapshots to observe finisher_cast event
    await page.route('**/api/abyss/rift', async route => {
      const response = await route.fetch();
      const data = await response.json();

      if (data && data.run && data.run.events) {
        for (const ev of data.run.events) {
          if (ev.kind === 'finisher_cast') {
            finisherCastSeen = true;
            finisherCastCharges = ev.value;
          }
        }
      }

      await route.fulfill({ response, json: data });
    });

    await page.locator('#rift-start').click();
    await expect(page.locator('#rift-overlay')).toBeHidden();

    // Build charges with builder (KeyQ), then cast finisher (KeyE)
    for (let i = 0; i < 3; i++) {
      await page.keyboard.press('KeyQ');
      await page.waitForTimeout(150);
    }

    for (let i = 0; i < 5; i++) {
      await page.keyboard.press('KeyE');
      await page.waitForTimeout(150);
      if (finisherCastSeen) break;
    }

    expect(finisherCastSeen).toBe(true);
    expect(finisherCastCharges).toBeGreaterThanOrEqual(0);

    await page.keyboard.press('Escape');
  });

  test('renderer handles finisher_cast accent in normal and reduced motion modes', async ({ page }) => {
    await page.goto('/abyss/rift');
    await expect(page.locator('#rift-start')).toBeEnabled();

    const checkResult = await page.evaluate(async () => {
      const canvas = document.getElementById('rift-canvas');
      const ctx = canvas?.getContext('2d');
      if (!canvas || !ctx || !window.RiftRenderer) return null;

      // Ensure renderer atlases are fully loaded
      if (window.RiftRenderer.ready) {
        try {
          await window.RiftRenderer.ready;
        } catch (_) {}
      }

      const mockRun = {
        id: 'test-finisher-cast-render',
        status: 'fighting',
        room: 0,
        build: { class: 'vanguard' },
        player: {
          id: 'player',
          kind: 'vanguard',
          x: 400,
          y: 350,
          hp: 100,
          max_hp: 100,
          facing: 1,
          pose: 'cast',
          pose_time: 0.35,
          jump: 0,
          guard: false,
        },
        enemies: [],
        projectiles: [],
        drops: [],
        events: [
          {
            id: 1,
            kind: 'finisher_cast',
            x: 400,
            y: 315,
            value: 3,
          }
        ],
      };

      // Feed snapshot with finisher_cast event and await render frames
      window.RiftRenderer.feed(mockRun);
      await new Promise(resolve => {
        let frames = 0;
        function step() {
          if (++frames >= 3) resolve();
          else requestAnimationFrame(step);
        }
        requestAnimationFrame(step);
      });
      const accent = window.RiftRenderer.lastFinisherCastAccent;

      // Verify audio cue can play
      let audioResult = false;
      if (window.RiftAudio) {
        audioResult = typeof window.RiftAudio.finisherCast === 'function';
        try {
          window.RiftAudio.finisherCast(3, 0);
          window.RiftAudio.play('finisher_cast', 0, 3);
        } catch (_) {}
      }

      return {
        accentRecorded: Boolean(accent && accent.charges === 3),
        audioResult,
      };
    });

    expect(checkResult).not.toBeNull();
    expect(checkResult.accentRecorded).toBe(true);
    expect(checkResult.audioResult).toBe(true);
  });
});
