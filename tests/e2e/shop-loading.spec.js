const { test, expect } = require('@playwright/test');

for (const viewport of [{ width: 1440, height: 900 }, { width: 390, height: 844 }]) {
  test(`shop mounts stock in batches and keeps keyboard inspection usable at ${viewport.width}px`, async ({ page }) => {
    await page.setViewportSize(viewport);
    await page.goto('/shop');
    const cards = page.locator('#shopGrid .shop-card');
    await expect(cards).toHaveCount(12);
    await expect(page.locator('#shopResultCount')).toHaveText('Showing 12 of 49 items');
    const initialIds = await cards.evaluateAll(elements => elements.map(element => element.dataset.id));
    const more = page.getByRole('button', { name: 'Show 12 more', exact: true });
    await more.focus();
    await page.keyboard.press('Enter');
    await expect(cards).toHaveCount(24);
    expect((await cards.evaluateAll(elements => elements.map(element => element.dataset.id))).slice(0, 12)).toEqual(initialIds);
    const inspect = cards.nth(12).locator('.shop-inspect-action');
    await expect(inspect).toBeFocused();
    await expect(inspect).toBeInViewport();
    await page.keyboard.press('Enter');
    await expect(page.locator('.item-inspector')).toBeVisible();
    await page.keyboard.press('Escape');
    await expect(inspect).toBeFocused();
    await page.setViewportSize({ width: viewport.width === 390 ? 1440 : 390, height: 900 });
    await expect(cards).toHaveCount(24);
    await page.reload();
    await expect(cards).toHaveCount(24);
    await page.locator('#shopReset').click();
    await expect(cards).toHaveCount(12);
    await expect(page.locator('#shopSearch')).toBeFocused();
    await page.setViewportSize(viewport);
    await expect(cards).toHaveCount(12);
  });

  test(`shop initial artwork stays below 1.5 MB at ${viewport.width}px`, async ({ page }) => {
    await page.setViewportSize(viewport);
    await page.goto('/shop');
    await page.waitForLoadState('networkidle');
    const resources = await page.evaluate(() => performance.getEntriesByType('resource').map(entry => ({ name: entry.name, bytes: entry.encodedBodySize })));
    const pngs = resources.filter(entry => /\.png(?:\?|$)/i.test(entry.name));
    expect(pngs.length).toBeGreaterThan(0);
    expect(pngs.reduce((total, entry) => total + entry.bytes, 0)).toBeLessThanOrEqual(1_500_000);
    expect(resources.some(entry => /\/abyss_catalog_icons\.js(?:\?|$)/.test(entry.name))).toBe(false);
    expect(await page.locator('*').count()).toBeLessThan(1500);
  });
}

for (const [url, total] of [['/shop', 49], ['/shop?buff_fixture=endless', 240]]) {
  test(`shop filters and sorts all ${total} offers while mounting only the current batch`, async ({ page }) => {
    await page.goto(url);
    if (total === 240) {
      // Use the public stock-page link so this test follows the server's query contract.
      await page.getByRole('link', { name: 'Next stock page' }).click();
    }
    const cards = page.locator('#shopGrid .shop-card');
    await expect(cards).toHaveCount(12);
    expect(await page.evaluate(() => shopCards.length)).toBe(total);
    expect(await page.locator('*').count()).toBeLessThan(total === 240 ? 2000 : 1500);
    const offers = await page.evaluate(() => shopCards.map(card => ({
      id: card.dataset.id, price: Number(card.dataset.price), order: Number(card.dataset.order),
      slot: card.dataset.slot, search: card.dataset.search,
      name: JSON.parse(card.querySelector('[data-item-inspect]').dataset.itemInspect).name,
    })));
    const target = offers[offers.length - 1];
    await expect(page.locator(`#shopGrid .shop-card[data-id="${target.id}"]`)).toHaveCount(0);
    await page.locator('#shopSearch').fill(target.name);
    await expect(page.locator(`#shopGrid .shop-card[data-id="${target.id}"]`)).toBeVisible();
    await page.locator('#shopReset').click();
    await page.locator('#shopSort').selectOption('price');
    expect(await cards.evaluateAll(elements => elements.map(element => element.dataset.id))).toEqual(
      [...offers].sort((a, b) => a.price - b.price || a.order - b.order).slice(0, 12).map(offer => offer.id)
    );
    await page.locator('#shopSlot').selectOption(target.slot);
    const matching = offers.filter(offer => offer.slot === target.slot).sort((a, b) => a.price - b.price || a.order - b.order);
    expect(await cards.evaluateAll(elements => elements.map(element => element.dataset.id))).toEqual(matching.slice(0, 12).map(offer => offer.id));
    while (await page.locator('#shopMore').isVisible()) await page.locator('#shopMore').click();
    expect(await cards.evaluateAll(elements => elements.map(element => element.dataset.id))).toEqual(matching.map(offer => offer.id));
    await page.locator('#shopReset').click();
    const commonQuery = await page.evaluate(() => {
      const searches = shopCards.map(card => card.dataset.search.toLocaleLowerCase());
      return ['a', 'e', 'i', 'o'].find(query => searches.filter(search => search.includes(query)).length > 12);
    });
    expect(commonQuery).toBeTruthy();
    await page.locator('#shopSearch').fill(commonQuery);
    await expect(cards).toHaveCount(12);
    await expect(page.locator('#shopMore')).toBeVisible();
    await page.locator('#shopReset').click();
    await expect(cards).toHaveCount(12);
  });
}

test('shop artwork and newly mounted inspection work without IntersectionObserver', async ({ page }) => {
  await page.addInitScript(() => { window.IntersectionObserver = undefined; });
  await page.goto('/shop');
  await page.getByRole('button', { name: 'Show 12 more', exact: true }).click();
  const card = page.locator('#shopGrid .shop-card').nth(12);
  await expect(card.locator('.shop-inspect-action')).toBeFocused();
  const artwork = card.locator('.item-art').first();
  await expect.poll(() => artwork.evaluate(element => getComputedStyle(element, '::before').backgroundImage)).toContain('abyss_catalog_');
  await card.locator('.shop-inspect-action').click();
  await expect(page.locator('.item-inspector')).toBeVisible();
});
