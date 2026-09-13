const { test, expect } = require('@playwright/test');
const { fulfillAbyssAPI } = require('./helpers/abyss');

function lootItem(id, slot, status, title) {
  const delta = status === 'downgrade' ? -5 : 5;
  return {
    id, slot, title, label: title, item_type: 'gear', gear_id: 'QUICKWINS_' + id,
    rarity: 'Rare', rarity_rank: 2, depth: 3, source: 'Dropped floor 3',
    is_upgrade: status === 'upgrade', can_equip_best: status === 'upgrade',
    cr: 105, cr_delta: delta, stat_power: 55, equip_on_bank: false,
    comparison: {
      status, power: 55, power_delta: delta,
      reasons: ['Review ' + title + ' against equipped gear.'],
      changes: [{ code: 'STR', label: 'STR', before: 50, after: 50 + delta,
        delta, percent: delta * 2, has_percent: true, combat: true }],
    },
  };
}

async function openLoot(page) {
  const state = { items: [
    lootItem(9701, 'MainHand', 'upgrade', 'Winning Test Blade'),
    lootItem(9702, 'Head', 'tradeoff', 'Tradeoff Test Crown'),
    lootItem(9703, 'MainHand', 'downgrade', 'Runner-up Test Blade'),
  ], mutationRequests: [] };
  await fulfillAbyssAPI(page, (path, body) => {
    if (path.endsWith('/loot/manifest')) return { ok: true, items: state.items };
    if (/\/loot\/(equip_best|sell_junk|reserve)$|\/bank$/.test(path)) state.mutationRequests.push({ path, body });
    return { ok: false, error: 'Read-only loot fixture' };
  });
  await page.goto('/abyss?active=1');
  await page.evaluate(() => refreshRunLootManifest());
  await expect(page.locator('#lootManifest [data-loot-id]')).toHaveCount(3);
  return state;
}

test('loot comparison counts and status filters reflect every mocked item', async ({ page }) => {
  const state = await openLoot(page);
  const counts = page.locator('#lootComparisonCounts');
  for (const status of ['upgrade', 'tradeoff', 'downgrade']) {
    await expect(counts).toContainText(status + ': 1');
  }
  for (const [status, id] of [['upgrade', '9701'], ['tradeoff', '9702'], ['downgrade', '9703']]) {
    await page.locator('#lootComparisonFilter').selectOption(status);
    const visible = page.locator('#lootManifest [data-loot-id]:visible');
    await expect(visible).toHaveCount(1);
    await expect(visible).toHaveAttribute('data-loot-id', id);
  }
  await page.locator('#lootComparisonFilter').selectOption('');
  await expect(page.locator('#lootManifest [data-loot-id]:visible')).toHaveCount(3);
  expect(state.mutationRequests).toEqual([]);
});

test('a refreshed recommendation preserves its expanded row and keyboard focus', async ({ page }) => {
  const errors = [];
  page.on('pageerror', error => errors.push(error.message));
  const state = await openLoot(page);
  const row = page.locator('#lootManifest [data-loot-id="9701"]');
  await row.click();
  await row.focus();
  await expect(row).toHaveAttribute('aria-expanded', 'true');
  await expect(row).toBeFocused();
  state.items[0] = { ...lootItem(9701, 'MainHand', 'tradeoff', 'Winning Test Blade'),
    reservation_warning: 'Your reserved blade now loses defense. Review before banking.' };
  await page.evaluate(() => refreshRunLootManifest());
  await expect(row).toHaveAttribute('aria-expanded', 'true');
  await expect(row.locator('.ab-loot-detail')).toBeVisible();
  await expect(row).toBeFocused();
  await expect(row.locator('.ab-loot-detail')).toContainText('tradeoff');
  await expect(row.locator('.ab-loot-detail')).toContainText(state.items[0].reservation_warning);
  await expect(page.locator('#lootComparisonCounts')).toContainText('tradeoff: 2');
  await expect(page.locator('#lootComparisonCounts')).toContainText('downgrade: 1');
  await expect(page.locator('#lootComparisonCounts')).not.toContainText('upgrade: 1');
  await expect(page.locator('#lootComparisonUpdated')).toContainText('1 recommendations changed');
  await expect(row.locator('.ab-equip-best')).toHaveCount(0);
  expect(state.mutationRequests).toEqual([]);
  expect(errors).toEqual([]);
});

test('equipment slot sorting keeps matching loot slots together', async ({ page }) => {
  const state = await openLoot(page);
  await page.locator('#lootSort').selectOption('slot');
  const rows = page.locator('#lootManifest [data-loot-id]:visible');
  await expect.poll(() => rows.evaluateAll(nodes => nodes.map(node => node.dataset.slot)))
    .toEqual(['Head', 'MainHand', 'MainHand']);
  await expect(rows.nth(0)).toHaveAttribute('data-loot-id', '9702');
  await expect(rows.nth(1)).toHaveAttribute('data-loot-id', '9701');
  await expect(rows.nth(2)).toHaveAttribute('data-loot-id', '9703');
  expect(state.mutationRequests).toEqual([]);
});

test('expanded loot explains its winning comparison, runner-up, and reservation warning', async ({ page }) => {
  const state = await openLoot(page);
  state.items[0].best_reason = 'Best eligible MainHand: gains 5 stat power without losing effective XP.';
  state.items[0].runner_up_id = 9703;
  state.items[0].runner_up_title = 'Runner-up Test Blade';
  state.items[0].reservation_warning = 'The reservation changed after your equipment changed; review before banking.';
  await page.evaluate(() => refreshRunLootManifest());
  const row = page.locator('#lootManifest [data-loot-id="9701"]');
  await row.click();
  const detail = row.locator('.ab-loot-detail');
  await expect(detail).toBeVisible();
  await expect(detail).toContainText(state.items[0].best_reason);
  await expect(detail).toContainText('Runner-up: Runner-up Test Blade');
  await expect(detail).toContainText(state.items[0].reservation_warning);
  expect(state.mutationRequests).toEqual([]);
});
