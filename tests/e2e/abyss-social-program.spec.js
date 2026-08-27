const { test, expect } = require('@playwright/test');

for (const viewport of [
  { name: 'desktop', width: 1440, height: 960 },
  { name: 'mobile', width: 390, height: 844 },
]) {
  test(`Fellowship social program is usable on ${viewport.name}`, async ({ page }) => {
    const pageErrors = [];
    page.on('pageerror', error => pageErrors.push(error.message));
    await page.setViewportSize(viewport);
    await page.goto('/abyss');
    await page.locator('[data-tab-key="social"]').click();

    const hub = page.locator('#abyssSocialHub');
    await expect(hub).toBeVisible();
    await expect(hub.getByText('Friends, cheers & server chat')).toBeVisible();
    await expect(hub.getByText('Cooperative operations')).toBeVisible();
    await expect(hub.getByPlaceholder('Exact player UID')).toHaveAttribute('maxlength', '96');
    await expect(hub.getByPlaceholder('Message the Abyss channel')).toHaveAttribute('maxlength', '240');
    await expect(hub.getByText('Token duels')).toBeVisible();
    await expect(hub.getByText('Five-player weekly raid')).toBeVisible();
    await expect(hub.getByText('Three-player tournament team')).toBeVisible();
    await expect(hub.getByText('Recipient-bound consumable exchange')).toBeVisible();

    expect(await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth)).toBeLessThanOrEqual(1);
    expect(pageErrors).toEqual([]);
  });
}
