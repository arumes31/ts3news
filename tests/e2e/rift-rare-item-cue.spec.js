const { test, expect } = require('@playwright/test');

test.describe('Distinct rare-item discovery cue (Proposal 0182)', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/abyss/rift');
    await expect(page.locator('#rift-start')).toBeEnabled();
  });

  test('rare_item cue is synthesised and routed through the sfx bus', async ({ page }) => {
    const result = await page.evaluate(async () => {
      const audio = window.RiftAudio;
      await audio.setActive(true);
      const before = audio.played;
      audio.play('rare_item', 0);
      const after = audio.played;
      return { before, after };
    });

    expect(result.after).toBe(result.before + 1);

    await page.evaluate(async () => {
      await window.RiftAudio.setActive(false);
    });
  });

  test('rare_item event from combat run triggers audio cue through renderer pipeline', async ({ page }) => {
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

      // Inject a rare_item event into the run
      run.counter = (run.counter || 0) + 1;
      run.events = run.events || [];
      run.events.push({ id: run.counter, kind: 'rare_item', x: 250, y: 400, value: 4 });

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

  test('combat captions reflect rarity-specific discovery labels', async ({ page }) => {
    await page.locator('.rift-settings > summary').click();
    await page.locator('#rift-combat-captions').check();
    await page.locator('#rift-start').click();
    await expect(page.locator('#rift-overlay')).toBeHidden();
    await page.keyboard.press('Escape');

    const captionsResults = await page.evaluate(async () => {
      const res = await fetch('/api/abyss/rift');
      const data = await res.json();
      const run = data.run;
      run.paused = false;

      // First call establishes baseline identity
      window.RiftFeedback.update(run, false, true);

      // 1. Rare item (value = 2)
      run.counter = (run.counter || 0) + 1;
      run.events = [{ id: run.counter, kind: 'rare_item', x: 200, y: 380, value: 2 }];
      window.RiftFeedback.update(run, false, true);
      const rareText = document.getElementById('rift-captions')?.textContent || '';

      // 2. Epic item (value = 3)
      run.counter = run.counter + 1;
      run.events = [{ id: run.counter, kind: 'rare_item', x: 200, y: 380, value: 3 }];
      window.RiftFeedback.update(run, false, true);
      const epicText = document.getElementById('rift-captions')?.textContent || '';

      // 3. Legendary item (value = 4)
      run.counter = run.counter + 1;
      run.events = [{ id: run.counter, kind: 'rare_item', x: 200, y: 380, value: 4 }];
      window.RiftFeedback.update(run, false, true);
      const legText = document.getElementById('rift-captions')?.textContent || '';

      return { rareText, epicText, legText };
    });

    expect(captionsResults.rareText).toContain('Rare item discovered');
    expect(captionsResults.epicText).toContain('Epic item discovered');
    expect(captionsResults.legText).toContain('Legendary item discovered');

    await page.evaluate(async () => {
      await window.RiftAudio.setActive(false);
    });
  });

  test('floating combat text distinguishes rare, epic, and legendary discoveries', async ({ page }) => {
    const textProps = await page.evaluate(() => {
      const renderer = window.RiftRenderer;
      const rare = renderer.combatText({ kind: 'rare_item', value: 2 });
      const epic = renderer.combatText({ kind: 'rare_item', value: 3 });
      const legendary = renderer.combatText({ kind: 'rare_item', value: 4 });
      return { rare, epic, legendary };
    });

    expect(textProps.rare.color).toBe('#2196f3');
    expect(textProps.rare.label).toContain('Rare Discovery');

    expect(textProps.epic.color).toBe('#9c27b0');
    expect(textProps.epic.label).toContain('Epic Discovery');

    expect(textProps.legendary.color).toBe('#ff9800');
    expect(textProps.legendary.label).toContain('Legendary Discovery');
  });
});
