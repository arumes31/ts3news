const { test, expect } = require('@playwright/test');

test('narrow reflow and enlarged HUD keep combat controls within the page', async ({ page }) => {
  // This is narrow reflow evidence; rift-native-zoom.spec.js verifies actual browser zoom.
  await page.setViewportSize({ width: 320, height: 640 });
  await page.goto('/abyss/rift');
  await expect(page.locator('#rift-start')).toBeEnabled();

  // 1. Assert no horizontal document scrolling at 320px viewport
  const overflow = await page.evaluate(() => {
    return {
      scrollWidth: document.documentElement.scrollWidth,
      innerWidth: window.innerWidth,
      bodyScrollWidth: document.body.scrollWidth,
    };
  });
  expect(overflow.scrollWidth).toBeLessThanOrEqual(overflow.innerWidth + 2);
  expect(overflow.bodyScrollWidth).toBeLessThanOrEqual(overflow.innerWidth + 2);

  // 2. Action bar buttons (Attack, Jump, Guard, Skills) are visible and fully within viewport
  const actionButtons = page.locator('.rift-actionbar button');
  const actionCount = await actionButtons.count();
  expect(actionCount).toBeGreaterThanOrEqual(3);

  for (let i = 0; i < actionCount; i++) {
    const btn = actionButtons.nth(i);
    await expect(btn).toBeVisible();
    const box = await btn.boundingBox();
    expect(box).not.toBeNull();
    expect(box.x).toBeGreaterThanOrEqual(0);
    expect(box.x + box.width).toBeLessThanOrEqual(322);
    expect(box.width).toBeGreaterThan(15);
    expect(box.height).toBeGreaterThan(15);
  }

  // 3. Class bar signature buttons (Q, E, R) are visible and fully within viewport
  const sigButtons = page.locator('.rift-signatures button');
  const sigCount = await sigButtons.count();
  for (let i = 0; i < sigCount; i++) {
    const btn = sigButtons.nth(i);
    await expect(btn).toBeVisible();
    const box = await btn.boundingBox();
    expect(box).not.toBeNull();
    expect(box.x).toBeGreaterThanOrEqual(0);
    expect(box.x + box.width).toBeLessThanOrEqual(322);
    expect(box.width).toBeGreaterThan(15);
    expect(box.height).toBeGreaterThan(15);
  }

  // 4. Combat controls receive click/pointer events without obstruction
  const attackBtn = page.locator('.rift-actionbar button[data-bind="attack"]');
  await attackBtn.click();

  const jumpBtn = page.locator('.rift-actionbar button[data-bind="jump"]');
  await jumpBtn.click();

  // 5. Override the HUD property on its owning element and prove it takes effect.
  const beforeFont=await page.locator('.rift-player-line').evaluate(node=>parseFloat(getComputedStyle(node).fontSize));
  await page.locator('#rift-app').evaluate(node=>node.style.setProperty('--rift-hud-scale','2'));
  expect(await page.locator('.rift-player-line').evaluate(node=>parseFloat(getComputedStyle(node).fontSize))).toBeCloseTo(beforeFont*2,1);

  const scaledActionCount = await actionButtons.count();
  for (let i = 0; i < scaledActionCount; i++) {
    const btn = actionButtons.nth(i);
    await expect(btn).toBeVisible();
    const box = await btn.boundingBox();
    expect(box).not.toBeNull();
    expect(box.x).toBeGreaterThanOrEqual(0);
    expect(box.x + box.width).toBeLessThanOrEqual(322);
  }

  // 6. Clean screenshot toggle cleanly hides combat controls
  const screenshotBtn = page.locator('#rift-screenshot-toggle');
  await screenshotBtn.click();
  await expect(page.locator('#rift-app')).toHaveClass(/rift-clean-screenshot/);
  await expect(page.locator('.rift-actionbar')).toBeHidden();
  await expect(page.locator('.rift-classbar')).toBeHidden();

  // Toggle screenshot mode off to restore controls
  await screenshotBtn.click();
  await expect(page.locator('#rift-app')).not.toHaveClass(/rift-clean-screenshot/);
  await expect(page.locator('.rift-actionbar')).toBeVisible();
  await expect(page.locator('.rift-classbar')).toBeVisible();
});
