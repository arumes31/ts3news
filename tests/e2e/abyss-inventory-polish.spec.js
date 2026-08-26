const { test, expect } = require('@playwright/test');

test('inventory charm motion and unidentified silhouette remain accessible and secret', async ({ page, request }) => {
  const styles = await request.get('/static/gear_inventory_motion.css');
  expect(styles.status()).toBe(200);
  const css = await styles.text();
  expect(css).toContain('@keyframes gear-charm-dangle');
  expect(css).toMatch(/@media\s*\(prefers-reduced-motion:\s*reduce\)/);
  expect(css).toMatch(/@media\s*\(forced-colors:\s*active\)/);

  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto('/inventory');

  const charm = page.locator('.inv-card[data-slot="Charm"]');
  await expect(charm).toContainText('Lucky Test Charm');
  await expect(charm).toHaveAttribute('title', 'Provenance · Abyss depth 18 · Boss: Gorgoroth the Firelord · 2026-08-20 UTC');
  await expect(charm.locator('.slot-gi').first()).toHaveCSS('animation-name', 'gear-charm-dangle');

  const mystery = page.locator('.inv-card[data-unidentified="true"]');
  await expect(mystery).toContainText('Unidentified Finger1');
  await expect(mystery).toContainText('Unknown · Finger1');
  await expect(mystery).not.toContainText('Secret Celestial Ring');
  await expect(mystery).not.toContainText('Celestial');
  await expect(mystery.locator('.rarity-silhouette')).toHaveAccessibleName('Unknown rarity silhouette');
  await expect(mystery).not.toHaveAttribute('title');
  expect(await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth)).toBeLessThanOrEqual(1);

  await page.emulateMedia({ reducedMotion: 'reduce' });
  await expect(charm.locator('.slot-gi').first()).toHaveCSS('animation-name', 'none');
});

test('Armoury preserves exact values and unidentified secrecy in compact mode', async ({ page }) => {
  await page.goto('/armory-fixture');

  const strength = page.locator('.stats-grid [data-item-number="123456"]');
  await expect(strength).toHaveText('123,456');
  await expect(page.getByText('Measured Test Blade')).toBeVisible();
  await expect(page.getByText(/Broken in/)).toBeVisible();
  await expect(page.locator('.gear-cell').filter({ hasText: 'Measured Test Blade' }).locator('.gear-meta').first())
    .toHaveAttribute('title', 'Provenance · Abyss depth 25 · Boss: Malakor the Voidweaver · 2026-08-21 UTC');

  const mystery = page.locator('.gear-cell').filter({ hasText: 'Unidentified Head' });
  await expect(mystery).toContainText('Unknown');
  await expect(mystery).not.toContainText('Secret Armory Crown');
  await expect(mystery).not.toContainText('Celestial');
  await expect(mystery).not.toContainText('987654');
  await expect(mystery.locator('.gear-meta').first()).not.toHaveAttribute('title');

  await page.locator('[data-item-number-toggle]').click();
  await expect(strength).toHaveText('123.5K');
  await expect(strength).toHaveAttribute('title', 'Exact value: 123,456');
  await expect(strength).toHaveAttribute('aria-label', '123.5K, exact value 123,456');
  expect(await page.evaluate(() => localStorage.getItem('ab_item_numbers'))).toBe('compact');
});

test('pity and drop-streak cards use opaque responsive surfaces', async ({ page, request }) => {
  const styles = await request.get('/static/abyss_pity.css');
  expect(styles.status()).toBe(200);
  const css = await styles.text();
  expect(css).toMatch(/@media\s*\(forced-colors:\s*active\)/);
  expect(css).toMatch(/@media\s*\(prefers-reduced-motion:\s*reduce\)/);

  await page.setViewportSize({ width: 390, height: 844 });
  await page.emulateMedia({ colorScheme: 'light', reducedMotion: 'reduce' });
  await page.goto('/abyss');

  const pity = page.locator('.ab-loot-pity');
  const streak = page.locator('.ab-drop-streak');
  await expect(pity).toHaveCSS('background-color', 'rgb(11, 18, 28)');
  await expect(streak).toHaveCSS('background-color', 'rgb(11, 18, 28)');
  await expect(pity).toContainText('Guaranteed Legendary');
  await expect(pity).toContainText('Celestial drought');
  await expect(streak).toContainText('loot find');
  const transitionSeconds = await pity.locator('#pityFill').evaluate(node =>
    Number.parseFloat(getComputedStyle(node).transitionDuration) || 0
  );
  expect(transitionSeconds).toBeLessThanOrEqual(0.001);
  expect(await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth)).toBeLessThanOrEqual(1);
});
