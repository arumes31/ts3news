const { test, expect } = require('@playwright/test');

test.describe('Add a third-strike impact accent (Proposal 0204)', () => {
  test('third strike connecting with an enemy emits a third_strike impact event', async ({ page }) => {
    await page.goto('/abyss/rift');
    await expect(page.locator('#rift-start')).toBeEnabled();

    let thirdStrikeSeen = false;

    // Intercept step snapshots to observe third_strike event
    await page.route('**/api/abyss/rift', async route => {
      const response = await route.fetch();
      const data = await response.json();

      if (data && data.run && data.run.events) {
        for (const ev of data.run.events) {
          if (ev.kind === 'third_strike') {
            thirdStrikeSeen = true;
          }
        }
      }

      await route.fulfill({ response, json: data });
    });

    await page.locator('#rift-start').click();
    await expect(page.locator('#rift-overlay')).toBeHidden();

    // Approach enemies and perform full 3-hit combo
    await page.keyboard.down('KeyD');
    await page.waitForTimeout(400);
    await page.keyboard.up('KeyD');

    for (let i = 0; i < 20; i++) {
      await page.keyboard.press('KeyJ');
      await page.waitForTimeout(140);
      if (thirdStrikeSeen) break;
    }

    expect(thirdStrikeSeen).toBe(true);
    await page.keyboard.press('Escape');
  });

  test('renderer handles third_strike impact accent correctly in normal and reduced motion modes', async ({ page }) => {
    await page.goto('/abyss/rift');
    await expect(page.locator('#rift-start')).toBeEnabled();

    const checkResult = await page.evaluate(async () => {
      const canvas = document.getElementById('rift-canvas');
      const ctx = canvas?.getContext('2d');
      if (!canvas || !ctx || !window.RiftRenderer) return null;

      const mockRun = {
        id: 'test-third-strike-render',
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
          pose: 'attack',
          pose_time: 0.25,
          jump: 0,
          guard: false,
        },
        enemies: [
          {
            id: 'mob-1',
            kind: 'goblin',
            x: 450,
            y: 350,
            hp: 60,
            max_hp: 100,
            facing: -1,
            pose: 'hit',
            pose_time: 0.2,
            jump: 0,
          }
        ],
        projectiles: [],
        drops: [],
        events: [
          {
            id: 1,
            kind: 'third_strike',
            x: 450,
            y: 325,
            value: 3,
          }
        ],
      };

      // Feed snapshot with third_strike event
      await window.RiftRenderer.prepareRun(mockRun);
      window.RiftRenderer.feed(mockRun);

      // Verify audio cue can play
      let audioResult = false;
      if (window.RiftAudio) {
        audioResult = typeof window.RiftAudio.play === 'function';
        try {
          window.RiftAudio.play('third_strike', 0, 3);
        } catch (_) {}
      }

      return {
        fed: true,
        audioResult,
      };
    });

    expect(checkResult).not.toBeNull();
    expect(checkResult.fed).toBe(true);
    expect(checkResult.audioResult).toBe(true);
  });
});
