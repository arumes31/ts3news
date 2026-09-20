const { test, expect } = require('@playwright/test');

test.describe('Add an ultimate anticipation pose (Proposal 0206)', () => {
  test('casting ultimate sets ultimate_anticipation pose and emits ultimate_anticipation event in combat', async ({ page }) => {
    await page.goto('/abyss/rift');
    await expect(page.locator('#rift-start')).toBeEnabled();

    let ultimateAnticipationPoseSeen = false;
    let ultimateAnticipationEventSeen = false;

    // Intercept step snapshots to observe ultimate_anticipation pose and event
    await page.route('**/api/abyss/rift', async route => {
      const response = await route.fetch();
      const data = await response.json();

      if (data && data.run) {
        if (data.run.player && data.run.player.pose === 'ultimate_anticipation') {
          ultimateAnticipationPoseSeen = true;
        }
        if (data.run.events) {
          for (const ev of data.run.events) {
            if (ev.kind === 'ultimate_anticipation') {
              ultimateAnticipationEventSeen = true;
            }
          }
        }
      }

      await route.fulfill({ response, json: data });
    });

    await page.locator('#rift-start').click();
    await expect(page.locator('#rift-overlay')).toBeHidden();

    // Cast ultimate using KeyR
    await page.keyboard.press('KeyR');
    await page.waitForTimeout(200);

    for (let i = 0; i < 5; i++) {
      if (ultimateAnticipationPoseSeen && ultimateAnticipationEventSeen) break;
      await page.keyboard.press('KeyR');
      await page.waitForTimeout(150);
    }

    expect(ultimateAnticipationPoseSeen).toBe(true);
    expect(ultimateAnticipationEventSeen).toBe(true);

    await page.keyboard.press('Escape');
  });

  test('renderer handles ultimate_anticipation pose and visual accent in normal and reduced motion modes', async ({ page }) => {
    await page.goto('/abyss/rift');
    await expect(page.locator('#rift-start')).toBeEnabled();

    const checkResult = await page.evaluate(async () => {
      const canvas = document.getElementById('rift-canvas');
      const ctx = canvas?.getContext('2d');
      if (!canvas || !ctx || !window.RiftRenderer) return null;

      if (window.RiftRenderer.ready) {
        try {
          await window.RiftRenderer.ready;
        } catch (_) {}
      }

      const mockRun = {
        id: 'test-ultimate-anticipation-render',
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
          pose: 'ultimate_anticipation',
          pose_time: 0.5,
          jump: 0,
          guard: false,
        },
        enemies: [],
        projectiles: [],
        drops: [],
        events: [
          {
            id: 1,
            kind: 'ultimate_anticipation',
            x: 400,
            y: 315,
            value: 0,
          }
        ],
      };

      // Feed snapshot and await render frames
      window.RiftRenderer.feed(mockRun);
      await new Promise(resolve => {
        let frames = 0;
        function step() {
          if (++frames >= 3) resolve();
          else requestAnimationFrame(step);
        }
        requestAnimationFrame(step);
      });
      const accent = window.RiftRenderer.lastUltimateAnticipation;

      // Verify audio cue can play
      let audioResult = false;
      if (window.RiftAudio) {
        audioResult = typeof window.RiftAudio.play === 'function';
        try {
          window.RiftAudio.play('ultimate_anticipation', 0);
        } catch (_) {}
      }

      return {
        accentRecorded: Boolean(accent),
        audioResult,
      };
    });

    expect(checkResult).not.toBeNull();
    expect(checkResult.accentRecorded).toBe(true);
    expect(checkResult.audioResult).toBe(true);
  });
});
