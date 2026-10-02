const { test, expect } = require('@playwright/test');

test.describe('Support forced-colors mode for menus and settings (Proposal 0153)', () => {
  test('rift.css stylesheet defines @media (forced-colors: active) rules for menus and settings', async ({ request }) => {
    const response = await request.get('/static/rift.css');
    expect(response.status()).toBe(200);
    const css = await response.text();

    expect(css).toMatch(/@media\s*\(forced-colors:\s*active\)/);
    expect(css).toContain('.rift-settings');
    expect(css).toContain('.rift-controls-dialog');
    expect(css).toContain('CanvasText');
    expect(css).toContain('Highlight');
    expect(css).toContain('ButtonBorder');
  });

  test('menus and settings retain high-contrast borders and system colors under forced-colors: active', async ({ page }) => {
    await page.emulateMedia({ forcedColors: 'active' });
    await page.goto('/abyss/rift');
    await expect(page.locator('#rift-start')).toBeEnabled();

    // Verify window matches forced-colors: active
    const isForcedColors = await page.evaluate(() => window.matchMedia('(forced-colors: active)').matches);
    expect(isForcedColors).toBe(true);

    // Verify Sound & display settings borders and inputs
    const settings = page.locator('.rift-settings');
    await settings.locator('> summary').click();

    const rangeInput = page.locator('#rift-effects-volume');
    await expect(rangeInput).toBeVisible();
    const rangeAccent = await rangeInput.evaluate((el) => window.getComputedStyle(el).accentColor);
    // In forced colors, accent-color computes to Highlight (or resolved system highlight RGB), not the default gold #d4b570 (rgb(212, 181, 112))
    expect(rangeAccent).toBeTruthy();
    expect(rangeAccent).not.toBe('rgb(212, 181, 112)');

    const selectCtrl = page.locator('#rift-transition-delay');
    await expect(selectCtrl).toBeVisible();
    const selectStyles = await selectCtrl.evaluate((el) => {
      const cs = window.getComputedStyle(el);
      return { borderStyle: cs.borderStyle, borderWidth: cs.borderWidth };
    });
    expect(selectStyles.borderStyle).toBe('solid');
    expect(parseInt(selectStyles.borderWidth, 10)).toBeGreaterThanOrEqual(1);

    // Verify buttons inside settings have visible borders
    const resetDisplayBtn = page.locator('#rift-reset-display');
    await expect(resetDisplayBtn).toBeVisible();
    const resetBorder = await resetDisplayBtn.evaluate((el) => window.getComputedStyle(el).borderStyle);
    expect(resetBorder).toBe('solid');
  });

  test('controls dialog surfaces visible system borders and pressed states in forced colors', async ({ page }) => {
    await page.emulateMedia({ forcedColors: 'active' });
    await page.goto('/abyss/rift');
    await expect(page.locator('#rift-start')).toBeEnabled();

    // Open controls dialog
    await page.locator('#rift-controls-open').click();
    const dialog = page.locator('#rift-controls-dialog');
    await expect(dialog).toBeVisible();

    const dialogStyles = await dialog.evaluate((el) => {
      const cs = window.getComputedStyle(el);
      return {
        borderStyle: cs.borderStyle,
        borderWidth: cs.borderWidth,
      };
    });
    expect(dialogStyles.borderStyle).toBe('solid');
    expect(parseInt(dialogStyles.borderWidth, 10)).toBeGreaterThanOrEqual(2);

    // Click a keybind button to trigger capture (aria-pressed="true")
    const attackBindBtn = page.locator('#rift-remap-attack');
    await expect(attackBindBtn).toBeVisible();
    await attackBindBtn.click();
    await expect(attackBindBtn).toHaveAttribute('aria-pressed', 'true');

    // In forced colors, pressed button must receive Highlight styling
    const pressedStyles = await attackBindBtn.evaluate((el) => {
      const cs = window.getComputedStyle(el);
      return {
        background: cs.backgroundColor,
        color: cs.color,
      };
    });
    expect(pressedStyles.background).toBeTruthy();

    // Cancel keybind capture with Escape
    await page.keyboard.press('Escape');
    await expect(attackBindBtn).not.toHaveAttribute('aria-pressed', 'true');

    // Close dialog
    await page.locator('#rift-controls-close').click();
    await expect(dialog).not.toBeVisible();
  });

  test('stage-top toggled buttons and campaign current route markers render with Highlight', async ({ page }) => {
    await page.emulateMedia({ forcedColors: 'active' });
    await page.goto('/abyss/rift');
    await expect(page.locator('#rift-start')).toBeEnabled();

    // Toggle sound button to pressed
    const soundBtn = page.locator('#rift-sound');
    await soundBtn.click();
    await expect(soundBtn).toHaveAttribute('aria-pressed', 'true');

    const soundBorder = await soundBtn.evaluate((el) => window.getComputedStyle(el).borderStyle);
    expect(soundBorder).toBe('solid');

    // Verify current route step indicator
    const currentStep = page.locator('.rift-route li.current > span');
    await expect(currentStep).toBeVisible();
    const stepBorder = await currentStep.evaluate((el) => window.getComputedStyle(el).borderStyle);
    expect(stepBorder).toBe('solid');
  });

  test('bestiary filter inputs and comparison table have distinct high-contrast borders', async ({ page }) => {
    await page.emulateMedia({ forcedColors: 'active' });
    await page.goto('/abyss/rift');
    await expect(page.locator('#rift-start')).toBeEnabled();

    // Open bestiary accordion
    const bestiary = page.locator('.rift-bestiary');
    await bestiary.locator('> summary').click();

    const searchInput = page.locator('#rift-monster-search');
    await expect(searchInput).toBeVisible();
    const searchBorder = await searchInput.evaluate((el) => window.getComputedStyle(el).borderStyle);
    expect(searchBorder).toBe('solid');

    // Open creature comparison
    const compare = page.locator('#rift-compare');
    await compare.locator('> summary').click();
    await expect(page.locator('#rift-compare-left')).toBeVisible();
    const leftSelectBorder = await page.locator('#rift-compare-left').evaluate((el) => window.getComputedStyle(el).borderStyle);
    expect(leftSelectBorder).toBe('solid');
  });
});
