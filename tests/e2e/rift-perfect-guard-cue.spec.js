const { test, expect } = require('@playwright/test');

test.describe('Distinct perfect-guard cue (Proposal 0178)', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/abyss/rift');
    await expect(page.locator('#rift-start')).toBeEnabled();
  });

  test('perfect guard cue is synthesised and routed through the sfx bus', async ({ page }) => {
    const result = await page.evaluate(async () => {
      const audio = window.RiftAudio;
      await audio.setActive(true);
      const before = audio.played;
      audio.play('perfect_guard', 0);
      const after = audio.played;
      return { before, after };
    });

    expect(result.after).toBe(result.before + 1);

    await page.evaluate(async () => {
      await window.RiftAudio.setActive(false);
    });
  });

  test('perfect_guard event from combat run triggers the audio cue through the renderer pipeline', async ({ page }) => {
    await page.locator('#rift-start').click();
    await expect(page.locator('#rift-overlay')).toBeHidden();
    await page.keyboard.press('Escape');

    const result = await page.evaluate(async () => {
      const audio = window.RiftAudio;
      await audio.setActive(true);
      const playedBefore = audio.played;

      const res = await fetch('/api/abyss/rift');
      const data = await res.json();
      const run = data.run;

      // Inject a perfect_guard event into the run
      run.counter = (run.counter || 0) + 1;
      run.events = run.events || [];
      run.events.push({ id: run.counter, kind: 'perfect_guard', x: run.player.x, y: run.player.y, value: 0 });

      // Call the renderer snapshot which processes events and plays sounds
      window.RiftRenderer?.snapshot?.(run, false);

      const playedAfter = audio.played;
      return { playedBefore, playedAfter };
    });

    expect(result.playedAfter).toBe(result.playedBefore + 1);

    await page.evaluate(async () => {
      await window.RiftAudio.setActive(false);
    });
  });

  test('perfect guard caption appears when combat captions are enabled', async ({ page }) => {
    await page.locator('.rift-settings > summary').click();
    await page.locator('#rift-combat-captions').check();
    await page.locator('#rift-start').click();
    await expect(page.locator('#rift-overlay')).toBeHidden();
    await page.keyboard.press('Escape');

    const captionText = await page.evaluate(async () => {
      const res = await fetch('/api/abyss/rift');
      const data = await res.json();
      const run = data.run;
      run.paused = false;

      // First call establishes identity and sets seen = run.counter
      window.RiftFeedback.update(run, false, true);

      // Second call with a new perfect_guard event at a higher counter
      run.counter = (run.counter || 0) + 1;
      run.events = run.events || [];
      run.events.push({ id: run.counter, kind: 'perfect_guard', x: run.player.x, y: run.player.y, value: 0 });

      window.RiftFeedback.update(run, false, true);

      const captions = document.getElementById('rift-captions');
      return captions ? captions.textContent : '';
    });

    expect(captionText).toContain('Perfect guard!');

    await page.evaluate(async () => {
      await window.RiftAudio.setActive(false);
    });
  });

  test('floating combat text distinguishes perfect guard with gold color and star icon', async ({ page }) => {
    const textProperties = await page.evaluate(() => {
      const renderer = window.RiftRenderer;
      return {
        perfectFull: renderer.combatText({ kind: 'perfect_guard', value: 0 }),
        perfectPartial: renderer.combatText({ kind: 'perfect_guard', value: 3.4 }),
        normalBlock: renderer.combatText({ kind: 'block', value: 3.4 })
      };
    });

    expect(textProperties.perfectFull.color).toBe('#ffd700');
    expect(textProperties.perfectFull.label).toBe('⭐ Perfect Guard');

    expect(textProperties.perfectPartial.color).toBe('#ffd700');
    expect(textProperties.perfectPartial.label).toBe('⭐ Perfect Guard -3');

    // Normal block is cyan and uses shield icon
    expect(textProperties.normalBlock.color).toBe('#7fd7ff');
    expect(textProperties.normalBlock.label).toBe('🛡️ Guarded -3');
  });
});
