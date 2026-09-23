const { test, expect } = require('@playwright/test');
test('paused arena pixels stay stable across every region', async ({ page }) => {
  await page.goto('/abyss/rift?scenario=checkpoint');
  await expect(page.locator('#rift-start')).toBeEnabled();
  const run = (await (await page.request.get('/api/abyss/rift')).json()).run;
  await page.evaluate(async () => { await window.RiftRenderer.ready; });
  for (let region = 0; region < 10; region++) {
    const frames = await page.evaluate(async ({ run, region }) => {
      const frame = () => new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(resolve)));
      const canvas = document.getElementById('rift-canvas');
      window.RiftDisplay.cameraSmooth = true;
      run.level.region = region;
      run.paused = false;
      run.player.x = 350;
      run.events = [];
      window.RiftRenderer.snapshot(structuredClone(run), false);
      await frame();
      run.paused = true;
      run.player.x = 850;
      run.counter++;
      run.events = [{ id: run.counter, kind: 'projectile_impact', x: 850, y: run.player.y, value: 0 }];
      window.RiftRenderer.snapshot(structuredClone(run), false);
      await frame();
      const first = canvas.toDataURL();
      await new Promise(resolve => setTimeout(resolve, 220));
      return first === canvas.toDataURL();
    }, { run, region });
    expect(frames, 'region ' + region + ' changed while paused').toBe(true);
  }
  const resumed = await page.evaluate(async run => {
    const canvas = document.getElementById('rift-canvas');
    const first = canvas.toDataURL();
    run.paused = false;
    window.RiftRenderer.snapshot(run, false);
    await new Promise(resolve => setTimeout(resolve, 200));
    return first !== canvas.toDataURL();
  }, run);
  expect(resumed).toBe(true);
});
