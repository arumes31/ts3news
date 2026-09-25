const { test, expect } = require('@playwright/test');

test.use({ hasTouch: true, isMobile: true, viewport: { width: 390, height: 844 } });

test('avoids virtual keyboard opening during combat and blurs text inputs on start/resume', async ({ page }) => {
  await page.goto('/abyss/rift');
  await expect(page.locator('#rift-start')).toBeEnabled();

  // Focus mission search before starting
  const search = page.locator('#rift-mission-search');
  await search.tap();
  await expect(search).toBeFocused();

  // Starting combat blurs the input
  await page.locator('#rift-start').tap();
  await expect(page.locator('#rift-overlay')).toBeHidden();
  await expect(search).not.toBeFocused();

  // Active element should not be an editable text input
  const activeTag = await page.evaluate(() => document.activeElement ? document.activeElement.tagName : '');
  expect(['INPUT', 'TEXTAREA'].includes(activeTag)).toBe(false);

  // Tapping canvas does not focus text inputs
  await page.locator('#rift-canvas').tap();
  expect(await page.evaluate(() => ['INPUT', 'TEXTAREA'].includes(document.activeElement?.tagName))).toBe(false);

  // Attempting to focus text input during combat immediately blurs it
  await page.evaluate(() => {
    const input = document.getElementById('rift-mission-search') || document.getElementById('rift-monster-search');
    if (input) input.focus();
  });
  const focusedDuringCombat = await page.evaluate(() => document.activeElement?.tagName);
  expect(['INPUT', 'TEXTAREA'].includes(focusedDuringCombat)).toBe(false);

  // Pausing allows normal text input focus
  await page.locator('#rift-pause').tap();
  await expect(page.locator('#rift-start')).toBeEnabled();

  // Open campaign and focus search while paused
  await page.locator('#rift-campaign > summary').tap();
  await search.tap();
  await expect(search).toBeFocused();

  // Resuming expedition dismisses the input again
  await page.locator('#rift-start').tap();
  await expect(page.locator('#rift-overlay')).toBeHidden();
  await expect(search).not.toBeFocused();

  await page.locator('#rift-pause').tap();
});

test('scrolls selected mission into view above keyboard in shortened viewport', async ({ page }) => {
  // Simulate mobile device with soft keyboard open (shortened visual height)
  await page.setViewportSize({ width: 390, height: 480 });
  await page.goto('/abyss/rift');
  await expect(page.locator('#rift-start')).toBeEnabled();

  const search = page.locator('#rift-mission-search');
  await search.tap();
  await search.fill('85');

  const card85 = page.locator('#rift-levels [data-level="85"]');
  await expect(card85).toBeVisible();

  // Select card 85 and verify its bounding box fits inside the reduced viewport
  await card85.tap();
  const box = await card85.boundingBox();
  expect(box).not.toBeNull();
  expect(box.y).toBeGreaterThanOrEqual(0);
  expect(box.y + box.height).toBeLessThanOrEqual(520); // within visible area above keyboard
});
