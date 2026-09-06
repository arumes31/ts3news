const { test, expect } = require('@playwright/test');

test('shop prices apply premium rarity floors to featured and buff-promoted gear', async ({ page }) => {
  await page.goto('/shop?buff_fixture=endless');
  const floors = { Mythic: 3_000_000, Divine: 6_000_000, Celestial: 8_000_000, Eternal: 10_000_000 };
  const offers = await page.evaluate(() => shopCards.map(card => ({
    rarity: JSON.parse(card.querySelector('[data-item-inspect]').dataset.itemInspect).rarity,
    price: Number(card.dataset.price),
    shown: Number(card.querySelector('.price').textContent.replace(/[^\d]/g, '')),
    reviewed: Number(card.querySelector('.shop-buy-action').dataset.price),
    featured: card.classList.contains('featured-item'),
  })));
  expect(offers.filter(offer => offer.rarity === 'Eternal')).toHaveLength(20);
  for (const offer of offers) {
    if (floors[offer.rarity]) expect(offer.price).toBeGreaterThanOrEqual(floors[offer.rarity]);
    expect(offer.shown).toBe(offer.price);
    expect(offer.reviewed).toBe(offer.price);
  }
  expect(offers.find(offer => offer.featured).price).toBeGreaterThan(3_000_000);
  await page.locator('#shopSearch').fill('Eternal');
  const eternal = page.locator('.shop-card').filter({ has: page.locator('.shop-eternal-bonus') }).first();
  const price = Number(await eternal.getAttribute('data-price'));
  await eternal.locator('.shop-buy-action').click();
  await expect(page.getByRole('dialog')).toContainText(`${new Intl.NumberFormat('en-US').format(price)} gold`);
  await page.keyboard.press('Escape');
  await expect(page.locator('#shopAvailableGold')).toHaveAttribute('data-value', '3000000000');
});
