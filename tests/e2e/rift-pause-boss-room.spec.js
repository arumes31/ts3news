const { test, expect } = require('@playwright/test');

test('pause before boss rooms holds progression before the boss room until explicit continuation', async ({ page }) => {
  await page.goto('/abyss/rift?scenario=checkpoint&room=preboss');
  await expect(page.locator('#rift-start')).toBeEnabled();

  const settings = page.locator('.rift-settings');
  await settings.locator('> summary').click();

  const pauseBoss = page.locator('#rift-pause-boss-room');
  await expect(pauseBoss).toBeVisible();
  await pauseBoss.check();

  await page.evaluate(() => history.replaceState(null, '', '/abyss/rift'));
  await page.reload();
  await expect(page.locator('#rift-start')).toBeEnabled();

  await settings.locator('> summary').click();
  await expect(pauseBoss).toBeChecked();
  await settings.locator('> summary').click();

  const before = (await (await page.request.get('/api/abyss/rift')).json()).run;
  expect(before.room).toBe(1);

  await page.locator('#rift-start').click();
  await expect(page.locator('#rift-transition')).toContainText('Boss room ahead');
  await expect(page.locator('#rift-next')).toContainText('Enter boss room');

  // Wait beyond default transition delay to ensure progression is held
  await page.waitForTimeout(1600);
  const held = (await (await page.request.get('/api/abyss/rift')).json()).run;
  expect(held.room).toBe(1);
  expect(held.status).toBe('cleared');

  // Explicitly continue into the boss room
  await page.locator('#rift-next').click();
  await expect.poll(async () => (await (await page.request.get('/api/abyss/rift')).json()).run.room).toBe(2);
});

test('ordinary room progression is not halted by pause before boss rooms', async ({ page }) => {
  await page.addInitScript(() => localStorage.setItem('riftPauseBossRoom', 'true'));
  await page.goto('/abyss/rift?scenario=checkpoint');
  await expect(page.locator('#rift-start')).toBeEnabled();

  const before = (await (await page.request.get('/api/abyss/rift')).json()).run;
  expect(before.room).toBe(0);

  await page.locator('#rift-start').click();
  await expect.poll(async () => (await (await page.request.get('/api/abyss/rift')).json()).run.room).toBe(1);
});
