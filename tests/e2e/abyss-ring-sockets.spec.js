const { test, expect } = require('@playwright/test');

test('Forge offers a quoted gem-preserving socket reroll only for rings', async ({ page }) => {
  const pageErrors = [];
  page.on('pageerror', error => pageErrors.push(error.message));
  await page.goto('/abyss?gear=1');
  await page.locator('.ab-tab[data-tab-key="forge"]').click();

  const picker = page.locator('#forgeItemSelect');
  const row = page.locator('#forgeRingSocketRow');
  const button = page.locator('#btnForgeRingSocketReroll');

  await picker.selectOption('equipped:MainHand');
  await expect(row).toBeHidden();

  await picker.selectOption('equipped:Finger1');
  await expect(row).toBeVisible();
  await expect(button).toBeEnabled();
  await expect(button).toContainText('5🔷');
  await expect(row).toContainText('1–3 sockets');
  await expect(row).toContainText('preserved');

  const contract = await page.evaluate(() => ({
    operation: abyssForgeCommitOperations['/api/abyss/reroll_ring_sockets'],
    hasAction: typeof forgeRingSocketReroll === 'function',
  }));
  expect(contract).toEqual({ operation: 'reroll_ring_sockets', hasAction: true });
  expect(pageErrors).toEqual([]);
});
