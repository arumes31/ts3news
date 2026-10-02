const { test, expect } = require('@playwright/test');

test.describe('Provide a large-target action-bar option (Proposal 0152)', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/abyss/rift');
    await expect(page.locator('#rift-start')).toBeEnabled();
  });

  test('setting is present in Sound & display settings and defaults to unchecked', async ({ page }) => {
    const settings = page.locator('.rift-settings');
    await settings.locator('> summary').click();

    const checkbox = page.locator('#rift-large-action-bar');
    await expect(checkbox).toBeAttached();
    expect(await checkbox.isChecked()).toBe(false);

    const label = page.locator('label[for="rift-large-action-bar"]');
    await expect(label).toHaveText('Large-target action bar');
  });

  test('toggling large-target action bar adds .rift-large-action-bar to #rift-app and expands button sizes', async ({ page }) => {
    const root = page.locator('#rift-app');
    await expect(root).not.toHaveClass(/rift-large-action-bar/);

    const attackBtn = page.locator('.rift-basics button[data-bind="attack"]');
    const boxDefault = await attackBtn.boundingBox();
    expect(boxDefault).not.toBeNull();

    // Toggle on large-target action bar
    await page.locator('.rift-settings > summary').click();
    const checkbox = page.locator('#rift-large-action-bar');
    await checkbox.check();
    await expect(root).toHaveClass(/rift-large-action-bar/);

    // Bounding box of attack button should be at least 64px x 64px
    const boxLarge = await attackBtn.boundingBox();
    expect(boxLarge.width).toBeGreaterThanOrEqual(64);
    expect(boxLarge.height).toBeGreaterThanOrEqual(64);

    // Jump and Guard buttons must also have large target sizes
    const jumpBtn = page.locator('.rift-basics button[data-bind="jump"]');
    const jumpBox = await jumpBtn.boundingBox();
    expect(jumpBox.width).toBeGreaterThanOrEqual(64);
    expect(jumpBox.height).toBeGreaterThanOrEqual(64);

    const guardBtn = page.locator('.rift-basics button[data-bind="guard"]');
    const guardBox = await guardBtn.boundingBox();
    expect(guardBox.width).toBeGreaterThanOrEqual(64);
    expect(guardBox.height).toBeGreaterThanOrEqual(64);
  });

  test('accessible preset enables large-target action bar by default', async ({ page }) => {
    await page.locator('.rift-settings > summary').click();
    await page.locator('#rift-display-preset').selectOption('accessible');
    await page.locator('#rift-apply-preset').click();

    const checkbox = page.locator('#rift-large-action-bar');
    await expect(checkbox).toBeChecked();
    await expect(page.locator('#rift-app')).toHaveClass(/rift-large-action-bar/);
  });

  test('setting persists across page reload and resets cleanly', async ({ page }) => {
    await page.locator('.rift-settings > summary').click();
    const checkbox = page.locator('#rift-large-action-bar');
    await checkbox.check();

    await page.reload();
    await expect(page.locator('#rift-app')).toHaveClass(/rift-large-action-bar/);
    await page.locator('.rift-settings > summary').click();
    await expect(page.locator('#rift-large-action-bar')).toBeChecked();

    // Reset display restores unchecked state
    await page.locator('#rift-reset-display').click();
    await expect(page.locator('#rift-large-action-bar')).not.toBeChecked();
    await expect(page.locator('#rift-app')).not.toHaveClass(/rift-large-action-bar/);
  });

  test('large-target action bar wraps cleanly on narrow screens without horizontal overflow', async ({ page }) => {
    await page.locator('.rift-settings > summary').click();
    await page.locator('#rift-large-action-bar').check();

    await page.setViewportSize({ width: 390, height: 844 });

    // Verify touch target sizes remain at least 48px x 48px on mobile viewports
    const attackBtn = page.locator('.rift-basics button[data-bind="attack"]');
    const boxMobile = await attackBtn.boundingBox();
    expect(boxMobile.width).toBeGreaterThanOrEqual(48);
    expect(boxMobile.height).toBeGreaterThanOrEqual(48);

    // No document horizontal overflow
    const overflows = await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth);
    expect(overflows).toBe(true);
  });
});
