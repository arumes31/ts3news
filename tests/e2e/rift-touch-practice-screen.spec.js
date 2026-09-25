const { test, expect } = require('@playwright/test');

test.use({ hasTouch: true, isMobile: true, viewport: { width: 390, height: 844 } });

test('touch-control practice screen renders calibration guide and interactive touch controls', async ({ page }) => {
  await page.goto('/abyss/rift?practice=touch');

  // Verify navigation link is present
  const practiceLink = page.locator('#rift-practice-links a[href*="practice=touch"]');
  await expect(practiceLink).toBeVisible();
  await expect(practiceLink).toHaveText('Touch control practice');

  // Verify guide details
  const guide = page.locator('#rift-practice-guide');
  await expect(guide).toBeVisible();
  await expect(page.locator('#rift-practice-title')).toHaveText('Touch control calibration');
  await expect(page.locator('#rift-practice-instructions')).toContainText('Calibrate on-screen touch controls');

  // Start the drill
  await expect(page.locator('#rift-start')).toBeEnabled();
  await page.locator('#rift-start').tap();
  await expect(page.locator('#rift-overlay')).toBeHidden();

  // Verify HUD displays practice room name
  await expect(page.locator('#rift-room')).toHaveText('Practice · Touch control calibration');

  // Movement pad is visible and interactive
  const touchPad = page.locator('.rift-touch');
  await expect(touchPad).toBeVisible();

  const moveRight = page.locator('.rift-touch button[data-move="right"]');
  await expect(moveRight).toBeVisible();

  // Tap movement right button
  await moveRight.dispatchEvent('pointerdown', { pointerId: 1, pointerType: 'touch', isPrimary: true });
  await page.waitForTimeout(100);
  await moveRight.dispatchEvent('pointerup', { pointerId: 1, pointerType: 'touch', isPrimary: true });

  // Verify drill remains active (free practice mode)
  await expect(page.locator('#rift-overlay')).toBeHidden();

  // Test reset button
  const resetBtn = page.locator('#rift-practice-reset');
  await expect(resetBtn).toBeEnabled();
  await resetBtn.tap();
  await expect(page.locator('#rift-overlay')).toBeVisible();
  await expect(page.locator('#rift-overlay-title')).toHaveText('Touch control calibration');
});
