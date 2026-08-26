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
  await expect(charm.locator('.slot-gi').first()).toHaveCSS('animation-name', 'gear-charm-dangle');

  const mystery = page.locator('.inv-card[data-unidentified="true"]');
  await expect(mystery).toContainText('Unidentified Finger1');
  await expect(mystery).toContainText('Unknown · Finger1');
  await expect(mystery).not.toContainText('Secret Celestial Ring');
  await expect(mystery).not.toContainText('Celestial');
  await expect(mystery.locator('.rarity-silhouette')).toHaveAccessibleName('Unknown rarity silhouette');
  expect(await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth)).toBeLessThanOrEqual(1);

  await page.emulateMedia({ reducedMotion: 'reduce' });
  await expect(charm.locator('.slot-gi').first()).toHaveCSS('animation-name', 'none');
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
