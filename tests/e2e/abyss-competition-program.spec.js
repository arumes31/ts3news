const { test, expect } = require('@playwright/test');

for (const viewport of [
  { name: 'desktop', width: 1440, height: 960 },
  { name: 'mobile', width: 390, height: 844 },
]) {
  test(`Abyss competition workspace is navigable on ${viewport.name}`, async ({ page }) => {
    const pageErrors = [];
    page.on('pageerror', error => pageErrors.push(error.message));
    await page.setViewportSize(viewport);
    await page.goto('/abyss');
    await page.locator('[data-tab-key="leaderboards"]').click();

    const competition = page.locator('#lb.ab-competition');
    await expect(competition).toBeVisible();
    await expect(competition.getByRole('heading', { name: '🏆 Abyss Competition' })).toBeVisible();
    await expect(competition.locator('select[name="lbtier"]')).toBeVisible();
    await expect(competition.locator('select[name="lbbuild"]')).toBeVisible();
    await expect(competition.locator('select[name="lbperiod"]')).toBeVisible();

    for (const key of ['depth', 'speed', 'economy', 'pact', 'bestiary', 'shame', 'streak', 'pets', 'bosses', 'wagers']) {
      const tab = competition.locator(`[data-comp-board="${key}"]`);
      await expect(tab).toBeVisible();
      await tab.click();
      await expect(competition.locator(`[data-comp-panel="${key}"]`)).toBeVisible();
      await expect(tab).toHaveAttribute('aria-selected', 'true');
    }

    await expect(competition.getByText('Past season snapshots')).toBeVisible();
    await expect(competition.getByText('Audit trail')).toBeVisible();
    await expect(competition.getByText('Personal depth · last 30 days')).toBeVisible();
    await expect(competition.locator('[data-shame-opt]')).toBeVisible();
    expect(await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth)).toBeLessThanOrEqual(1);
    expect(pageErrors).toEqual([]);
  });
}
