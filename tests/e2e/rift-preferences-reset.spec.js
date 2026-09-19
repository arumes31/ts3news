const { test, expect } = require('@playwright/test');

test('master reset restores display, audio, and workflow preferences to defaults without affecting saved game data', async ({ page }) => {
  // Seed a campaign favorite and a bestiary bookmark to verify they survive preference reset
  await page.addInitScript(() => {
    if (!sessionStorage.getItem('preferencesResetSeed')) {
      localStorage.setItem('riftCampaignView', JSON.stringify({ version: 1, favorites: [2], startCollapsed: true }));
      localStorage.setItem('riftMonsterBookmarks', JSON.stringify({ version: 1, keys: ['goblin_scout'] }));
      sessionStorage.setItem('preferencesResetSeed', '1');
    }
  });

  await page.goto('/abyss/rift');
  await expect(page.locator('#rift-start')).toBeEnabled();

  const settings = page.locator('.rift-settings');
  await settings.locator('> summary').click();

  // 1. Mutate display settings
  await page.locator('#rift-text-scale').selectOption('1.25');
  await page.locator('#rift-compact-hud').check();
  await page.locator('#rift-shake-intensity').selectOption('0.5');
  await page.locator('#rift-background-particles').uncheck();

  // 2. Mutate audio settings
  await page.locator('#rift-interface-volume').evaluate(el => { el.value = '20'; el.dispatchEvent(new Event('input')); });
  await page.locator('#rift-mono-audio').check();
  await page.locator('#rift-steady-ambience').check();

  // 3. Mutate workflow options
  await page.locator('#rift-confirm-boss').check();
  await page.locator('#rift-pause-boss-room').check();
  await page.locator('#rift-pause-new-region').check();
  await page.locator('#rift-transition-delay').selectOption('5');
  await expect(page.locator('#rift-campaign-start-collapsed')).toBeChecked();

  // Reload and verify settings persisted
  await page.reload();
  await expect(page.locator('#rift-start')).toBeEnabled();
  await settings.locator('> summary').click();

  await expect(page.locator('#rift-text-scale')).toHaveValue('1.25');
  await expect(page.locator('#rift-compact-hud')).toBeChecked();
  await expect(page.locator('#rift-shake-intensity')).toHaveValue('0.5');
  await expect(page.locator('#rift-background-particles')).not.toBeChecked();
  await expect(page.locator('#rift-interface-volume')).toHaveValue('20');
  await expect(page.locator('#rift-mono-audio')).toBeChecked();
  await expect(page.locator('#rift-steady-ambience')).toBeChecked();
  await expect(page.locator('#rift-confirm-boss')).toBeChecked();
  await expect(page.locator('#rift-pause-boss-room')).toBeChecked();
  await expect(page.locator('#rift-pause-new-region')).toBeChecked();
  await expect(page.locator('#rift-transition-delay')).toHaveValue('5');
  await expect(page.locator('#rift-campaign-start-collapsed')).toBeChecked();

  // 4. Click master preferences reset button
  const masterReset = page.locator('#rift-reset-preferences');
  await expect(masterReset).toBeVisible();
  await masterReset.click();

  const status = page.locator('#rift-preferences-status');
  await expect(status).toContainText('restored to defaults');

  // Verify UI controls immediately reflect defaults
  await expect(page.locator('#rift-text-scale')).toHaveValue('1');
  await expect(page.locator('#rift-compact-hud')).not.toBeChecked();
  await expect(page.locator('#rift-shake-intensity')).toHaveValue('0');
  await expect(page.locator('#rift-background-particles')).toBeChecked();
  await expect(page.locator('#rift-interface-volume')).toHaveValue('65');
  await expect(page.locator('#rift-mono-audio')).not.toBeChecked();
  await expect(page.locator('#rift-steady-ambience')).not.toBeChecked();
  await expect(page.locator('#rift-confirm-boss')).not.toBeChecked();
  await expect(page.locator('#rift-pause-boss-room')).not.toBeChecked();
  await expect(page.locator('#rift-pause-new-region')).not.toBeChecked();
  await expect(page.locator('#rift-transition-delay')).toHaveValue('1.2');
  await expect(page.locator('#rift-campaign-start-collapsed')).not.toBeChecked();

  // Reload to verify reset preferences persist
  await page.reload();
  await expect(page.locator('#rift-start')).toBeEnabled();
  await settings.locator('> summary').click();

  await expect(page.locator('#rift-text-scale')).toHaveValue('1');
  await expect(page.locator('#rift-compact-hud')).not.toBeChecked();
  await expect(page.locator('#rift-shake-intensity')).toHaveValue('0');
  await expect(page.locator('#rift-background-particles')).toBeChecked();
  await expect(page.locator('#rift-interface-volume')).toHaveValue('65');
  await expect(page.locator('#rift-mono-audio')).not.toBeChecked();
  await expect(page.locator('#rift-steady-ambience')).not.toBeChecked();
  await expect(page.locator('#rift-confirm-boss')).not.toBeChecked();
  await expect(page.locator('#rift-pause-boss-room')).not.toBeChecked();
  await expect(page.locator('#rift-pause-new-region')).not.toBeChecked();
  await expect(page.locator('#rift-transition-delay')).toHaveValue('1.2');
  await expect(page.locator('#rift-campaign-start-collapsed')).not.toBeChecked();

  // Verify game data is unharmed: favorites and monster bookmarks survived
  const campaignView = await page.evaluate(() => JSON.parse(localStorage.getItem('riftCampaignView') || '{}'));
  expect(campaignView.favorites).toEqual([2]);
  expect(campaignView.startCollapsed).toBe(false);

  const bookmarks = await page.evaluate(() => JSON.parse(localStorage.getItem('riftMonsterBookmarks') || '{}'));
  expect(bookmarks.keys).toEqual(['goblin_scout']);

  // Verify no expedition was created
  const res = await page.request.get('/api/abyss/rift');
  expect((await res.json()).run).toBeNull();
});
