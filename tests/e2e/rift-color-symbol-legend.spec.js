const { test, expect } = require('@playwright/test');

test.describe('Provide an explanation of color and symbol legends (Proposal 0155)', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/abyss/rift');
    await expect(page.locator('#rift-start')).toBeEnabled();
  });

  test('field guide includes comprehensive color and symbol legends section', async ({ page }) => {
    const fieldGuide = page.locator('#rift-field-guide');
    await fieldGuide.locator('> summary').click();

    const legendSection = page.locator('#rift-legend-guide');
    await expect(legendSection).toBeVisible();

    // Check heading
    const title = legendSection.locator('h3');
    await expect(title).toHaveText('Color and symbol legends');

    // Verify 4 categories are present
    const cards = legendSection.locator('.rift-legend-card');
    await expect(cards).toHaveCount(4);

    // 1. Combat floating text
    const combatCard = cards.nth(0);
    await expect(combatCard.locator('h4')).toHaveText(/Combat floating text/i);
    await expect(combatCard).toContainText('Guarded hit');
    await expect(combatCard).toContainText('Damage dealt');
    await expect(combatCard).toContainText('Damage taken');
    await expect(combatCard).toContainText('Healing');
    await expect(combatCard).toContainText('Barrier');
    await expect(combatCard).toContainText('Pickup confirmed');

    // 2. Health thresholds
    const healthCard = cards.nth(1);
    await expect(healthCard.locator('h4')).toHaveText(/Health thresholds/i);
    await expect(healthCard).toContainText('Healthy');
    await expect(healthCard).toContainText('Wounded');
    await expect(healthCard).toContainText('Critical');

    // 3. Battlefield shapes & hazards
    const shapesCard = cards.nth(2);
    await expect(shapesCard.locator('h4')).toHaveText(/Battlefield shapes/i);
    await expect(shapesCard).toContainText('Hostile projectile');
    await expect(shapesCard).toContainText('Friendly projectile');
    await expect(shapesCard).toContainText('Hazard area');

    // 4. Navigation, risk & loot
    const navCard = cards.nth(3);
    await expect(navCard.locator('h4')).toHaveText(/Navigation, risk & loot/i);
    await expect(navCard).toContainText('AT RISK');
    await expect(navCard).toContainText('Favorite mission');
    await expect(navCard).toContainText('Cleared mission');
    await expect(navCard).toContainText('Directional cues');
  });

  test('sound & display settings has button to open and scroll directly to color and symbol legend', async ({ page }) => {
    const settings = page.locator('.rift-settings');
    await settings.locator('> summary').click();

    const openLegendBtn = page.locator('#rift-open-legend-reference');
    await expect(openLegendBtn).toBeVisible();
    await expect(openLegendBtn).toHaveText('Color & symbol legend');
    // Button with visible text must NOT have redundant hover tooltip
    expect(await openLegendBtn.getAttribute('title')).toBeNull();

    // Field guide starts closed
    const fieldGuide = page.locator('#rift-field-guide');
    await expect(fieldGuide).not.toHaveAttribute('open');

    // Clicking button opens field guide and focuses legend title
    await openLegendBtn.click();
    await expect(fieldGuide).toHaveAttribute('open');
    const legendSection = page.locator('#rift-legend-guide');
    await expect(legendSection).toBeVisible();

    const title = legendSection.locator('h3');
    await expect(title).toBeFocused();
  });

  test('legend grid wraps cleanly on mobile viewports without horizontal scroll overflow', async ({ page }) => {
    await page.setViewportSize({ width: 390, height: 844 });
    const fieldGuide = page.locator('#rift-field-guide');
    await fieldGuide.locator('> summary').click();

    const legendSection = page.locator('#rift-legend-guide');
    await expect(legendSection).toBeVisible();

    const sample = legendSection.locator('.rift-legend-sample').first();
    await expect(sample).toBeVisible();

    const overflows = await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth + 2);
    expect(overflows).toBe(true);
  });

  test('color and symbol legends render high-contrast borders in forced-colors mode', async ({ page }) => {
    await page.emulateMedia({ forcedColors: 'active' });
    const fieldGuide = page.locator('#rift-field-guide');
    await fieldGuide.locator('> summary').click();

    const card = page.locator('.rift-legend-card').first();
    await expect(card).toBeVisible();

    const borderStyle = await card.evaluate((el) => window.getComputedStyle(el).borderStyle);
    expect(borderStyle).toBe('solid');

    const sample = page.locator('.rift-legend-sample').first();
    const sampleBorder = await sample.evaluate((el) => window.getComputedStyle(el).borderStyle);
    expect(sampleBorder).toBe('solid');
  });
});
