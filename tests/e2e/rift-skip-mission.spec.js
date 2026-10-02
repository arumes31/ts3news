const { test, expect } = require('@playwright/test');

test('provides an accessible skip link directly to the mission picker', async ({ page }) => {
  await page.goto('/abyss/rift');
  await expect(page.locator('#rift-start')).toBeEnabled();

  const skipMission = page.locator('#rift-skip-mission');
  await expect(skipMission).toBeAttached();
  expect(await skipMission.getAttribute('href')).toBe('#rift-campaign');
  expect(await skipMission.textContent()).toBe('Skip to mission picker');

  // 1. Skip link is off-screen before focus
  const initialBox = await skipMission.boundingBox();
  expect(initialBox.y).toBeLessThan(0);

  // 2. Tabbing from page top: Tab 1 -> skip-controls, Tab 2 -> skip-mission
  await page.keyboard.press('Tab');
  await expect(page.locator('#rift-skip-controls')).toBeFocused();

  await page.keyboard.press('Tab');
  await expect(skipMission).toBeFocused();

  const focusedBox = await skipMission.boundingBox();
  expect(focusedBox.y).toBeGreaterThanOrEqual(0);
  expect(focusedBox.y).toBeLessThan(100);

  // 3. Activating skip link via Enter focuses the selected mission card
  await page.keyboard.press('Enter');

  const selectedMission = page.locator('#rift-levels button[aria-pressed="true"]');
  await expect(selectedMission).toBeFocused();
  const missionBox = await selectedMission.boundingBox();
  expect(missionBox.y).toBeGreaterThanOrEqual(0);

  // 4. If details#rift-campaign is collapsed, activating the skip link automatically expands it
  await page.evaluate(() => {
    const campaign = document.getElementById('rift-campaign');
    if (campaign) campaign.open = false;
    window.scrollTo(0, 0);
  });
  expect(await page.locator('#rift-campaign').getAttribute('open')).toBeNull();

  await skipMission.focus();
  await page.keyboard.press('Enter');

  await expect(page.locator('#rift-campaign')).toHaveAttribute('open', '');
  await expect(selectedMission).toBeFocused();

  // 5. Clicking the skip link also expands and focuses the mission picker
  await page.evaluate(() => {
    const campaign = document.getElementById('rift-campaign');
    if (campaign) campaign.open = false;
    window.scrollTo(0, 0);
  });
  await skipMission.focus();
  await skipMission.click();

  await expect(page.locator('#rift-campaign')).toHaveAttribute('open', '');
  await expect(selectedMission).toBeFocused();
});
