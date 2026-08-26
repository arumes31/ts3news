const { test, expect } = require('@playwright/test');

test('wishlist targets three items and presents an accessible opaque shop panel', async ({ page, request }) => {
  const initial = await request.get('/api/abyss/loot/wishlist');
  const initialData = await initial.json();
  for (const item of initialData.wishlist.selected || []) {
    await request.post('/api/abyss/loot/wishlist', { data: { gear_id: item.id } });
  }

  const pageErrors = [];
  page.on('pageerror', error => pageErrors.push(error.message));
  await page.setViewportSize({ width: 1280, height: 900 });
  await page.goto('/abyss');
  await page.locator('.ab-tab[data-tab-key="shop"]').click();

  const panel = page.locator('#abyssWishlist');
  await expect(panel).toBeVisible();
  await expect(panel).toHaveCSS('background-color', 'rgb(11, 18, 32)');
  await expect(panel.locator('#abyssWishlistCount')).toHaveText('0 / 3 selected');
  await expect(panel.locator('#abyssWishlistProgress')).toHaveAttribute('aria-valuenow', '0');

  for (let count = 1; count <= 3; count += 1) {
    await panel.locator('.ab-wishlist-candidate:not([disabled])').first().click();
    await expect(panel.locator('#abyssWishlistCount')).toHaveText(`${count} / 3 selected`);
  }
  await expect(panel.locator('.ab-wishlist-item.is-selected')).toHaveCount(3);
  await expect(panel.locator('.ab-wishlist-candidate').first()).toBeDisabled();

  await panel.locator('.ab-wishlist-remove').first().click();
  await expect(panel.locator('#abyssWishlistCount')).toHaveText('2 / 3 selected');
  await expect(panel.locator('#abyssWishlistStatus')).toContainText('progress reset');

  const search = panel.locator('#abyssWishlistSearch');
  await search.fill('weapon');
  await page.waitForTimeout(300);
  await expect(panel.locator('.ab-wishlist-candidate').first()).toBeVisible();
  await expect(search).toHaveAttribute('aria-controls', 'abyssWishlistCandidates');
  expect(pageErrors).toEqual([]);
  expect(await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth)).toBeLessThanOrEqual(1);

  await page.setViewportSize({ width: 390, height: 844 });
  await expect(search).toHaveCSS('min-height', '44px');
  await expect(panel.locator('.ab-wishlist-remove').first()).toHaveCSS('min-height', '44px');
  expect(await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth)).toBeLessThanOrEqual(1);

  const css = await request.get('/static/abyss_wishlist.css');
  expect(css.status()).toBe(200);
  expect(await css.text()).toMatch(/@media\s*\(forced-colors:\s*active\)/);
});
