const { test, expect } = require('@playwright/test');

test('shop permanent tokens cancel safely and keep separate price ladders across reloads', async ({ page }) => {
  const requests = [];
  page.on('request', request => { if (request.url().endsWith('/api/shop/buffs')) requests.push(request.postDataJSON()); });
  await page.goto('/shop');
  const rarity = page.locator('#shopBuff-rarity');
  const quantity = page.locator('#shopBuff-quantity');
  await rarity.getByRole('button', { name: 'Buy rarity token' }).click();
  const dialog = page.getByRole('dialog');
  await expect(dialog).toContainText('25,000,000 → 24,000,000');
  await expect(dialog).toContainText('0.0% → 0.1%');
  await expect(dialog).toContainText('up to 20');
  await page.keyboard.press('Escape');
  expect(requests).toHaveLength(0);
  await expect(rarity.getByRole('button')).toBeFocused();
  await rarity.getByRole('button').click();
  await dialog.getByRole('button', { name: 'Buy for 1,000,000 gold' }).click();
  await expect(rarity.locator('.shop-buff-action')).toHaveAttribute('data-owned', '1');
  await expect(rarity.locator('.shop-buff-action')).toHaveAttribute('data-price', '2000000');
  await expect(quantity.locator('.shop-buff-action')).toHaveAttribute('data-price', '1000000');
  await expect(page.locator('#exState')).toHaveAttribute('data-gold', '24000000');
  await quantity.getByRole('button').click();
  await dialog.getByRole('button', { name: 'Buy for 1,000,000 gold' }).click();
  await expect(quantity.locator('.shop-buff-action')).toHaveAttribute('data-owned', '1');
  await expect(page.locator('#exState')).toHaveAttribute('data-gold', '23000000');
  await page.reload();
  await expect(rarity.locator('.shop-buff-action')).toHaveAttribute('data-owned', '1');
  await expect(quantity.locator('.shop-buff-action')).toHaveAttribute('data-owned', '1');
  expect(requests).toEqual([
    { kind: 'rarity', amount: 1, expected_owned: 0, expected_gold: 25000000 },
    { kind: 'quantity', amount: 1, expected_owned: 0, expected_gold: 24000000 },
  ]);
});

for (const failure of ['stale', 'unconfirmed']) {
  test(`shop ${failure} token purchase locks shared wallet mutations and offers recovery`, async ({ page }) => {
    await page.route('**/api/shop/buffs', route => failure === 'stale'
      ? route.fulfill({ json: { ok: false, review_required: true, error: 'Your token count changed. Refresh to verify.' } })
      : route.abort('connectionreset'));
    await page.goto('/shop');
    await page.getByRole('button', { name: 'Buy rarity token' }).click();
    await page.getByRole('dialog').getByRole('button', { name: 'Buy for 1,000,000 gold' }).click();
    await expect(page.locator('#shopBuff-rarity .shop-local-status')).toBeVisible();
    await expect.poll(() => page.locator('.shop-buy-action,.shop-exchange-action,.shop-buff-action').evaluateAll(buttons => buttons.every(button => button.disabled))).toBe(true);
    await page.locator('#shopMore').click();
    await expect(page.locator('.shop-card')).toHaveCount(24);
    await expect(page.locator('.shop-card').nth(12).locator('.shop-buy-action')).toBeDisabled();
    await expect(page.getByRole('button', { name: 'Refresh to verify the unconfirmed shop action' })).toBeEnabled();
    await expect(page.locator('#exState')).toHaveAttribute('data-gold', '25000000');
  });
}

test('shop buffs show deterministic boosted stock and fractional quantity on mobile', async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto('/shop?buff_fixture=boosted');
  await expect(page.locator('.shop-card')).toHaveCount(12);
  expect(await page.evaluate(() => shopCards.length)).toBe(97);
  expect(await page.evaluate(() => shopCards.filter(card => card.querySelector('.shop-rarity-boost')).length)).toBe(20);
  const ids = await page.evaluate(() => shopCards.map(card => card.dataset.id));
  await page.reload();
  expect(await page.evaluate(() => shopCards.map(card => card.dataset.id))).toEqual(ids);
  await page.getByRole('link', { name: 'Jump to permanent shop buffs' }).click();
  await expect(page.getByRole('heading', { name: 'Permanent shop buffs' })).toBeInViewport();
  const action = page.getByRole('button', { name: 'Buy quantity token' });
  await action.scrollIntoViewIfNeeded();
  await expect(action).toBeDisabled();
  await expect(page.locator('#shopBuff-quantity .shop-buff-affordability')).toContainText('Need');
  expect((await action.boundingBox()).height).toBeGreaterThanOrEqual(44);
  expect(await page.evaluate(() => document.documentElement.scrollWidth-document.documentElement.clientWidth)).toBeLessThanOrEqual(1);
});

test('shop tokens keep growing beyond 100 percent and large stock remains paged', async ({ page }) => {
  await page.goto('/shop?buff_fixture=endless');
  const rarity = page.locator('#shopBuff-rarity');
  const quantity = page.locator('#shopBuff-quantity');
  await expect(rarity).toContainText('1000.0% bonus');
  expect(await page.evaluate(() => shopCards.filter(card => card.querySelector('.shop-eternal-bonus')).length)).toBe(20);
  expect(await page.evaluate(() => shopCards.length)).toBe(241);
  await expect(page.locator('.shop-card')).toHaveCount(12);
  await rarity.getByRole('button').click();
  await expect(page.getByRole('dialog')).toContainText('1000.0% → 1000.1%');
  await page.getByRole('dialog').getByRole('button', { name: 'Buy for 1,000,000,000 gold' }).click();
  await expect(rarity.locator('button')).toHaveAttribute('data-owned', '10001');
  await expect(rarity.locator('button')).toHaveAttribute('data-price', '1000000000');
  await expect(rarity.getByRole('button')).toBeEnabled();
  await expect(quantity.locator('button')).toHaveAttribute('data-price', '1000000000');
  const boosted = await page.evaluate(() => shopCards.filter(card => card.querySelector('.shop-rarity-boost')).map(card => card.dataset.id));
  await quantity.getByRole('button').click();
  await page.getByRole('dialog').getByRole('button', { name: 'Buy for 1,000,000,000 gold' }).click();
  await expect(quantity.locator('button')).toHaveAttribute('data-owned', '10001');
  expect(await page.evaluate(() => shopCards.filter(card => card.querySelector('.shop-rarity-boost')).map(card => card.dataset.id))).toEqual(boosted);
  await page.getByRole('link', { name: 'Next stock page' }).click();
  await expect(page.locator('#shopGrid')).toHaveAttribute('data-stock-page', '1');
  await expect(page.locator('.shop-card')).toHaveCount(12);
  expect(await page.evaluate(() => shopCards.length)).toBe(240);
  await expect(page.locator('.featured-item')).toHaveCount(0);
  const purchase = page.waitForRequest('**/api/shop/buy');
  await page.route('**/api/shop/buy', route => route.fulfill({ json: { ok: false, review_required: true, error: 'Fixture purchase review complete.' } }));
  await page.locator('.shop-buy-action').first().click();
  await page.getByRole('dialog').getByRole('button', { name: /^Buy for/ }).click();
  expect((await purchase).postDataJSON().stock_page).toBe(1);
});


test('bulk tokens review the full price, cancel safely, and purchase both kinds', async ({ page }) => {
  const requests = [];
  const errors = [];
  page.on('pageerror', error => errors.push(error.message));
  page.on('request', request => { if (request.url().endsWith('/api/shop/buffs')) requests.push(request.postDataJSON()); });
  await page.goto('/shop');
  const rarity = page.locator('#shopBuff-rarity');
  const quantity = page.locator('#shopBuff-quantity');
  const amount = rarity.getByRole('spinbutton');
  const action = rarity.locator('.shop-buff-action');
  for (const invalid of ['', '0', '-1', '1.5', '999999999999999999']) {
    await amount.fill(invalid);
    await expect(action).toBeDisabled();
  }
  await amount.fill('7');
  await expect(rarity.locator('.shop-buff-total')).toHaveText('Total: 28,000,000 gold');
  await expect(rarity.locator('.shop-buff-affordability')).toHaveText('Need 3,000,000 more gold');
  await expect(action).toBeDisabled();
  await amount.fill('5');
  await action.click();
  const dialog = page.getByRole('dialog');
  await expect(dialog).toContainText('5 permanent rarity tokens');
  await expect(dialog).toContainText('Exact total: 15,000,000 gold');
  await expect(dialog).toContainText('0.0% → 0.5%');
  await expect(dialog).toContainText('25,000,000 → 10,000,000 gold');
  await expect(dialog).toContainText('Next rarity token: 6,000,000 gold');
  await expect(amount).toBeDisabled();
  await page.keyboard.press('Escape');
  expect(requests).toHaveLength(0);
  await expect(amount).toBeEnabled();
  await expect(action).toBeFocused();
  await action.click();
  await dialog.getByRole('button', { name: 'Buy for 15,000,000 gold' }).click();
  await expect(action).toHaveAttribute('data-owned', '5');
  await expect(quantity.locator('.shop-buff-action')).toHaveAttribute('data-price', '1000000');
  await quantity.getByRole('spinbutton').fill('3');
  await quantity.getByRole('button', { name: 'Buy 3 quantity tokens' }).click();
  await expect(dialog).toContainText('0.0% → 0.3%');
  await dialog.getByRole('button', { name: 'Buy for 6,000,000 gold' }).click();
  await expect(quantity.locator('.shop-buff-action')).toHaveAttribute('data-owned', '3');
  await expect(page.locator('#exState')).toHaveAttribute('data-gold', '4000000');
  await page.reload();
  await expect(action).toHaveAttribute('data-owned', '5');
  await expect(quantity.locator('.shop-buff-action')).toHaveAttribute('data-owned', '3');
  expect(requests).toEqual([
    { kind: 'rarity', amount: 5, expected_owned: 0, expected_gold: 25000000 },
    { kind: 'quantity', amount: 3, expected_owned: 0, expected_gold: 10000000 },
  ]);
  expect(errors).toEqual([]);
});

test('bulk tokens at the price cap remain usable on mobile', async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto('/shop?buff_fixture=endless');
  const rarity = page.locator('#shopBuff-rarity');
  await rarity.getByRole('spinbutton').fill('3');
  await rarity.getByRole('button', { name: 'Buy 3 rarity tokens' }).click();
  const dialog = page.getByRole('dialog');
  await expect(dialog).toContainText('1000.0% → 1000.3%');
  await expect(dialog).toContainText('3,000,000,000 → 0 gold');
  await expect(dialog).toContainText('Next rarity token: 1,000,000,000 gold');
  await dialog.getByRole('button', { name: 'Buy for 3,000,000,000 gold' }).click();
  await expect(rarity.locator('.shop-buff-action')).toHaveAttribute('data-owned', '10003');
  await expect(page.locator('#exState')).toHaveAttribute('data-gold', '0');
  await rarity.scrollIntoViewIfNeeded();
  expect((await rarity.getByRole('spinbutton').boundingBox()).height).toBeGreaterThanOrEqual(44);
  expect(await page.evaluate(() => document.documentElement.scrollWidth-document.documentElement.clientWidth)).toBeLessThanOrEqual(1);
  await page.screenshot({ path: 'test-results/shop-bulk-mobile.png' });
});


test('a rejected bulk purchase keeps the revised amount after cancelling another review', async ({ page }) => {
  await page.route('**/api/shop/buffs', route => route.fulfill({ json: { ok: false, error: 'Purchase could not be saved.' } }));
  await page.goto('/shop');
  const rarity = page.locator('#shopBuff-rarity');
  await rarity.getByRole('spinbutton').fill('5');
  await rarity.getByRole('button', { name: 'Buy 5 rarity tokens' }).click();
  await page.getByRole('dialog').getByRole('button', { name: 'Buy for 15,000,000 gold' }).click();
  await expect(rarity.getByRole('button')).toBeEnabled();
  await rarity.getByRole('spinbutton').fill('3');
  await rarity.getByRole('button', { name: 'Buy 3 rarity tokens' }).click();
  await page.keyboard.press('Escape');
  await expect(rarity.getByRole('button')).toHaveText('Buy 3 rarity tokens');
  await expect(rarity.getByRole('spinbutton')).toHaveValue('3');
  await expect(page.locator('#exState')).toHaveAttribute('data-gold', '25000000');
});
