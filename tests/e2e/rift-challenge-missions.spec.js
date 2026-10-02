const { test, expect } = require('@playwright/test');

test('identifies missions compatible with active challenge, displays status and badges, and filters by challenge compatibility', async ({ page }) => {
  // Test with gold_rush challenge fixture (requires Champion or Mythic difficulty)
  await page.goto('/abyss/rift?challenge=gold_rush');
  await expect(page.locator('#rift-start')).toBeEnabled();

  const challengeStatus = page.locator('#rift-mission-challenge');
  await expect(challengeStatus).toBeVisible();
  await expect(challengeStatus).toContainText('Active challenge:');
  await expect(challengeStatus).toContainText('Requires Champion or Mythic difficulty');

  // Mission 1 is Wayfarer, so it is incompatible with gold_rush
  await expect(challengeStatus.locator('.rift-challenge-status.incompatible')).toBeVisible();
  await expect(challengeStatus.locator('.rift-challenge-status')).toHaveText('✗ Incompatible');

  const mission1 = page.locator('#rift-levels button[data-level="1"]');
  await expect(mission1).not.toHaveClass(/challenge-compatible/);
  await expect(mission1.locator('.rift-challenge-badge')).not.toBeVisible();

  // Find a Champion or Mythic mission (e.g. Mission 65)
  const mission65 = page.locator('#rift-levels button[data-level="65"]');
  await expect(mission65).toHaveClass(/challenge-compatible/);
  const badge65 = mission65.locator('.rift-challenge-badge');
  await expect(badge65).toBeVisible();
  await expect(badge65).toHaveText('Challenge');

  // Select Mission 65 and verify mission plan updates to compatible
  await mission65.click();
  await expect(challengeStatus.locator('.rift-challenge-status.compatible')).toBeVisible();
  await expect(challengeStatus.locator('.rift-challenge-status')).toHaveText('✓ Compatible');

  // Verify challenge filter checkbox and label
  const challengeCheck = page.locator('#rift-challenge-only');
  await expect(challengeCheck).toBeVisible();
  await expect(challengeCheck).not.toBeChecked();

  // Clicking label toggles the checkbox
  const challengeLabel = page.locator('label[for="rift-challenge-only"]');
  await challengeLabel.click();
  await expect(challengeCheck).toBeChecked();

  // In gold_rush, Wayfarer (Mission 1) is hidden, Champion (Mission 65) remains visible
  await expect(mission1).toBeHidden();
  await expect(mission65).toBeVisible();

  // All visible missions must have challenge-compatible class
  const visibleCards = page.locator('#rift-levels button[data-level]:not([hidden])');
  const count = await visibleCards.count();
  expect(count).toBeGreaterThan(0);
  expect(count).toBeLessThan(100);
  for (let i = 0; i < count; i++) {
    await expect(visibleCards.nth(i)).toHaveClass(/challenge-compatible/);
  }
  await expect(page.locator('#rift-filter-count')).toHaveText(`${count} of 100 missions`);

  // Clear filters button resets challenge-only filter
  await page.locator('#rift-clear-filters').click();
  await expect(challengeCheck).not.toBeChecked();
  await expect(mission1).toBeVisible();
  await expect(page.locator('#rift-filter-count')).toHaveText('100 of 100 missions');

  // Test persistence: re-enable filter and reload
  await challengeCheck.check();
  await expect(challengeCheck).toBeChecked();
  await page.reload();
  await expect(page.locator('#rift-challenge-only')).toBeChecked();
  await expect(page.locator('#rift-levels button[data-level="1"]')).toBeHidden();
});

test('handles double_hazards challenge criteria correctly', async ({ page }) => {
  await page.goto('/abyss/rift?challenge=double_hazards');
  await expect(page.locator('#rift-start')).toBeEnabled();

  const challengeStatus = page.locator('#rift-mission-challenge');
  await expect(challengeStatus).toBeVisible();
  await expect(challengeStatus).toContainText('Requires at least 2 hazard zones across the route');

  // Check filter toggle
  await page.locator('#rift-challenge-only').check();
  const visibleCards = page.locator('#rift-levels button[data-level]:not([hidden])');
  const count = await visibleCards.count();
  expect(count).toBeGreaterThan(0);
  for (let i = 0; i < count; i++) {
    await expect(visibleCards.nth(i)).toHaveClass(/challenge-compatible/);
  }
});
