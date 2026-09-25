const { test, expect } = require('@playwright/test');

test.use({ hasTouch: true, isMobile: true, viewport: { width: 390, height: 844 } });

test('offers fixed pad by default and allows switching to joystick-style movement with persistence and reset', async ({ page }) => {
  await page.goto('/abyss/rift');
  await expect(page.locator('#rift-start')).toBeEnabled();

  // Open settings
  const settings = page.locator('.rift-settings');
  if (!(await settings.evaluate(el => el.open))) {
    await page.locator('.rift-settings > summary').tap();
  }

  const modeSelect = page.locator('#rift-touch-mode');
  await expect(modeSelect).toBeVisible();
  await expect(modeSelect).toHaveValue('pad');

  const fixedPad = page.locator('.rift-touch');
  const joystick = page.locator('#rift-touch-joystick');

  // 1. Verify default: fixed pad visible, joystick hidden
  await expect(fixedPad).toBeVisible();
  await expect(joystick).toBeHidden();

  // 2. Switch to joystick
  await modeSelect.selectOption('joystick');
  await expect(page.locator('#rift-app')).toHaveAttribute('data-touch-mode', 'joystick');
  await expect(fixedPad).toBeHidden();
  await expect(joystick).toBeVisible();

  // 3. Verify persistence across reload
  await page.reload();
  await expect(page.locator('#rift-app')).toHaveAttribute('data-touch-mode', 'joystick');
  await expect(page.locator('.rift-touch')).toBeHidden();
  await expect(page.locator('#rift-touch-joystick')).toBeVisible();

  // 4. Test Reset touch layout restores fixed pad
  if (!(await settings.evaluate(el => el.open))) {
    await page.locator('.rift-settings > summary').tap();
  }
  const resetBtn = page.locator('#rift-reset-touch-layout');
  await resetBtn.tap();

  await expect(modeSelect).toHaveValue('pad');
  await expect(page.locator('#rift-app')).toHaveAttribute('data-touch-mode', 'pad');
  await expect(fixedPad).toBeVisible();
  await expect(joystick).toBeHidden();

  // 5. Switch back to joystick and test drag functionality in combat
  await modeSelect.selectOption('joystick');
  await expect(joystick).toBeVisible();

  // Start combat to test joystick dragging
  await expect(page.locator('#rift-start')).toBeEnabled();
  await page.locator('#rift-start').tap();
  await expect(page.locator('#rift-overlay')).toBeHidden();

  const joystickBase = page.locator('#rift-joystick-base');
  const joystickStick = page.locator('#rift-joystick-stick');
  await expect(joystickBase).toBeVisible();
  await expect(joystickStick).toBeVisible();

  const box = await joystickBase.boundingBox();
  expect(box).not.toBeNull();
  const centerX = box.x + box.width / 2;
  const centerY = box.y + box.height / 2;

  // Drag stick to the right
  await joystickBase.dispatchEvent('pointerdown', { pointerId: 1, pointerType: 'touch', isPrimary: true, clientX: centerX, clientY: centerY, button: 0 });
  await page.waitForTimeout(50);
  await joystickBase.dispatchEvent('pointermove', { pointerId: 1, pointerType: 'touch', isPrimary: true, clientX: centerX + 30, clientY: centerY });
  await page.waitForTimeout(100);

  // Stick knob has been displaced
  const transform = await joystickStick.evaluate(el => el.style.transform);
  expect(transform).toContain('px');
  expect(transform).not.toBe('translate(0px, 0px)');

  // Release joystick
  await joystickBase.dispatchEvent('pointerup', { pointerId: 1, pointerType: 'touch', isPrimary: true, clientX: centerX + 30, clientY: centerY, button: 0 });
  await page.waitForTimeout(50);
  const releasedTransform = await joystickStick.evaluate(el => el.style.transform);
  expect(releasedTransform).toBe('translate(0px, 0px)');
});
