const { test, expect } = require('@playwright/test');
for (const mode of ['normal', 'reduced', 'still', 'offscreen']) {
  test('projectile collision burst: ' + mode, async ({ page }) => {
    await page.emulateMedia({ reducedMotion: mode === 'reduced' ? 'reduce' : 'no-preference' });
    await page.goto('/abyss/rift?scenario=checkpoint');
    await expect(page.locator('#rift-start')).toBeEnabled();
    const run = (await (await page.request.get('/api/abyss/rift')).json()).run;
    await page.evaluate(({ run, mode }) => {
      window.impactStrokes = 0;
      const ctx = document.getElementById('rift-canvas').getContext('2d');
      const stroke = ctx.stroke.bind(ctx);
      ctx.stroke = (...args) => { if (ctx.strokeStyle === '#ffe1b5') window.impactStrokes++; return stroke(...args); };
      if (mode === 'still') window.RiftDisplay.motionIntensity = 0;
      run.paused = false;
      run.counter++;
      run.events = [{ id: run.counter, kind: 'projectile_impact', x: mode === 'offscreen' ? -2000 : run.player.x, y: run.player.y, value: 0 }];
      window.RiftRenderer.snapshot(run, false);
    }, { run, mode });
    if (mode === 'normal') {
      await expect.poll(() => page.evaluate(() => window.impactStrokes)).toBeGreaterThan(0);
      await page.waitForTimeout(400);
      const count = await page.evaluate(() => window.impactStrokes);
      await page.waitForTimeout(150);
      expect(await page.evaluate(() => window.impactStrokes)).toBe(count);
    } else {
      await page.waitForTimeout(350);
      expect(await page.evaluate(() => window.impactStrokes)).toBe(0);
    }
  });
}
