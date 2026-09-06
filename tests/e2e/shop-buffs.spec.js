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
    { kind: 'rarity', expected_owned: 0, expected_gold: 25000000 },
    { kind: 'quantity', expected_owned: 0, expected_gold: 24000000 },
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
