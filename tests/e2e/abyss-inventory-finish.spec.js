const { test, expect } = require('@playwright/test');

test('inventory recovers from an interrupted purchase without an unhandled error', async ({ page }) => {
  const errors = [];
  page.on('pageerror', error => errors.push(error.message));
  await page.route('**/api/inventory/buyback', route => route.abort());
  await page.goto('/inventory');
  await page.locator('.buyback-card button').click();
  await page.getByRole('dialog').getByRole('button', { name: /Buy back/ }).click();
  await expect(page.locator('#invMsg')).toContainText(/could not confirm/i);
  await expect(page.locator('#invMsg')).toHaveAttribute('role', 'status');
  await expect(page.locator('.buyback-card button')).toBeEnabled();
  expect(errors).toEqual([]);
});

test('buyback refreshes restored gear and its slot filter from the server', async ({ page }) => {
  let purchased = false;
  await page.route('**/inventory', async route => {
    const response = await route.fetch();
    let html = await response.text();
    if (purchased) html = html.replace(/(<div class="inv-grid">)/, '$1<div class="inv-card" data-id="3" data-slot="Head">Restored audit helm</div>');
    await route.fulfill({ response, body: html });
  });
  await page.route('**/api/inventory/buyback', route => {
    purchased = true;
    return route.fulfill({ json: { ok: true, gold: 14000 } });
  });
  await page.goto('/inventory');
  await page.locator('.buyback-card button').click();
  await page.getByRole('dialog').getByRole('button', { name: /Buy back/ }).click();
  await expect(page.getByText('Restored audit helm')).toBeVisible();
  await expect(page.locator('#inventorySlotStatus')).toContainText('3 items');
  await page.locator('#inventorySlot').selectOption('Head');
  await expect(page.locator('#inventorySlotStatus')).toContainText('1 item for Head');
});

test('pouch next limits advance and disappear at masterwork', async ({ page }) => {
  let level = 1;
  await page.route('**/api/inventory/pouch/upgrade', route => {
    level++;
    return route.fulfill({ json: { ok: true, level, gold: 4000000, stack_cap: 5 + level, carry_cap: 8 + level, next_cost: level < 3 ? 5000000 : 0 } });
  });
  await page.goto('/inventory');
  for (const rank of [2, 3]) {
    await page.locator('#pouchUpgradeButton').click();
    await page.getByRole('dialog').getByRole('button', { name: /Tailor ·/ }).click();
    await expect(page.locator('#pouchRank')).toHaveText(String(rank));
    if (rank === 2) await expect(page.locator('.pouch-caps')).toContainText('next 8');
  }
  await expect(page.locator('.pouch-caps small:visible')).toHaveCount(0);
  await expect(page.locator('.pouch-copy')).toContainText('Masterwork tailoring complete');
});

test('inventory exposes separate inspect and item-specific actions with tablet targets', async ({ page }) => {
  await page.setViewportSize({ width: 768, height: 1000 });
  await page.goto('/inventory');
  await expect(page.locator('[role="button"] button')).toHaveCount(0);
  const inspect = page.getByRole('button', { name: 'Inspect Lucky Test Charm', exact: true });
  await expect(inspect).toHaveJSProperty('tagName', 'BUTTON');
  await inspect.focus();
  await page.keyboard.press('Enter');
  await expect(page.getByRole('dialog')).toBeVisible();
  await page.keyboard.press('Escape');
  await expect(inspect).toBeFocused();
  for (const name of ['Equip Lucky Test Charm', 'Vendor Lucky Test Charm']) {
    const button = page.getByRole('button', { name, exact: true });
    await expect(button).toBeVisible();
    expect((await button.boundingBox()).height).toBeGreaterThanOrEqual(44);
  }
});

test('rarity text remains readable on inventory surfaces', async ({ page }) => {
  await page.goto('/inventory');
  const ratio = await page.locator('.inv-name').first().evaluate(el => {
    const channels = getComputedStyle(el).color.match(/[\d.]+/g).slice(0, 3).map(Number);
    const luminance = values => values.map(v => v / 255).map(v => v <= .04045 ? v / 12.92 : ((v + .055) / 1.055) ** 2.4).reduce((sum, v, i) => sum + v * [.2126, .7152, .0722][i], 0);
    return (luminance(channels) + .05) / (luminance([17, 25, 37]) + .05);
  });
  expect(ratio).toBeGreaterThanOrEqual(4.5);
});

test('offline recovery offers refresh and prevents repeating an uncertain purchase', async ({ page }) => {
  let posts = 0;
  const errors = [];
  page.on('pageerror', error => errors.push(error.message));
  await page.goto('/inventory');
  await page.route('**/inventory', route => route.abort());
  await page.route('**/api/inventory/buyback', route => { posts++; return route.abort(); });
  await page.locator('.buyback-card button').click();
  await page.getByRole('dialog').getByRole('button', { name: /Buy back/ }).click();
  await expect(page.getByRole('button', { name: 'Refresh inventory', exact: true })).toBeFocused();
  await expect(page.locator('.buyback-card button')).toBeDisabled();
  await expect(page.locator('#invMsg')).toContainText('Reconnect and refresh');
  expect(posts).toBe(1);
  expect(errors).toEqual([]);
});

test('cancel and rejected purchases preserve controls and announce the reason', async ({ page }) => {
  let posts = 0;
  await page.route('**/api/inventory/buyback', route => {
    posts++;
    return route.fulfill({ json: { ok: false, error: 'Not enough gold.' } });
  });
  await page.goto('/inventory');
  const button = page.locator('.buyback-card button');
  await button.click();
  await page.keyboard.press('Escape');
  await expect(button).toBeFocused();
  expect(posts).toBe(0);
  await button.click();
  await page.getByRole('dialog').getByRole('button', { name: /Buy back/ }).click();
  await expect(button).toBeEnabled();
  await expect(page.locator('#invMsg')).toHaveText('Not enough gold.');
  expect(posts).toBe(1);
});

test('inventory does not request the unrelated full catalog manifest', async ({ page }) => {
  const manifests = [];
  page.on('request', request => { if (request.url().includes('abyss_catalog_icons.js')) manifests.push(request.url()); });
  await page.goto('/inventory');
  await page.getByRole('button', { name: 'Inspect Lucky Test Charm', exact: true }).click();
  await expect(page.getByRole('dialog')).toContainText('Lucky');
  expect(manifests).toEqual([]);
});
