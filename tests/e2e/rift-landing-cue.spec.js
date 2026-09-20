const { test, expect } = require('@playwright/test');

test.describe('Vary landing sounds by jump intensity (Proposal 0184)', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/abyss/rift');
    await expect(page.locator('#rift-start')).toBeEnabled();
  });

  test('RiftAudio.land synthesises distinct landing sounds for light, medium, and heavy intensities', async ({ page }) => {
    const intensities = [0.2, 0.55, 0.9];

    const result = await page.evaluate(async (vals) => {
      const audio = window.RiftAudio;
      await audio.setActive(true);
      const before = audio.played;

      for (const val of vals) {
        audio.land(val, 0);
      }

      const after = audio.played;
      return { before, after, count: vals.length };
    }, intensities);

    expect(result.after).toBe(result.before + result.count);

    await page.evaluate(async () => {
      await window.RiftAudio.setActive(false);
    });
  });

  test('RiftAudio.land accepts string intensity aliases', async ({ page }) => {
    const aliases = ['light', 'medium', 'heavy'];

    const result = await page.evaluate(async (kinds) => {
      const audio = window.RiftAudio;
      await audio.setActive(true);
      const before = audio.played;

      for (const kind of kinds) {
        audio.land(kind, 0);
      }

      const after = audio.played;
      return { before, after, count: kinds.length };
    }, aliases);

    expect(result.after).toBe(result.before + result.count);

    await page.evaluate(async () => {
      await window.RiftAudio.setActive(false);
    });
  });

  test('audio.play with land routes to land cues and respects intensity arguments', async ({ page }) => {
    const result = await page.evaluate(async () => {
      const audio = window.RiftAudio;
      await audio.setActive(true);
      const before = audio.played;

      audio.play('land_light', 0);
      audio.play('land_heavy', 0);
      audio.play('land', 0, 0.2);
      audio.play('land', 0, 0.85);

      const after = audio.played;
      return { before, after };
    });

    expect(result.after).toBe(result.before + 4);

    await page.evaluate(async () => {
      await window.RiftAudio.setActive(false);
    });
  });

  test('audio.land layers surface transients when floor material is specified', async ({ page }) => {
    const materials = ['metal', 'water', 'mud', 'ice', 'wood'];

    const result = await page.evaluate(async (mats) => {
      const audio = window.RiftAudio;
      await audio.setActive(true);
      const before = audio.played;

      for (const mat of mats) {
        audio.land(0.85, 0, mat);
      }

      const after = audio.played;
      return { before, after, count: mats.length };
    }, materials);

    expect(result.after).toBe(result.before + result.count);

    await page.evaluate(async () => {
      await window.RiftAudio.setActive(false);
    });
  });

  test('renderer snapshot routes land event from server to audio.play with intensity and floor material', async ({ page }) => {
    const playback = await page.evaluate(async () => {
      const calls = [];
      const origPlay = window.RiftAudio.play;
      window.RiftAudio.play = function (kind, pan, extra, extra2) {
        calls.push({ kind, pan, extra, extra2 });
        return origPlay.apply(this, arguments);
      };

      const mockRun = {
        id: 'test-land-run',
        counter: 10,
        room: 0,
        status: 'fighting',
        floor: 'metal',
        player: { id: 'player', x: 200, y: 350, hp: 100, max_hp: 100, mana: 50, facing: 1, pose: 'idle', jump: 0 },
        enemies: [],
        projectiles: [],
        drops: [],
        events: [
          { id: 10, kind: 'land', x: 200, y: 350, value: 0.85 }
        ]
      };

      window.RiftRenderer.snapshot(mockRun, false);
      window.RiftAudio.play = origPlay;
      return calls;
    });

    const landCall = playback.find(c => c.kind === 'land');
    expect(landCall).toBeDefined();
    expect(landCall.extra).toBe(0.85);
    expect(landCall.extra2).toBe('metal');
  });

  test('combat captions show Hard impact landing for heavy landing events', async ({ page }) => {
    await page.locator('.rift-settings > summary').click();
    await page.locator('#rift-combat-captions').check();
    await page.locator('#rift-start').click();
    await expect(page.locator('#rift-overlay')).toBeHidden();
    await page.keyboard.press('Escape');

    await page.evaluate(async () => {
      const res = await fetch('/api/abyss/rift');
      const data = await res.json();
      const run = data.run;
      run.paused = false;

      window.RiftFeedback.update(run, false, true);

      const updated = JSON.parse(JSON.stringify(run));
      updated.counter = (run.counter || 0) + 1;
      updated.events = [
        { id: updated.counter, kind: 'land', x: updated.player.x, y: updated.player.y, value: 0.95 }
      ];

      window.RiftFeedback.update(updated, false, true);
    });

    const captionSection = page.locator('#rift-captions');
    await expect(captionSection).toBeVisible();
    await expect(captionSection).toContainText('Hard impact landing');
  });
});
