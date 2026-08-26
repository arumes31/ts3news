const { test, expect } = require('@playwright/test');

test('Consumable Pouch tailoring raises authoritative stack and carry limits', async ({ page }) => {
  let request;
  await page.route('**/api/inventory/pouch/upgrade', async route => {
    request = route.request().postDataJSON();
    await route.fulfill({
      contentType: 'application/json',
      body: JSON.stringify({
        ok: true, level: 2, gold: 4_000_000, stack_cap: 7, carry_cap: 10,
        next_cost: 5_000_000, msg: 'Consumable Pouch tailored to rank 2.',
      }),
    });
  });

  await page.goto('/inventory');
  const pouch = page.locator('#pouchTailoring');
  await expect(pouch).toContainText('Consumable Pouch · Rank 1/3');
  await expect(pouch.locator('#pouchStackCap')).toHaveText('6');
  await expect(pouch.locator('#pouchCarryCap')).toHaveText('9');
  await pouch.getByRole('button', { name: /Tailor rank 2/ }).click();

  const dialog = page.getByRole('dialog');
  await expect(dialog).toContainText('1,000,000');
  await expect(dialog).toContainText('+1 Abyss loot stack · +1 equipped run carry.');
  await dialog.getByRole('button', { name: /Tailor · 1,000,000g/ }).click();

  await expect.poll(() => request).toEqual({});
  await expect(pouch).toContainText('Rank 2/3');
  await expect(pouch.locator('#pouchStackCap')).toHaveText('7');
  await expect(pouch.locator('#pouchCarryCap')).toHaveText('10');
  await expect(pouch.getByRole('button')).toContainText('Tailor rank 3 · 🪙 5,000,000');
  await expect(page.locator('#goldPill')).toContainText('4,000,000');
});
