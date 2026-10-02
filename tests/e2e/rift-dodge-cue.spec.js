const { test, expect } = require('@playwright/test');

test.describe('Distinct successful-dodge cue (Proposal 0177)', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/abyss/rift');
    await expect(page.locator('#rift-start')).toBeEnabled();
  });

  test('dodge cue is synthesised and routed through the sfx bus', async ({ page }) => {
    const result = await page.evaluate(async () => {
      const audio = window.RiftAudio;
      await audio.setActive(true);
      const before = audio.played;
      audio.play('dodge', 0);
      const after = audio.played;
      return { before, after };
    });

    expect(result.after).toBe(result.before + 1);

    await page.evaluate(async () => {
      await window.RiftAudio.setActive(false);
    });
  });

  test('dodge event from combat run triggers the audio cue through the renderer pipeline', async ({ page }) => {
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

      // Inject a dodge event into the run
      run.counter = (run.counter || 0) + 1;
      run.events = run.events || [];
      run.events.push({ id: run.counter, kind: 'dodge', x: run.player.x, y: run.player.y, value: 0 });

      // Call the renderer snapshot which processes events and plays sounds
      window.RiftRenderer?.snapshot?.(run, false);

      const playedAfter = audio.played;
      return { playedBefore, playedAfter };
    });

    // The renderer processes events and calls audio.play(event.kind, pan)
    expect(result.playedAfter).toBe(result.playedBefore + 1);

    await page.evaluate(async () => {
      await window.RiftAudio.setActive(false);
    });
  });

  test('dodge caption appears when combat captions are enabled', async ({ page }) => {
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

      // Second call with a new dodge event at a higher counter
      run.counter = (run.counter || 0) + 1;
      run.events = run.events || [];
      run.events.push({ id: run.counter, kind: 'dodge', x: run.player.x, y: run.player.y, value: 0 });

      window.RiftFeedback.update(run, false, true);

      const captions = document.getElementById('rift-captions');
      return captions ? captions.textContent : '';
    });

    expect(captionText).toContain('Dodged!');

    await page.evaluate(async () => {
      await window.RiftAudio.setActive(false);
    });
  });
});
