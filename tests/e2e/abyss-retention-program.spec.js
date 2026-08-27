const { test, expect } = require('@playwright/test');

for (const viewport of [
  { name: 'desktop', width: 1440, height: 1000 },
  { name: 'mobile', width: 390, height: 844 },
]) {
  test(`season retention workspace remains coherent on ${viewport.name}`, async ({ page }) => {
    const pageErrors = [];
    const claims = [];
    page.on('pageerror', error => pageErrors.push(error.message));
    await page.setViewportSize(viewport);
    await page.route('**/api/abyss/retention/**', async route => {
      const path = new URL(route.request().url()).pathname;
      const body = route.request().postDataJSON() || {};
      claims.push({ path, body });
      if (path.endsWith('/login')) {
        await route.fulfill({
          contentType: 'application/json',
          body: JSON.stringify({ ok: true, gold: 124456, tokens: 43, reward_gold: 1000, reward_tokens: 0 }),
        });
        return;
      }
      await route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({ ok: true, depth: body.depth, name: 'Endless Banner · Rank 1', already_owned: false }),
      });
    });
    await page.goto('/abyss?endless=1');
    await page.locator('[data-tab-key="season"]').click();

    await expect(page.locator('#abyssSeasonJourney')).toBeVisible();
    const journal = page.locator('#abyssSeasonJournal');
    await expect(journal).toBeVisible();
    await expect(journal.locator('[data-season-objective]')).toHaveCount(50);
    await journal.locator('summary').click();
    await expect(journal.locator('.ab-season-finale')).toContainText('Chronicle Mantle');
    await expect(journal.getByRole('button', { name: 'Complete all 50 objectives' })).toBeDisabled();

    const retention = page.locator('#abyssRetentionWorkspace');
    await expect(retention).toBeVisible();
    await expect(retention.locator('.ab-login-days li')).toHaveCount(28);
    await expect(retention.locator('#abyssWeeklyDigest')).toContainText('1 floor from your record');
    await expect(retention.locator('#abyssEndlessMode')).toContainText('Combat power · +0%');

    const login = retention.locator('#abyssLoginClaim');
    await login.click();
    await expect(login).toHaveText('✓ Claimed');
    await expect(retention.locator('.ab-login-days .current')).toHaveClass(/claimed/);
    await expect.poll(() => claims.some(claim => claim.path.endsWith('/login'))).toBe(true);

    const endless = retention.locator('[data-endless-depth="125"]');
    await endless.getByRole('button', { name: 'Claim cosmetic' }).click();
    await expect(endless).toHaveClass(/owned/);
    await expect(endless.getByRole('button')).toHaveText('✓ Owned');
    await expect.poll(() => claims.some(claim => claim.path.endsWith('/endless') && claim.body.depth === 125)).toBe(true);

    expect(await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth)).toBeLessThanOrEqual(1);
    expect(pageErrors).toEqual([]);
  });
}
