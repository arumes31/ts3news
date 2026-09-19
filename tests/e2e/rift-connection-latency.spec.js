const { test, expect } = require('@playwright/test');

test('displays connection latency in combat overview without alarming color at normal values', async ({ page }) => {
  await page.goto('/abyss/rift');
  await expect(page.locator('#rift-start')).toBeEnabled();

  const latencyIndicator = page.locator('#rift-latency');
  await expect(latencyIndicator).toBeVisible();
  await expect(latencyIndicator).toHaveAttribute('role', 'status');

  // Initial load performs API request, so latency is recorded
  await expect(latencyIndicator).toContainText('ms');
  await expect(latencyIndicator).toHaveAttribute('data-state', 'normal');

  // Check that styling at normal values uses non-alarming calm color (#bad0b5)
  const color = await latencyIndicator.evaluate(el => window.getComputedStyle(el).color);
  expect(color).toBe('rgb(186, 208, 181)');

  // Start expedition to perform combat requests
  await page.locator('#rift-start').click();
  await expect(page.locator('.rift-combat-overview')).toBeVisible();

  // Wait for combat step request to update latency
  await expect(latencyIndicator).toHaveText(/\d+\s*ms/);
  await expect(latencyIndicator).toHaveAttribute('data-state', 'normal');
  await expect(latencyIndicator).toHaveAttribute('aria-label', /Connection latency: \d+ ms \(normal\)/);

  // Verify clean screenshot mode hides combat overview and latency
  const screenshotBtn = page.locator('#rift-screenshot-toggle');
  await screenshotBtn.click();
  await expect(screenshotBtn).toHaveAttribute('aria-pressed', 'true');
  await expect(latencyIndicator).not.toBeVisible();

  // Restore HUD
  await screenshotBtn.click();
  await expect(screenshotBtn).toHaveAttribute('aria-pressed', 'false');
  await expect(latencyIndicator).toBeVisible();
  await expect(latencyIndicator).toHaveText(/\d+\s*ms/);
});
