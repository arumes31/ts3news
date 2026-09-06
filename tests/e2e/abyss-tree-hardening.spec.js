const { test, expect } = require('@playwright/test');

test.beforeEach(async ({ page }) => {
  await page.emulateMedia({ reducedMotion: 'reduce' });
});

test('invalid saved discovery lists recover without breaking search or bookmarks', async ({ page }) => {
  const errors = [];
  page.on('pageerror', error => errors.push(error.message));
  await page.addInitScript(() => {
    localStorage.setItem('abyssTreeRecentNodes', '{}');
    localStorage.setItem('abyssTreeBookmarks', 'null');
  });
  await page.goto('/abyss/tree');
  expect.soft(errors, 'Malformed saved lists must not throw during initialization').toEqual([]);
  await page.locator('#treeSearch').fill('Limit Break');
  await expect(page.locator('#treeResultCount')).toHaveText('1 result');
  await page.locator('#treeDiscoveryDetails > summary').click();
  await page.locator('#treeResultList button').click();
  await expect(page.locator('#treeInspectorTitle')).toContainText('Limit Break');
  await page.locator('#treeBookmark').click();
  await expect(page.locator('#treeBookmarks button')).toContainText('Limit Break');
  expect(errors).toEqual([]);
});

test('long keystone effects keep their own space before desktop metadata', async ({ page }) => {
  await page.setViewportSize({ width: 1440, height: 900 });
  await page.goto('/abyss/tree?node=4969');
  await expect(page.locator('#treeInspectorTitle')).toContainText('Limit Break');
  const bounds = await page.locator('#treeInspectorEffects').evaluate(effects => {
    const range = document.createRange();
    range.selectNodeContents(effects);
    const text = range.getBoundingClientRect();
    const card = effects.getBoundingClientRect();
    const metadata = document.getElementById('treeInspectorGrid').getBoundingClientRect();
    return { height: card.height, contentHeight: text.height, contentBottom: text.bottom, metadataTop: metadata.top };
  });
  expect(bounds.height, 'The effects block must reserve room for its content').toBeGreaterThanOrEqual(bounds.contentHeight - 1);
  expect(bounds.contentBottom, 'Effects must end before the metadata starts').toBeLessThanOrEqual(bounds.metadataTop + 1);
});

test('path tooltip quotes the same points as the explicit inspector action', async ({ page }) => {
  const writes = [];
  await page.route('**/api/abyss/tree/allocate', async route => {
    writes.push(route.request().postDataJSON());
    await route.fulfill({ json: { ok: false, error: 'Unexpected allocation captured by test.' } });
  });
  await page.setViewportSize({ width: 1440, height: 900 });
  await page.goto('/abyss/tree?node=4969');
  await expect(page.locator('#treeInspectorTitle')).toContainText('Limit Break');
  const actionText = await page.locator('#treeInspectorAllocate').textContent();
  const points = actionText.match(/(\d+)\s+points?/i);
  expect(points, 'Limit Break must expose an allocation path price').not.toBeNull();
  // Nearby cross-sector nodes overlap at this zoom; discovery isolates the target.
  await page.locator('#treeSearch').fill('Limit Break');
  await page.locator('#treeSvg .tn[data-id="4969"]').hover();
  const tooltip = page.locator('#treeTip');
  await expect(tooltip).toBeVisible();
  await expect(tooltip).toContainText(new RegExp(`\\b${points[1]}\\s+(?:points?|pts?)\\b`, 'i'));
  await expect(tooltip).toContainText(/inspect/i);
  await expect(tooltip).not.toContainText(/click to auto-allocate/i);
  await page.locator('#treeInspectorAllocate').click();
  await expect(page.locator('#sharedModalCard')).toContainText(new RegExp(`\\b${points[1]}\\s+points?\\b`, 'i'));
  expect(writes, 'Reviewing the quote must not allocate any nodes').toEqual([]);
  await page.locator('#modalCancelBtn').click();
  await expect(page.locator('#sharedModal')).not.toHaveClass(/open/);
  expect(writes, 'Cancelling the quote must not allocate any nodes').toEqual([]);
});

test('Canvas redraws its pixels when high contrast changes and restores its original palette', async ({ page }) => {
  await page.goto('/abyss/tree');
  await page.locator('#treeSavedBuildsDetails > summary').click();
  await page.locator('#treeCanvasBtn').click();
  const canvas = page.locator('#treeCanvas');
  await expect(canvas).toBeVisible();
  // Background requests need not settle; only the Canvas atlas images must be ready.
  await page.waitForFunction(() => Object.values(treeAtlasImages).every(image => image.complete && image.naturalWidth > 0));
  const pixels = () => canvas.evaluate(element => {
    const data = element.getContext('2d').getImageData(0, 0, element.width, element.height).data;
    let hash = 2166136261;
    // Sample RGB across the bitmap instead of hashing millions of channel values.
    for (let index = 0; index < data.length; index += 64) {
      hash = Math.imul(hash ^ data[index], 16777619);
      hash = Math.imul(hash ^ data[index + 1], 16777619);
      hash = Math.imul(hash ^ data[index + 2], 16777619);
    }
    return hash >>> 0;
  });
  const standard = await pixels();
  await page.locator('#treeContrastToggle').click();
  await expect(page.locator('#treeContrastToggle')).toHaveAttribute('aria-pressed', 'true');
  await expect.poll(pixels, { timeout: 15000, message: 'The Canvas bitmap itself must reflect high contrast' }).not.toBe(standard);
  await page.locator('#treeContrastToggle').click();
  await expect.poll(pixels, { timeout: 15000, message: 'Disabling high contrast restores the stationary Canvas view' }).toBe(standard);
});

test.describe('mobile cost filters', () => {
  test.use({ viewport: { width: 390, height: 844 }, isMobile: true, hasTouch: true });

  test('numeric costs have visible labels and touch-sized inputs', async ({ page }) => {
    await page.goto('/abyss/tree');
    await page.locator('#treeDiscoveryDetails > summary').tap();
    for (const id of ['treeMinCost', 'treeMaxCost']) {
      const input = page.locator(`#${id}`);
      await expect(input).toBeVisible();
      const bounds = await input.boundingBox();
      expect.soft(bounds.height, `${id} must meet the project 44px touch target`).toBeGreaterThanOrEqual(44);
      const labels = await input.evaluate(element => Array.from(element.labels || []).map(label => {
        const box = label.getBoundingClientRect();
        const style = getComputedStyle(label);
        return { text: label.textContent.trim(), visible: box.width > 1 && box.height > 1 && style.visibility !== 'hidden' && style.display !== 'none' };
      }));
      expect.soft(labels.some(label => label.visible && /cost|points?/i.test(label.text)), `${id} needs a persistent visible cost label`).toBe(true);
    }
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth + 1)).toBe(true);
  });
});
