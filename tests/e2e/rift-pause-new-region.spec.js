const { test, expect } = require('@playwright/test');

test('pause before a new region holds progression before entering a new region until explicit continuation', async ({ page }) => {
  await page.goto('/abyss/rift?scenario=checkpoint&room=final&level=10');
  await expect(page.locator('#rift-start')).toBeEnabled();

  const settings = page.locator('.rift-settings');
  await settings.locator('> summary').click();

  const pauseRegion = page.locator('#rift-pause-new-region');
  await expect(pauseRegion).toBeVisible();
  await pauseRegion.check();

  await page.evaluate(() => history.replaceState(null, '', '/abyss/rift'));
  await page.reload();
  await expect(page.locator('#rift-start')).toBeEnabled();

  await settings.locator('> summary').click();
  await expect(pauseRegion).toBeChecked();
  await settings.locator('> summary').click();

  const before = (await (await page.request.get('/api/abyss/rift')).json()).run;
  expect(before.level.id).toBe(10);
  expect(before.room).toBe(2);

  await page.locator('#rift-start').click();
  await expect(page.locator('#rift-transition')).toContainText('Approaching Ember Forge');
  await expect(page.locator('#rift-next')).toContainText('Enter next region');

  // Wait beyond default transition delay to ensure progression is held
  await page.waitForTimeout(1600);
  const held = (await (await page.request.get('/api/abyss/rift')).json()).run;
  expect(held.level.id).toBe(10);
  expect(held.room).toBe(2);
  expect(held.status).toBe('cleared');

  // Explicitly continue into the new region
  await page.locator('#rift-next').click();
  await expect.poll(async () => (await (await page.request.get('/api/abyss/rift')).json()).run?.level?.id).toBe(11);
});

test('non-region-boundary level transition is not halted by pause before a new region', async ({ page }) => {
  await page.addInitScript(() => localStorage.setItem('riftPauseNewRegion', 'true'));
  await page.goto('/abyss/rift?scenario=checkpoint&room=final&level=1');
  await expect(page.locator('#rift-start')).toBeEnabled();

  const before = (await (await page.request.get('/api/abyss/rift')).json()).run;
  expect(before.level.id).toBe(1);
  expect(before.room).toBe(2);

  await page.locator('#rift-start').click();
  await expect.poll(async () => (await (await page.request.get('/api/abyss/rift')).json()).run?.level?.id).toBe(2);
});
