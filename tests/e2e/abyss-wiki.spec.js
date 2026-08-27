const { test, expect } = require('@playwright/test');

for (const viewport of [
  { name: 'desktop', width: 1440, height: 960 },
  { name: 'mobile', width: 390, height: 844 },
]) {
  test(`generated Abyss field archive is searchable on ${viewport.name}`, async ({ page }) => {
    const pageErrors = [];
    page.on('pageerror', error => pageErrors.push(error.message));
    await page.setViewportSize(viewport);
    await page.goto('/abyss');
    await page.locator('[data-tab-key="lore"]').click();

    const wiki = page.locator('#abyssWiki');
    await expect(wiki).toBeVisible();
    await expect(wiki.getByRole('heading', { name: /Delver's Field Archive/ })).toBeVisible();
    await expect(wiki.locator('[data-wiki-tab]')).toHaveCount(4);

    const search = wiki.locator('#abyssWikiSearch');
    await search.fill('ABYSS_OFFHAND');
    await expect(wiki.locator('[data-wiki-panel="gear"] .ab-wiki-entry:visible')).toHaveCount(1);

    await wiki.locator('[data-wiki-tab="affixes"]').click();
    await search.fill('bloodlust');
    const bloodlust = wiki.locator('[data-wiki-panel="affixes"] .ab-wiki-entry:visible');
    await expect(bloodlust).toHaveCount(1);
    await expect(bloodlust).toContainText('Daily affix');
    await expect(wiki.locator('#abyssWikiStatus')).toContainText('1 affixes record');

    expect(await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth)).toBeLessThanOrEqual(1);
    expect(pageErrors).toEqual([]);
  });
}
