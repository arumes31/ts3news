const { test, expect } = require('@playwright/test');

test.describe('Action labels understandable without keycap symbols (Proposal 0159)', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/abyss/rift');
    await expect(page.locator('#rift-start')).toBeEnabled();
  });

  test('action buttons have clean accessible names and aria-keyshortcuts without keycap symbol clutter', async ({ page }) => {
    const attackBtn = page.locator('.rift-basics button[data-bind="attack"]');
    const jumpBtn = page.locator('.rift-basics button[data-bind="jump"]');
    const guardBtn = page.locator('.rift-basics button[data-bind="guard"]');
    const pauseBtn = page.locator('#rift-pause');

    // Keycaps must be aria-hidden="true" so screen readers and speech control don't hear "J Attack", "Space Jump", etc.
    await expect(attackBtn.locator('kbd')).toHaveAttribute('aria-hidden', 'true');
    await expect(jumpBtn.locator('kbd')).toHaveAttribute('aria-hidden', 'true');
    await expect(guardBtn.locator('kbd')).toHaveAttribute('aria-hidden', 'true');
    await expect(pauseBtn.locator('kbd')).toHaveAttribute('aria-hidden', 'true');

    // Accessible names must match the action directly
    await expect(attackBtn).toHaveAttribute('aria-label', 'Attack');
    await expect(jumpBtn).toHaveAttribute('aria-label', 'Jump');
    await expect(guardBtn).toHaveAttribute('aria-label', 'Guard');

    // aria-keyshortcuts must convey the shortcut metadata
    await expect(attackBtn).toHaveAttribute('aria-keyshortcuts', 'J');
    await expect(jumpBtn).toHaveAttribute('aria-keyshortcuts', 'Space / K');
    await expect(guardBtn).toHaveAttribute('aria-keyshortcuts', 'L');

    // Buttons with visible text must NOT have title tooltips
    await expect(attackBtn).not.toHaveAttribute('title');
    await expect(jumpBtn).not.toHaveAttribute('title');
    await expect(guardBtn).not.toHaveAttribute('title');
    await expect(pauseBtn).not.toHaveAttribute('title');
  });

  test('pause button transitions between Pause and Resume with isolated action label and keycap', async ({ page }) => {
    const pauseBtn = page.locator('#rift-pause');

    // Start expedition
    await page.locator('#rift-start').click();
    await expect(pauseBtn).toBeEnabled();

    // In playing state
    await expect(pauseBtn).toHaveAttribute('aria-label', 'Pause expedition');
    await expect(pauseBtn).toHaveAttribute('aria-keyshortcuts', 'Escape');
    await expect(pauseBtn.locator('.rift-action-label')).toHaveText('Pause');
    await expect(pauseBtn.locator('kbd')).toHaveAttribute('aria-hidden', 'true');

    // Click pause
    await pauseBtn.click();
    await expect(page.locator('#rift-overlay')).toBeVisible();

    // In paused state
    await expect(pauseBtn).toHaveAttribute('aria-label', 'Resume expedition');
    await expect(pauseBtn.locator('.rift-action-label')).toHaveText('Resume');
    await expect(pauseBtn.locator('kbd')).toHaveAttribute('aria-hidden', 'true');
  });

  test('action labels remain legible and understandable when keycaps are hidden', async ({ page }) => {
    // Add .rift-no-keycaps utility class to simulate non-keycap / touch / restricted view
    await page.evaluate(() => document.getElementById('rift-app').classList.add('rift-no-keycaps'));

    const attackLabel = page.locator('.rift-basics button[data-bind="attack"] .rift-action-label');
    const jumpLabel = page.locator('.rift-basics button[data-bind="jump"] .rift-action-label');
    const guardLabel = page.locator('.rift-basics button[data-bind="guard"] .rift-action-label');
    const attackKbd = page.locator('.rift-basics button[data-bind="attack"] kbd');

    // kbd is hidden
    await expect(attackKbd).not.toBeVisible();

    // Action labels remain completely visible and clearly legible
    await expect(attackLabel).toBeVisible();
    await expect(attackLabel).toHaveText('Attack');
    await expect(jumpLabel).toBeVisible();
    await expect(jumpLabel).toHaveText('Jump');
    await expect(guardLabel).toBeVisible();
    await expect(guardLabel).toHaveText('Guard');
  });

  test('equipped skills and signature buttons provide clean action labels with aria-keyshortcuts', async ({ page }) => {
    await page.locator('#rift-start').click();

    const skillButtons = page.locator('#rift-skills button');
    await expect(skillButtons.first()).toBeVisible();
    const count = await skillButtons.count();
    expect(count).toBeGreaterThan(0);

    for (let i = 0; i < count; i++) {
      const btn = skillButtons.nth(i);
      const kbd = btn.locator('kbd');
      await expect(kbd).toHaveAttribute('aria-hidden', 'true');
      await expect(btn).toHaveAttribute('aria-keyshortcuts', String(i + 1));
      await expect(btn.locator('.rift-action-label')).toBeVisible();
      // Button with visible text must not have hover tooltip
      await expect(btn).not.toHaveAttribute('title');
    }

    const signatureButtons = page.locator('#rift-signatures button');
    const sigCount = await signatureButtons.count();
    expect(sigCount).toBeGreaterThan(0);

    for (let i = 0; i < sigCount; i++) {
      const btn = signatureButtons.nth(i);
      const kbd = btn.locator('kbd');
      await expect(kbd).toHaveAttribute('aria-hidden', 'true');
      await expect(btn.locator('.rift-action-label')).toBeVisible();
      await expect(btn).not.toHaveAttribute('title');
    }
  });

  test('remapping controls preserves accessible action names while updating shortcuts', async ({ page }) => {
    await page.locator('#rift-controls-open').click();
    const dialog = page.locator('#rift-controls-dialog');
    await expect(dialog).toBeVisible();

    // Select leftHanded preset
    await page.locator('#rift-key-preset').selectOption('leftHanded');
    await page.locator('#rift-apply-keys').click();
    await page.locator('#rift-controls-close').click();

    const attackBtn = page.locator('.rift-basics button[data-bind="attack"]');
    // Keycap updated to A
    await expect(attackBtn.locator('kbd')).toHaveText('A');
    // But accessible action name remains Attack
    await expect(attackBtn).toHaveAttribute('aria-label', 'Attack');
    await expect(attackBtn).toHaveAttribute('aria-keyshortcuts', 'A');
    await expect(attackBtn.locator('.rift-action-label')).toHaveText('Attack');

    // Reset back to defaults
    await page.locator('#rift-controls-open').click();
    await page.locator('#rift-reset-keys').click();
    await page.locator('#rift-controls-close').click();
    await expect(attackBtn.locator('kbd')).toHaveText('J');
    await expect(attackBtn).toHaveAttribute('aria-label', 'Attack');
  });
});
