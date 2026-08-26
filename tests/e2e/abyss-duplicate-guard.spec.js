const { test, expect } = require('@playwright/test');
const { fulfillAbyssAPI } = require('./helpers/abyss');

test('recent gear duplicate guard refreshes from the authoritative manifest', async ({ page }) => {
  await fulfillAbyssAPI(page, path => {
    if (path.endsWith('/loot/manifest')) {
      return {
        ok: true,
        items: [],
        recent_gear_protected: 7,
        duplicate_floor_window: 20,
      };
    }
    return { ok: false, error: `unexpected API request: ${path}` };
  });

  await page.goto('/abyss?active=1');
  await page.evaluate(() => window.refreshRunLootManifest());

  const guard = page.locator('#lootDuplicateGuard');
  await expect(guard).toBeVisible();
  await expect(guard).toContainText('7 catalog IDs');
  await expect(guard).toContainText('previous 20 cleared floors');
  await expect(guard).toHaveAttribute('data-count', '7');
  await expect(guard).toHaveAttribute('data-window', '20');

  const overflow = await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth);
  expect(overflow).toBeLessThanOrEqual(1);

  const css = await page.request.get('/static/abyss_duplicate_guard.css');
  expect(css.ok()).toBeTruthy();
  const body = await css.text();
  expect(body).toContain('.ab-duplicate-guard');
  expect(body).toContain('forced-colors');
});
