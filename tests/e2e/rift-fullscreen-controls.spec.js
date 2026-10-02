const { test, expect } = require('@playwright/test');

test('when enabled, clicking fullscreen opens controls dialog first, and entering fullscreen from dialog works', async ({ page }) => {
  await page.goto('/abyss/rift');
  await expect(page.locator('#rift-start')).toBeEnabled();

  const settings = page.locator('.rift-settings');
  await settings.locator('> summary').click();

  const fsOption = page.locator('#rift-fullscreen-controls');
  await expect(fsOption).toBeVisible();
  await fsOption.check();

  // Reload to verify persistence
  await page.reload();
  await expect(page.locator('#rift-start')).toBeEnabled();

  await settings.locator('> summary').click();
  await expect(fsOption).toBeChecked();
  await settings.locator('> summary').click();

  // Click fullscreen button
  const fsButton = page.locator('#rift-fullscreen');
  await expect(fsButton).toHaveAttribute('title', 'Fullscreen battlefield');
  await fsButton.click();

  // Controls dialog is displayed before entering fullscreen
  const controlsDialog = page.locator('#rift-controls-dialog');
  await expect(controlsDialog).toBeVisible();
  expect(await page.evaluate(() => document.fullscreenElement)).toBeNull();
  await expect(page.locator('#rift-status')).toContainText('Review your controls before entering fullscreen');

  // Proceed into fullscreen from dialog
  const dialogFs = page.locator('#rift-controls-fullscreen');
  await expect(dialogFs).toBeVisible();
  await dialogFs.click();

  await expect(controlsDialog).not.toBeVisible();
  await expect.poll(async () => page.evaluate(() => document.fullscreenElement?.id)).toBe('rift-viewport');

  // Exit fullscreen
  await page.evaluate(() => document.exitFullscreen());
  await expect.poll(async () => page.evaluate(() => document.fullscreenElement)).toBeNull();

  // Clicking fullscreen button again opens controls dialog when preference is enabled
  await fsButton.click();
  await expect(controlsDialog).toBeVisible();
  await page.locator('#rift-controls-close').click();
  await expect(controlsDialog).not.toBeVisible();
});

test('when disabled by default, clicking fullscreen enters fullscreen immediately', async ({ page }) => {
  await page.goto('/abyss/rift');
  await expect(page.locator('#rift-start')).toBeEnabled();

  const fsButton = page.locator('#rift-fullscreen');
  await fsButton.click();

  const controlsDialog = page.locator('#rift-controls-dialog');
  await expect(controlsDialog).not.toBeVisible();
  await expect.poll(async () => page.evaluate(() => document.fullscreenElement?.id)).toBe('rift-viewport');
});
