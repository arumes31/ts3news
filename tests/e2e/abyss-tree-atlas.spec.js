const { test, expect } = require('@playwright/test');

test('every live Skill Web node uses its unique discipline atlas icon', async ({ page }) => {
  const pageErrors = [];
  const legacyIconRequests = [];
  page.on('pageerror', error => pageErrors.push(error.message));
  page.on('request', request => {
    if (request.url().includes('/static/icons/')) legacyIconRequests.push(request.url());
  });
  await page.emulateMedia({ reducedMotion: 'reduce' });
  await page.goto('/abyss/tree');

  const nodeCount = await page.evaluate(() => NODES.length);
  expect(nodeCount).toBeGreaterThan(5000);
  await expect(page.locator('.tn-pixel-icon')).toHaveCount(nodeCount);

  const art = await page.locator('.tn-pixel-icon').evaluateAll(icons => icons.map(icon => ({
    signature: icon.dataset.artSignature,
    sheet: icon.dataset.artSheet,
    cell: Number(icon.dataset.artCell),
    href: icon.querySelector('image').getAttribute('href'),
  })));
  expect(new Set(art.map(icon => icon.signature)).size).toBe(nodeCount);
  expect(new Set(art.map(icon => `${icon.sheet}:${icon.cell}`)).size).toBe(nodeCount);
  expect([...new Set(art.map(icon => icon.sheet))].sort()).toEqual([
    'tree_arcane', 'tree_fortune', 'tree_shadow', 'tree_vitality', 'tree_void', 'tree_war',
  ]);
  expect(legacyIconRequests).toEqual([]);

  const atlasURLs = [...new Set(art.map(icon => icon.href))];
  expect(atlasURLs).toHaveLength(6);
  for (const url of atlasURLs) {
    const response = await page.request.get(url);
    expect(response.status(), url).toBe(200);
    expect(response.headers()['content-type']).toContain('image/png');
  }

  const search = page.locator('#treeSearch');
  await search.fill('Limit Break');
  await expect(page.locator('.tn-pixel-icon.hit')).toHaveCount(1);
  expect(await page.locator('.tn-pixel-icon.tree-nav-hidden').count()).toBe(nodeCount - 1);
  await search.fill('');

  await page.locator('#treeContrastToggle').click();
  const highContrastFilter = await page.locator('.tn-pixel-icon').first().evaluate(icon => getComputedStyle(icon).filter);
  expect(highContrastFilter).not.toBe('none');

  await page.locator('#treeCanvasBtn').click();
  await expect(page.locator('#treeCanvas')).toBeVisible();
  await expect.poll(() => page.locator('#treeCanvas').evaluate(canvas => {
    const context = canvas.getContext('2d');
    const width = Math.min(canvas.width, 400);
    const height = Math.min(canvas.height, 300);
    const x = Math.max(0, Math.floor((canvas.width - width) / 2));
    const y = Math.max(0, Math.floor((canvas.height - height) / 2));
    return context.getImageData(x, y, width, height).data.some((value, index) => index % 4 === 3 && value > 0);
  })).toBe(true);

  await page.setViewportSize({ width: 390, height: 844 });
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth + 1)).toBe(true);
  expect(pageErrors).toEqual([]);
});
