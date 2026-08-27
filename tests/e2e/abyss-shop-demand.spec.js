const { test, expect } = require('@playwright/test');
const { fulfillAbyssAPI } = require('./helpers/abyss');

test('Token Shop discloses and charges the server-wide demand price', async ({ page }) => {
  let purchase;
  await fulfillAbyssAPI(page, (path, body) => {
    if (path.endsWith('/shop/buy')) {
      purchase = body;
      return { ok: true, tokens: 572, gold: 12_000_000, msg: 'Purchased Great Health Potions ×3!' };
    }
    return { ok: false, error: 'unexpected e2e request' };
  });

  await page.goto('/abyss');
  await page.locator('#abyss-tab-shop').click();
  const item = page.locator('#abyssTokenShop [data-shop-category="supplies"]').filter({ hasText: 'Great Health Potions' });
  await expect(item).toContainText('7-day market');
  await expect(item).toContainText('hot · +10%');
  await expect(item).toContainText('24 bought');
  const buy = item.getByRole('button');
  await expect(buy).toContainText('🜲 7');
  await expect(buy).toHaveAttribute('title', /Base 6 tokens · market \+10% from 24 prior-window purchases · exact total 7 tokens · fee 0/);
  await buy.click();

  await expect.poll(() => purchase).toEqual({ item: 'great_potions', quoted_cost: 7 });
  await expect(page.locator('#abToastHost')).toContainText('Purchased Great Health Potions ×3!');
});
