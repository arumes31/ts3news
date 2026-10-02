const { test, expect } = require('@playwright/test');

test('displays boss attack name during windup in HUD and telegraphs', async ({ page }) => {
  await page.goto('/abyss/rift?scenario=boss-windup');
  await expect(page.locator('#rift-start')).toBeEnabled();

  // Resume the paused boss-room fight
  await page.locator('#rift-start').click();

  // Verify boss meter and attack name display
  const bossSection = page.locator('#rift-boss');
  await expect(bossSection).toBeVisible();

  const bossAttack = page.locator('#rift-boss-attack');
  await expect(bossAttack).toBeVisible();
  await expect(bossAttack).toContainText('⚡ Mossbound Slam');
  await expect(bossAttack).toContainText('windup');

  // Verify clean screenshot mode hides boss attack banner
  const screenshotBtn = page.locator('#rift-screenshot-toggle');
  await screenshotBtn.click();
  await expect(screenshotBtn).toHaveAttribute('aria-pressed', 'true');
  await expect(bossSection).not.toBeVisible();
  await expect(bossAttack).not.toBeVisible();

  // Restore HUD
  await screenshotBtn.click();
  await expect(screenshotBtn).toHaveAttribute('aria-pressed', 'false');
  await expect(bossSection).toBeVisible();
  await expect(bossAttack).toBeVisible();
  await expect(bossAttack).toContainText('⚡ Mossbound Slam');
});
