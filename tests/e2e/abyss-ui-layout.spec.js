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

test('low-health descent locks every run action while confirmation is open', async ({ page }) => {
  await page.goto('/abyss?active=1');
  await page.evaluate(() => {
    const health = document.getElementById('hpBar');
    health.setAttribute('aria-valuenow', '20');
    health.setAttribute('aria-valuemax', '100');
  });

  await page.locator('#btnDescend').click();
  await expect(page.locator('#sharedModalCard')).toContainText('Descend anyway?');
  await expect(page.locator('#abyssControls')).toHaveAttribute('aria-busy', 'true');
  await expect(page.locator('#btnDescend')).toBeDisabled();
  await expect(page.locator('#btnBank')).toBeDisabled();

  await page.locator('#modalCancelBtn').click();
  await expect(page.locator('#abyssControls')).toHaveAttribute('aria-busy', 'false');
  await expect(page.locator('#btnDescend')).toBeEnabled();
});

test('a lost core-action response offers a safe manual retry with the same idempotency key', async ({ page }) => {
  const requestKeys = [];
  await page.route('**/api/abyss/descend', async route => {
    requestKeys.push(route.request().headers()['idempotency-key']);
    if (requestKeys.length <= 2) {
      await route.abort('connectionreset');
      return;
    }
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ ok: true, depth: 2 }),
    });
  });
  await page.goto('/abyss?active=1');

  const result = await page.evaluate(async () => {
    const body = { interactive: true };
    const failed = await window.abPost('/api/abyss/descend', body);
    window.__abyssRetryResult = null;
    window.abToast(failed.error, false, window.abyssRetryOptions(failed, async () => {
      window.__abyssRetryResult = await window.abPost('/api/abyss/descend', body);
    }));
    return failed;
  });
  expect(result).toMatchObject({ ok: false, error: 'network error', retry_safe: true });
  await expect(page.locator('.ab-toast-retry')).toHaveText('↻ Retry safely');
  await page.locator('.ab-toast-retry').click();
  await expect.poll(() => page.evaluate(() => window.__abyssRetryResult)).toMatchObject({ ok: true, depth: 2 });
  expect(requestKeys).toHaveLength(3);
  expect(requestKeys[0]).toBeTruthy();
  expect(requestKeys[1]).toBe(requestKeys[0]);
  expect(requestKeys[2]).toBe(requestKeys[0]);
});

test('session-expiry and API-latency feedback stay compact and readable', async ({ page }) => {
  await page.goto('/abyss?active=1');
  await page.evaluate(() => {
    document.cookie = `ts3session_exp=${Math.floor(Date.now() / 1000) + 5 * 60}; path=/`;
    window.__abyssFeedback.checkSessionExpiry();
    window.__abyssFeedback.recordLatency(140, true);
  });

  await expect(page.locator('#abyssSessionWarning')).toBeVisible();
  await expect(page.locator('#abyssSessionWarning')).toContainText('expires in 5 minutes');
  await expect(page.locator('#abyssAPILatency')).toHaveAttribute('data-state', 'fast');
  await expect(page.locator('#abyssAPILatency')).toContainText('API · 140 ms');
  await expect(page.locator('#abyssAPILatency')).toHaveAttribute('aria-label', /140 milliseconds, responsive/);

  const overflow = await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth);
  expect(overflow).toBeLessThanOrEqual(1);
});
