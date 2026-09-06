const fs = require('fs');
const path = require('path');
const { test, expect } = require('@playwright/test');

const shopTemplate = fs.readFileSync(path.join(__dirname, '../../internal/bot/webassets/shop.html'), 'utf8');
const artworkScript = shopTemplate.slice(shopTemplate.indexOf('var shopPageSize=12;'), shopTemplate.indexOf('function filterShopStock(){'));

async function mountArtworkFixture(page) {
  await page.setContent(`
    <style>
      body { margin: 0; }
      .shop-card { height: 800px; }
      .item-art { display: block; width: 64px; height: 64px; }
    </style>
    <main id="shopGrid">
      <article class="shop-card"><span class="item-art" data-shop-art-pending></span><button>Inspect first item</button></article>
      <article class="shop-card"><span class="item-art" data-shop-art-pending></span><button>Inspect second item</button></article>
    </main>
  `);
  await page.addScriptTag({ content: artworkScript });
  await page.evaluate(() => document.querySelectorAll('.shop-card').forEach(window.observeShopArt));
}

test('keyboard focus reveals its mounted shop card while intersection delivery is delayed', async ({ page }) => {
  await page.evaluate(() => {
    window.IntersectionObserver = class {
      observe() {}
      unobserve() {}
      disconnect() {}
    };
  });
  await mountArtworkFixture(page);
  const first = page.locator('.item-art').first();
  const second = page.locator('.item-art').nth(1);
  await expect(first).toHaveAttribute('data-shop-art-pending', '');
  await page.getByRole('button', { name: 'Inspect first item', exact: true }).focus();
  await expect(first).not.toHaveAttribute('data-shop-art-pending');
  await expect(second).toHaveAttribute('data-shop-art-pending', '');
});

test('partially visible shop cards reveal their artwork before the icon intersects', async ({ page }) => {
  await mountArtworkFixture(page);
  await page.evaluate(() => {
    window.shopArtObserver.disconnect();
    document.querySelectorAll('.item-art').forEach(art => art.setAttribute('data-shop-art-pending', ''));
    window.scrollTo(0, 300);
  });
  await expect(page.locator('.shop-card').first()).toBeInViewport();
  await expect(page.locator('.item-art').first()).not.toBeInViewport();
  await page.evaluate(() => document.querySelectorAll('.shop-card').forEach(window.observeShopArt));
  await expect(page.locator('.item-art').first()).not.toHaveAttribute('data-shop-art-pending');
});
