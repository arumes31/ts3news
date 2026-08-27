const { test, expect } = require('@playwright/test');

for (const viewport of [
  { name: 'desktop', width: 1440, height: 1000 },
  { name: 'mobile', width: 390, height: 844 },
]) {
  test(`layout readability controls stay usable on ${viewport.name}`, async ({ page }) => {
    const pageErrors = [];
    page.on('pageerror', error => pageErrors.push(error.message));
    await page.setViewportSize(viewport);
    await page.goto('/abyss?gear=1');

    await page.locator('#abyss-tab-forge').click();
    const search = page.locator('#forgeItemSearch');
    await expect(search).toBeVisible();
    await search.fill('Tideglass');
    const result = page.getByRole('button', { name: 'Select Tideglass Focus for forging' });
    await expect(result).toHaveCount(1);
    await result.scrollIntoViewIfNeeded();
    await result.focus();
    await result.press('Enter');
    await expect(page.locator('#forgeItemSelect')).toHaveValue(/equipped:/);
    await expect(page.locator('#forgePickerCount')).toHaveText('1 item shown');

    await page.locator('#abSettingsBtn').click();
    await page.locator('#abSettingsSearch').fill('Run loot row density');
    const density = page.locator('#abSettingsRows .ab-set-row').filter({ hasText: 'Run loot row density' });
    await expect(density).toBeVisible();
    await density.locator('select').selectOption('compact');
    await expect(page.locator('body')).toHaveClass(/ab-loot-density-compact/);
    await page.locator('#modalOkBtn').click();

    await page.evaluate(() => {
      window.renderConsumables([{ ID: 'small_health_potion', Name: 'Small Health Potion', Type: 'Healing', Duration: 3, Description: 'Restore health.' }]);
      window.__abyssLayoutReadability.syncPrepConsumables({ options: [{ kind: 'item', id: 'small_health_potion', count: 2, cooldown: 2 }] });
    });
    await expect(page.locator('#cons-small_health_potion .ab-cons-state')).toHaveText('×2 · 2 rounds cooldown');
    await expect(page.locator('#cons-small_health_potion button')).toBeDisabled();
    await page.evaluate(() => window.__abyssLayoutReadability.syncPrepConsumables({ options: [{ kind: 'item', id: 'small_health_potion', count: 2, cooldown: 0 }] }));
    await expect(page.locator('#cons-small_health_potion .ab-cons-state')).toHaveText('×2 · ready');
    await expect(page.locator('#cons-small_health_potion button')).toBeEnabled();

    await page.reload();
    await expect(page.locator('body')).toHaveClass(/ab-loot-density-compact/);
    expect(await page.evaluate(() => localStorage.getItem('ab_loot_density'))).toBe('compact');
    expect(pageErrors).toEqual([]);
  });
}
