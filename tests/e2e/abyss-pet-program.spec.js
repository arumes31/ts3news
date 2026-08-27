const { test, expect } = require('@playwright/test');

test('companion stable exposes its complete lifecycle without layout regressions', async ({ page }) => {
  const pageErrors = [];
  page.on('pageerror', error => pageErrors.push(error.message));
  await page.setViewportSize({ width: 1440, height: 960 });
  await page.goto('/abyss');

  const sidebar = page.locator('.ab-side-pets');
  await expect(sidebar).toContainText('Ember · damage');
  await expect(sidebar).toContainText('4.2k/5.0k');
  await expect(sidebar.getByRole('progressbar', { name: 'Ember health' })).toHaveAttribute('value', '4200');

  await page.locator('[data-tab-key="social"]').click();
  const stable = page.locator('#abyssSocialHub');
  await expect(stable).toBeVisible();
  await expect(stable.locator('.ab-pet-card')).toHaveCount(3);
  await expect(stable.getByText('Companion power board')).toBeVisible();
  await expect(stable.locator('.ab-pet-power-board')).toContainText('#1 Ember');
  await expect(stable.getByLabel('Companion gift code')).toHaveAttribute('maxlength', '15');

  const reserve = stable.locator('.ab-pet-card[data-pet-id="102"]');
  await expect(reserve.getByLabel('Feed')).toContainText('Small Health Potion ×2');
  await expect(reserve.getByRole('button', { name: 'Daycare' })).toBeVisible();
  await expect(reserve.getByLabel('Expedition')).toContainText('Prism');
  await expect(reserve.getByLabel('Cosmetic')).toContainText('20 🜲');
  await expect(reserve.getByLabel('Voice')).toHaveValue('bold');
  await expect(reserve.getByLabel('Fuse donor')).toContainText('Spark');
  await expect(reserve.getByRole('button', { name: 'Gift' })).toBeVisible();
  await stable.getByText('Pet memorials · 1').click();
  await expect(stable.getByRole('button', { name: /Revive · Companion Scroll/ })).toBeVisible();

  await page.locator('[data-tab-key="lore"]').click();
  const bestiaryRow = page.locator('.ab-bestiary-table tr[data-name="Fixture Stalker"]');
  await expect(bestiaryRow).toHaveAttribute('data-capture-pct', '35');
  await expect(bestiaryRow).toContainText('🌀 35%');

  expect(await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth)).toBeLessThanOrEqual(1);
  expect(pageErrors).toEqual([]);
});
