const { test, expect } = require('@playwright/test');

test('vendor buyback restores a recent exact item for the disclosed handling fee', async ({ page }) => {
  let request;
  await page.route('**/api/inventory/buyback', async route => {
    request = route.request().postDataJSON();
    await route.fulfill({
      contentType: 'application/json',
      body: JSON.stringify({ ok: true, gold: 14_000, msg: 'Item returned to your inventory.' }),
    });
  });

  await page.goto('/inventory');
  const panel = page.locator('#vendorBuybacks');
  await expect(panel.getByRole('heading', { name: 'Buy back recently sold gear' })).toBeVisible();
  await expect(panel).toContainText('Lucky Test Charm');
  await expect(panel).toContainText('41 durability');
  await expect(panel).toContainText('received 🪙 10.0k');

  await panel.getByRole('button', { name: /Buy back/ }).click();
  const dialog = page.getByRole('dialog');
  await expect(dialog).toContainText('11,000');
  await expect(dialog).toContainText('disclosed 10% vendor handling fee');
  await dialog.getByRole('button', { name: /Buy back · 11,000g/ }).click();

  await expect.poll(() => request).toEqual({ id: 77 });
  await expect(panel.locator('.buyback-card')).toHaveCount(0);
  await expect(panel).toContainText('No recent vendor sales');
  await expect(page.locator('#goldPill')).toContainText('14,000');
  await expect(page.locator('#invMsg')).toContainText('Item returned to your inventory');
});
