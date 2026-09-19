const { test, expect } = require('@playwright/test');

test.describe('Document the limits of nonvisual real-time combat support (Proposal 0160)', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/abyss/rift');
    await expect(page.locator('#rift-start')).toBeEnabled();
  });

  test('nonvisual accessibility guide is present in field guide with full landmark structure', async ({ page }) => {
    const fieldGuide = page.locator('#rift-field-guide');
    const accessGuide = fieldGuide.locator('#rift-accessibility-guide');
    const title = accessGuide.locator('#rift-accessibility-guide-title');

    await expect(accessGuide).toHaveAttribute('aria-labelledby', 'rift-accessibility-guide-title');
    await expect(title).toHaveText('Nonvisual combat support & limits');

    // Verify the three distinct structured articles
    const cards = accessGuide.locator('.rift-accessibility-card');
    await expect(cards).toHaveCount(3);

    const headers = accessGuide.locator('.rift-accessibility-card h4');
    await expect(headers.nth(0)).toHaveText('Supported nonvisual & assistive features');
    await expect(headers.nth(1)).toHaveText('Technical limits of real-time combat');
    await expect(headers.nth(2)).toHaveText('Recommended nonvisual strategies');

    // Verify key documented details
    const textContent = await accessGuide.textContent();
    expect(textContent).toContain('Polite live region alerts');
    expect(textContent).toContain('Accessible encounter summaries');
    expect(textContent).toContain('Directional audio & threat cues');
    expect(textContent).toContain('Continuous spatial positioning');
    expect(textContent).toContain('Depth & melee alignment');
    expect(textContent).toContain('Dynamic floor hazards');
    expect(textContent).toContain('Frequent tactical pauses');
    expect(textContent).toContain('Dedicated practice drills');
  });

  test('settings button opens field guide and focuses accessibility section title', async ({ page }) => {
    const settings = page.locator('.rift-settings');
    const openBtn = settings.locator('#rift-open-accessibility-reference');
    const fieldGuide = page.locator('#rift-field-guide');
    const guideTitle = page.locator('#rift-accessibility-guide-title');

    // Buttons with visible text must NOT have hover tooltips
    await expect(openBtn).toHaveText('Nonvisual combat & accessibility');
    await expect(openBtn).not.toHaveAttribute('title');

    // Field guide should initially be closed
    await expect(fieldGuide).not.toHaveAttribute('open', '');

    // Open settings details and click reference button
    await settings.locator('> summary').click();
    await expect(openBtn).toBeVisible();
    await openBtn.click();

    // Field guide must now be open
    await expect(fieldGuide).toHaveAttribute('open', '');

    // The title of the accessibility guide section must have received focus
    await expect(guideTitle).toBeFocused();
  });

  test('nonvisual guide is resilient to 390px mobile viewport without horizontal overflow', async ({ page }) => {
    await page.setViewportSize({ width: 390, height: 844 });
    const fieldGuide = page.locator('#rift-field-guide');
    await fieldGuide.evaluate(el => { el.open = true; });

    const accessGuide = page.locator('#rift-accessibility-guide');
    await expect(accessGuide).toBeVisible();

    const isOverflowing = await page.evaluate(() => {
      const guide = document.getElementById('rift-accessibility-guide');
      return guide.scrollWidth > document.documentElement.clientWidth;
    });
    expect(isOverflowing).toBe(false);
  });

  test('forced-colors mode preserves accessibility card borders', async ({ page }) => {
    await page.emulateMedia({ forcedColors: 'active' });
    const fieldGuide = page.locator('#rift-field-guide');
    await fieldGuide.evaluate(el => { el.open = true; });

    const card = page.locator('.rift-accessibility-card').first();
    await expect(card).toBeVisible();

    const borderStyle = await card.evaluate(el => window.getComputedStyle(el).borderStyle);
    expect(borderStyle).toBe('solid');
  });
});
