const { test, expect } = require('@playwright/test');

test('shows the range of an equipped skill on request via hover, hotkey, range toggle, and display settings', async ({ page }) => {
  await page.goto('/abyss/rift');
  await expect(page.locator('#rift-start')).toBeEnabled();

  const rangeToggle = page.locator('#rift-range-toggle');
  await expect(rangeToggle).toBeVisible();
  await expect(rangeToggle).toHaveAttribute('title', 'Toggle skill range preview');
  await expect(rangeToggle).toHaveAttribute('aria-label', 'Toggle skill range preview');
  await expect(rangeToggle).toHaveAttribute('aria-pressed', 'false');

  // Start expedition to enter combat
  await page.locator('#rift-start').click();
  await expect(page.locator('.rift-combat-signals')).toBeVisible();

  const skillRangeSignal = page.locator('#rift-skill-range');
  await expect(skillRangeSignal).toBeVisible();
  await expect(skillRangeSignal).toContainText('Skill range: None requested');

  // 1. Request range by focusing the first equipped skill button
  const skillButtons = page.locator('#rift-skills button');
  await expect(skillButtons.first()).toBeVisible();
  const firstSkillName = await skillButtons.first().locator('.rift-action-label').textContent();

  await skillButtons.first().focus();
  await expect(skillRangeSignal).toContainText('Range: ' + firstSkillName);
  await expect(skillRangeSignal).toHaveText(/Area|Projectile|Self/);

  // Blur restores default state when unpinned
  await skillButtons.first().blur();
  await expect(skillRangeSignal).toContainText('Skill range: None requested');

  // 2. Request range via top toggle button (#rift-range-toggle)
  await rangeToggle.click();
  await expect(rangeToggle).toHaveAttribute('aria-pressed', 'true');
  await expect(skillRangeSignal).toContainText('Range: ' + firstSkillName);

  // 3. Cycle to next equipped skill
  const secondSkillName = await skillButtons.nth(1).locator('.rift-action-label').textContent();
  await rangeToggle.click();
  await expect(rangeToggle).toHaveAttribute('aria-pressed', 'true');
  await expect(skillRangeSignal).toContainText('Range: ' + secondSkillName);

  // 4. Request specific skill via Shift+Digit1 shortcut
  await page.keyboard.press('Shift+Digit1');
  await expect(skillRangeSignal).toContainText('Range: ' + firstSkillName);

  // 5. Clean screenshot mode hides range display signal
  const screenshotBtn = page.locator('#rift-screenshot-toggle');
  await screenshotBtn.click();
  await expect(screenshotBtn).toHaveAttribute('aria-pressed', 'true');
  await expect(skillRangeSignal).not.toBeVisible();

  // Restore HUD
  await screenshotBtn.click();
  await expect(screenshotBtn).toHaveAttribute('aria-pressed', 'false');
  await expect(skillRangeSignal).toBeVisible();

  // 6. Sound & display settings toggle and label activation
  const settings = page.locator('.rift-settings');
  await settings.locator('> summary').click();

  const rangeCheck = page.locator('#rift-show-skill-range');
  const rangeLabel = page.locator('label[for="rift-show-skill-range"]');
  await expect(rangeCheck).toBeVisible();
  await expect(rangeCheck).not.toBeChecked();

  // Clicking label toggles checkbox
  await rangeLabel.click();
  await expect(rangeCheck).toBeChecked();

  // 7. Master preferences reset restores setting to unchecked
  await page.locator('#rift-reset-preferences').click();
  await expect(rangeCheck).not.toBeChecked();
});
