const { test, expect } = require('@playwright/test');
const { fulfillAbyssAPI } = require('./helpers/abyss');

for (const width of [390, 1440]) {
  test(`repeated descent reaches defeat without stale controls at ${width}px`, async ({ page }) => {
    let descents = 0;
    await fulfillAbyssAPI(page, path => {
      if (!path.endsWith('/descend')) return { ok: false, error: 'Fixture endpoint unavailable' };
      descents += 1;
      return {
        ok: true, victory: descents < 3, depth: 12 + descents, risk: 20,
        hp: descents < 3 ? 900 - descents * 100 : 0, max_hp: 1000,
        gold: 5000, tokens: 12, escrow: 3456 + descents * 100,
        logs: [`Journey floor ${12 + descents}`], loot: [], dura: [], timeline: [],
        consumables: [], can_revive: true, can_last_stand: true,
        revive_chance_pct: 48, last_stand_cost: 8,
      };
    });
    await page.setViewportSize({ width, height: 900 });
    await page.goto('/abyss?active=1');
    await page.evaluate(() => { window.reduceMotion = true; });
    const descend = page.locator(width === 390 ? '[data-mobile-action="btnDescend"]' : '#btnDescend');
    for (let floor = 13; floor <= 15; floor += 1) {
      await descend.click();
      await expect(page.locator('#depthNum')).toHaveText(String(floor));
      await expect.poll(() => page.evaluate(() => window.busy)).toBe(false);
      if (floor < 15) await expect(descend).toBeEnabled();
    }
    expect(descents).toBe(3);
    await expect(page.locator('#btnRevive')).toBeVisible();
    await expect(page.locator('#btnConcede')).toBeVisible();
    await expect(page.locator('#btnDescend')).toBeHidden();
    await expect(page.locator('#abyssMobileActions')).toBeHidden();
  });
}

test('partially committed cursed elevator failure keeps completed floors and releases controls', async ({ page }) => {
  await fulfillAbyssAPI(page, path => path.endsWith('/descend') ? {
    ok: false, cursed_elevator: true, error: 'Encounter service unavailable after two cleared floors.',
    depth: 14, hp: 760, max_hp: 1000, escrow: 9000, gold: 5000, tokens: 12, risk: 20,
    floor_results: [13, 14].map(depth => ({ depth, hp: 760, max_hp: 1000, victory: true,
      logs: [`Committed floor ${depth}`], loot: [], dura: [], timeline: [] })),
  } : { ok: false, error: 'Fixture endpoint unavailable' });
  await page.goto('/abyss?active=1');
  await page.evaluate(() => { window.reduceMotion = true; window.__journeyFloors = [];
    document.addEventListener('abyss:batch-floor', event => window.__journeyFloors.push(event.detail.depth)); });
  await page.locator('#btnDescend').click();
  await expect.poll(() => page.evaluate(() => window.__journeyFloors)).toEqual([13, 14]);
  await expect(page.locator('#depthNum')).toHaveText('14');
  await expect.poll(() => page.evaluate(() => ({ depth: curDepth, escrow: curEscrow, busy })) )
    .toEqual({ depth: 14, escrow: 9000, busy: false });
  await expect(page.locator('#btnDescend')).toBeEnabled();
  await expect(page.locator('#abToastHost')).toContainText('Encounter service unavailable');
  await expect(page.locator('#autoContinueEnabled')).not.toBeChecked();
});
