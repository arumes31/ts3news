const { test, expect } = require('@playwright/test');

test('shows loading-progress count for critical atlases', async ({ page }) => {
  await page.goto('/abyss/rift');
  await expect(page.locator('#rift-start')).toBeEnabled();

  const progress = page.locator('#rift-atlas-progress');
  await expect(progress).toBeVisible();
  await expect(progress).toHaveAttribute('role', 'status');
  await expect(progress).toHaveAttribute('aria-live', 'polite');

  // Verify critical atlas loading progress indicators
  await expect(progress).toHaveAttribute('data-total', '8');
  await expect(progress).toHaveAttribute('data-loaded', '8');
  await expect(progress).toHaveText('Critical atlases loaded (8/8)');

  // Verify exposed RiftRenderer properties
  const rendererData = await page.evaluate(() => {
    const r = window.RiftRenderer;
    return {
      atlasProgress: r ? r.atlasProgress : null,
      criticalKeys: r && typeof r.getCriticalAtlasKeys === 'function' ? r.getCriticalAtlasKeys() : null,
    };
  });

  expect(rendererData.atlasProgress).toEqual({
    loaded: 8,
    total: 8,
    ready: true,
  });

  expect(rendererData.criticalKeys).toEqual([
    'area',
    'regions',
    'props',
    'mobs',
    'items',
    'effects',
    'terrainCover',
    'platformSurface',
  ]);

  // Starting an expedition hides the overlay (and its progress indicator)
  await page.locator('#rift-start').click();
  await expect(page.locator('#rift-overlay')).toBeHidden();
  await expect(progress).toBeHidden();
});
