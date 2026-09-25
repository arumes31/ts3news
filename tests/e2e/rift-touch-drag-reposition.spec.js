const { test, expect } = require('@playwright/test');

test.use({ hasTouch: true, isMobile: true, viewport: { width: 390, height: 844 } });

test('supports drag-to-reposition touch control editor with save, cancel, reset, and persistence', async ({ page }) => {
  await page.goto('/abyss/rift');
  await expect(page.locator('#rift-start')).toBeEnabled();

  const settings = page.locator('.rift-settings');
  if (!(await settings.evaluate(el => el.open))) {
    await page.locator('.rift-settings > summary').tap();
  }

  const editBtn = page.locator('#rift-edit-touch-layout');
  await expect(editBtn).toBeVisible();

  const editor = page.locator('#rift-touch-editor');
  await expect(editor).toBeHidden();

  // 1. Enter edit mode
  await editBtn.tap();
  await expect(editor).toBeVisible();
  await expect(page.locator('#rift-app')).toHaveAttribute('data-touch-editing', 'true');

  const pad = page.locator('.rift-touch');
  await expect(pad).toBeVisible();

  // 2. Drag movement pad
  const padBox = await pad.boundingBox();
  expect(padBox).not.toBeNull();
  const startX = padBox.x + padBox.width / 2;
  const startY = padBox.y + padBox.height / 2;

  await pad.dispatchEvent('pointerdown', { pointerId: 1, pointerType: 'touch', isPrimary: true, clientX: startX, clientY: startY, button: 0 });
  await page.waitForTimeout(50);
  await pad.dispatchEvent('pointermove', { pointerId: 1, pointerType: 'touch', isPrimary: true, clientX: startX + 40, clientY: startY + 25 });
  await page.waitForTimeout(50);
  await pad.dispatchEvent('pointerup', { pointerId: 1, pointerType: 'touch', isPrimary: true, clientX: startX + 40, clientY: startY + 25, button: 0 });
  await page.waitForTimeout(50);

  // Check CSS variables updated
  const offsetX = await page.evaluate(() => document.getElementById('rift-app').style.getPropertyValue('--rift-touch-offset-x'));
  const offsetY = await page.evaluate(() => document.getElementById('rift-app').style.getPropertyValue('--rift-touch-offset-y'));
  expect(offsetX).toBe('40px');
  expect(offsetY).toBe('25px');

  // 3. Save layout
  await page.locator('#rift-save-touch-layout').tap();
  await expect(editor).toBeHidden();
  await expect(page.locator('#rift-app')).not.toHaveAttribute('data-touch-editing');

  // Verify saved in localStorage
  const savedData = await page.evaluate(() => localStorage.getItem('riftTouchOffsets'));
  expect(savedData).toContain('"x":40');
  expect(savedData).toContain('"y":25');

  // 4. Reload and verify persistence
  await page.reload();
  const reloadedX = await page.evaluate(() => document.getElementById('rift-app').style.getPropertyValue('--rift-touch-offset-x'));
  const reloadedY = await page.evaluate(() => document.getElementById('rift-app').style.getPropertyValue('--rift-touch-offset-y'));
  expect(reloadedX).toBe('40px');
  expect(reloadedY).toBe('25px');

  // 5. Test Cancel reverts changes
  if (!(await settings.evaluate(el => el.open))) {
    await page.locator('.rift-settings > summary').tap();
  }
  await page.locator('#rift-edit-touch-layout').tap();
  await expect(editor).toBeVisible();

  // Drag pad further
  await pad.dispatchEvent('pointerdown', { pointerId: 2, pointerType: 'touch', isPrimary: true, clientX: startX, clientY: startY, button: 0 });
  await pad.dispatchEvent('pointermove', { pointerId: 2, pointerType: 'touch', isPrimary: true, clientX: startX - 30, clientY: startY - 20 });
  await pad.dispatchEvent('pointerup', { pointerId: 2, pointerType: 'touch', isPrimary: true, clientX: startX - 30, clientY: startY - 20, button: 0 });

  // Cancel edit
  await page.locator('#rift-cancel-touch-layout').tap();
  await expect(editor).toBeHidden();
  const revertedX = await page.evaluate(() => document.getElementById('rift-app').style.getPropertyValue('--rift-touch-offset-x'));
  const revertedY = await page.evaluate(() => document.getElementById('rift-app').style.getPropertyValue('--rift-touch-offset-y'));
  expect(revertedX).toBe('40px');
  expect(revertedY).toBe('25px');

  // 6. Test Reset touch layout clears offsets
  if (!(await settings.evaluate(el => el.open))) {
    await page.locator('.rift-settings > summary').tap();
  }
  await page.locator('#rift-reset-touch-layout').tap();
  const resetX = await page.evaluate(() => document.getElementById('rift-app').style.getPropertyValue('--rift-touch-offset-x'));
  const resetY = await page.evaluate(() => document.getElementById('rift-app').style.getPropertyValue('--rift-touch-offset-y'));
  expect(resetX).toBe('0px');
  expect(resetY).toBe('0px');
  const clearedStorage = await page.evaluate(() => localStorage.getItem('riftTouchOffsets'));
  expect(clearedStorage).toBeNull();
});
