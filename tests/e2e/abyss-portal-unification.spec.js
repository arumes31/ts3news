const { test, expect } = require('@playwright/test');

const portalRoutes = [
  { path: '/armory-fixture', heading: 'Armoury' },
  { path: '/inventory', heading: 'Inventory' },
  { path: '/shop', heading: 'Shop' },
  { path: '/ah', heading: 'Auction House' },
  { path: '/leaderboards', heading: 'Leaderboards' },
];

async function expectCatalogArtwork(art) {
  await expect.poll(() => art.evaluate(node => {
    if (node.hasAttribute('data-shop-art-pending')) return false;
    const background = getComputedStyle(node, '::before').backgroundImage;
    if (!background.includes('abyss_catalog_')) return false;
    const url = background.match(/^url\(["']?(.*?)["']?\)$/)?.[1];
    if (!url) return false;
    let state = node.__e2eCatalogArtwork;
    if (!state || state.url !== url) {
      state = node.__e2eCatalogArtwork = { url, decoded: false };
      const image = new Image();
      image.src = url;
      // Poll the outcome without letting a stalled download extend the assertion.
      image.decode().then(() => { state.decoded = image.naturalWidth > 0; }, () => {});
    }
    return state.decoded;
  }), { message: 'Visible catalog artwork has loaded and decoded' }).toBe(true);
}

test('portal surfaces share the Abyss console theme without layout or script failures', async ({ page }) => {
  const failures = [];
  page.on('pageerror', error => failures.push(error.message));
  page.on('console', message => {
    if (message.type() === 'error') failures.push(message.text());
  });

  for (const route of portalRoutes) {
    await page.goto(route.path);
    await expect(page.locator('body')).toHaveClass(/delver-shell/);
    await expect(page.getByRole('heading', { name: new RegExp(route.heading, 'i') }).first()).toBeVisible();
    await expect(page.locator('link[href*="abyss_portal.css"]')).toHaveCount(1);
    if (route.path === '/armory-fixture') {
      await expect(page.locator('body')).toHaveCSS('background-color', 'rgb(12, 9, 8)');
      const scene = page.locator('.armory-scene');
      await expect(scene).toBeVisible();
      expect(await scene.evaluate(image => image.complete && image.naturalWidth > 1000)).toBe(true);
    } else {
      const background = await page.locator('body').evaluate(node => getComputedStyle(node).backgroundImage);
      expect(background).toContain('linear-gradient');
    }
    expect(await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth))
      .toBeLessThanOrEqual(1);
  }

  expect(failures).toEqual([]);
});

test('armoury, inventory, shop, and auction use atlas art and expose every special', async ({ page }) => {
  for (const path of ['/armory-fixture', '/inventory', '/shop', '/ah']) {
    await page.goto(path);
    const trigger = path === '/inventory' ? page.locator('.inv-inspect').first() : page.locator('.item-inspect-trigger[data-item-inspect]').first();
    await expect(trigger).toBeVisible();
    const art = path === '/inventory' ? page.locator('.inv-card .item-art').first() : trigger.locator('.item-art').first();
    await expect(art).toBeVisible();
    await expectCatalogArtwork(art);
    await expect(art).toHaveCSS('background-color', 'rgba(0, 0, 0, 0)');
    await expect(art).toHaveCSS('border-top-width', '0px');

    await trigger.click();
    const inspector = page.locator('.item-inspector');
    await expect(inspector).toBeVisible();
    await expect(inspector.getByRole('heading', { name: 'Global stats' })).toBeVisible();
    await expect(inspector.getByRole('heading', { name: 'All specials' })).toBeVisible();
    await page.keyboard.press('Escape');
    await expect(inspector).toBeHidden();
  }

  await page.goto('/armory-fixture');
  await page.locator('.gear-cell.item-inspect-trigger').first().click();
  await expect(page.locator('.item-inspector-special')).toHaveCount(2);

  await page.goto('/shop');
  await expect(page.locator('.shop-card')).toHaveCount(12);
  // Each rendered offer resolves to catalog artwork when it enters view.
  for (const art of await page.locator('.shop-card .item-art').all()) {
    await art.scrollIntoViewIfNeeded();
    await expectCatalogArtwork(art);
  }
  expect(new Set(await page.locator('.shop-card .item-art').evaluateAll(nodes =>
    nodes.map(node => node.dataset.artFamily)
  )).size).toBeGreaterThan(1);
});
