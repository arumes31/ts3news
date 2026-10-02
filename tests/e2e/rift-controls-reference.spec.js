const { test, expect } = require('@playwright/test');

test.describe('Plain-text combat-controls reference (Proposal 0143)', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/abyss/rift');
    await expect(page.locator('#rift-start')).toBeEnabled();
  });

  test('provides a structured plain-text combat controls reference inside controls dialog', async ({ page }) => {
    await page.locator('#rift-controls-open').click();
    const dialog = page.locator('#rift-controls-dialog');
    await expect(dialog).toBeVisible();

    const refDetails = page.locator('#rift-controls-reference');
    await expect(refDetails).toBeAttached();
    // Collapsed by default when opened via standard controls button
    expect(await refDetails.getAttribute('open')).toBeNull();

    // Expand reference details
    await refDetails.locator('> summary').click();
    await expect(refDetails).toHaveAttribute('open', '');

    const refText = page.locator('#rift-controls-reference-text');
    await expect(refText).toBeVisible();
    const content = await refText.textContent();

    expect(content).toContain('RIFT BRAWL COMBAT CONTROLS REFERENCE');
    expect(content).toContain('Movement:');
    expect(content).toContain('Move up:            W / ArrowUp');
    expect(content).toContain('Move left:          A / ArrowLeft');
    expect(content).toContain('Move down:          S / ArrowDown');
    expect(content).toContain('Move right:         D / ArrowRight');

    expect(content).toContain('Combat Actions:');
    expect(content).toContain('Attack:             J');
    expect(content).toContain('Jump:               Space / K');
    expect(content).toContain('Guard:              L');
    expect(content).toContain('Class builder:      Q');
    expect(content).toContain('Class finisher:     E');
    expect(content).toContain('Equipped skill 1:   1');
    expect(content).toContain('Equipped skill 2:   2');
    expect(content).toContain('Equipped skill 3:   3');
    expect(content).toContain('Ultimate:           R');
    expect(content).toContain('Pause / resume:     Escape');

    expect(content).toContain('Combat Shortcuts:');
    expect(content).toContain('Clean screenshot:   F4 or Alt+Shift+H');
    expect(content).toContain('Skill range preview: Alt+Shift+R');
    expect(content).toContain('Pin skill range:    Shift+1 / Shift+2 / Shift+3');
    expect(content).toContain('Loadout reference:  Alt+Shift+L');
    expect(content).toContain('Quick pause:        Escape');
  });

  test('reference dynamically updates on layout change, mouse rebind, and guard toggle', async ({ page }) => {
    await page.locator('#rift-controls-open').click();
    const refDetails = page.locator('#rift-controls-reference');
    await refDetails.locator('> summary').click();
    const refText = page.locator('#rift-controls-reference-text');

    // 1. Change layout preset to leftHanded
    await page.locator('#rift-key-preset').selectOption('leftHanded');
    await page.locator('#rift-apply-keys').click();

    let content = await refText.textContent();
    expect(content).toContain('Move up:            ArrowUp');
    expect(content).toContain('Attack:             A');
    expect(content).toContain('Guard:              S');
    expect(content).toContain('Class finisher:     W');
    expect(content).toContain('Ultimate:           E');

    // 2. Assign mouse attack
    await page.locator('#rift-mouse-attack').selectOption('0');
    content = await refText.textContent();
    expect(content).toContain('Attack:             A (Mouse: Left button)');

    // 3. Toggle guard mode checkbox
    const toggleGuard = page.locator('#rift-toggle-guard');
    await toggleGuard.check();
    content = await refText.textContent();
    expect(content).toContain('Toggle mode');

    // 4. Reset all keys restores defaults
    await page.locator('#rift-reset-keys').click();
    content = await refText.textContent();
    expect(content).toContain('Move up:            W / ArrowUp');
    expect(content).toContain('Attack:             J');
    expect(content).toContain('Guard:              L');
  });

  test('supports copying the plain-text reference with accessible status', async ({ page }) => {
    await page.locator('#rift-controls-open').click();
    await page.locator('#rift-controls-reference > summary').click();

    const copyBtn = page.locator('#rift-copy-controls-reference');
    const copyStatus = page.locator('#rift-copy-controls-status');
    await expect(copyBtn).toBeVisible();

    await copyBtn.click();
    await expect(copyStatus).toHaveText('Controls reference copied to clipboard.');
  });

  test('can open plain-text reference directly from Settings and returns focus on close', async ({ page }) => {
    const settings = page.locator('.rift-settings');
    await settings.locator('> summary').click();

    const openRefBtn = page.locator('#rift-open-controls-reference');
    await expect(openRefBtn).toBeVisible();
    await openRefBtn.click();

    const dialog = page.locator('#rift-controls-dialog');
    await expect(dialog).toBeVisible();

    // The reference details is automatically expanded
    const refDetails = page.locator('#rift-controls-reference');
    await expect(refDetails).toHaveAttribute('open', '');

    // Focus is placed on the plain-text reference text
    await expect(page.locator('#rift-controls-reference-text')).toBeFocused();

    // Close the dialog via close button
    await page.locator('#rift-controls-close').click();
    await expect(dialog).toBeHidden();

    // Focus returns directly to the opener (#rift-open-controls-reference)
    await expect(openRefBtn).toBeFocused();
  });

  test('focus is trapped within controls dialog when reference is open', async ({ page }) => {
    await page.locator('#rift-controls-open').click();
    await page.locator('#rift-controls-reference > summary').click();

    // Focus the last element in footer
    const closeBtn = page.locator('#rift-controls-close');
    await closeBtn.focus();
    await expect(closeBtn).toBeFocused();

    // Tab wraps to the first focusable element in dialog
    await page.keyboard.press('Tab');
    expect(await page.evaluate(() => document.getElementById('rift-controls-dialog').contains(document.activeElement))).toBe(true);

    // Shift-Tab wraps back to close button
    await page.keyboard.down('Shift');
    await page.keyboard.press('Tab');
    await page.keyboard.up('Shift');
    await expect(closeBtn).toBeFocused();
  });
});
