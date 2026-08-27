const { test, expect } = require('@playwright/test');

test('event encounters use a compact stage and never expose invalid memory values', async ({ page }) => {
  await page.goto('/abyss?active=1&room=forge_floor');

  const stage = page.locator('#abyssStage');
  const panel = page.locator('#nonCombatPanel');
  await expect(stage).toHaveClass(/ab-event-stage/);
  await expect(panel).toBeVisible();
  await expect(panel).toContainText(/the silent anvil/i);
  await expect(page.locator('.ab-elevator')).toBeHidden();
  await expect(page.locator('body')).not.toContainText('NaN%');

  const stageBox = await stage.boundingBox();
  const panelBox = await panel.boundingBox();
  expect(stageBox.height).toBeLessThan(850);
  expect(panelBox.y - stageBox.y).toBeLessThan(160);
  expect(panelBox.width).toBeGreaterThan(420);
});

test('boss record and cosmetic collection stay in their assigned tabs', async ({ page }) => {
  await page.goto('/abyss');

  const record = page.locator('#abyssBestKill');
  const cosmetics = page.locator('.ab-boss-cosmetics');
  await expect(cosmetics).toBeVisible();
  await expect(record).toBeHidden();

  await page.locator('.ab-tab[data-tab-key="shop"]').click();
  await expect(cosmetics).toBeHidden();
  await expect(record).toBeHidden();

  await page.locator('.ab-tab[data-tab-key="leaderboards"]').click();
  await expect(record).toBeVisible();
  await expect(cosmetics).toBeHidden();
});

test('collection mastery and two badge slots remain usable at desktop and mobile widths', async ({ page }) => {
  await page.goto('/abyss');

  const collections = page.locator('.ab-collections');
  await expect(collections).toBeVisible();
  await collections.locator('summary').click();
  await expect(collections).toContainText('Mossbound');
  await expect(collections).toContainText('25% · +2% loot find');

  await page.locator('#abyssRareActions summary').click();
  await page.getByRole('button', { name: 'Choose prefix' }).click();
  await expect(page.locator('#sharedModalCard')).toContainText('Choose prefix badge');
  await page.evaluate(() => closeModal());
  await page.getByRole('button', { name: 'Choose suffix' }).click();
  await expect(page.locator('#sharedModalCard')).toContainText('Choose suffix badge');
  await page.evaluate(() => closeModal());

  await page.setViewportSize({ width: 390, height: 844 });
  const widths = await page.evaluate(() => ({ body: document.body.scrollWidth, viewport: window.innerWidth }));
  expect(widths.body).toBeLessThanOrEqual(widths.viewport + 1);
});
