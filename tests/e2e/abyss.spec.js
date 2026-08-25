const { test, expect } = require('@playwright/test');

async function fulfillAbyssAPI(page, handler) {
  await page.route('**/api/abyss/**', async route => {
    const request = route.request();
    const body = request.postDataJSON?.() || {};
    const response = handler(new URL(request.url()).pathname, body);
    await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(response) });
  });
}

test('enter sends the selected run setup', async ({ page }) => {
  let entered = false;
  await fulfillAbyssAPI(page, path => {
    if (path.endsWith('/enter')) {
      entered = true;
      return { ok: true, free_entry: true };
    }
    return { ok: false, error: 'unexpected e2e request' };
  });
  await page.goto('/abyss');
  await page.evaluate(() => { window.reduceMotion = true; });
  await page.locator('#btnEnter').click();
  await expect.poll(() => entered).toBe(true);
});

test('a victorious descend can preview and commit bank', async ({ page }) => {
  let committed = false;
  await fulfillAbyssAPI(page, (path, body) => {
    if (path.endsWith('/descend')) return {
      ok: true, victory: true, depth: 13, risk: 18, hp: 900, max_hp: 1000,
      gold: 5000, tokens: 12, bonus: 750, escrow: 3750, logs: [], loot: [],
      dura: [], timeline: [], consumables: [], run_floors_cleared: 3,
    };
    if (path.endsWith('/bank') && body.preview) return {
      ok: true, escrow: 3750, depth_bonus_pct: 13, depth_bonus: 488,
      streak_bonus_pct: 0, streak_bonus: 0, payout: 4238, tokens_grant: 2,
      loot_count: 0, capped: false, partial: false,
    };
    if (path.endsWith('/bank')) {
      committed = true;
      return { ok: true, banked: 4238, depth: 13, mult: 1.13, gold: 9238, tokens: 14 };
    }
    return { ok: false, error: 'unexpected e2e request' };
  });
  await page.goto('/abyss?active=1');
  await page.evaluate(() => { window.reduceMotion = true; });
  await page.locator('#btnDescend').click();
  await expect(page.locator('#abStatus')).toContainText('survived');
  await page.locator('#btnBank').click();
  await page.locator('#modalOkBtn').click();
  await expect.poll(() => committed).toBe(true);
});

test('a fatal descend exposes the revive and concede decision', async ({ page }) => {
  await fulfillAbyssAPI(page, path => path.endsWith('/descend') ? {
    ok: true, victory: false, depth: 13, risk: 72, hp: 0, max_hp: 1000,
    gold: 5000, tokens: 12, escrow: 3750, logs: ['A fatal test blow lands.'],
    loot: [], dura: [], timeline: [], can_revive: true, can_last_stand: true,
    revive_chance_pct: 48, revive_streak: 0, last_stand_cost: 8,
  } : { ok: false, error: 'unexpected e2e request' });
  await page.goto('/abyss?active=1');
  await page.evaluate(() => { window.reduceMotion = true; });
  await page.locator('#btnDescend').click();
  await expect(page.locator('#abStatus')).toContainText('Choose');
  await expect(page.locator('#btnRevive')).toBeVisible();
  await expect(page.locator('#btnConcede')).toBeVisible();
});
