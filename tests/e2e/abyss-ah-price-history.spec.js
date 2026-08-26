const { test, expect } = require('@playwright/test');

test('Auction listings show exact bounded server sale history', async ({ page }) => {
  await page.route('**/api/ah/notices', async route => {
    await route.fulfill({ contentType: 'application/json', body: JSON.stringify({ ok: true, notices: [] }) });
  });

  await page.goto('/ah');
  const listing = page.locator('.ah-row').filter({ hasText: 'Cinder Test Blade' });
  await expect(listing).toBeVisible();

  const trend = listing.locator('.ah-spark');
  await expect(trend).toBeVisible();
  await expect(trend).toHaveClass(/trend-up/);
  await expect(trend).toHaveAttribute('title', 'Last 4 sales · min 900g · latest 1600g · max 1600g');
  await expect(trend.getByRole('img')).toHaveAttribute(
    'aria-label',
    'Price history: minimum 900 gold, latest 1600 gold, maximum 1600 gold',
  );
  await expect(trend.locator('polyline')).toHaveAttribute('points', '0,27 33,17 66,24 100,3');
  await expect(trend).toContainText('4 sales');
});
