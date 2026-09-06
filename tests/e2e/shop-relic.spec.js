const { test, expect } = require('@playwright/test');

test('shop relic showcase gives the featured item a full row without duplicating stock', async ({ page }) => {
  await page.setViewportSize({ width: 1440, height: 1000 });
  await page.goto('/shop');
  const featured = page.locator('.shop-card.featured-item');
  await expect(featured.locator('.shop-relic-stage')).toBeVisible();
  await expect(page.locator('.shop-card')).toHaveCount(49);
  const grid = await page.locator('#shopGrid').boundingBox();
  const card = await featured.boundingBox();
  expect(Math.abs(card.width - grid.width)).toBeLessThan(2);
  expect((await featured.locator('.item-art').boundingBox()).width).toBeGreaterThanOrEqual(128);
  await expect(featured.locator('.shop-item-stats').first()).toBeVisible();
  await expect(featured.locator('.shop-item-specials')).toBeVisible();
  await expect(featured.locator('.shop-buy-action')).toHaveCount(1);
  await page.locator('#shopSearch').fill('no such relic');
  await expect(featured).toBeHidden();
  await page.locator('#shopReset').click();
  await expect(featured).toBeVisible();
});

test('shop relic lighting follows the pointer and resets when hidden or left', async ({ page }) => {
  await page.goto('/shop');
  const stage = page.locator('.shop-relic-stage');
  await stage.hover({ position: { x: 40, y: 40 } });
  await expect(stage).toHaveAttribute('data-lit', 'true');
  const tilt = await stage.locator('.shop-relic-mount').evaluate(element => getComputedStyle(element).transform);
  expect(tilt).not.toBe('none');
  await page.locator('#shopSearch').hover();
  await expect(stage).not.toHaveAttribute('data-lit', 'true');
  await expect.poll(() => stage.evaluate(element => element.style.getPropertyValue('--relic-rx'))).toBe('');
  await stage.hover();
  await page.locator('#shopSearch').fill('no such relic');
  await expect(stage).not.toHaveAttribute('data-lit', 'true');
});

test('shop relic inspector reveal keeps focus, immediate dismissal and other dialogs intact', async ({ page }) => {
  await page.goto('/shop');
  const featured = page.locator('.featured-item');
  const inspect = featured.locator('.shop-inspect-action');
  const dialog = page.getByRole('dialog');
  await inspect.focus();
  await page.keyboard.press('Enter');
  expect(await dialog.locator('.shop-relic-inspector').evaluate(element => element.getAnimations({ subtree: true }).some(animation => animation.playState === 'running'))).toBe(true);
  await expect(dialog.locator('.shop-relic-inspector')).toBeVisible();
  await expect(dialog).toBeFocused();
  await page.keyboard.press('Escape');
  await expect(dialog).toBeHidden();
  await expect(inspect).toBeFocused();
  expect(await page.locator('.shop-relic-inspector').evaluate(element => element.getAnimations({ subtree: true }).length)).toBe(0);
  await featured.locator('.shop-relic-stage').click();
  await expect(dialog.locator('.shop-relic-inspector')).toBeVisible();
  await dialog.getByRole('button', { name: 'Close', exact: true }).click();
  await expect(inspect).toBeFocused();
  await featured.locator('.shop-buy-action').click();
  await expect(dialog.locator('.shop-relic-inspector')).toHaveCount(0);
  await expect(dialog).toContainText('Balance: 25,000,000');
  await page.keyboard.press('Escape');
  await page.locator('.shop-card:not(.featured-item) .shop-inspect-action').first().click();
  await expect(dialog.locator('.item-inspector')).toBeVisible();
  await expect(dialog.locator('.shop-relic-inspector')).toHaveCount(0);
});

test('shop relic falls back to a static display when animation APIs are unavailable', async ({ page }) => {
  await page.addInitScript(() => { Element.prototype.animate = undefined; });
  await page.goto('/shop');
  const stage = page.locator('.shop-relic-stage');
  await stage.hover();
  await expect(stage).not.toHaveAttribute('data-lit', 'true');
  await page.locator('.featured-item .shop-inspect-action').click();
  await expect(page.locator('.shop-relic-inspector')).toBeVisible();
  expect(await page.locator('.shop-relic-inspector').evaluate(element => element.getAnimations({ subtree: true }).length)).toBe(0);
  await page.keyboard.press('Escape');
  await expect(page.locator('.featured-item .shop-inspect-action')).toBeFocused();
});

test('shop relic is static with reduced motion and reflows on narrow screens', async ({ page }) => {
  await page.emulateMedia({ reducedMotion: 'reduce' });
  await page.goto('/shop');
  const stage = page.locator('.shop-relic-stage');
  await stage.hover();
  await expect(stage).not.toHaveAttribute('data-lit', 'true');
  await page.locator('.featured-item .shop-inspect-action').click();
  await expect(page.locator('.shop-relic-inspector')).toBeVisible();
  expect(await page.locator('.shop-relic-inspector').evaluate(element => element.getAnimations({ subtree: true }).length)).toBe(0);
  await page.keyboard.press('Escape');
  for (const width of [320, 390, 768, 1024, 1920]) {
    await page.setViewportSize({ width, height: 1000 });
    expect(await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth)).toBeLessThanOrEqual(1);
    const clipped = await page.locator('.featured-item,.shop-relic-stage,.featured-item .shop-card-copy').evaluateAll(elements => elements.filter(element => element.scrollWidth > element.clientWidth + 1).map(element => element.className));
    expect(clipped, `relic clipping at ${width}px`).toEqual([]);
  }
});

test('shop relic uses a touch fallback and does not appear on later stock pages', async ({ browser }) => {
  const context = await browser.newContext({ viewport: { width: 390, height: 844 }, isMobile: true, hasTouch: true, baseURL: `http://127.0.0.1:${process.env.ABYSS_E2E_PORT || '18082'}` });
  const page = await context.newPage();
  const errors = [];
  page.on('pageerror', error => errors.push(error.message));
  await page.goto('/shop?buff_fixture=endless');
  const stage = page.locator('.shop-relic-stage');
  await expect(stage).toBeVisible();
  await stage.tap();
  await expect(page.locator('.shop-relic-inspector')).toBeVisible();
  expect(await page.locator('.shop-relic-inspector').evaluate(element => element.getAnimations({ subtree: true }).length)).toBe(0);
  await page.getByRole('button', { name: 'Close', exact: true }).tap();
  await expect(stage).not.toHaveAttribute('data-lit', 'true');
  await page.getByRole('link', { name: 'Next stock page' }).click();
  await expect(page.locator('.shop-card')).toHaveCount(240);
  await expect(stage).toHaveCount(0);
  expect(errors).toEqual([]);
  await context.close();
});
