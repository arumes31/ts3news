const { test, expect } = require('@playwright/test');
const { fulfillAbyssAPI } = require('./helpers/abyss');

function victory(count, extra = {}) {
  return { ok: true, victory: true, depth: 12 + count, hp: 750, max_hp: 1000,
    gold: 5000, tokens: 12, escrow: 3500 + count * 100, bonus: 100, risk: 20,
    logs: [], loot: [], dura: [], timeline: [], consumables: [], ...extra };
}

async function enable(page, count = '3') {
  await page.goto('/abyss?active=1');
  await page.evaluate(() => { window.reduceMotion = true; });
  await page.locator('#autoContinueCount').fill(count);
  await page.locator('#autoContinueEnabled').check();
}

test('auto-continue performs the selected normal interactive descents and then stops', async ({ page }) => {
  const requests = [];
  await fulfillAbyssAPI(page, (path, body) => {
    if (!path.endsWith('/descend')) return { ok: false };
    requests.push(body); return victory(requests.length);
  });
  await enable(page, '2');
  await page.locator('#btnDescend').click();
  await expect(page.locator('#autoContinueStatus')).toContainText('Complete', { timeout: 15000 });
  expect(requests).toEqual([{ interactive: true }, { interactive: true }]);
  await expect(page.locator('#autoContinueEnabled')).not.toBeChecked();
  await expect(page.locator('#depthNum')).toHaveText('14');
});

for (const [name, result] of [
  ['defeat', { victory: false, hp: 0, can_revive: true }],
  ['low health', { hp: 200 }],
  ['sanctuary', { noncombat: true, floor_type: 'rest', event_state: {} }],
  ['path choice', { choose_floor: true, options: [{ index: 0, label: 'Combat' }] }],
  ['boon choice', { boon_draft: { pending: true, options: [] } }],
  ['error', { ok: false, error: 'Encounter unavailable' }],
]) {
  test(`auto-continue stops for ${name}`, async ({ page }) => {
    let requests = 0;
    await fulfillAbyssAPI(page, path => {
      if (!path.endsWith('/descend')) return { ok: false };
      requests += 1; return victory(requests, result);
    });
    await enable(page);
    await page.locator('#btnDescend').click();
    await expect(page.locator('#autoContinueEnabled')).not.toBeChecked({ timeout: 15000 });
    expect(requests).toBe(1);
    expect(await page.evaluate(() => abyssAutoContinue.remaining)).toBe(0);
  });
}

test('Stop cancels a pending continuation and reload never re-arms it', async ({ page }) => {
  let requests = 0;
  await fulfillAbyssAPI(page, path => path.endsWith('/descend') ? victory(++requests) : { ok: false });
  await enable(page);
  await page.locator('#btnDescend').click();
  await expect(page.locator('#autoContinueStatus')).toContainText('Next descent');
  await page.locator('#autoContinueStop').click();
  await page.clock.install();
  await page.clock.fastForward(5000);
  expect(requests).toBe(1);
  await page.reload();
  await expect(page.locator('#autoContinueEnabled')).not.toBeChecked();
});

test('an oversized count performs at most 30 descents', async ({ page }) => {
  let requests = 0;
  await fulfillAbyssAPI(page, path => path.endsWith('/descend') ? victory(++requests) : { ok: false });
  await enable(page, '99');
  await page.clock.install();
  await page.locator('#btnDescend').click();
  for (let count = 1; count <= 30; count += 1) {
    await expect.poll(() => requests).toBe(count);
    await expect.poll(() => page.evaluate(() => busy)).toBe(false);
    if (count < 30) await page.clock.fastForward(2500);
  }
  await expect(page.locator('#autoContinueStatus')).toContainText('Complete');
  await page.clock.fastForward(10000);
  expect(requests).toBe(30);
});
