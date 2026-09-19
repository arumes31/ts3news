const { test, expect } = require('@playwright/test');

test('displays resource gains as small transient counters in HUD and canvas', async ({ page }) => {
  await page.goto('/abyss/rift?subclass=geomancer');
  await expect(page.locator('#rift-start')).toBeEnabled();
  await page.locator('#rift-start').click();
  await expect(page.locator('#rift-overlay')).toBeHidden();
  await expect(page.locator('#rift-enemy-count')).toContainText('remaining');

  const resourceGain = page.locator('#rift-resource-gain');
  const hpGain = page.locator('#rift-hp-gain');
  const manaGain = page.locator('#rift-mana-gain');

  // Initial state: transient counters are hidden
  await expect(resourceGain).toBeHidden();
  await expect(hpGain).toBeHidden();
  await expect(manaGain).toBeHidden();

  // Test HUD triggerTransientCounter helper directly
  await page.evaluate(() => {
    const hud = window.RiftHUD;
    hud.triggerTransientCounter(document.getElementById('rift-resource-gain'), '+1 Resonance', 'resource');
  });

  await expect(resourceGain).toBeVisible();
  await expect(resourceGain).toHaveText('+1 Resonance');
  await expect(resourceGain).toHaveAttribute('data-kind', 'resource');

  // Check that it auto-dismisses after expiry
  await expect(resourceGain).toBeHidden({ timeout: 3000 });

  // Test live builder skill cast (KeyQ) generating class resource
  await page.keyboard.press('KeyQ');

  // When the server responds, resource increases to 1 and triggers +1 Stone (or Resonance/Charge)
  await expect(resourceGain).toBeVisible({ timeout: 5000 });
  await expect(resourceGain).toHaveText(/\+1 (Stone|Resonance|Charge)/);
  await expect(resourceGain).toHaveAttribute('data-kind', 'resource');

  // Test health and mana transient counter display
  await page.evaluate(() => {
    const hud = window.RiftHUD;
    hud.triggerTransientCounter(document.getElementById('rift-hp-gain'), '+25 HP', 'health');
    hud.triggerTransientCounter(document.getElementById('rift-mana-gain'), '+15 MP', 'mana');
  });

  await expect(hpGain).toBeVisible();
  await expect(hpGain).toHaveText('+25 HP');
  await expect(hpGain).toHaveAttribute('data-kind', 'health');

  await expect(manaGain).toBeVisible();
  await expect(manaGain).toHaveText('+15 MP');
  await expect(manaGain).toHaveAttribute('data-kind', 'mana');

  // Verify canvas event rendering for resource gains
  const canvasRenderSuccess = await page.evaluate(async () => {
    const renderer = window.RiftRenderer;
    const runResp = await (await fetch('/api/abyss/rift')).json();
    const mockRun = JSON.parse(JSON.stringify(runResp.run));
    mockRun.events = [
      { id: mockRun.counter + 1, kind: 'resource', x: 200, y: 300, value: 1 },
      { id: mockRun.counter + 2, kind: 'heal', x: 200, y: 280, value: 30 },
      { id: mockRun.counter + 3, kind: 'barrier', x: 200, y: 260, value: 50 },
    ];
    mockRun.counter += 3;
    renderer.snapshot(mockRun, false);

    // Wait 2 animation frames to confirm render loop executes without exception
    await new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(resolve)));
    return true;
  });
  expect(canvasRenderSuccess).toBe(true);

  // Clean screenshot hides HUD transient counters
  const screenshotBtn = page.locator('#rift-screenshot-toggle');
  await screenshotBtn.click();
  await expect(resourceGain).toBeHidden();
  await expect(hpGain).toBeHidden();
  await expect(manaGain).toBeHidden();
});

test('mobile viewport at 390px does not overflow when transient counters are shown', async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto('/abyss/rift?subclass=geomancer');
  await expect(page.locator('#rift-start')).toBeEnabled();
  await page.locator('#rift-start').click();
  await expect(page.locator('#rift-overlay')).toBeHidden();

  await page.evaluate(() => {
    const hud = window.RiftHUD;
    hud.triggerTransientCounter(document.getElementById('rift-resource-gain'), '+1 Resonance', 'resource');
    hud.triggerTransientCounter(document.getElementById('rift-hp-gain'), '+40 HP', 'health');
  });

  await expect(page.locator('#rift-resource-gain')).toBeVisible();

  const isOverflowing = await page.evaluate(() => {
    return document.documentElement.scrollWidth > window.innerWidth;
  });
  expect(isOverflowing).toBe(false);
});
