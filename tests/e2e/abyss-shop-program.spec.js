const { test, expect } = require('@playwright/test');
const { fulfillAbyssAPI } = require('./helpers/abyss');

test('Shop program presents the unified wallet, bundles, gifting, and legacy exchange', async ({ page }) => {
  const pageErrors = [];
  let bundleRequest = null;
  page.on('pageerror', error => pageErrors.push(error.message));
  await fulfillAbyssAPI(page, (path, body) => {
    if (path.endsWith('/shop/bundle')) {
      bundleRequest = body;
      return { ok: true, tokens: 554, gold: 12_000_000, loyalty_punches: 1, msg: 'Delver Supply Crate opened for 18 tokens.' };
    }
    return { ok: false, error: 'unexpected e2e request' };
  });

  await page.goto('/abyss');
  await page.locator('#abyss-tab-shop').click();
  const program = page.locator('.ab-shop-program');
  await expect(program).toBeVisible();
  await expect(program.locator('#shopWalletGold')).not.toBeEmpty();
  await expect(program.locator('#shopWalletTokens')).toContainText('🜲');
  await expect(program.locator('[data-shop-material]')).toHaveCount(4);
  await expect(program.locator('.ab-shop-flash')).toContainText('DAILY MARKET FLASH');
  await expect(program.locator('#shopLoyaltyValue')).toHaveText(/\d+ \/ 10/);

  await program.locator('summary').click();
  await expect(program.getByText('Delver Supply Crate')).toBeVisible();
  await expect(program.getByText('passive Insurance Charm')).toBeVisible();
  await expect(program.locator('#abyssGiftRecipient')).toBeVisible();
  await expect(program.getByRole('button', { name: /Create gift/ })).toContainText('2.5kg fee');
  await expect(program.getByText(/Tokens never hard reset/)).toBeVisible();

  await program.getByRole('button', { name: '🜲 18' }).first().click();
  await expect.poll(() => bundleRequest).toEqual({ bundle: 'delver_supply', quoted_cost: 18 });
  await expect(program.locator('#shopLoyaltyValue')).toHaveText('1 / 10');
  await expect(page.locator('#abToastHost')).toContainText('Delver Supply Crate opened');

  const overflow = await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth);
  expect(overflow).toBeLessThanOrEqual(1);
  expect(pageErrors).toEqual([]);
});

test('Shop program collapses to one column on a narrow viewport', async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto('/abyss');
  await page.locator('#abyss-tab-shop').click();
  const program = page.locator('.ab-shop-program');
	await program.locator('details').evaluate(details => { details.open = true; });
  await expect(program.locator('.ab-shop-program-grid')).toHaveCSS('grid-template-columns', /\d+(\.\d+)?px/);
  const overflow = await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth);
  expect(overflow).toBeLessThanOrEqual(1);
});
