const { test, expect } = require('@playwright/test');
const {
  expectDocumentStructure,
  expectSurfaceMonitorsClean,
  expectVisibleSurface,
  monitorSurface,
} = require('./helpers/abyss-surface');

const sections = ['season', 'progression', 'observatory', 'shop', 'forge', 'social', 'lore', 'leaderboards'];
const viewports = [
  { name: 'desktop', width: 1440, height: 900 },
  { name: 'mobile', width: 390, height: 844 },
];

for (const viewport of viewports) {
  test(`every Abyss workspace has a valid rendered surface on ${viewport.name}`, async ({ page }) => {
    const monitors = monitorSurface(page);
    await page.setViewportSize({ width: viewport.width, height: viewport.height });
    await page.emulateMedia({ reducedMotion: 'reduce', colorScheme: 'light' });
    await page.goto('/abyss?gear=1');

    await expectDocumentStructure(page);

    for (const key of sections) {
      const tab = page.locator(`[data-tab-key="${key}"]`);
      await tab.click();
      await expect(tab).toHaveAttribute('aria-selected', 'true');
      await expectVisibleSurface(page, `[data-abyss-section="${key}"]`, key);
    }

    expectSurfaceMonitorsClean(monitors);
  });
}
