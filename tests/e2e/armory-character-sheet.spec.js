const { test, expect } = require('@playwright/test');

test('equipment hover previews every effect and dismisses without opening a dialog', async ({ page }) => {
  await page.goto('/armory-fixture');
  const blade = page.getByRole('button', { name: 'Inspect Measured Test Blade', exact: true });
  await blade.hover();
  const tooltip = page.getByRole('tooltip');
  await expect(tooltip).toBeVisible();
  await expect(tooltip).toContainText('Measured Test Blade');
  await expect(tooltip).toContainText('Vampiric');
  await expect(tooltip).toContainText('Berserk');
  await expect(tooltip).toContainText('37 / 100 durability');
  await expect(page.getByRole('dialog')).not.toBeVisible();
  await tooltip.hover();
  await expect(tooltip).toBeVisible();
  await page.keyboard.press('Escape');
  await expect(tooltip).toBeHidden();
});

test('keyboard preview stays in the viewport and click retains the full item record', async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto('/armory-fixture');
  const blade = page.getByRole('button', { name: 'Inspect Measured Test Blade', exact: true });
  await blade.focus();
  const tooltip = page.getByRole('tooltip');
  await expect(tooltip).toBeVisible();
  const box = await tooltip.boundingBox();
  expect(box.x).toBeGreaterThanOrEqual(8);
  expect(box.x + box.width).toBeLessThanOrEqual(382);
  expect(box.y).toBeGreaterThanOrEqual(8);
  expect(box.y + box.height).toBeLessThanOrEqual(836);
  await page.keyboard.press('Enter');
  await expect(tooltip).toBeHidden();
  await expect(page.locator('.item-inspector')).toContainText('Measured Test Blade');
  await page.keyboard.press('Escape');
  await expect(blade).toBeFocused();
});

test('the character sheet keeps all thirty slots accessible across desktop and mobile', async ({ page }) => {
  await page.goto('/armory-fixture');
  await expect(page.locator('.armory-scene')).toHaveJSProperty('complete', true);
  expect(await page.locator('.armory-scene').evaluate(image => image.naturalWidth)).toBeGreaterThan(1000);
  await page.evaluate(() => document.fonts.ready);
  expect(await page.evaluate(() => document.fonts.check('600 24px Cinzel'))).toBe(true);
  for (const width of [320, 390, 768, 1440]) {
    await page.setViewportSize({ width, height: 1000 });
    await expect(page.locator('.armory-equipment .gear-cell')).toHaveCount(30);
    await expect(page.locator('.armory-portrait')).toBeVisible();
    expect(await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth)).toBeLessThanOrEqual(1);
    if (width >= 700) {
      const left = await page.locator('.armory-gear-left').boundingBox();
      const portrait = await page.locator('.armory-portrait').boundingBox();
      const right = await page.locator('.armory-gear-right').boundingBox();
      expect(left.x + left.width).toBeLessThanOrEqual(portrait.x);
      expect(right.x).toBeGreaterThanOrEqual(portrait.x + portrait.width);
    }
  }
});

test('tooltip respects hidden item stats and renders supplied text safely', async ({ page }) => {
  await page.goto('/armory-fixture');
  const mystery = page.locator('.gear-cell[data-slot="Head"]');
  await mystery.hover();
  await expect(page.getByRole('tooltip')).toContainText('Hidden combat stats are inactive');
  await expect(page.getByRole('tooltip')).not.toContainText('987,654');
  const blade = page.getByRole('button', { name: 'Inspect Measured Test Blade', exact: true });
  await blade.evaluate(node => {
    const item = JSON.parse(node.dataset.itemInspect);
    item.name = '<img src=x onerror=alert(1)>';
    node.dataset.itemInspect = JSON.stringify(item);
  });
  await blade.hover();
  await expect(page.getByRole('tooltip')).toContainText('<img src=x onerror=alert(1)>');
  await expect(page.getByRole('tooltip').locator('img')).toHaveCount(0);
});
