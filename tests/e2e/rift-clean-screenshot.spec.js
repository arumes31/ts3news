const { test, expect } = require('@playwright/test');

test('clean screenshot mode hides on-screen HUD elements and toggles via button, hotkey, and settings', async ({ page }) => {
  await page.goto('/abyss/rift');
  await expect(page.locator('#rift-start')).toBeEnabled();

  const screenshotBtn = page.locator('#rift-screenshot-toggle');
  await expect(screenshotBtn).toBeVisible();
  await expect(screenshotBtn).toHaveAttribute('title', 'Clean screenshot mode (hide HUD)');
  await expect(screenshotBtn).toHaveAttribute('aria-label', 'Clean screenshot mode (hide HUD)');
  await expect(screenshotBtn).toHaveAttribute('aria-pressed', 'false');

  // Start expedition to enter combat and display active HUD
  await page.locator('#rift-start').click();
  await expect(page.locator('#rift-vitals')).toBeVisible();
  await expect(page.locator('.rift-combat-overview')).toBeVisible();
  await expect(page.locator('.rift-actionbar')).toBeVisible();
  await expect(page.locator('.rift-classbar')).toBeVisible();
  await expect(page.locator('.rift-combat-signals')).toBeVisible();

  // 1. Toggle clean screenshot mode via top button
  await screenshotBtn.click();
  await expect(screenshotBtn).toHaveAttribute('aria-pressed', 'true');
  await expect(page.locator('#rift-app')).toHaveClass(/rift-clean-screenshot/);

  // Assert on-screen HUD elements are hidden
  await expect(page.locator('#rift-vitals')).not.toBeVisible();
  await expect(page.locator('.rift-combat-overview')).not.toBeVisible();
  await expect(page.locator('.rift-actionbar')).not.toBeVisible();
  await expect(page.locator('.rift-classbar')).not.toBeVisible();
  await expect(page.locator('.rift-combat-signals')).not.toBeVisible();

  // 2. Toggle back via F4 hotkey
  await page.keyboard.press('F4');
  await expect(screenshotBtn).toHaveAttribute('aria-pressed', 'false');
  await expect(page.locator('#rift-app')).not.toHaveClass(/rift-clean-screenshot/);

  // Assert on-screen HUD elements are restored
  await expect(page.locator('#rift-vitals')).toBeVisible();
  await expect(page.locator('.rift-combat-overview')).toBeVisible();
  await expect(page.locator('.rift-actionbar')).toBeVisible();
  await expect(page.locator('.rift-classbar')).toBeVisible();
  await expect(page.locator('.rift-combat-signals')).toBeVisible();

  // 3. Toggle via settings checkbox
  const settings = page.locator('.rift-settings');
  await settings.locator('> summary').click();
  const cleanCheck = page.locator('#rift-clean-screenshot');
  await expect(cleanCheck).toBeVisible();
  await expect(cleanCheck).not.toBeChecked();

  // Click setting label
  const cleanLabel = page.locator('label[for="rift-clean-screenshot"]');
  await cleanLabel.click();
  await expect(cleanCheck).toBeChecked();
  await expect(screenshotBtn).toHaveAttribute('aria-pressed', 'true');
  await expect(page.locator('#rift-vitals')).not.toBeVisible();

  // 4. Persistence across reload
  await page.reload();
  await expect(page.locator('#rift-app')).toHaveClass(/rift-clean-screenshot/);
  await expect(screenshotBtn).toHaveAttribute('aria-pressed', 'true');
  await settings.locator('> summary').click();
  await expect(cleanCheck).toBeChecked();

  // 5. Master reset restores cleanScreenshot to false
  await page.locator('#rift-reset-display').click();
  await expect(cleanCheck).not.toBeChecked();
  await expect(screenshotBtn).toHaveAttribute('aria-pressed', 'false');
  await expect(page.locator('#rift-app')).not.toHaveClass(/rift-clean-screenshot/);
});
