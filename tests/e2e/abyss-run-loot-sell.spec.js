const { test, expect } = require('@playwright/test');
const { fulfillAbyssAPI } = require('./helpers/abyss');

test('ordinary run loot can be sold into the at-risk cache at its exact quote', async ({ page }) => {
  let sale;
  await fulfillAbyssAPI(page, (path, body) => {
    if (path.endsWith('/loot/sell_junk')) {
      sale = body;
      return { ok: true, value: 321, escrow: 3777, msg: 'Junk converted into 321g of at-risk run cache.' };
    }
    if (path.endsWith('/loot/manifest')) return { ok: true, items: [] };
    return { ok: false, error: 'unexpected e2e request' };
  });

  await page.goto('/abyss?active=1');
  await page.evaluate(() => {
    const manifest = document.getElementById('lootManifest');
    manifest.replaceChildren(window.buildAuthoritativeRunLootRow({
      id: 42,
      label: 'Worn Test Blade',
      source: 'Dropped floor 3',
      item_type: 'gear',
      gear_id: 'E2E_JUNK',
      slot: 'MainHand',
      rarity: 'Common',
      rarity_rank: 0,
      can_sell_junk: true,
      sell_value: 321,
    }));
  });

  const sell = page.getByRole('button', { name: 'Sell junk · +321g cache', exact: true });
  await expect(sell).toBeVisible();
  await sell.click();
  const dialog = page.getByRole('dialog');
  await expect(dialog).toContainText('+321g of at-risk run cache');
  await expect(dialog).toContainText('destroyed immediately and cannot be recovered');
  await dialog.getByRole('button', { name: 'Sell · +321g cache' }).click();

  await expect.poll(() => sale).toEqual({ id: 42, quoted_gold: 321 });
  await expect(page.locator('#escrowVal')).toHaveAttribute('data-raw', '3777');
  await expect(page.locator('#lootManifest .abyss-side-loot')).toHaveCount(0);
  await expect(page.locator('#abToastHost')).toContainText('Junk converted into 321g of at-risk run cache.');
  expect(await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth)).toBeLessThanOrEqual(1);

  const css = await page.request.get('/static/abyss_run_loot_sell.css');
  expect(css.status()).toBe(200);
  expect(await css.text()).toContain('.ab-sell-junk');
});
