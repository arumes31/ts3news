const { test, expect } = require('@playwright/test');

test('Hall of Delvers commissions a cosmetic monument in an opaque public plaza', async ({ page }) => {
  let purchase;
  await page.route('**/api/abyss/plaza/buy', async route => {
    purchase = route.request().postDataJSON();
    await route.fulfill({
      contentType: 'application/json',
      body: JSON.stringify({
        ok: true,
        key: purchase.key,
        gold: 4_750_000,
        msg: 'Bronze Delver Bust now stands in the Hall of Delvers.',
      }),
    });
  });

  await page.goto('/abyss/plaza');
  await expect(page.getByRole('heading', { name: 'Hall of Delvers' })).toBeVisible();
  await expect(page.locator('.plaza-catalog-card')).toHaveCount(4);
  await expect(page.locator('.plaza-exhibit img')).toHaveCount(0);
  await expect(page.locator('body')).toHaveCSS('background-color', 'rgb(8, 10, 15)');

  const commission = page.locator('[data-monument="bronze_bust"] button');
  await commission.click();
  const dialog = page.getByRole('dialog');
  await expect(dialog).toContainText('Permanent vanity only · no stats · no resale.');
  await expect(dialog).toContainText('250,000');
  await dialog.getByRole('button', { name: /Commission · 250,000g/ }).click();

  await expect.poll(() => purchase).toEqual({ key: 'bronze_bust' });
  await expect(commission).toBeDisabled();
  await expect(commission).toHaveText('✓ Standing');
  await expect(page.locator('#goldPill')).toContainText('4,750,000');
  await expect(page.locator('#plazaToast')).toContainText('now stands in the Hall of Delvers');
});
