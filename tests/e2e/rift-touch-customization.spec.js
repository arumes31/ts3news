const { test, expect } = require('@playwright/test');

test.use({ hasTouch: true, isMobile: true, viewport: { width: 390, height: 844 } });

test('enlarges movement-pad targets on coarse pointer without enlarging HUD', async ({ page }) => {
  await page.goto('/abyss/rift?practice=skills');
  await expect(page.locator('#rift-start')).toBeEnabled();
  await page.locator('#rift-start').tap();
  await expect(page.locator('#rift-overlay')).toBeHidden();

  // Movement pad buttons meet enlarged target criteria (>=56px)
  const padButtons = page.locator('.rift-touch button');
  const count = await padButtons.count();
  expect(count).toBe(4);

  const padSizes = await padButtons.evaluateAll(nodes =>
    nodes.map(n => {
      const r = n.getBoundingClientRect();
      return { width: r.width, height: r.height };
    })
  );

  for (const s of padSizes) {
    expect(s.width).toBeGreaterThanOrEqual(56);
    expect(s.height).toBeGreaterThanOrEqual(56);
  }

  // HUD elements remain standard compact size (not scaled up with the pad)
  const vitalsName = page.locator('#rift-name');
  const nameFontSize = await vitalsName.evaluate(el => window.getComputedStyle(el).fontSize);
  expect(parseFloat(nameFontSize)).toBeLessThanOrEqual(16);

  const meterHeight = await page.locator('#rift-vitals .rift-meter.hp').evaluate(el => el.getBoundingClientRect().height);
  expect(meterHeight).toBeLessThanOrEqual(16);

  const stageBtnFontSize = await page.locator('.rift-stage-top button').first().evaluate(el => window.getComputedStyle(el).fontSize);
  expect(parseFloat(stageBtnFontSize)).toBeLessThanOrEqual(12);
});

test('supports left and right placement, opacity, and scale customization with persistence and reset', async ({ page }) => {
  await page.goto('/abyss/rift');
  await expect(page.locator('#rift-start')).toBeEnabled();

  // Open settings
  const settings = page.locator('.rift-settings');
  if (!(await settings.evaluate(el => el.open))) {
    await page.locator('.rift-settings > summary').tap();
  }

  const layoutSelect = page.locator('#rift-touch-layout');
  const opacityInput = page.locator('#rift-touch-opacity');
  const scaleSelect = page.locator('#rift-touch-scale');
  const resetBtn = page.locator('#rift-reset-touch-layout');

  await expect(layoutSelect).toBeVisible();
  await expect(opacityInput).toBeVisible();
  await expect(scaleSelect).toBeVisible();
  await expect(resetBtn).toBeVisible();

  // 1. Test left / right placement
  await layoutSelect.selectOption('left');
  await expect(page.locator('.rift-touch')).toHaveCSS('justify-content', 'flex-start');
  expect(await page.locator('#rift-app').getAttribute('data-touch-layout')).toBe('left');

  await layoutSelect.selectOption('right');
  await expect(page.locator('.rift-touch')).toHaveCSS('justify-content', 'flex-end');
  expect(await page.locator('#rift-app').getAttribute('data-touch-layout')).toBe('right');

  // 2. Test opacity slider (e.g. 60%)
  await opacityInput.fill('60');
  await opacityInput.dispatchEvent('change');
  await expect(page.locator('.rift-touch')).toHaveCSS('opacity', '0.6');
  expect(await page.evaluate(() => localStorage.getItem('riftTouchOpacity'))).toBe('60');

  // 3. Test scale setting (e.g. 120%)
  await scaleSelect.selectOption('1.2');
  const scaleVal = await page.evaluate(() => {
    return getComputedStyle(document.getElementById('rift-app')).getPropertyValue('--rift-touch-scale');
  });
  expect(scaleVal).toBe('1.2');
  expect(await page.evaluate(() => localStorage.getItem('riftTouchScale'))).toBe('1.2');

  // 4. Persistence across page reload
  await page.reload();
  await expect(page.locator('#rift-start')).toBeEnabled();
  await expect(page.locator('.rift-touch')).toHaveCSS('opacity', '0.6');
  await expect(page.locator('.rift-touch')).toHaveCSS('justify-content', 'flex-end');
  const reloadedScale = await page.evaluate(() => {
    return getComputedStyle(document.getElementById('rift-app')).getPropertyValue('--rift-touch-scale');
  });
  expect(reloadedScale).toBe('1.2');

  // 5. Reset touch layout restores defaults
  if (!(await settings.evaluate(el => el.open))) {
    await page.locator('.rift-settings > summary').tap();
  }
  await resetBtn.tap();

  await expect(layoutSelect).toHaveValue('center');
  await expect(page.locator('.rift-touch')).toHaveCSS('justify-content', 'center');
  await expect(page.locator('.rift-touch')).toHaveCSS('opacity', '1');
  const resetScale = await page.evaluate(() => {
    return getComputedStyle(document.getElementById('rift-app')).getPropertyValue('--rift-touch-scale');
  });
  expect(resetScale).toBe('1');
});
