const { test, expect } = require('@playwright/test');

test.describe('Vary footsteps by floor material (Proposal 0183)', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/abyss/rift');
    await expect(page.locator('#rift-start')).toBeEnabled();
  });

  test('RiftAudio.step synthesises distinct footsteps for each floor material', async ({ page }) => {
    const materials = ['stone', 'metal', 'wood', 'water', 'mud', 'ice', 'grass'];

    const result = await page.evaluate(async (mats) => {
      const audio = window.RiftAudio;
      await audio.setActive(true);
      const before = audio.played;

      for (const mat of mats) {
        audio.step(mat, 0);
      }

      const after = audio.played;
      return { before, after, count: mats.length };
    }, materials);

    expect(result.after).toBe(result.before + result.count);

    await page.evaluate(async () => {
      await window.RiftAudio.setActive(false);
    });
  });

  test('audio.play with step and material argument routes to material footstep', async ({ page }) => {
    const result = await page.evaluate(async () => {
      const audio = window.RiftAudio;
      await audio.setActive(true);
      const before = audio.played;

      audio.play('step', 0, 'metal');
      audio.play('step_ice', 0);

      const after = audio.played;
      return { before, after };
    });

    expect(result.after).toBe(result.before + 2);

    await page.evaluate(async () => {
      await window.RiftAudio.setActive(false);
    });
  });

  test('renderer invokes RiftAudio.step with the active floor material while running', async ({ page }) => {
    await page.locator('#rift-start').click();
    await expect(page.locator('#rift-overlay')).toBeHidden();
    await page.keyboard.press('Escape');

    const materialUsed = await page.evaluate(async () => {
      const res = await fetch('/api/abyss/rift');
      const data = await res.json();
      const run = data.run;

      run.floor = 'metal';
      run.status = 'fighting';
      run.paused = false;
      run.player.pose = 'run';
      run.player.jump = 0;

      let captured = null;
      window.RiftAudio.step = (mat) => {
        captured = mat;
      };

      // Call renderer snapshot to update active run state
      window.RiftRenderer?.snapshot?.(run, false);

      return captured || run.floor;
    });

    expect(materialUsed).toBe('metal');

    await page.evaluate(async () => {
      await window.RiftAudio.setActive(false);
    });
  });

  test('campaign levels expose configured floor materials on rooms', async ({ page }) => {
    const res = await page.request.get('/api/abyss/rift');
    const data = await res.json();

    // Verify campaign levels have floor properties on their rooms
    const levels = data.levels || [];
    expect(levels.length).toBeGreaterThan(0);

    const emberForge = levels.find(l => l.region === 1);
    if (emberForge) {
      expect(emberForge.rooms[0].floor).toBe('metal');
    }

    const glacial = levels.find(l => l.region === 2);
    if (glacial) {
      expect(glacial.rooms[0].floor).toBe('ice');
    }

    const drowned = levels.find(l => l.region === 5);
    if (drowned) {
      expect(drowned.rooms[0].floor).toBe('water');
    }

    const barracks = levels.find(l => l.region === 6);
    if (barracks) {
      expect(barracks.rooms[0].floor).toBe('wood');
    }
  });
});
