const { test, expect } = require('@playwright/test');
const { fulfillAbyssAPI } = require('./helpers/abyss');

async function expandWorkbench(workbench) {
  if (!await workbench.evaluate(node => node.open)) await workbench.locator(':scope > summary').click();
  await workbench.locator('details').evaluateAll(sections => sections.forEach(section => { section.open = true; }));
}

async function openMainShop(page) {
  await page.goto('/shop');
  await expect(page.locator('.shop-card')).not.toHaveCount(0);
  const workbench = page.locator('#shopWorkbench');
  await expandWorkbench(workbench);
}

async function openAbyssShop(page) {
  await fulfillAbyssAPI(page, () => ({ ok: false, error: 'Unmocked shop fixture request' }));
  await page.goto('/abyss');
  await page.locator('#abyss-tab-shop').click();
  const workbench = page.locator('#abyssShopWorkbench');
  await expandWorkbench(workbench);
}

function mainCard(page, id) {
  return page.locator('.shop-card').filter({ has: page.locator('.shop-buy-action') })
    .and(page.locator('[data-id="' + id + '"]'));
}

function abyssPotion(page) {
  return page.locator('#abyssTokenShop [data-shop-key="great_potions"]');
}

function abyssBuy(page) {
  return abyssPotion(page).locator('button[onclick*="abyssShopBuy"]');
}

function exactNumber(value) {
  return new RegExp(String(value).split('').join('[,\\s]?'));
}

test('main shop price, rarity, element and no-loss filters constrain visible stock', async ({ page }) => {
  await openMainShop(page);
  const first = page.locator('.shop-card').first();
  const price = Number(await first.getAttribute('data-price'));
  const metadata = (await first.locator('.inv-meta').first().textContent()).split('·').map(value => value.trim());
  await page.locator('#swMinPrice').fill(String(price));
  await page.locator('#swMaxPrice').fill(String(price));
  await expect.poll(() => page.locator('.shop-card:visible').evaluateAll(cards =>
    cards.map(card => Number(card.dataset.price)))).toEqual(expect.arrayContaining([price]));
  const prices = await page.locator('.shop-card:visible').evaluateAll(cards => cards.map(card => Number(card.dataset.price)));
  expect(prices.every(value => value === price)).toBe(true);
  await page.locator('#swMinPrice').fill('');
  await page.locator('#swMaxPrice').fill('');
  await page.locator('#swRarity').selectOption({ label: metadata[0] });
  await expect(page.locator('.shop-card:visible').first().locator('.inv-meta').first()).toContainText(metadata[0]);
  const rarityRows = await page.locator('.shop-card:visible .inv-meta').evaluateAll(nodes =>
    nodes.filter(node => node === node.parentElement.querySelector('.inv-meta')).map(node => node.textContent.split('·')[0].trim()));
  expect(rarityRows.every(value => value === metadata[0])).toBe(true);
  if (metadata[2]) {
    await page.locator('#swElement').selectOption({ label: metadata[2] });
    await expect(page.locator('.shop-card:visible').first().locator('.inv-meta').first()).toContainText(metadata[2]);
  }
  await page.locator('#swNoLoss').check();
  await expect(page.locator('.shop-card:visible .shop-stat-loss')).toHaveCount(0);
});

test('watchlist and visual preferences survive reload without a purchase', async ({ page }) => {
  const purchases = [];
  await page.route('**/api/shop/buy', route => { purchases.push(route.request().postDataJSON()); return route.abort(); });
  await openMainShop(page);
  const first = page.locator('.shop-card').first();
  const id = await first.getAttribute('data-id');
  const name = await first.getAttribute('data-name');
  await first.locator('.sw-watch').click();
  await expect(page.locator('#swWatchlist')).toContainText(name);
  await page.locator('#swDensity').selectOption('compact');
  await page.locator('#swContrast').selectOption('high');
  await page.reload();
  const workbench = page.locator('#shopWorkbench');
  await expandWorkbench(workbench);
  await expect(page.locator('#swWatchlist')).toContainText(name);
  await expect(page.locator('#swDensity')).toHaveValue('compact');
  await expect(page.locator('#swContrast')).toHaveValue('high');
  await page.locator('#swWishlistOnly').check();
  await expect(page.locator('.shop-card:visible')).toHaveCount(1);
  await expect(mainCard(page, id)).toBeVisible();
  expect(purchases).toEqual([]);
});

test('purchase plan totals exact prices, filters planned items and removes an entry', async ({ page }) => {
  await openMainShop(page);
  const entries = await page.locator('.shop-card').evaluateAll(cards => cards.slice(0, 2).map(card =>
    ({ id: card.dataset.id, name: card.dataset.name, price: Number(card.dataset.price) })));
  for (const entry of entries) await mainCard(page, entry.id).locator('.sw-plan').click();
  await expect(page.locator('#swPlanList .sw-plan-remove')).toHaveCount(2);
  await expect(page.locator('#swPlanSummary')).toContainText(exactNumber(entries[0].price + entries[1].price));
  await page.locator('#swGoldReserve').fill('20000000');
  await page.locator('#swSpendLimit').fill('3000000');
  await page.locator('#swPlanOnly').check();
  await expect(page.locator('.shop-card:visible')).toHaveCount(2);
  await page.locator('#swPlanList .sw-plan-remove').first().click();
  await expect(page.locator('#swPlanList .sw-plan-remove')).toHaveCount(1);
  await expect(page.locator('.shop-card:visible')).toHaveCount(1);
  const remainingPrice = Number(await page.locator('.shop-card:visible').getAttribute('data-price'));
  await expect(page.locator('#swPlanSummary')).toContainText(exactNumber(remainingPrice));
});

test('comparison shows selected item identities and a saved search restores its query', async ({ page }) => {
  await openMainShop(page);
  const entries = await page.locator('.shop-card').evaluateAll(cards => cards.slice(0, 2).map(card =>
    ({ id: card.dataset.id, name: card.dataset.name })));
  for (const entry of entries) await mainCard(page, entry.id).locator('.sw-compare').click();
  for (const entry of entries) await expect(page.locator('#swCompareTable')).toContainText(entry.name);
  await page.locator('#shopSearch').fill(entries[0].name);
  await page.locator('#swSearchName').fill('Exact test item');
  await page.locator('#swSaveSearch').click();
  await page.locator('#shopSearch').fill('No matching test item whatsoever');
  await expect(page.locator('.shop-card:visible')).toHaveCount(0);
  await page.locator('#swSavedSearches').selectOption({ label: 'Exact test item' });
  await page.locator('#swApplySearch').click();
  await expect(page.locator('#shopSearch')).toHaveValue(entries[0].name);
  await expect(mainCard(page, entries[0].id)).toBeVisible();
});

test('maximum buff quantity respects both gold reserve and spend limit', async ({ page }) => {
  const purchases = [];
  await page.route('**/api/shop/buffs', route => { purchases.push(route.request().postDataJSON()); return route.abort(); });
  await openMainShop(page);
  await page.locator('#swGoldReserve').fill('20000000');
  await page.locator('#swSpendLimit').fill('3000000');
  await page.locator('#swBuffMax-rarity').click();
  // Fixture wallet is25m; reserve leaves5m and spend limit allows3m.
  // The first two escalating tokens cost1m+2m; three would cost6m.
  await expect(page.locator('#shopBuffAmount-rarity')).toHaveValue('2');
  await expect(page.locator('#shopBuff-rarity .shop-buff-total')).toContainText('3,000,000');
  await expect(page.locator('#swBuffProjection-rarity')).toContainText('Planned bonus increase: 0.2%');
  for (const amount of ['0', '']) {
    await page.locator('#shopBuffAmount-rarity').fill(amount);
    await expect(page.locator('#swBuffProjection-rarity')).toHaveText('Choose an amount to preview the bonus increase.');
  }
  expect(purchases).toEqual([]);
});

test('Abyss search and currency filters combine without changing the wallet', async ({ page }) => {
  await openAbyssShop(page);
  const gold = await page.locator('#shopWalletGold').textContent();
  const tokens = await page.locator('#shopWalletTokens').textContent();
  await page.locator('#asSearch').fill('Great Health Potions');
  await expect(abyssPotion(page)).toBeVisible();
  await page.locator('#asCurrency').selectOption('gold');
  await expect(abyssPotion(page)).toBeHidden();
  await page.locator('#asCurrency').selectOption('tokens');
  await expect(abyssPotion(page)).toBeVisible();
  const visibleNames = await page.locator('#abyssTokenShop [data-shop-key]:visible').evaluateAll(rows =>
    rows.map(row => row.dataset.shopName));
  expect(visibleNames.length).toBeGreaterThan(0);
  expect(visibleNames.every(name => /great health potions/i.test(name))).toBe(true);
  await expect(page.locator('#shopWalletGold')).toHaveText(gold);
  await expect(page.locator('#shopWalletTokens')).toHaveText(tokens);
});

test('Abyss duplicate purchase attempts share one pending request and permanent feedback', async ({ page }) => {
  await openAbyssShop(page);
  let pending;
  let release;
  const requests = [];
  await page.route('**/api/abyss/shop/buy', async route => {
    requests.push(route.request().postDataJSON());
    pending = route;
    await new Promise(resolve => { release = resolve; });
  });
  try {
    await abyssBuy(page).evaluate(button => { button.click(); button.click(); });
    await expect.poll(() => requests.length).toBe(1);
    await expect(abyssBuy(page)).toBeDisabled();
    await pending.fulfill({ json: { ok: true, tokens: 572, gold: 12000000, msg: 'One verified potion purchase.' } });
    release();
    release = null;
    await expect(page.locator('#asStatus')).toContainText('One verified potion purchase.');
    await expect(abyssBuy(page)).toBeEnabled();
    expect(requests).toEqual([{ item: 'great_potions', quoted_cost: 7 }]);
  } finally {
    if (release) { await pending.abort(); release(); }
  }
});

for (const failure of ['network', 'unreadable response']) {
  test(`Abyss ${failure} locks purchases until explicit verification`, async ({ page }) => {
    await openAbyssShop(page);
    const requests = [];
    await page.route('**/api/abyss/shop/buy', route => {
      requests.push(route.request().postDataJSON());
      return failure === 'network' ? route.abort('connectionreset') :
        route.fulfill({ status: 200, contentType: 'text/html', body: '<h1>Unconfirmed upstream response</h1>' });
    });
    await abyssBuy(page).click();
    await expect(page.locator('#asStatus')).toContainText(/verify|unconfirmed|unknown/i);
    await expect(abyssBuy(page)).toBeDisabled();
    await expect.poll(() => page.locator('#abyssTokenShop button[onclick*="abyssShopBuy"]').evaluateAll(buttons =>
      buttons.length > 0 && buttons.every(button => button.disabled))).toBe(true);
    await expect(page.locator('#asVerify')).toBeVisible();
    const reloaded = page.waitForEvent('domcontentloaded');
    await page.locator('#asVerify').click();
    await reloaded;
    await page.locator('#abyss-tab-shop').click();
    await expect(abyssBuy(page)).toBeEnabled();
    expect(requests).toHaveLength(1);
  });
}

test('Abyss price rejection updates the quoted cost and permits a normal retry', async ({ page }) => {
  await openAbyssShop(page);
  const requests = [];
  await page.route('**/api/abyss/shop/buy', route => {
    requests.push(route.request().postDataJSON());
    return route.fulfill({ json: requests.length === 1 ? {
      ok: false, price_changed: true, current_cost: 9, error: 'Market price changed to9 tokens. Review and retry.',
    } : { ok: true, tokens: 570, gold: 12000000, msg: 'Purchased at the refreshed9-token price.' } });
  });
  await abyssBuy(page).click();
  await expect(page.locator('#asStatus')).toContainText('Market price changed');
  await expect(abyssBuy(page)).toBeEnabled();
  await expect(abyssBuy(page)).toContainText('9');
  await expect(abyssPotion(page)).toHaveAttribute('data-token-cost', '9');
  await abyssBuy(page).click();
  await expect(page.locator('#asStatus')).toContainText('Purchased at the refreshed9-token price.');
  expect(requests).toEqual([{ item: 'great_potions', quoted_cost: 7 }, { item: 'great_potions', quoted_cost: 9 }]);
});

test('closed shop comparisons keep stat tables detached and reveal intact values on demand', async ({ page }) => {
  await openMainShop(page);
  const card = page.locator('.shop-card').first();
  const details = card.locator('details.item-tools');
  const comparison = JSON.parse(await card.getAttribute('data-comparison-detail'));
  const rows = comparison.all_stats || comparison.changes;
  expect(rows.length).toBeGreaterThan(0);
  await expect(details).not.toHaveAttribute('open', '');
  await expect(page.locator('.shop-card details.item-tools .item-stat-table')).toHaveCount(0);
  await details.locator(':scope > summary').click();
  const table = details.locator('.item-stat-table');
  await expect(table).toBeVisible();
  await expect(table.locator('thead th')).toHaveText(['Stat', 'Current', 'Candidate', 'Difference', 'Change %']);
  await details.getByLabel('Show unchanged').check();
  await details.getByLabel('Show flavour').check();
  await expect(table.locator('tbody tr')).toHaveCount(rows.length);
  for (const [index, stat] of rows.entries()) {
    const row = table.locator('tbody tr').nth(index);
    await expect(row.locator('th')).toContainText(stat.label);
    for (const [column, value] of [stat.before, stat.after, stat.delta].entries()) {
      await expect(row.locator('td span').nth(column)).toHaveAttribute('title', String(value));
    }
  }
  const rendered = await table.innerText();
  await details.locator(':scope > summary').click();
  await expect(details.locator('.item-stat-table')).toHaveCount(0);
  await details.locator(':scope > summary').click();
  await expect(details.locator('.item-stat-table')).toHaveText(rendered, { useInnerText: true });
});

test('external Abyss wallet updates immediately refresh affordability and filtered offers', async ({ page }) => {
  await openAbyssShop(page);
  const purchases = [];
  await page.route('**/api/abyss/shop/buy', route => { purchases.push(route.request().postDataJSON()); return route.abort(); });
  const tokenCost = Number(await abyssPotion(page).getAttribute('data-token-cost'));
  await page.locator('#asAffordable').check();
  await expect(abyssPotion(page)).toBeVisible();
  await page.evaluate(({ gold, tokens }) => { window.setGold(gold); window.setTokens(tokens); },
    { gold: 999, tokens: tokenCost - 1 });
  await expect(abyssPotion(page)).toBeHidden();
  await expect(page.locator('#shopWalletGold')).toContainText('999');
  await expect(page.locator('#shopWalletTokens')).toContainText(String(tokenCost - 1));
  await page.locator('#asAffordable').uncheck();
  await expect(abyssBuy(page)).toBeDisabled();
  await expect(abyssPotion(page).locator('.sw-offer-status')).toContainText('Need 1 more tokens');
  await page.locator('#asAffordable').check();
  await page.evaluate(({ gold, tokens }) => { window.setGold(gold); window.setTokens(tokens); },
    { gold: 888, tokens: tokenCost });
  await expect(abyssPotion(page)).toBeVisible();
  await expect(page.locator('#shopWalletGold')).toContainText('888');
  await expect(page.locator('#shopWalletTokens')).toContainText(String(tokenCost));
  await expect(abyssBuy(page)).toBeEnabled();
  await expect(abyssPotion(page).locator('.sw-offer-status')).toContainText('Wallet after: 0 tokens');
  expect(purchases).toEqual([]);
});

test('Abyss gifts normalize recipient and code whitespace and clear only a redeemed code', async ({ page }) => {
  await openAbyssShop(page);
  await page.locator('.ab-shop-program-details > summary').click();
  const requests = [];
  await page.route('**/api/abyss/shop/gift_create', route => {
    requests.push(route.request().postDataJSON());
    return route.fulfill({ json: { ok: true, code: 'GIFT-123', msg: 'Bound gift created.' } });
  });
  await page.locator('#abyssGiftRecipient').fill('  Fixture Friend  ');
  await page.locator('.ab-shop-gift [type="submit"]').click();
  await expect(page.locator('#abyssGiftRecipient')).toHaveValue('Fixture Friend');
  await expect(page.locator('#abyssGiftOutput')).toContainText('Gift code: GIFT-123');
  await expect(page.locator('#abyssGiftOutput').getByRole('button', { name: 'Copy gift code' })).toBeVisible();
  expect(requests).toEqual([{ recipient: 'Fixture Friend', item: 'great_potions', quoted_cost: 7 }]);
  const codes = [];
  await page.route('**/api/abyss/shop/gift_redeem', route => {
    codes.push(route.request().postDataJSON());
    return route.fulfill({ json: codes.length === 1 ?
      { ok: false, error: 'Gift is temporarily unavailable.' } :
      { ok: true, msg: 'Gift redeemed successfully.' } });
  });
  await page.locator('#abyssGiftCode').fill('  GIFT - 123  ');
  const redeem = page.locator('.ab-shop-gift').getByRole('button', { name: 'Redeem', exact: true });
  await redeem.click();
  await expect(page.locator('#abyssGiftOutput')).toContainText('Gift is temporarily unavailable.');
  await expect(page.locator('#abyssGiftCode')).toHaveValue('GIFT-123');
  await redeem.click();
  await expect(page.locator('#abyssGiftOutput')).toContainText('Gift redeemed successfully.');
  await expect(page.locator('#abyssGiftCode')).toHaveValue('');
  expect(codes).toEqual([{ code: 'GIFT-123' }, { code: 'GIFT-123' }]);
});

test('nine loyalty punches preview free token offers while gold prices and gift delivery remain charged', async ({ page }) => {
  await page.route('**/abyss', async route => {
    const response = await route.fetch();
    let html = await response.text();
    html = html.replace('data-loyalty="0"', 'data-loyalty="9"');
    // The fixture has token offers only; add the real catalog gold service before scripts initialize.
    html = html.replace('<div class="ab-shop">', '<div class="ab-shop">' +
      '<div class="ab-shop-item" data-order="99" data-shop-key="emergency_revive" ' +
      'data-shop-name="Emergency Revive Potion" data-shop-description="Revive once" ' +
      'data-token-cost="0" data-gold-cost="100000" data-base-cost="0" data-demand="0" ' +
      'data-deal="false" data-owned="false" data-shop-category="supplies">' +
      '<b>Emergency Revive Potion</b><button data-shop-buy ' +
      'onclick="abyssShopBuy(\'emergency_revive\',0,this)">100,000 gold</button></div>');
    await route.fulfill({ response, body: html });
  });
  await openAbyssShop(page);
  await expect(page.locator('#shopLoyaltyValue')).toHaveText('9 / 10');
  await expect(page.locator('#shopLoyaltyNotice')).toContainText('Gold services still cost gold');
  await page.locator('.ab-shop-program-details > summary').click();
  const fee = Number(await page.locator('.ab-shop-program').getAttribute('data-gift-fee'));
  expect(fee).toBeGreaterThan(0);
  await expect(page.locator('#asGiftPrice')).toContainText('Supply: 0 tokens');
  await expect(page.locator('#asGiftPrice')).toContainText(new RegExp('delivery: ' + exactNumber(fee).source + ' gold'));
  await page.evaluate(() => window.setTokens(0));
  await page.locator('#asAffordable').check();
  await expect(abyssPotion(page)).toBeVisible();
  await expect(abyssBuy(page)).toBeEnabled();
  await expect(abyssBuy(page)).toHaveAccessibleName(/for 0 tokens/);
  await expect(abyssPotion(page).locator('.sw-offer-status')).toContainText('Next loyalty purchase: free');
  const goldOffer = page.locator('#abyssTokenShop [data-shop-key="emergency_revive"]');
  await expect(goldOffer).toBeVisible();
  await expect(goldOffer.locator('[data-shop-buy]')).toHaveAccessibleName(/for 100,000 gold/);
  await page.evaluate(() => window.setGold(99999));
  await expect(goldOffer).toBeHidden();
  await expect(abyssPotion(page)).toBeVisible();
  await page.locator('#asAffordable').uncheck();
  await expect(goldOffer.locator('[data-shop-buy]')).toBeDisabled();
  await expect(goldOffer.locator('.sw-offer-status')).toContainText('Need 1 more gold');
  await expect(page.locator('#shopLoyaltyValue')).toHaveText('9 / 10');
});

test('gift price changes refresh the preview and use the new quoted cost on retry', async ({ page }) => {
  await openAbyssShop(page);
  await page.locator('.ab-shop-program-details > summary').click();
  const requests = [];
  await page.route('**/api/abyss/shop/gift_create', route => {
    requests.push(route.request().postDataJSON());
    return route.fulfill({ json: requests.length === 1 ?
      { ok: false, current_cost: 9, error: 'Gift supply price changed. Review 9 tokens before retrying.' } :
      { ok: true, code: 'GIFT-NEW-PRICE', msg: 'Gift created at the revised price.' } });
  });
  await page.locator('#abyssGiftRecipient').fill('Fixture Friend');
  await expect(page.locator('#asGiftPrice')).toContainText('Supply: 7 tokens');
  const create = page.locator('.ab-shop-gift [type="submit"]');
  await create.click();
  await expect(page.locator('#abyssGiftOutput')).toContainText('Gift supply price changed');
  await expect(page.locator('#asGiftPrice')).toContainText('Supply: 9 tokens');
  await expect(create).toBeEnabled();
  await create.click();
  await expect(page.locator('#abyssGiftOutput')).toContainText('GIFT-NEW-PRICE');
  expect(requests).toEqual([
    { recipient: 'Fixture Friend', item: 'great_potions', quoted_cost: 7 },
    { recipient: 'Fixture Friend', item: 'great_potions', quoted_cost: 9 },
  ]);
});

test('main element choices use rendered offer elements and filter distinct stock', async ({ page }) => {
  const seeded = [];
  await page.route('**/shop', async route => {
    const response = await route.fetch();
    let html = await response.text();
    html = html.replace(/(<article class="shop-card[^>]*>[\s\S]*?<div class="inv-meta">)([^<]*)(<\/div>)/g,
      (match, prefix, metadata, suffix) => {
        if (seeded.length >= 2) return match;
        const element = seeded.length === 0 ? 'Fire' : 'Frost';
        const id = prefix.match(/data-id="([^"]+)"/)[1];
        seeded.push({ id, element });
        prefix = prefix.replace(/ data-element="[^"]*"/, '').replace('<article ', '<article data-element="' + element + '" ');
        return prefix + metadata.split('·').slice(0, 2).join('·').trim() + ' · ' + element + suffix;
      });
    await route.fulfill({ response, body: html });
  });
  await openMainShop(page);
  expect(seeded).toHaveLength(2);
  for (const entry of seeded) {
    await expect(page.locator('#swElement option').filter({ hasText: entry.element })).toHaveCount(1);
    await page.locator('#swElement').selectOption(entry.element);
    await expect(mainCard(page, entry.id)).toBeVisible();
    const other = seeded.find(value => value.id !== entry.id);
    await expect(mainCard(page, other.id)).toBeHidden();
    const visible = await page.locator('.shop-card:visible').evaluateAll(cards => cards.map(card => card.dataset.element));
    expect(visible.length).toBeGreaterThan(0);
    expect(visible.every(element => element === entry.element)).toBe(true);
  }
});

test('saved main views restore advanced stat filters and the matching offers', async ({ page }) => {
  await openMainShop(page);
  await page.locator('#shopAdvancedFilters > summary').click();
  await page.locator('#shopStatCode').selectOption('HP');
  await page.locator('#shopMinStat').fill('80');
  await expect(page.locator('.shop-card:visible')).not.toHaveCount(0);
  const expected = await page.locator('.shop-card:visible').evaluateAll(cards => cards.map(card => card.dataset.id));
  await page.locator('#swSearchName').fill('Health threshold');
  await page.locator('#swSaveSearch').click();
  await page.locator('#shopStatCode').selectOption('STR');
  await page.locator('#shopMinStat').fill('999999999');
  await expect(page.locator('.shop-card:visible')).toHaveCount(0);
  await page.locator('#swSavedSearches').selectOption({ label: 'Health threshold' });
  await page.locator('#swApplySearch').click();
  await expect(page.locator('#shopStatCode')).toHaveValue('HP');
  await expect(page.locator('#shopMinStat')).toHaveValue('80');
  await expect.poll(() => page.locator('.shop-card:visible').evaluateAll(cards => cards.map(card => card.dataset.id))).toEqual(expected);
});

test('locating a watched offer clears conflicting slot and advanced stat filters', async ({ page }) => {
  await openMainShop(page);
  const first = page.locator('.shop-card').first();
  const id = await first.getAttribute('data-id');
  const slot = await first.getAttribute('data-slot');
  await first.locator('.sw-watch').click();
  const otherSlot = await page.locator('#shopSlot option').evaluateAll((options, current) =>
    options.find(option => option.value && option.value !== current).value, slot);
  await page.locator('#shopSlot').selectOption(otherSlot);
  await page.locator('#shopAdvancedFilters > summary').click();
  await page.locator('#shopMinStat').fill('999999999');
  await expect(mainCard(page, id)).toBeHidden();
  await page.locator('#swWatchlist').getByRole('button', { name: 'Locate', exact: true }).click();
  await expect(page.locator('#shopSlot')).toHaveValue('');
  await expect(page.locator('#shopMinStat')).toHaveValue('');
  await expect(mainCard(page, id)).toBeVisible();
  await expect(mainCard(page, id).locator('.shop-inspect-action')).toBeFocused();
});

test('main reserved-budget filtering recalculates when the displayed gold balance changes', async ({ page }) => {
  await openMainShop(page);
  const first = page.locator('.shop-card').first();
  const id = await first.getAttribute('data-id');
  const price = Number(await first.getAttribute('data-price'));
  await page.locator('#swGoldReserve').fill('100');
  await page.locator('#swWithinBudget').check();
  await page.evaluate(gold => { shopCurrentGold = gold; renderShopBalances(); }, price + 100);
  await expect(mainCard(page, id)).toBeVisible();
  await page.evaluate(gold => { shopCurrentGold = gold; renderShopBalances(); }, price + 99);
  await expect(mainCard(page, id)).toBeHidden();
  await page.evaluate(gold => { shopCurrentGold = gold; renderShopBalances(); }, price + 100);
  await expect(mainCard(page, id)).toBeVisible();
  await expect(page.locator('#swGoldReserve')).toHaveValue('100');
});

test('open Abyss workbench keeps usable width and unclipped sections across viewport sizes', async ({ page }) => {
  await page.setViewportSize({ width: 320, height: 844 });
  await openAbyssShop(page);
  const workbench = page.locator('#abyssShopWorkbench');
  for (const width of [320, 390, 768, 1440]) {
    await page.setViewportSize({ width, height: 844 });
    await expect(workbench).toBeVisible();
    const layout = await workbench.evaluate(node => {
      const rect = node.getBoundingClientRect();
      return {
        width: rect.width, left: rect.left, right: rect.right,
        sections: [...node.querySelectorAll('.sw-section-body')].map(section => {
          const bounds = section.getBoundingClientRect();
          return { left: bounds.left, right: bounds.right, width: bounds.width,
            clientWidth: section.clientWidth, scrollWidth: section.scrollWidth };
        }),
      };
    });
    expect(layout.width, 'workbench width at viewport ' + width).toBeGreaterThanOrEqual(240);
    expect(layout.left).toBeGreaterThanOrEqual(0);
    expect(layout.right).toBeLessThanOrEqual(width + 1);
    expect(layout.sections.length).toBeGreaterThan(0);
    for (const section of layout.sections) {
      expect(section.width, 'section width at viewport ' + width).toBeGreaterThan(0);
      expect(section.left).toBeGreaterThanOrEqual(layout.left);
      expect(section.right).toBeLessThanOrEqual(layout.right + 1);
      expect(section.scrollWidth, 'section overflow at viewport ' + width).toBeLessThanOrEqual(section.clientWidth + 1);
    }
    await page.locator('#asSearch').fill('Great Health Potions');
    await expect(abyssPotion(page)).toBeVisible();
    await page.locator('#asSearch').fill('');
  }
});
