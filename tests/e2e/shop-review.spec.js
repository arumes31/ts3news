const { test, expect } = require('@playwright/test');

test('shop keeps stats, specials and equipped tradeoffs inline and filters persist', async ({ page }) => {
  await page.goto('/shop');
  const mismatches = await page.evaluate(() => shopCards.flatMap(card => {
    const item = JSON.parse(card.querySelector('[data-item-inspect]').dataset.itemInspect);
    const details = card.querySelector('.shop-item-details');
    const values = [...details.querySelectorAll(':scope > .shop-item-stats > div')].map(row => [row.querySelector('dt').textContent, Number(row.querySelector('dd').textContent.replaceAll(',', ''))]);
    return item.stats.filter(stat => !values.some(([label, value]) => label === stat.label && value === stat.value)).map(stat => stat.label)
      .concat(item.specials.filter(special => !details.textContent.includes(special.name) || !details.textContent.includes(special.description)).map(special => special.name));
  }));
  expect(mismatches).toEqual([]);
  await expect(page.locator('.shop-comparison').first()).toContainText('Compared with Equipped');
  await expect(page.locator('.shop-stat-loss').first()).toBeVisible();
  await page.locator('#shopSlot').selectOption('Head');
  await page.locator('#shopSort').selectOption('price');
  const prices = await page.locator('.shop-card:visible').evaluateAll(cards => cards.map(card => Number(card.dataset.price)));
  expect(prices.length).toBeGreaterThan(0);
  expect(prices).toEqual([...prices].sort((a,b) => a-b));
  await page.reload();
  await expect(page.locator('#shopSlot')).toHaveValue('Head');
  await page.locator('#shopReset').click();
  await expect(page.locator('.shop-card')).toHaveCount(12);
  await expect(page.locator('#shopResultCount')).toHaveText('Showing 12 of 49 items');
  await page.locator('#shopUpgrades').check();
  expect(await page.locator('.shop-card:visible').evaluateAll(cards => cards.every(card => card.dataset.upgrade === 'true'))).toBe(true);
  await page.locator('#shopReset').click();
  await page.locator('#shopSearch').fill('no such equipment');
  await expect(page.locator('#shopEmpty')).toContainText('reset the filters');
});

test('XP conversion reviews exact level loss and cancellation spends nothing', async ({ page }) => {
  const commits = [];
  page.on('request', request => {
    if (request.url().endsWith('/api/shop/exchange') && !request.postDataJSON().preview) commits.push(request.postDataJSON());
  });
  await page.goto('/shop');
  await page.locator('#x2gAmount').fill('10000');
  await expect(page.locator('#x2gPreview')).toContainText('Your level will drop');
  await page.getByRole('button', { name: 'Review gold exchange' }).click();
  const dialog = page.getByRole('dialog');
  await expect(dialog).toHaveAccessibleName('Confirm level loss');
  await expect(dialog).toContainText('10,000 XP → 5,000 gold');
  await expect(dialog).toContainText('25,000,000 → 25,005,000');
  await expect(dialog.getByRole('button', { name: 'Keep my resources' })).toBeFocused();
  await page.keyboard.press('Escape');
  await expect(dialog).toBeHidden();
  await expect(page.getByRole('button', { name: 'Review gold exchange' })).toBeFocused();
  expect(commits).toHaveLength(0);
  await page.getByRole('button', { name: 'Review gold exchange' }).click();
  await dialog.getByRole('button', { name: 'Spend XP and lower level' }).click();
  await expect(page.locator('#exState')).toContainText('0 XP');
  await expect(page.locator('#exState')).toContainText('Lvl 1');
  expect(commits).toHaveLength(1);
  expect(commits[0].confirm_level_loss).toBe(true);
  expect(commits[0].expected.xp).toBe(10000);
});

test('gold conversion exposes the next rate and refreshes the next preview', async ({ page }) => {
  await page.goto('/shop');
  await expect(page.locator('#g2xPreview')).toContainText('Next purchase: 11,000 gold per XP.');
  await page.getByRole('button', { name: 'Review XP exchange' }).click();
  await page.getByRole('dialog').getByRole('button', { name: 'Confirm exchange' }).click();
  await expect(page.locator('#g2xRate')).toHaveText('11,000 gold = 1 XP');
  await expect(page.locator('#g2xPreview')).toContainText('Next purchase: 12,000 gold per XP.');
  await expect(page.locator('#g2xAmount')).toHaveValue('11000');
});

test('a changed wallet requires another review and leaves a persistent contextual error', async ({ page }) => {
  await page.route('**/api/shop/exchange', route => route.request().postDataJSON().preview ? route.continue() : route.fulfill({ json: { ok:false, review_required:true, error:'Your balance changed. Review a fresh preview before converting.' } }));
  await page.goto('/shop');
  await page.getByRole('button', { name: 'Review XP exchange' }).click();
  await page.getByRole('dialog').getByRole('button', { name: 'Confirm exchange' }).click();
  const error = page.locator('.ex-card').first().locator('.shop-local-status');
  await expect(error).toContainText('Your balance changed');
  await page.waitForTimeout(3400); // Regression: the previous toast vanished after 3.2 seconds.
  await expect(error).toBeVisible();
  await expect(page.getByRole('button', { name: 'Review XP exchange' })).toBeEnabled();
  await expect(page.locator('#exState')).toHaveAttribute('data-gold', '25000000');
});

test('unconfirmed conversion locks all wallet mutations but keeps verification reachable', async ({ page }) => {
  await page.route('**/api/shop/exchange', route => route.request().postDataJSON().preview ? route.continue() : route.abort('connectionreset'));
  await page.goto('/shop');
  await page.getByRole('button', { name: 'Review XP exchange' }).click();
  await page.getByRole('dialog').getByRole('button', { name: 'Confirm exchange' }).click();
  await expect(page.locator('#shopMsg')).toContainText('result is unconfirmed');
  await expect.poll(() => page.locator('.shop-buy-action,.shop-exchange-action').evaluateAll(buttons => buttons.every(button => button.disabled))).toBe(true);
  await page.locator('#shopMore').click();
  await expect(page.locator('.shop-card')).toHaveCount(24);
  await expect(page.locator('.shop-card').nth(12).locator('.shop-buy-action')).toBeDisabled();
  await expect(page.getByRole('button', { name: 'Refresh to verify the unconfirmed shop action' })).toBeEnabled();
});

test('older previews cannot overwrite newer input and decimal XP is not silently truncated', async ({ page }) => {
  let release;
  await page.route('**/api/shop/exchange', async route => {
    const body = route.request().postDataJSON();
    if (body.preview && body.direction === 'xp_to_gold' && body.amount === 100) await new Promise(resolve => { release = resolve; });
    await route.continue();
  });
  await page.goto('/shop');
  await expect.poll(() => Boolean(release)).toBe(true);
  await page.locator('#x2gAmount').fill('200');
  await expect(page.locator('#x2gPreview')).toContainText('200 XP → 100 gold');
  const oldResponse = page.waitForResponse(response => response.url().endsWith('/api/shop/exchange') && response.request().postDataJSON().amount === 100);
  release();
  await oldResponse;
  await expect(page.locator('#x2gPreview')).toContainText('200 XP → 100 gold');
  await page.locator('#x2gAmount').fill('2.5');
  await expect(page.locator('#x2gPreview')).toContainText('positive whole amount');
});

test('shop search and rarity text remain readable on mobile', async ({ page }) => {
  await page.setViewportSize({ width:390, height:844 });
  await page.goto('/shop');
  const input = await page.locator('#shopSearch').boundingBox();
  expect(input.width).toBeGreaterThan(260);
  await expect(page.locator('.shop-card:visible')).toHaveCount(12);
  const contrast = await page.locator('.shop-card .inv-name').evaluateAll(titles => {
    const canvas = document.createElement('canvas'); canvas.width=canvas.height=1;
    const context=canvas.getContext('2d');
    const luminance=rgb => rgb.map(v => v/255).map(v => v<=0.04045?v/12.92:((v+0.055)/1.055)**2.4).reduce((sum,v,i) => sum+v*[0.2126,0.7152,0.0722][i],0);
    return titles.map(title => {
      context.clearRect(0,0,1,1);context.fillStyle=getComputedStyle(title).color;context.fillRect(0,0,1,1);
      return (luminance([...context.getImageData(0,0,1,1).data].slice(0,3))+0.05)/(luminance([17,25,37])+0.05);
    });
  });
  expect(Math.min(...contrast)).toBeGreaterThanOrEqual(4.5);
  expect(await page.evaluate(() => document.documentElement.scrollWidth-document.documentElement.clientWidth)).toBeLessThanOrEqual(1);
});

test('purchase review shows exact remaining gold and returns keyboard focus', async ({ page }) => {
  await page.goto('/shop');
  const button = page.locator('.shop-buy-action').first();
  const price=Number(await button.getAttribute('data-price'));
  await button.click();
  await expect(page.getByRole('dialog')).toContainText(`Balance: 25,000,000 → ${new Intl.NumberFormat('en-US').format(25_000_000-price)} gold.`);
  await page.keyboard.press('Escape');
  await expect(button).toBeFocused();
});

test('a rejected purchase clears pending state on offers detached while waiting', async ({ page }) => {
  let release;
  await page.route('**/api/shop/buy', async route => {
    await new Promise(resolve => { release = resolve; });
    await route.fulfill({ json: { ok: false, error: 'This purchase was rejected. Please try again.' } });
  });
  await page.goto('/shop');
  const originalBuy = page.locator('.shop-buy-action').first();
  const originalItem = await originalBuy.getAttribute('data-item-name');
  await originalBuy.click();
  await page.getByRole('dialog').getByRole('button', { name: /^Buy for/ }).click();
  await expect.poll(() => Boolean(release)).toBe(true);
  await page.locator('#shopSearch').fill('no such equipment');
  await expect(page.locator('.shop-card')).toHaveCount(0);
  release();
  await expect(page.locator('#shopMsg')).toContainText('This purchase was rejected');
  await page.locator('#shopReset').click();
  await expect(originalBuy).toHaveAttribute('data-item-name', originalItem);
  await expect(originalBuy).toBeEnabled();
  await originalBuy.click();
  await expect(page.getByRole('dialog')).toHaveAccessibleName('Confirm purchase');
  await page.keyboard.press('Escape');
  await expect(originalBuy).toBeFocused();
});

test('desktop exchange panel keeps both review actions reachable in a short viewport', async ({ page }) => {
  await page.setViewportSize({ width:1440, height:800 });
  await page.goto('/shop');
  await expect(page.locator('#x2gPreview')).toContainText('Your level will drop');
  const panel = page.locator('.shop-exchange');
  expect(await panel.evaluate(element => element.clientHeight)).toBeLessThanOrEqual(704);
  const action = page.getByRole('button', { name:'Review gold exchange' });
  await action.scrollIntoViewIfNeeded();
  const bounds = await action.boundingBox();
  expect(bounds.y).toBeGreaterThanOrEqual(0);
  expect(bounds.y+bounds.height).toBeLessThanOrEqual(800);
  await action.click();
  await expect(page.getByRole('dialog')).toHaveAccessibleName('Confirm level loss');
});

test('shop polish preserves heading structure and comfortable desktop controls', async ({ page }) => {
  await page.setViewportSize({ width:1440, height:1000 });
  await page.goto('/shop');
  await expect(page.locator('.shop-card .inv-name .upgrade-badge')).toHaveCount(0);
  await expect(page.locator('.shop-item-badges').first()).toContainText('Featured');
  await expect(page.locator('.shop-item-specials h4')).toHaveCount(12);
  expect(await page.evaluate(() => shopCards.filter(card => card.querySelector('.shop-item-specials h4')).length)).toBe(49);
  const compactControls = await page.locator('.shop-console button,.shop-console input:not([type=checkbox]),.shop-console select,.shop-exchange-jump').evaluateAll(elements => elements.filter(element => element.getClientRects().length && element.getBoundingClientRect().height<44).map(element => element.id||element.textContent));
  expect(compactControls).toEqual([]);
  await page.locator('#shopSearch').focus();
  expect(await page.locator('#shopSearch').evaluate(element => getComputedStyle(element).outlineStyle)).toBe('solid');
  await page.keyboard.press('Tab');
  await expect(page.locator('#shopSlot')).toBeFocused();
  const actions = page.locator('.shop-card-actions').first();
  const buttonWidth = await actions.locator('.shop-buy-action').evaluate(element => element.getBoundingClientRect().width);
  expect(buttonWidth).toBeGreaterThan(120);
});

test('shop polish reflows without clipping at narrow, intermediate and wide sizes', async ({ page }) => {
  await page.goto('/shop');
  for (const width of [320,768,1024,1920]) {
    await page.setViewportSize({ width, height:1000 });
    expect(await page.evaluate(() => document.documentElement.scrollWidth-document.documentElement.clientWidth)).toBeLessThanOrEqual(1);
    const clipped = await page.locator('.shop-toolbar,.shop-filters,.shop-card:visible,.shop-item-stats,.shop-card-actions,.ex-card').evaluateAll(elements => elements.filter(element => element.getClientRects().length && element.scrollWidth>element.clientWidth+1).map(element => element.className));
    expect(clipped, `clipped shop elements at ${width}px`).toEqual([]);
  }
  await page.emulateMedia({ reducedMotion:'reduce' });
  const card = page.locator('.shop-card').first();
  await card.hover();
  expect(await card.evaluate(element => getComputedStyle(element).transitionDuration)).toBe('0s');
});
