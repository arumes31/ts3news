const { test, expect } = require('@playwright/test');
const { fulfillAbyssAPI } = require('./helpers/abyss');

function runeQuote(consequence) {
  const balance = { gold: 1000, tokens: 20, materials: {} };
  const stats = { STR: 60, INT: 20 };
  return {
    schema_version: 1,
    operation: 'etch_rune',
    irreversible: false,
    success_chance: 1,
    chance_explanation: 'Guaranteed.',
    failure_explanation: 'None.',
    pity_explanation: 'No pity required.',
    undo_available: true,
    undo_window_seconds: 60,
    confirmation_phrase: 'ETCH RUNE',
    cost: { gold: 150, tokens: 0, materials: {} },
    cost_minimum: { gold: 150, tokens: 0, materials: {} },
    cost_maximum: { gold: 150, tokens: 0, materials: {} },
    balance_before: balance,
    balance_after: { ...balance, gold: 850 },
    outcome: {
      minimum_stats: stats,
      expected_stats: stats,
      maximum_stats: stats,
      minimum_cr: 80,
      expected_cr: 80,
      maximum_cr: 80,
      gained_effects: ['Fire rune'],
      lost_effects: [],
      consequences: [consequence],
    },
    warnings: [],
    durability_before: 100,
    durability_after: 100,
    sockets_before: 0,
    sockets_after: 0,
    set_after: '',
    tradeable_after: true,
    recovery: {},
    cost_explanation: 'Known rune price.',
  };
}

test('Forge previews the authoritative offensive rune resonance', async ({ page }) => {
  const pageErrors = [];
  let quoteRequest = null;
  page.on('pageerror', error => pageErrors.push(error.message));
  await fulfillAbyssAPI(page, (path, body) => {
    if (path.endsWith('/forge/quote')) {
      quoteRequest = body;
      return {
        ok: true,
        quote: runeQuote('Fire rune grants +5% resonance to matching Fire attacks; it deals 2.0× against Air, 0.5× against Water, and 1.0× otherwise.'),
      };
    }
    if (path.endsWith('/transmog')) {
      return { ok: true, owned: 0, total: 240, gold: 1000, appearances: [] };
    }
    if (path.endsWith('/forge/receipts')) {
      return { ok: true, receipts: [] };
    }
    return { ok: false, error: 'unexpected e2e request' };
  });

  await page.setViewportSize({ width: 1280, height: 900 });
  await page.goto('/abyss?gear=1');
  await page.locator('.ab-tab[data-tab-key="forge"]').click();
  await page.locator('#forgeItemSelect').selectOption('equipped:MainHand');
  await page.locator('#forgeRuneFamily').selectOption('offensive');
  await page.locator('#forgeRuneElement').selectOption('Fire');
  await page.locator('#forgePlanOperation').selectOption('etch_rune');

  await expect.poll(() => quoteRequest).not.toBeNull();
  expect(quoteRequest.operation).toBe('etch_rune');
  expect(quoteRequest.parameters.rune).toBe('Fire');
  expect(quoteRequest.parameters.rune_family).toBe('offensive');
  await expect(page.locator('#forgeQuoteLost')).toContainText('+5% resonance');
  await expect(page.locator('#forgeQuoteLost')).toContainText('2.0× against Air');
  await expect(page.locator('#forgeQuotePanel')).toHaveAttribute('aria-busy', 'false');
  expect(pageErrors).toEqual([]);
  expect(await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth)).toBeLessThanOrEqual(1);
});
