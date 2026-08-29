const { test, expect } = require('@playwright/test');

const portalRoutes = [
  { path: '/armory-fixture', heading: 'Armoury' },
  { path: '/inventory', heading: 'Inventory' },
  { path: '/shop', heading: 'Shop' },
  { path: '/ah', heading: 'Auction House' },
  { path: '/leaderboards', heading: 'Leaderboards' },
];

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
    const background = await page.locator('body').evaluate(node => getComputedStyle(node).backgroundImage);
    expect(background).toContain('linear-gradient');
    expect(await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth))
      .toBeLessThanOrEqual(1);
  }

  expect(failures).toEqual([]);
});

test('armoury, inventory, shop, and auction use atlas art and expose every special', async ({ page }) => {
  for (const path of ['/armory-fixture', '/inventory', '/shop', '/ah']) {
    await page.goto(path);
    const trigger = page.locator('.item-inspect-trigger[data-item-inspect]').first();
    await expect(trigger).toBeVisible();
    const art = trigger.locator('.item-art').first();
    await expect(art).toBeVisible();
    const atlas = await art.evaluate(node => getComputedStyle(node, '::before').backgroundImage);
    expect(atlas).toContain('abyss_catalog_');
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
  await expect(page.locator('.shop-card')).toHaveCount(49);
  const shopAtlases = await page.locator('.shop-card .item-art').evaluateAll(nodes =>
    nodes.map(node => getComputedStyle(node, '::before').backgroundImage)
  );
  expect(shopAtlases.every(asset => asset.includes('abyss_catalog_'))).toBe(true);
  expect(new Set(await page.locator('.shop-card .item-art').evaluateAll(nodes =>
    nodes.map(node => node.dataset.artFamily)
  )).size).toBeGreaterThan(1);
});
