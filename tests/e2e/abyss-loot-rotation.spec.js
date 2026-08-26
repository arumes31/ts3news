const { test, expect } = require('@playwright/test');

test('weekly featured drops disclose exact weighting and UTC reset responsively', async ({ page }) => {
  const pageErrors = [];
  page.on('pageerror', error => pageErrors.push(error.message));
  await page.setViewportSize({ width: 1280, height: 900 });
  await page.goto('/abyss');

  const panel = page.locator('#abyssFeaturedDrops');
  await expect(panel).toBeVisible();
  await expect(panel).toContainText('WEEKLY LOOT SIGNAL · 2026-W35');
  await expect(panel).toContainText('2× selection weight');
  await expect(panel).toContainText('Resets Mon 31 Aug · 00:00 UTC');
  await expect(panel.locator('[role="listitem"]')).toHaveCount(3);
  await expect(panel.locator('.ab-featured-weight')).toHaveCount(3);
  await expect(panel).toContainText('Rarity odds and pity are unchanged');
  await expect(panel).toHaveCSS('background-color', 'rgb(10, 16, 27)');
  expect(pageErrors).toEqual([]);
  expect(await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth)).toBeLessThanOrEqual(1);

  await page.setViewportSize({ width: 390, height: 844 });
  await expect(panel.locator('[role="listitem"]').first()).toHaveCSS('min-height', '44px');
  expect(await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth)).toBeLessThanOrEqual(1);
  const css = await page.request.get('/static/abyss_featured_drops.css');
  expect(css.status()).toBe(200);
  expect(await css.text()).toMatch(/@media\s*\(forced-colors:\s*active\)/);
});
