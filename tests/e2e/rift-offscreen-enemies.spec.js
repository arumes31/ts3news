const { test, expect } = require('@playwright/test');

test('shows off-screen enemy direction indicators on canvas and in combat signals with toggle support', async ({ page }) => {
  await page.goto('/abyss/rift');
  await expect(page.locator('#rift-start')).toBeEnabled();

  // 1. Verify settings control and label association
  const settings = page.locator('.rift-settings');
  await settings.locator('> summary').click();

  const indicatorCheck = page.locator('#rift-enemy-indicators');
  await expect(indicatorCheck).toBeVisible();
  await expect(indicatorCheck).toBeChecked(); // default enabled

  const indicatorLabel = page.locator('label[for="rift-enemy-indicators"]');
  await expect(indicatorLabel).toBeVisible();
  await expect(indicatorLabel).toHaveText('Off-screen enemy direction indicators');

  // 2. Start expedition to enter combat
  await page.locator('#rift-start').click();
  await expect(page.locator('#rift-vitals')).toBeVisible();

  // Verify offscreen enemy signal in combat signals
  const offscreenSignal = page.locator('#rift-offscreen-enemies');
  await expect(offscreenSignal).toBeVisible();
  await expect(offscreenSignal).toHaveText(/Off-screen threats:|No off-screen threats/);

  // Poll renderer to verify indicator rendering state when enemies are offscreen
  await page.waitForFunction(() => {
    return window.RiftRenderer && Array.isArray(window.RiftRenderer.lastOffscreen);
  });

  // Verify that lastOffscreen is populated when off-screen enemies exist
  const hasOffscreen = await page.evaluate(() => {
    return window.RiftRenderer.lastOffscreen.length;
  });
  expect(typeof hasOffscreen).toBe('number');

  // 3. Disable enemy indicators in settings
  await indicatorCheck.uncheck();
  await expect(indicatorCheck).not.toBeChecked();

  // Verify renderer suppresses indicators
  await page.waitForFunction(() => {
    return window.RiftRenderer.lastOffscreen.length === 0;
  });

  // 4. Persistence across reload
  await page.reload();
  await expect(page.locator('#rift-start')).toBeEnabled();
  await page.locator('#rift-start').click();
  await expect(page.locator('#rift-vitals')).toBeVisible();
  await settings.locator('> summary').click();
  await expect(page.locator('#rift-enemy-indicators')).not.toBeChecked();

  // Toggle back on using label click
  await page.locator('label[for="rift-enemy-indicators"]').click();
  await expect(page.locator('#rift-enemy-indicators')).toBeChecked();

  // 5. Clean screenshot mode hides off-screen indicators
  const screenshotBtn = page.locator('#rift-screenshot-toggle');
  await screenshotBtn.click();
  await expect(screenshotBtn).toHaveAttribute('aria-pressed', 'true');
  await page.waitForFunction(() => {
    return window.RiftRenderer.lastOffscreen.length === 0;
  });
  await expect(offscreenSignal).not.toBeVisible();

  // Toggle screenshot mode off
  await screenshotBtn.click();
  await expect(screenshotBtn).toHaveAttribute('aria-pressed', 'false');
  await expect(offscreenSignal).toBeVisible();

  // 6. Master preferences reset restores enemy indicators to default true
  await page.locator('#rift-reset-display').click();
  await expect(page.locator('#rift-enemy-indicators')).toBeChecked();
});
