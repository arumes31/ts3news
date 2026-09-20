const { test, expect } = require('@playwright/test');

test.describe('Distinct treasure-goblin escape cue (Proposal 0181)', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/abyss/rift');
    await expect(page.locator('#rift-start')).toBeEnabled();
  });

  test('treasure_escape cue is synthesised and routed through the voice bus', async ({ page }) => {
    const result = await page.evaluate(async () => {
      const audio = window.RiftAudio;
      await audio.setActive(true);
      const before = audio.played;
      audio.play('treasure_escape', 0);
      const after = audio.played;
      return { before, after };
    });

    expect(result.after).toBe(result.before + 1);

    await page.evaluate(async () => {
      await window.RiftAudio.setActive(false);
    });
  });

  test('treasure_escape event from combat run triggers the audio cue through the renderer pipeline', async ({ page }) => {
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

      // Inject a treasure_escape event into the run
      run.counter = (run.counter || 0) + 1;
      run.events = run.events || [];
      run.events.push({ id: run.counter, kind: 'treasure_escape', x: 200, y: 380, value: 0 });

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

  test('bestiary monster details for treasure goblin displays Preview escape button and previews cue', async ({ page }) => {
    const roster = (await (await page.request.get('/api/abyss/rift')).json()).bestiary;
    const goblin = roster.find(unit => unit.kind === 'treasure');
    expect(goblin).toBeDefined();

    await page.locator('.rift-bestiary > summary').click();
    await page.getByRole('button', { name: 'Inspect ' + goblin.name, exact: true }).click();

    const escapeBtn = page.getByRole('button', { name: 'Preview escape', exact: true });
    await expect(escapeBtn).toBeVisible();

    await page.evaluate(() => {
      window.previewCues = [];
      const preview = window.RiftAudio.previewCue;
      window.RiftAudio.previewCue = kind => {
        window.previewCues.push(kind);
        return preview(kind);
      };
    });

    const before = await page.evaluate(() => window.RiftAudio.played);
    await escapeBtn.click();
    await expect(page.locator('#rift-monster-sound-status')).toContainText('Previewing escape');
    expect(await page.evaluate(() => window.previewCues)).toEqual(['treasure_escape']);
    expect(await page.evaluate(() => window.RiftAudio.played)).toBe(before + 1);

    // Ensure run was not started
    expect((await (await page.request.get('/api/abyss/rift')).json()).run).toBeNull();
  });

  test('treasure goblin escape caption appears when combat captions are enabled', async ({ page }) => {
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

      // Second call with a new treasure_escape event at a higher counter
      run.counter = (run.counter || 0) + 1;
      run.events = run.events || [];
      run.events.push({ id: run.counter, kind: 'treasure_escape', x: 200, y: 380, value: 0 });

      window.RiftFeedback.update(run, false, true);

      const captions = document.getElementById('rift-captions');
      return captions ? captions.textContent : '';
    });

    expect(captionText).toContain('Treasure goblin escaped');

    await page.evaluate(async () => {
      await window.RiftAudio.setActive(false);
    });
  });
});
