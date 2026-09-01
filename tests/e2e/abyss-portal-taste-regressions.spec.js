const { test, expect } = require('@playwright/test');

async function expectActiveNavInView(page) {
  const nav = page.locator('.nav');
  const active = nav.locator('a.active');
  const [navBox, activeBox] = await Promise.all([nav.boundingBox(), active.boundingBox()]);
  expect(activeBox.x).toBeGreaterThanOrEqual(navBox.x - 1);
  expect(activeBox.x + activeBox.width).toBeLessThanOrEqual(navBox.x + navBox.width + 1);
}

test('armoury leads with auditable combat readiness and the complete loadout', async ({ page }) => {
  await page.setViewportSize({ width: 1440, height: 900 });
  await page.goto('/armory-fixture');

  await expect(page.getByRole('heading', { name: 'Armoury', exact: true })).toBeVisible();
  const readiness = page.getByRole('region', { name: 'Combat readiness' });
  await expect(readiness).toBeVisible();
  await expect(readiness.getByRole('progressbar', { name: 'Current health' })).toBeVisible();
  await expect(readiness.getByText('Mana capacity')).toBeVisible();

  const equipment = page.getByRole('region', { name: 'Equipped loadout' });
  await expect(equipment).toBeVisible();
  await expect(equipment.locator('.gear-cell')).toHaveCount(30);
  await expect(equipment.getByText('37 / 100 durability')).toBeVisible();
  await expect(equipment.getByRole('button', { name: 'Review identification cost for Head' })).toBeVisible();

  const equipmentBox = await equipment.boundingBox();
  expect(equipmentBox.y).toBeLessThan(900);

  await page.setViewportSize({ width: 390, height: 844 });
  const durability = equipment.locator('.gear-durability').first();
  expect(await durability.evaluate(node => node.scrollWidth - node.clientWidth)).toBeLessThanOrEqual(1);
});

test('armoury identification reviews the signed price and preserves the irreversible commit contract', async ({ page }) => {
  await page.route('**/api/abyss/forge/quote', async route => {
    expect(route.request().postDataJSON()).toEqual({ operation: 'identify', slot: 'Head', parameters: {} });
    await route.fulfill({
      contentType: 'application/json',
      body: JSON.stringify({ ok: true, quote: { cost: { gold: 1000 }, token: 'signed-identify-token', confirmation_phrase: 'FORGE IDENTIFY' } }),
    });
  });
  await page.route('**/api/abyss/identify', async route => {
    await route.fulfill({
      contentType: 'application/json',
      body: JSON.stringify({ ok: true, msg: 'Item identified.', gold: '12,500' }),
    });
  });
  await page.goto('/armory-fixture');

  await page.getByRole('button', { name: 'Review identification cost for Head' }).click();
  const dialog = page.getByRole('dialog');
  await expect(dialog).toContainText('1,000 gold');

  const commitRequest = page.waitForRequest('**/api/abyss/identify');
  await dialog.getByRole('button', { name: 'Identify item' }).click();
  const request = await commitRequest;
  expect(request.headers()['x-abyss-forge-quote']).toBe('signed-identify-token');
  expect(request.headers()['idempotency-key']).toMatch(/^armory-identify-/);
  expect(request.postDataJSON()).toEqual({ slot: 'Head', confirmation: 'FORGE IDENTIFY' });
});

test('armoury returns focus to identification after a cancelled review', async ({ page }) => {
  await page.route('**/api/abyss/forge/quote', route => route.fulfill({
    contentType: 'application/json',
    body: JSON.stringify({ ok: true, quote: { cost: { gold: 1000 }, token: 'signed-identify-token', confirmation_phrase: 'FORGE IDENTIFY' } }),
  }));
  await page.goto('/armory-fixture');

  const identifyButton = page.getByRole('button', { name: 'Review identification cost for Head' });
  await identifyButton.click();
  await page.getByRole('dialog').getByRole('button', { name: 'Keep unidentified' }).click();
  await expect(identifyButton).toBeEnabled();
  await expect(identifyButton).toBeFocused();
});

test('armoury locks an ambiguous identification commit until refresh', async ({ page }) => {
  await page.route('**/api/abyss/forge/quote', route => route.fulfill({
    contentType: 'application/json',
    body: JSON.stringify({ ok: true, quote: { cost: { gold: 1000 }, token: 'signed-identify-token', confirmation_phrase: 'FORGE IDENTIFY' } }),
  }));
  await page.route('**/api/abyss/identify', route => route.abort('connectionreset'));
  await page.goto('/armory-fixture');

  const identifyButton = page.getByRole('button', { name: 'Review identification cost for Head' });
  await identifyButton.click();
  await page.getByRole('dialog').getByRole('button', { name: 'Identify item' }).click();
  await expect(page.locator('#armoryStatus')).toContainText('result is unconfirmed');
  await expect(identifyButton).toBeDisabled();
  await expect(identifyButton).toHaveText('Locked');
  const recovery = identifyButton.locator('xpath=..').getByRole('button', { name: 'Refresh to verify the unconfirmed identification' });
  await expect(recovery).toBeEnabled();
  await page.getByRole('dialog').getByRole('button', { name: 'Close' }).click();
  await expect(recovery).toBeFocused();
});

test('armoury identification retains the direct rollback path when Forge quotes are disabled', async ({ page }) => {
  let quoteRequests = 0;
  await page.route('**/api/abyss/forge/quote', async route => {
    quoteRequests += 1;
    await route.fulfill({ contentType: 'application/json', body: JSON.stringify({ ok: false }) });
  });
  await page.route('**/api/abyss/identify', async route => {
    await route.fulfill({
      contentType: 'application/json',
      body: JSON.stringify({ ok: true, msg: 'Item identified.', gold: '12,500' }),
    });
  });
  await page.goto('/armory-fixture?forge=0');

  await page.getByRole('button', { name: 'Review identification cost for Head' }).click();
  const dialog = page.getByRole('dialog');
  await expect(dialog).toContainText('Exact-price review is temporarily unavailable');

  const commitRequest = page.waitForRequest('**/api/abyss/identify');
  await dialog.getByRole('button', { name: 'Identify item' }).click();
  const request = await commitRequest;
  expect(quoteRequests).toBe(0);
  expect(request.headers()['x-abyss-forge-quote']).toBeUndefined();
  expect(request.postDataJSON()).toEqual({ slot: 'Head', confirmation: 'FORGE IDENTIFY' });
});

test('shop makes the rotating stock immediately searchable and every purchase unambiguous', async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto('/shop');
  await expectActiveNavInView(page);
  await expect(page.locator('.logout')).toBeVisible();

  const market = page.getByRole('region', { name: 'Rotating item stock' });
  await expect(market).toBeVisible();
  await expect(market.getByText('Fixed server prices')).toBeVisible();
  await expect(page.locator('#shopMsg')).toHaveAttribute('aria-live', 'polite');
  await expect(page.locator('[role="button"] button')).toHaveCount(0);

  const firstBuy = market.getByRole('button', { name: /^Buy .+ for .+ gold$/ }).first();
  await expect(firstBuy).toBeVisible();
  const firstCard = market.locator('.shop-card').first();
  const exactFirstPrice = new Intl.NumberFormat('en-US').format(Number(await firstCard.getAttribute('data-price')));
  await expect(firstCard.locator('.price')).toContainText(exactFirstPrice);
  await expect(firstBuy).toHaveAccessibleName(new RegExp(` for ${exactFirstPrice.replaceAll(',', '\\,')} gold$`));
  const firstBuyBox = await firstBuy.boundingBox();
  expect(firstBuyBox.y + firstBuyBox.height).toBeLessThanOrEqual(844);
  await expect(market.locator('.shop-card:visible')).toHaveCount(12);
  const showMore = market.getByRole('button', { name: 'Show 12 more' });
  const firstRevealedInspect = market.locator('.shop-card').nth(12).getByRole('button', { name: /^Inspect / });
  await expect(showMore).toBeVisible();
  await showMore.click();
  await expect(market.locator('.shop-card:visible')).toHaveCount(24);
  await expect(firstRevealedInspect).toBeFocused();
  await expect(page.getByRole('link', { name: 'Jump to currency exchange' })).toBeVisible();

  const search = market.getByRole('searchbox', { name: 'Search rotating stock' });
  await search.fill('Vine-Whip Belt');
  await expect(market.locator('.shop-card:visible')).toHaveCount(1);
  await expect(market.locator('#shopResultCount')).toContainText('1 item');
  await expect(market.getByRole('button', { name: /Buy Vine-Whip Belt for .* gold/ })).toBeVisible();

  expect(await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth))
    .toBeLessThanOrEqual(1);
});

test('shop keeps an ambiguous purchase locked until the player refreshes', async ({ page }) => {
  await page.route('**/api/shop/buy', route => route.abort('connectionreset'));
  await page.goto('/shop');

  const buyButton = page.getByRole('button', { name: /^Buy .+ for .+ gold$/ }).first();
  await buyButton.click();
  const purchaseReview = page.getByRole('dialog');
  await expect(purchaseReview).toContainText(/Exact total: .* gold/);
  await expect(purchaseReview).toContainText(/auto-equip|delivered to inventory/);
  await purchaseReview.getByRole('button', { name: /^Buy for .* gold$/ }).click();
  await expect(page.locator('#shopMsg')).toContainText('result is unconfirmed');
  await expect(buyButton).toBeDisabled();
  await expect(buyButton).toHaveText('Locked');
  await expect.poll(() => page.locator('.shop-buy-action').evaluateAll(buttons =>
    buttons.every(button => button.disabled)
  )).toBe(true);
  const recovery = buyButton.locator('xpath=..').getByRole('button', { name: 'Refresh to verify the unconfirmed shop action' });
  await expect(recovery).toBeEnabled();
  await expect(recovery).toBeFocused();
});

test('shop refreshes exact comparisons after the server confirms an auto-equip', async ({ page }) => {
  let shopLoads = 0;
  let releaseBuyResponse;
  page.on('request', request => {
    if (request.isNavigationRequest() && new URL(request.url()).pathname === '/shop') shopLoads += 1;
  });
  await page.route('**/api/shop/buy', async route => {
    await new Promise(resolve => { releaseBuyResponse = resolve; });
    await route.fulfill({
      contentType: 'application/json',
      body: JSON.stringify({ ok: true, bought: 'Fixture Relic and equipped!', gold: 20_000_000, equipped: true }),
    });
  });
  await page.goto('/shop');

  const buyButton = page.getByRole('button', { name: /^Buy .+ for .+ gold$/ }).first();
  await buyButton.click();
  const response = page.waitForResponse('**/api/shop/buy');
  await page.getByRole('dialog').getByRole('button', { name: /^Buy for .* gold$/ }).click();
  await expect.poll(() => typeof releaseBuyResponse).toBe('function');
  await expect.poll(() => page.locator('.shop-buy-action').evaluateAll(buttons =>
    buttons.every(button => button.disabled)
  )).toBe(true);
  releaseBuyResponse();
  await response;

  await expect(page.locator('#shopMsg')).toContainText('Refreshing equipment comparisons');
  await expect(page.locator('.shop-buy-action').first()).toBeDisabled();
  await expect.poll(() => shopLoads).toBe(2);
  await expect(page.getByRole('heading', { name: 'Shop', exact: true })).toBeVisible();
});

test('auction listings become self-contained market cards on mobile without clipped actions', async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await page.route('**/api/ah/notices', async route => {
    await route.fulfill({ contentType: 'application/json', body: JSON.stringify({ ok: true, notices: [] }) });
  });
  await page.goto('/ah');
  await expectActiveNavInView(page);
  await expect(page.locator('.logout')).toBeVisible();

  await expect(page.getByRole('search', { name: 'Search auction listings' })).toBeVisible();
  const listing = page.locator('.ah-row').filter({ hasText: 'Cinder Test Blade' });
  await expect(listing).toBeVisible();
  await listing.getByRole('button', { name: 'Inspect Cinder Test Blade' }).click();
  const inspectorDialog = page.getByRole('dialog');
  await expect(page.locator('.item-inspector')).toBeVisible();
  await expect(inspectorDialog).toBeFocused();
  await page.keyboard.press('Shift+Tab');
  await expect(inspectorDialog.getByRole('button', { name: 'Close' })).toBeFocused();
  await page.keyboard.press('Escape');
  await expect(listing.getByRole('button', { name: 'Buy Cinder Test Blade for 1600 gold' })).toBeVisible();
  await expect(listing.getByRole('button', { name: 'Bid on Cinder Test Blade' })).toBeVisible();
  await expect(listing.getByText('4 sales')).toBeVisible();
  await expect(listing.getByText('in 2h 15m')).toBeVisible();
  await expect(listing.getByText('01 Sep 2026 · 01:15 UTC')).toBeVisible();

  expect(await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth))
    .toBeLessThanOrEqual(1);
  const listingBox = await listing.boundingBox();
  expect(listingBox.x).toBeGreaterThanOrEqual(0);
  expect(listingBox.x + listingBox.width).toBeLessThanOrEqual(390);
});

test('auction reviews exact material-sale proceeds before committing', async ({ page }) => {
  let fillRequest;
  await page.route('**/api/ah/notices', route => route.fulfill({
    contentType: 'application/json',
    body: JSON.stringify({ ok: true, notices: [] }),
  }));
  await page.route('**/api/ah/material_fill', route => {
    fillRequest = route.request();
    return route.fulfill({ contentType: 'application/json', body: JSON.stringify({ ok: true, msg: 'Filled.' }) });
  });
  await page.goto('/ah');

  await page.getByRole('button', { name: 'Fill' }).click();
  const quantityDialog = page.getByRole('dialog');
  const quantity = quantityDialog.getByRole('textbox', { name: /How many units/ });
  await quantity.fill('3');
  await quantityDialog.getByRole('button', { name: 'OK' }).click();

  const review = page.getByRole('dialog');
  await expect(review).toContainText('Exact proceeds: 375 gold');
  expect(fillRequest).toBeUndefined();
  await review.getByRole('button', { name: 'Sell for 375 gold' }).click();
  await expect.poll(() => fillRequest && fillRequest.postDataJSON()).toEqual({ id: 17, count: 3 });
});


test('auction keeps an ambiguous purchase locked and sends an idempotency key', async ({ page }) => {
  let requestKey = '';
  await page.route('**/api/ah/notices', route => route.fulfill({
    contentType: 'application/json',
    body: JSON.stringify({ ok: true, notices: [] }),
  }));
  await page.route('**/api/ah/buy', route => {
    requestKey = route.request().headers()['idempotency-key'];
    return route.abort('connectionreset');
  });
  await page.goto('/ah');

  const buyButton = page.locator('.ah-row').filter({ hasText: 'Cinder Test Blade' }).locator('.auction-buy-button');
  await buyButton.click();
  await page.getByRole('dialog').getByRole('button', { name: 'Buy for 1,600 gold' }).click();

  await expect(page.locator('#ahMsg')).toContainText('result is unconfirmed');
  await expect(buyButton).toBeDisabled();
  await expect(buyButton).toHaveText('Locked');
  const recovery = buyButton.locator('xpath=..').getByRole('button', { name: 'Refresh to verify the unconfirmed market action' });
  await expect(recovery).toBeEnabled();
  await expect(recovery).toBeFocused();
  expect(requestKey).toMatch(/^ah-/);
});

test('auction requires an exact non-cancellable bid review before reserving gold', async ({ page }) => {
  let bidRequest;
  await page.route('**/api/ah/notices', route => route.fulfill({
    contentType: 'application/json',
    body: JSON.stringify({ ok: true, notices: [] }),
  }));
  await page.route('**/api/ah/bid', route => {
    bidRequest = route.request();
    return route.fulfill({ contentType: 'application/json', body: JSON.stringify({ ok: true, msg: 'Bid placed.', gold: 24200 }) });
  });
  await page.goto('/ah');

  await page.getByRole('button', { name: 'Bid on Cinder Test Blade' }).click();
  const amountDialog = page.getByRole('dialog');
  await amountDialog.getByRole('textbox', { name: /Enter a whole-gold bid/ }).fill('800');
  await amountDialog.getByRole('button', { name: 'OK' }).click();

  const review = page.getByRole('dialog');
  await expect(review).toContainText('Reserve exactly 800 gold');
  await expect(review).toContainText('cannot be cancelled');
  expect(bidRequest).toBeUndefined();
  await review.getByRole('button', { name: 'Reserve 800 gold' }).click();
  await expect.poll(() => bidRequest && bidRequest.postDataJSON()).toEqual({ id: 'listing-history', amount: 800 });
});

test('auction rejects an unavailable bid range before opening the amount prompt', async ({ page }) => {
  await page.route('**/api/ah/notices', route => route.fulfill({
    contentType: 'application/json',
    body: JSON.stringify({ ok: true, notices: [] }),
  }));
  await page.goto('/ah');

  const bidButton = page.getByRole('button', { name: 'Bid on Cinder Test Blade' });
  await bidButton.evaluate(button => { button.dataset.currentBid = '1599'; });
  await bidButton.click();

  await expect(page.getByRole('dialog')).toHaveCount(0);
  await expect(page.locator('#ahMsg')).toContainText('bid range is unavailable');
});

test('auction material-order cancellation stops before later prompts or a request', async ({ page }) => {
  let orderRequests = 0;
  await page.route('**/api/ah/notices', route => route.fulfill({
    contentType: 'application/json',
    body: JSON.stringify({ ok: true, notices: [] }),
  }));
  await page.route('**/api/ah/material_order', route => {
    orderRequests += 1;
    return route.fulfill({ contentType: 'application/json', body: JSON.stringify({ ok: true }) });
  });
  await page.goto('/ah');

  await page.getByRole('button', { name: 'Post buy order' }).click();
  await page.getByRole('dialog').getByRole('button', { name: 'OK' }).click();
  const quantityDialog = page.getByRole('dialog');
  await expect(quantityDialog).toContainText('Quantity to buy');
  await quantityDialog.getByRole('button', { name: 'Cancel' }).click();
  await expect(page.getByRole('dialog')).toHaveCount(0);
  expect(orderRequests).toBe(0);
});
