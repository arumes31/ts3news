const { test, expect } = require('@playwright/test');

test('Forge clearly exposes the UTC daily free identification', async ({ page }) => {
  const pageErrors = [];
  page.on('pageerror', error => pageErrors.push(error.message));
  await page.goto('/abyss?gear=1');
  await page.locator('.ab-tab[data-tab-key="forge"]').click();

  const picker = page.locator('#forgeItemSelect');
  await picker.selectOption('inv:98');

  const identify = page.locator('#btnForgeIdentify');
  await expect(identify).toBeVisible();
  await expect(identify).toHaveAttribute('data-daily-free', 'true');
  await expect(identify).toContainText('Daily free');

  const identifyAll = page.locator('#btnForgeIdentifyAll');
  await expect(identifyAll).toHaveAttribute('data-daily-free', 'true');
  await expect(identifyAll).toContainText('first free');
  await expect(page.locator('.ab-daily-identify')).toContainText('UTC day');
  expect(pageErrors).toEqual([]);
});
