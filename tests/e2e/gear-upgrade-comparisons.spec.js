const { test, expect } = require('@playwright/test');
const { fulfillAbyssAPI } = require('./helpers/abyss');

test('shop exposes before/after percentages, comparison filters and stat power sorting', async ({ page }) => {
  await page.goto('/shop');
  await expect(page.locator('.shop-comparison').first()).toContainText('→');
  await expect(page.locator('.shop-comparison').first()).toContainText('%');
  await page.locator('#shopComparison').selectOption('tradeoff');
  const statuses = await page.locator('.shop-card:visible').evaluateAll(cards => cards.map(c => c.dataset.comparison));
  expect(statuses.length).toBeGreaterThan(0);
  expect(statuses.every(status => status === 'tradeoff')).toBe(true);
  await page.reload();
  await expect(page.locator('#shopComparison')).toHaveValue('tradeoff');
  await page.locator('#shopReset').click();
  for (const [sort, key] of [['power', 'power'], ['gain', 'gain']]) {
    await page.locator('#shopSort').selectOption(sort);
    const values = await page.locator('.shop-card:visible').evaluateAll((cards, key) => cards.map(c => Number(c.dataset[key])), key);
    expect(values.length).toBeGreaterThan(1);
    expect(values).toEqual([...values].sort((a,b) => b-a));
  }
});

test('item inspector explains stat metadata and raw versus effective XP', async ({ page }) => {
  await page.goto('/shop');
  await page.locator('.shop-card [data-item-inspect]').first().click();
  const inspector = page.locator('.item-inspector');
  await expect(inspector).toBeVisible();
  await expect(inspector).toContainText('Stat power');
  await expect(inspector).toContainText('Raw bonus');
  await expect(inspector).toContainText('Effective bonus');
  expect(await inspector.locator('.item-inspector-stats [title]').count()).toBeGreaterThan(0);
});

test('structured loot comparisons render exact changes and escape recommendation text', async ({ page }) => {
  const loot = { id: 9002, item_type: 'gear', slot: 'Head', rarity: 'Rare', label: 'Review Helm', title: 'Review Helm',
    is_upgrade: false, cr: 20, cr_delta: 2, stat_power: 15,
    comparison: { status: 'tradeoff', power: 15, reasons: ['Loses defense. <img src=x onerror=alert(1)>'],
      changes: [{ code: 'DEF', label: 'DEF', before: 100, after: 90, delta: -10, percent: -10, has_percent: true, combat: true }] } };
  await page.route('**/api/abyss/loot/manifest', route => route.fulfill({ json: { ok: true, items: [loot] } }));
  await page.goto('/abyss?active=1');
  await page.evaluate(() => refreshRunLootManifest());
  const row = page.locator('[data-loot-id="9002"]');
  await row.click();
  await expect(row.locator('.ab-loot-detail')).toContainText('100 → 90');
  await expect(row.locator('.ab-loot-detail')).toContainText('-10.0%');
  await expect(row.locator('.ab-loot-detail')).toContainText('tradeoff');
  await expect(row.locator('.ab-loot-detail img')).toHaveCount(0);
  await row.hover();
  await expect(page.locator('.ab-hovercard')).toContainText('100 → 90');
});

test('forge separates current stats from guaranteed minimum and includes stamina', async ({ page }) => {
  const errors=[]; page.on('pageerror',e=>errors.push(e.message));
  await fulfillAbyssAPI(page, path => {
    if (path.endsWith('/forge/quote')) return { ok: true, quote: {
      operation: 'reinforce', current: { Stats: { DEF: 50, STA: 9 } }, current_cr: 50,
      success_chance: 1, chance_explanation: 'Guaranteed', cost: { gold: 100 },
      balance_before: {gold:1000,tokens:0,materials:{}}, balance_after: {gold:900,tokens:0,materials:{}},
      outcome: { minimum_stats:{DEF:51,STA:9},expected_stats:{DEF:51,STA:9},maximum_stats:{DEF:51,STA:9},
        minimum_cr:50.9,expected_cr:50.9,maximum_cr:50.9,gained_effects:[],lost_effects:[],consequences:[] }, warnings: [], recovery: {} } };
    if(path.endsWith('/forge/receipts')) return {ok:true,receipts:[]};
    if(path.endsWith('/transmog')) return {ok:true,owned:0,total:0,gold:1000,appearances:[]};
    return {ok:false,error:'unexpected e2e request'};
  });
  await page.goto('/abyss?gear=1');
  await page.locator('.ab-tab[data-tab-key="forge"]').click();
  await page.locator('#forgeItemSelect').selectOption('equipped:MainHand');
  await page.locator('#forgePlanOperation').selectOption('reinforce');
  const outcomes=page.locator('#forgeQuoteOutcomes .forge-outcome');
  await expect(outcomes).toHaveCount(4);
  await expect(outcomes.nth(0)).toContainText('DEF +50');
  await expect(outcomes.nth(0)).toContainText('STA +9');
  await expect(outcomes.nth(1)).toContainText('DEF +51');
  for(const width of [390,1280]) {
    await page.setViewportSize({width,height:900});
    expect(await page.locator('#forgeQuoteOutcomes').evaluate(el=>el.scrollWidth-el.clientWidth)).toBeLessThanOrEqual(1);
  }
  expect(errors).toEqual([]);
});

test('fresh loot comparisons show losses and filter by the authoritative upgrade decision', async ({ page }) => {
  const errors = [];
  page.on('pageerror', error => errors.push(error.message));
  const loot = {
    id: 9001, depth: 1, item_type: 'gear', slot: 'Head', rarity: 'Eternal',
    label: 'Old drop', title: 'Old drop', source: 'Dropped floor 1',
    score: 30, cr: 165, cr_delta: 44.1, is_upgrade: false,
    stat_changes: [{ label: 'STR', value: -80 }, { label: 'DEF', value: 9 }],
    xp_bonus_delta: 20, empty_slot: false, unidentified: false,
  };
  await page.route('**/api/abyss/loot/manifest', route => route.fulfill({ json: { ok: true, items: [loot] } }));
  await page.goto('/abyss?active=1');
  await page.evaluate(() => refreshRunLootManifest());
  const row = page.locator('[data-loot-id="9001"]');
  await row.click();
  await expect(row.locator('.ab-loot-detail')).toContainText('STR -80');
  await expect(row.locator('.ab-loot-detail')).toContainText('DEF +9');
  await expect(row.locator('.ab-loot-detail')).toContainText('+44.1');
  await expect(row.locator('.ab-loot-detail')).toContainText('No automatic upgrade');
  await expect(row.locator('.ab-equip-best')).toHaveCount(0);
  await page.evaluate(() => setLootFilter('up'));
  await expect(row).toBeHidden();
  loot.is_upgrade = true;
  loot.can_equip_best = true;
  loot.cr_delta = -0.1;
  loot.stat_changes = [{ label: 'MNA', value: 1 }];
  await page.evaluate(() => refreshRunLootManifest());
  await expect(row).toBeVisible();
  await expect(row.locator('.ab-equip-best')).toBeVisible();
  expect(errors).toEqual([]);
});

test('shop upgrade badges never accompany lost item stats or effective XP', async ({ page }) => {
  await page.goto('/shop');
  const bad = await page.evaluate(() => shopCards.filter(card => card.dataset.upgrade === 'true').filter(card => {
    const losses = card.querySelectorAll('.shop-stat-loss');
    const comparison = card.querySelector('.shop-comparison');
    return losses.length || /Effective XP bonus -/.test(comparison?.textContent || '');
  }).map(card => card.dataset.name));
  expect(bad).toEqual([]);
  await expect(page.locator('.shop-comparison').first()).toContainText('Rarity-weighted CR');
});
