const { test } = require('@playwright/test');
const {
  expectDocumentStructure,
  expectSurfaceMonitorsClean,
  expectVisibleSurface,
  monitorSurface,
} = require('./helpers/abyss-surface');

const session = '0123456789abcdef0123456789abcdef';
const surfaces = [
  { name: 'Skill Web', path: '/abyss/tree', ready: '#treeStage' },
  { name: 'Hall of Delvers', path: '/abyss/plaza', ready: '.plaza-shell' },
  { name: 'Operations', path: '/abyss/ops', ready: '.ops-shell' },
  { name: 'Spectator', path: `/abyss/spectate?session=${session}`, ready: '.ab-spectator' },
];
const viewports = [
  { name: 'desktop', width: 1440, height: 900 },
  { name: 'mobile', width: 390, height: 844 },
];

for (const viewport of viewports) {
  test(`dedicated Abyss surfaces remain valid on ${viewport.name}`, async ({ page }) => {
    await page.setViewportSize({ width: viewport.width, height: viewport.height });
    await page.emulateMedia({ reducedMotion: 'reduce', colorScheme: 'light' });

    for (const surface of surfaces) {
      const monitors = monitorSurface(page);
      await page.goto(surface.path);
      await page.locator(surface.ready).waitFor({ state: 'visible' });
      await expectDocumentStructure(page);
      await expectVisibleSurface(page, 'body', surface.name);
      expectSurfaceMonitorsClean(monitors);
    }
  });
}
