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
  await expect(page.locator('.ab-daily-identify')).toContainText('identified automatically');
  await expect(page.locator('.ab-daily-identify')).toContainText('1–100g per item by tier');
  await expect(page.locator('.ab-daily-identify')).toContainText('free at 0g');

  // Exercise the fallback after the daily benefit is spent with an empty wallet.
  await page.evaluate(() => {
    document.getElementById('btnForgeIdentify').dataset.dailyFree = 'false';
    document.getElementById('goldPill').textContent = '0g';
    updateForgeOptions();
    updateForgeAffordability();
  });
  await expect(identify).toContainText('1–100g · free at 0g');
  await expect(identify).not.toHaveClass(/ab-cantafford/);
  expect(pageErrors).toEqual([]);
});
