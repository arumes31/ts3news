const { test, expect } = require('@playwright/test');

test('indicates when server is catching up after a stall in combat overview', async ({ page }) => {
  // In normal start, catchup indicator is hidden
  await page.goto('/abyss/rift');
  await expect(page.locator('#rift-start')).toBeEnabled();

  const catchupNode = page.locator('#rift-catchup');
  await expect(catchupNode).toBeHidden();
  await expect(catchupNode).toHaveAttribute('role', 'status');

  // Load server-catchup scenario
  await page.goto('/abyss/rift?scenario=server-catchup');
  await expect(page.locator('#rift-start')).toBeEnabled();

  await expect(catchupNode).toBeVisible();
  await expect(catchupNode).toContainText('⚡ Catching up…');

  // Verify clean screenshot mode hides catchup notice
  const screenshotBtn = page.locator('#rift-screenshot-toggle');
  await screenshotBtn.click();
  await expect(screenshotBtn).toHaveAttribute('aria-pressed', 'true');
  await expect(catchupNode).toBeHidden();

  // Restore HUD
  await screenshotBtn.click();
  await expect(screenshotBtn).toHaveAttribute('aria-pressed', 'false');
  await expect(catchupNode).toBeVisible();
  await expect(catchupNode).toContainText('⚡ Catching up…');
});
