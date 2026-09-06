const { test, expect } = require('@playwright/test');

async function emptyNotices(page) {
  await page.route('**/api/ah/notices*', route => route.fulfill({
    json: { ok: true, notices: [], unseen_count: 0, next_before: 0, has_more: false },
  }));
}

async function openNotices(page) {
  const button = page.getByRole('button', { name: /Market notices/i });
  await expect(button).toBeVisible();
  await button.click();
  const panel = page.getByRole('region', { name: 'Market notices', exact: true });
  await expect(panel).toBeVisible();
  return panel;
}

async function openOrderForm(page) {
  const orders = page.getByRole('tab', { name: 'Orders', exact: true });
  await expect(orders).toBeVisible();
  await orders.click();
  await page.getByRole('button', { name: 'Post buy order', exact: true }).click();
  const form = page.locator('form').filter({ has: page.getByLabel('Gold per unit', { exact: true }) });
  await expect(form).toHaveCount(1);
  await expect(form).toBeVisible();
  await expect(page.getByRole('dialog')).toBeHidden();
  return form;
}

test('notices stay available after viewing and reload, and only explicit acknowledgement marks the shown IDs', async ({ page }) => {
  const notices = [
    { id: 105, kind: 'outbid', message: 'You were outbid on Cinder Test Blade.', amount: 900, when: '05 Sep 2026 · 10:00 UTC', when_iso: '2026-09-05T10:00:00Z', seen: false },
    { id: 104, kind: 'sale', message: 'Your Dust order was filled.', amount: 500, when: '05 Sep 2026 · 09:00 UTC', when_iso: '2026-09-05T09:00:00Z', seen: false },
    { id: 99, kind: 'refund', message: 'Older bid refund received.', amount: 400, when: '04 Sep 2026 · 10:00 UTC', when_iso: '2026-09-04T10:00:00Z', seen: false },
  ];
  const acknowledgements = [];
  await page.route('**/api/ah/notices*', async route => {
    const request = route.request();
    if (request.method() === 'POST') {
      const body = request.postDataJSON();
      acknowledgements.push(body);
      notices.forEach(notice => { if (body.ids?.includes(notice.id)) notice.seen = true; });
      await route.fulfill({ json: { ok: true, unseen_count: notices.filter(notice => !notice.seen).length } });
      return;
    }
    const before = Number(new URL(request.url()).searchParams.get('before') || 0);
    const shown = before ? notices.filter(notice => notice.id < before) : notices.slice(0, 2);
    await route.fulfill({ json: { ok: true, notices: shown, unseen_count: notices.filter(notice => !notice.seen).length, next_before: before ? 0 : 104, has_more: !before } });
  });

  await page.goto('/ah');
  let panel = await openNotices(page);
  await expect(panel).toContainText(notices[0].message);
  await expect(panel).toContainText(notices[1].message);
  expect(acknowledgements).toEqual([]);
  await page.reload();
  panel = await openNotices(page);
  await expect(panel).toContainText(notices[0].message);
  expect(acknowledgements).toEqual([]);

  await panel.getByRole('button', { name: 'Mark shown as read', exact: true }).click();
  await expect.poll(() => acknowledgements).toEqual([{ ids: [105, 104] }]);
  await expect(page.locator('#ahNoticeCount')).toHaveText('1');
  await expect(panel).toContainText(notices[0].message);
  await page.reload();
  panel = await openNotices(page);
  await expect(panel).toContainText(notices[0].message);
  await expect(page.locator('#ahNoticeCount')).toHaveText('1');
  expect(acknowledgements).toHaveLength(1);
});

test('notice loading can be retried, older pages remain reachable, and notice content renders as text', async ({ page }) => {
  let failRead = true;
  const requests = [];
  const literalMessage = '<img src=x onerror="alert(1)"> Watch price decreased & gold returned.';
  await page.route('**/api/ah/notices*', async route => {
    const request = route.request();
    requests.push({ method: request.method(), before: new URL(request.url()).searchParams.get('before') });
    if (failRead) {
      await route.fulfill({ status: 503, json: { ok: false, error: 'Market notices are temporarily unavailable.' } });
      return;
    }
    const older = Boolean(new URL(request.url()).searchParams.get('before'));
    await route.fulfill({ json: {
      ok: true,
      notices: [{ id: older ? 9 : 10, kind: 'watch', message: older ? 'Earlier watch alert.' : literalMessage, amount: 250, when: '05 Sep 2026 · 10:00 UTC', when_iso: '2026-09-05T10:00:00Z', seen: false }],
      unseen_count: 2, next_before: older ? 0 : 10, has_more: !older,
    } });
  });
  await page.goto('/ah');
  const panel = await openNotices(page);
  await expect(panel).toContainText('Market notices are temporarily unavailable.');
  failRead = false;
  await panel.getByRole('button', { name: /Retry/i }).click();
  await expect(panel.getByText(literalMessage, { exact: true })).toBeVisible();
  await expect(panel.locator('img')).toHaveCount(0);
  await panel.getByRole('button', { name: 'Older notices', exact: true }).click();
  await expect(panel).toContainText('Earlier watch alert.');
  expect(requests.some(request => request.method === 'GET' && request.before === '10')).toBe(true);
  expect(requests.every(request => request.method === 'GET')).toBe(true);
});

test('My trading groups listing history, transactions and orders into keyboard reachable tabs', async ({ page }) => {
  await emptyNotices(page);
  await page.goto('/ah');
  const tabs = page.getByRole('tablist', { name: 'My trading', exact: true });
  await expect(tabs).toBeVisible();
  const listings = tabs.getByRole('tab', { name: 'Listings', exact: true });
  const transactions = tabs.getByRole('tab', { name: 'Transactions', exact: true });
  const orders = tabs.getByRole('tab', { name: 'Orders', exact: true });
  const listingPanel = page.getByRole('tabpanel', { name: 'Listings', exact: true });
  await expect(listings).toHaveAttribute('aria-selected', 'true');
  await expect(listingPanel.getByRole('heading', { name: 'Sell from Inventory', exact: true })).toBeVisible();
  await expect(listingPanel.getByRole('tab', { name: /^Sold/ })).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Material Buy Orders', exact: true })).toBeHidden();

  await listings.focus();
  await page.keyboard.press('ArrowRight');
  await expect(transactions).toBeFocused();
  await expect(transactions).toHaveAttribute('aria-selected', 'true');
  await expect(page.getByRole('tabpanel', { name: 'Transactions', exact: true })).toContainText('No auction history yet.');
  await expect(listingPanel).toBeHidden();
  await page.keyboard.press('End');
  await expect(orders).toBeFocused();
  await expect(orders).toHaveAttribute('aria-selected', 'true');
  const orderPanel = page.getByRole('tabpanel', { name: 'Orders', exact: true });
  await expect(orderPanel.getByRole('heading', { name: 'Material Buy Orders', exact: true })).toBeVisible();
  await expect(orderPanel.getByRole('row').filter({ hasText: 'Crafter' })).toContainText('dust');
  await page.keyboard.press('Home');
  await expect(listings).toBeFocused();
  await expect(listings).toHaveAttribute('aria-selected', 'true');
});

test('a material order reviews its complete inline draft and submits the exact escrow only after confirmation', async ({ page }) => {
  await emptyNotices(page);
  const posted = [];
  await page.route('**/api/ah/material_order', async route => {
    posted.push(route.request().postDataJSON());
    await route.fulfill({ json: { ok: true, msg: 'Order placed. Reserved 1,000 gold.', gold: 24000 } });
  });
  await page.goto('/ah');
  const form = await openOrderForm(page);
  await form.getByLabel('Material', { exact: true }).selectOption('shard');
  await form.getByLabel('Quantity', { exact: true }).fill('8');
  await form.getByLabel('Gold per unit', { exact: true }).fill('125');
  await expect(form).toContainText(/Escrow[\s\S]*1,000 gold/i);
  await form.getByRole('button', { name: 'Review order', exact: true }).click();
  const review = page.getByRole('dialog', { name: 'Confirm buy order', exact: true });
  await expect(review).toContainText('8 shard');
  await expect(review).toContainText('125 gold');
  await expect(review).toContainText('1,000 gold');
  expect(posted).toEqual([]);
  await review.getByRole('button', { name: 'Edit order', exact: true }).click();
  await expect(review).toBeHidden();
  await expect(form.getByLabel('Material', { exact: true })).toHaveValue('shard');
  await expect(form.getByLabel('Quantity', { exact: true })).toHaveValue('8');
  await expect(form.getByLabel('Gold per unit', { exact: true })).toHaveValue('125');
  expect(posted).toEqual([]);
  await form.getByRole('button', { name: 'Review order', exact: true }).click();
  await review.getByRole('button', { name: 'Reserve 1,000 gold', exact: true }).click();
  await expect(page.locator('#ahMsg')).toContainText('Order placed. Reserved 1,000 gold.');
  expect(posted).toEqual([{ material: 'shard', count: 8, unit_price: 125 }]);
});

for (const invalid of [
  { name: 'fractional quantities', count: '2.5', unit: '125' },
  { name: 'escrow beyond the safe integer range', count: '10000', unit: '9007199254740991' },
]) {
  test(`a material order rejects ${invalid.name} while preserving the draft`, async ({ page }) => {
    await emptyNotices(page);
    const posted = [];
    await page.route('**/api/ah/material_order', async route => {
      posted.push(route.request().postDataJSON());
      await route.fulfill({ json: { ok: false, error: 'Invalid order should not reach the server.' } });
    });
    await page.goto('/ah');
    const form = await openOrderForm(page);
    await form.getByLabel('Material', { exact: true }).selectOption('core');
    await form.getByLabel('Quantity', { exact: true }).fill(invalid.count);
    await form.getByLabel('Gold per unit', { exact: true }).fill(invalid.unit);
    await form.getByRole('button', { name: 'Review order', exact: true }).click();
    await expect(page.getByRole('dialog')).toBeHidden();
    await expect(form.getByLabel('Material', { exact: true })).toHaveValue('core');
    await expect(form.getByLabel('Quantity', { exact: true })).toHaveValue(invalid.count);
    await expect(form.getByLabel('Gold per unit', { exact: true })).toHaveValue(invalid.unit);
    expect(posted).toEqual([]);
    await form.getByLabel('Quantity', { exact: true }).fill('3');
    await form.getByLabel('Gold per unit', { exact: true }).fill('125');
    await form.getByRole('button', { name: 'Review order', exact: true }).click();
    await expect(page.getByRole('dialog', { name: 'Confirm buy order', exact: true })).toContainText('375 gold');
    expect(posted).toEqual([]);
  });
}

test('a rejected material order leaves a persistent error and a complete editable draft', async ({ page }) => {
  await emptyNotices(page);
  const posted = [];
  await page.route('**/api/ah/material_order', async route => {
    posted.push(route.request().postDataJSON());
    await route.fulfill({ json: { ok: false, error: 'Your balance changed. Please review the order again.' } });
  });
  await page.goto('/ah');
  const form = await openOrderForm(page);
  await form.getByLabel('Material', { exact: true }).selectOption('dust');
  await form.getByLabel('Quantity', { exact: true }).fill('8');
  await form.getByLabel('Gold per unit', { exact: true }).fill('125');
  await form.getByRole('button', { name: 'Review order', exact: true }).click();
  await page.getByRole('dialog').getByRole('button', { name: 'Reserve 1,000 gold', exact: true }).click();
  await expect(page.getByText('Your balance changed. Please review the order again.', { exact: true }).first()).toBeVisible();
  await expect(form.getByLabel('Material', { exact: true })).toHaveValue('dust');
  await expect(form.getByLabel('Quantity', { exact: true })).toHaveValue('8');
  await expect(form.getByLabel('Gold per unit', { exact: true })).toHaveValue('125');
  await expect(form.getByRole('button', { name: 'Review order', exact: true })).toBeEnabled();
  await form.getByLabel('Quantity', { exact: true }).fill('4');
  await expect(form).toContainText(/Escrow[\s\S]*500 gold/i);
  expect(posted).toEqual([{ material: 'dust', count: 8, unit_price: 125 }]);
});

for (const status of [200, 503]) {
  test(`an uncertain order response (${status}) locks the draft until verification`, async ({ page }) => {
    await emptyNotices(page);
    const posted = [];
    await page.route('**/api/ah/material_order', async route => {
      posted.push(route.request().postDataJSON());
      await route.fulfill({ status, json: { ok: false, unconfirmed: true, error: 'The order result is unconfirmed. Refresh to verify before ordering again.' } });
    });
    await page.goto('/ah');
    const form = await openOrderForm(page);
    await form.getByLabel('Quantity', { exact: true }).fill('8');
    await form.getByLabel('Gold per unit', { exact: true }).fill('125');
    await form.getByRole('button', { name: 'Review order', exact: true }).click();
    await page.getByRole('dialog').getByRole('button', { name: 'Reserve 1,000 gold', exact: true }).click();
    await expect(form.getByRole('alert')).toContainText('unconfirmed');
    await expect(form.getByLabel('Quantity', { exact: true })).toHaveValue('8');
    await expect(form.getByLabel('Quantity', { exact: true })).toBeDisabled();
    await expect(form.getByRole('button', { name: 'Unconfirmed market action locked until refresh', exact: true })).toBeDisabled();
    await expect(form.getByRole('button', { name: 'Refresh to verify the unconfirmed market action', exact: true })).toBeEnabled();
    await expect(page.getByRole('button', { name: 'Post buy order', exact: true })).toBeDisabled();
    expect(posted).toEqual([{ material: 'dust', count: 8, unit_price: 125 }]);
  });
}

test('the item inspector explains the auction purchase and reaches a cancellable review before spending gold', async ({ page }) => {
  await emptyNotices(page);
  const purchases = [];
  await page.route('**/api/ah/buy', async route => {
    purchases.push(route.request().postDataJSON());
    await route.fulfill({ json: { ok: true, bought: 'Cinder Test Blade', gold: 23400 } });
  });
  await page.goto('/ah');
  const inspect = page.getByRole('button', { name: 'Inspect Cinder Test Blade', exact: true });
  await inspect.click();
  const inspector = page.getByRole('dialog');
  await expect(inspector).toContainText(/1,?600 gold/);
  await expect(inspector).toContainText(/inventory/i);
  await inspector.getByRole('button', { name: 'Review purchase', exact: true }).click();
  const review = page.getByRole('dialog', { name: 'Confirm purchase', exact: true });
  await expect(review).toContainText('Exact total: 1,600 gold');
  await expect(review).toContainText('buyer fee: 0 gold');
  expect(purchases).toEqual([]);
  await review.getByRole('button', { name: 'Cancel', exact: true }).click();
  await expect(review).toBeHidden();
  expect(purchases).toEqual([]);
  await expect(page.getByRole('dialog').getByRole('button', { name: 'Review purchase', exact: true })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Buy Cinder Test Blade for 1600 gold', exact: true })).toBeEnabled();
  await page.getByRole('dialog').getByRole('button', { name: 'Review purchase', exact: true }).click();
  await review.getByRole('button', { name: 'Buy for 1,600 gold', exact: true }).click();
  await expect(page.locator('#ahMsg')).toContainText('Bought Cinder Test Blade');
  expect(purchases).toEqual([{ id: 'listing-history' }]);
});

for (const width of [390, 1024, 1440]) {
  test(`auction trading remains readable without page or local table overflow at ${width}px`, async ({ page }) => {
    await page.setViewportSize({ width, height: 900 });
    await emptyNotices(page);
    await page.goto('/ah');
    await expect(page.getByRole('heading', { name: 'Auction House', exact: true })).toBeVisible();
    const note = page.locator('.auction-action-note').first();
    await expect(note).toBeVisible();
    expect.soft(await note.evaluate(element => parseFloat(getComputedStyle(element).fontSize))).toBeGreaterThanOrEqual(14);
    expect.soft(await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth)).toBeLessThanOrEqual(1);
    const overflow = await page.locator('.auction-table-wrap:visible').evaluateAll(wrappers => wrappers.map(element => element.scrollWidth - element.clientWidth));
    expect.soft(overflow.every(pixels => pixels <= 1), `Visible table overflow: ${overflow.join(', ')}px`).toBe(true);
    if (width === 390) {
      for (const name of ['Buy Cinder Test Blade for 1600 gold', 'Bid on Cinder Test Blade', 'Watch Cinder Test Blade']) {
        const bounds = await page.getByRole('button', { name, exact: true }).boundingBox();
        expect.soft(bounds.height, `${name} target height`).toBeGreaterThanOrEqual(44);
        expect.soft(bounds.width, `${name} target width`).toBeGreaterThanOrEqual(44);
      }
    }
    const orders = page.getByRole('tab', { name: 'Orders', exact: true });
    await expect(orders).toBeVisible();
    await orders.click();
    await expect(page.getByRole('row').filter({ hasText: 'Crafter' })).toBeVisible();
    const orderOverflow = await page.locator('.auction-table-wrap:visible').evaluateAll(wrappers => wrappers.map(element => element.scrollWidth - element.clientWidth));
    expect(orderOverflow.every(pixels => pixels <= 1), `Visible table overflow with Orders open: ${orderOverflow.join(', ')}px`).toBe(true);
    expect(await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth)).toBeLessThanOrEqual(1);
  });
}
