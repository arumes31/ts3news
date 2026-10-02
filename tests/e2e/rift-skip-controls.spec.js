const { test, expect } = require('@playwright/test');

test('provides an accessible skip link directly to battlefield controls', async ({ page }) => {
  await page.goto('/abyss/rift');
  await expect(page.locator('#rift-start')).toBeEnabled();

  const skipLink = page.locator('#rift-skip-controls');
  await expect(skipLink).toBeAttached();
  expect(await skipLink.getAttribute('href')).toBe('#rift-actionbar');
  expect(await skipLink.textContent()).toBe('Skip to battlefield controls');

  // 1. Skip link is off-screen before focus
  const initialBox = await skipLink.boundingBox();
  expect(initialBox.y).toBeLessThan(0);

  // 2. Tabbing into the page focuses the skip link and brings it into view
  await page.keyboard.press('Tab');
  await expect(skipLink).toBeFocused();

  const focusedBox = await skipLink.boundingBox();
  expect(focusedBox.y).toBeGreaterThanOrEqual(0);
  expect(focusedBox.y).toBeLessThan(100);

  // 3. Activating the skip link via Enter jumps directly to battlefield controls
  await page.keyboard.press('Enter');

  const attackBtn = page.locator('#rift-actionbar button[data-bind="attack"]');
  await expect(attackBtn).toBeFocused();

  // 4. Combat controls are visible in viewport after skipping
  const attackBox = await attackBtn.boundingBox();
  expect(attackBox.y).toBeGreaterThanOrEqual(0);

  // 5. Clicking the skip link also navigates and focuses combat controls
  await page.evaluate(() => window.scrollTo(0, 0));
  await skipLink.focus();
  await skipLink.click();
  await expect(attackBtn).toBeFocused();
});
