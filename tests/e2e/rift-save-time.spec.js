const { test, expect } = require('@playwright/test');

test('shows the last confirmed save time in combat overview', async ({ page }) => {
  await page.goto('/abyss/rift');
  await expect(page.locator('#rift-start')).toBeEnabled();

  const savedAt = page.locator('#rift-saved-at');
  await expect(savedAt).toBeVisible();
  await expect(savedAt).toHaveAttribute('role', 'status');

  // Before starting, shows placeholder
  await expect(savedAt).toHaveText('Save: --');

  // Start expedition
  await page.locator('#rift-start').click();
  await expect(page.locator('.rift-combat-overview')).toBeVisible();

  // After combat step request, SavedAtMS is set and display updates to HH:MM:SS
  await expect(savedAt).toHaveText(/Save: \d{2}:\d{2}:\d{2}/);
  await expect(savedAt).toHaveAttribute('aria-label', /Last confirmed save time:/);

  // Verify clean screenshot mode hides combat overview and save time
  const screenshotBtn = page.locator('#rift-screenshot-toggle');
  await screenshotBtn.click();
  await expect(screenshotBtn).toHaveAttribute('aria-pressed', 'true');
  await expect(savedAt).not.toBeVisible();

  // Restore HUD
  await screenshotBtn.click();
  await expect(screenshotBtn).toHaveAttribute('aria-pressed', 'false');
  await expect(savedAt).toBeVisible();
  await expect(savedAt).toHaveText(/Save: \d{2}:\d{2}:\d{2}/);
});
