const { test, expect } = require('@playwright/test');

test('clicking setting labels activates and toggles their associated controls', async ({ page }) => {
  await page.goto('/abyss/rift');
  await expect(page.locator('#rift-start')).toBeEnabled();

  const settings = page.locator('.rift-settings');
  await settings.locator('> summary').click();

  // Test static checkboxes in .rift-settings
  const reducedCheck = page.locator('#rift-reduced');
  const reducedLabel = page.locator('label[for="rift-reduced"]');
  await expect(reducedCheck).not.toBeChecked();
  await reducedLabel.click();
  await expect(reducedCheck).toBeChecked();
  await reducedLabel.click();
  await expect(reducedCheck).not.toBeChecked();

  const bossCheck = page.locator('#rift-confirm-boss');
  const bossLabel = page.locator('label[for="rift-confirm-boss"]');
  await expect(bossCheck).not.toBeChecked();
  await bossLabel.click();
  await expect(bossCheck).toBeChecked();

  const pauseBossCheck = page.locator('#rift-pause-boss-room');
  const pauseBossLabel = page.locator('label[for="rift-pause-boss-room"]');
  await expect(pauseBossCheck).not.toBeChecked();
  await pauseBossLabel.click();
  await expect(pauseBossCheck).toBeChecked();

  const pauseRegionCheck = page.locator('#rift-pause-new-region');
  const pauseRegionLabel = page.locator('label[for="rift-pause-new-region"]');
  await expect(pauseRegionCheck).not.toBeChecked();
  await pauseRegionLabel.click();
  await expect(pauseRegionCheck).toBeChecked();

  const fsControlsCheck = page.locator('#rift-fullscreen-controls');
  const fsControlsLabel = page.locator('label[for="rift-fullscreen-controls"]');
  await expect(fsControlsCheck).not.toBeChecked();
  await fsControlsLabel.click();
  await expect(fsControlsCheck).toBeChecked();

  // Test dynamically created display settings
  const compactCheck = page.locator('#rift-compact-hud');
  const compactLabel = page.locator('label[for="rift-compact-hud"]');
  await expect(compactCheck).not.toBeChecked();
  await compactLabel.click();
  await expect(compactCheck).toBeChecked();

  const particlesCheck = page.locator('#rift-background-particles');
  const particlesLabel = page.locator('label[for="rift-background-particles"]');
  await expect(particlesCheck).toBeChecked();
  await particlesLabel.click();
  await expect(particlesCheck).not.toBeChecked();

  // Test dynamically created audio settings
  const steadyCheck = page.locator('#rift-steady-ambience');
  const steadyLabel = page.locator('label[for="rift-steady-ambience"]');
  await expect(steadyCheck).not.toBeChecked();
  await steadyLabel.click();
  await expect(steadyCheck).toBeChecked();

  const monoCheck = page.locator('#rift-mono-audio');
  const monoLabel = page.locator('label[for="rift-mono-audio"]');
  await expect(monoCheck).not.toBeChecked();
  await monoLabel.click();
  await expect(monoCheck).toBeChecked();

  // Test campaign setting
  const collapseCheck = page.locator('#rift-campaign-start-collapsed');
  const collapseLabel = page.locator('label[for="rift-campaign-start-collapsed"]');
  await expect(collapseCheck).not.toBeChecked();
  await collapseLabel.click();
  await expect(collapseCheck).toBeChecked();

  // Test controls dialog bindings & pointer labels
  await page.locator('#rift-controls-open').click();
  const controlsDialog = page.locator('#rift-controls-dialog');
  await expect(controlsDialog).toBeVisible();

  const toggleGuardCheck = page.locator('#rift-toggle-guard');
  const toggleGuardLabel = page.locator('label[for="rift-toggle-guard"]');
  await expect(toggleGuardCheck).not.toBeChecked();
  await toggleGuardLabel.click();
  await expect(toggleGuardCheck).toBeChecked();

  // Test clicking key remap label enters key rebind capture
  const attackRemapLabel = page.locator('label[for="rift-remap-attack"]');
  await expect(attackRemapLabel).toBeVisible();
  await attackRemapLabel.click();

  const bindingStatus = page.locator('#rift-binding-status');
  await expect(bindingStatus).toContainText('Press a physical key for Attack');

  // Cancel capture with Escape
  await page.keyboard.press('Escape');
  await expect(bindingStatus).toContainText('Binding unchanged');

  await page.locator('#rift-controls-close').click();
  await expect(controlsDialog).not.toBeVisible();
});
