const { test, expect } = require('@playwright/test');

test('armory distinguishes rating from effective combat chance on desktop and mobile', async ({ page }) => {
  const errors = [];
  page.on('pageerror', error => errors.push(error.message));
  for (const width of [1280, 390]) {
    await page.setViewportSize({ width, height: 844 });
    await page.goto('/armory-fixture');
    await expect(page.getByText('CRT rating · 25.00% chance', { exact: true })).toBeVisible();
    await expect(page.getByText('DGE rating · 12.50% chance', { exact: true })).toBeVisible();
    expect(await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth)).toBeLessThanOrEqual(1);
  }
  expect(errors).toEqual([]);
});

test('forge labels rating and diminishing passives without promising raw percentages', async ({ page }) => {
  await page.goto('/abyss');
  await page.locator('.ab-tab[data-tab-key="forge"]').click();
  await expect(page.locator('option[value="focused"]').first()).toHaveText('Focused (+10% Critical rating)');
  await expect(page.locator('#recalibrateStatSelect option[value="CRT"]')).toHaveText('Critical rating');
});
