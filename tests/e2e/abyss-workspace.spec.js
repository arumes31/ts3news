const { test, expect } = require('@playwright/test');

const sections = ['season', 'progression', 'shop', 'forge', 'social', 'lore', 'leaderboards'];

test('workspace navigation reaches every Abyss section without losing context', async ({ page }) => {
  const pageErrors = [];
  page.on('pageerror', error => pageErrors.push(error.message));
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto('/abyss');
  await page.emulateMedia({ reducedMotion: 'reduce' });

  const rail = page.locator('.ab-tabs');
  await expect(rail).toHaveAttribute('role', 'tablist');
  await expect(rail.locator('[data-tab-key]')).toHaveCount(sections.length);

  for (const key of sections) {
    const tab = rail.locator(`[data-tab-key="${key}"]`);
    const panels = page.locator(`[data-abyss-section="${key}"]`);
    await expect(tab).toBeVisible();
    await expect(panels).not.toHaveCount(0);
    await tab.click();
    await expect(tab).toHaveAttribute('aria-selected', 'true');
    await expect(tab).toHaveAttribute('tabindex', '0');
    await expect(panels.first()).toBeVisible();
    await expect.poll(() => page.evaluate(activeKey => {
      return Array.from(document.querySelectorAll('[data-abyss-section]')).every(panel =>
        panel.dataset.abyssSection === activeKey ? !panel.hidden : panel.hidden);
    }, key)).toBe(true);
  }

  await expect.poll(() => rail.evaluate(node => node.scrollLeft)).toBeGreaterThan(0);
  expect(await rail.evaluate(node => node.scrollWidth > node.clientWidth)).toBe(true);
  expect(await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth)).toBeLessThanOrEqual(1);

  const leaderboards = rail.locator('[data-tab-key="leaderboards"]');
  await leaderboards.focus();
  await leaderboards.press('Home');
  await expect(rail.locator('[data-tab-key="season"]')).toHaveAttribute('aria-selected', 'true');
  await rail.locator('[data-tab-key="season"]').press('End');
  await expect(leaderboards).toHaveAttribute('aria-selected', 'true');

  await rail.locator('[data-tab-key="social"]').click();
  await page.goto('/abyss');
  await expect(page.locator('[data-tab-key="social"]')).toHaveAttribute('aria-selected', 'true');
  await expect(page.locator('#abyssSocialHub')).toBeVisible();

  await page.goto('/abyss#lore');
  await expect(page.locator('[data-tab-key="lore"]')).toHaveAttribute('aria-selected', 'true');
  const search = page.locator('#abyssLibrarySearch');
  await search.fill('definitely-no-match');
  await expect(page.locator('.ab-library-results')).toHaveText('0 matches');
  await search.fill('');

  await page.goto('/abyss#lb');
  await expect(page.locator('[data-tab-key="leaderboards"]')).toHaveAttribute('aria-selected', 'true');
  await expect(page.locator('#lb')).toBeVisible();
  expect(pageErrors).toEqual([]);
});
