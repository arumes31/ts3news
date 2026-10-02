const { test, expect } = require('@playwright/test');

test.use({ hasTouch: true, isMobile: true });

test('keeps touch controls usable in landscape orientation with side-by-side ergonomics', async ({ page }) => {
  await page.setViewportSize({ width: 844, height: 390 });
  await page.goto('/abyss/rift?practice=skills&subclass=vanguard');
  await expect(page.locator('#rift-start')).toBeEnabled();

  // In landscape, orientation hint is hidden
  await expect(page.locator('#rift-orientation-hint')).toBeHidden();

  await page.locator('#rift-start').tap();
  await expect(page.locator('#rift-overlay')).toBeHidden();

  // Controls band uses 2-column grid in landscape
  const controlsBand = page.locator('.rift-controls-band');
  await expect(controlsBand).toHaveCSS('display', 'grid');

  // Movement pad on left thumb side by default
  const touchPad = page.locator('.rift-touch');
  await expect(touchPad).toBeVisible();
  const touchBox = await touchPad.boundingBox();
  expect(touchBox).not.toBeNull();

  const actionbar = page.locator('#rift-actionbar');
  await expect(actionbar).toBeVisible();
  const actionBox = await actionbar.boundingBox();
  expect(actionBox).not.toBeNull();

  // Touch pad is to the left of the actionbar
  expect(touchBox.x).toBeLessThan(actionBox.x);

  // Test movement interaction via touch tap
  const rightBtn = touchPad.locator('button[data-move="right"]');
  await expect(rightBtn).toBeVisible();
  await rightBtn.tap();

  // Test combat interaction via touch tap
  const attackBtn = actionbar.locator('button[data-hold="attack"]');
  await expect(attackBtn).toBeVisible();
  await attackBtn.tap();

  // Test layout switching to right-handed touch placement
  await page.locator('#rift-pause').tap();
  const settings = page.locator('.rift-settings');
  if (!(await settings.evaluate(el => el.open))) {
    await page.locator('.rift-settings > summary').tap();
  }
  const layoutSelect = page.locator('#rift-touch-layout');
  await layoutSelect.selectOption('right');

  // Resume and verify touch pad is now to the right of the actionbar
  await page.locator('#rift-start').tap();
  await expect(page.locator('#rift-overlay')).toBeHidden();

  const touchBoxRight = await touchPad.boundingBox();
  const actionBoxLeft = await actionbar.boundingBox();
  expect(touchBoxRight.x).toBeGreaterThan(actionBoxLeft.x);

  // No horizontal scroll overflow
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true);
});

test('keeps touch controls usable in portrait orientation with stacked layout and accessible touch targets', async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto('/abyss/rift?practice=skills&subclass=elementalist');
  await expect(page.locator('#rift-start')).toBeEnabled();

  // In portrait, orientation hint is visible
  await expect(page.locator('#rift-orientation-hint')).toBeVisible();

  await page.locator('#rift-start').tap();
  await expect(page.locator('#rift-overlay')).toBeHidden();

  // Controls are vertically stacked
  const viewport = page.locator('#rift-viewport');
  const actionbar = page.locator('#rift-actionbar');
  const classbar = page.locator('.rift-classbar');
  const touchPad = page.locator('.rift-touch');

  await expect(viewport).toBeVisible();
  await expect(actionbar).toBeVisible();
  await expect(classbar).toBeVisible();
  await expect(touchPad).toBeVisible();

  const vBox = await viewport.boundingBox();
  const aBox = await actionbar.boundingBox();
  const cBox = await classbar.boundingBox();
  const tBox = await touchPad.boundingBox();

  expect(vBox.y).toBeLessThan(aBox.y);
  expect(aBox.y).toBeLessThan(cBox.y);
  expect(cBox.y).toBeLessThan(tBox.y);

  // Touch targets >= 48px
  const sizes = await page.locator('.rift-actionbar button, .rift-signatures button, .rift-touch button').evaluateAll(nodes =>
    nodes.filter(node => node.getClientRects().length).map(node => {
      const r = node.getBoundingClientRect();
      return { id: node.id || node.dataset.bind || node.dataset.move || node.textContent, width: r.width, height: r.height };
    })
  );
  expect(sizes.length).toBeGreaterThan(8);
  expect(sizes.filter(r => r.width < 44 || r.height < 44)).toEqual([]);

  // Touch interaction functions cleanly
  const leftBtn = touchPad.locator('button[data-move="left"]');
  await leftBtn.tap();

  const jumpBtn = actionbar.locator('button[data-hold="jump"]');
  await jumpBtn.tap();

  // No horizontal scroll overflow
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true);
});
