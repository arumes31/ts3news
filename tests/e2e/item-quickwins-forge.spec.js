const { test, expect } = require('@playwright/test');
const { fulfillAbyssAPI } = require('./helpers/abyss');

function forgeQuote(request, marker, overrides = {}) {
  const balance = { gold: 10000, tokens: 100, materials: { dust: 100 } };
  return {
    schema_version: 1,
    operation: request.operation,
    token: 'fixture-' + marker,
    expires_at: new Date(Date.now() + 120000).toISOString(),
    current_item: { ID: marker, Name: marker, Slot: request.slot, Temper: 4, Stats: { STR: 99, MNA: 20 } },
    current_cr: 432.1,
    success_chance: 1,
    chance_explanation: '100.0% success: temper guard guarantees this attempt.',
    failure_explanation: 'The guard converts a failed roll into success.',
    pity_explanation: 'The active guard guarantees the displayed outcome.',
    cost: { gold: 100, tokens: 0, materials: {} },
    cost_minimum: { gold: 100, tokens: 0, materials: {} },
    cost_maximum: { gold: 100, tokens: 0, materials: {} },
    balance_before: balance,
    balance_after: { ...balance, gold: 9900 },
    outcome: {
      minimum_stats: { STR: 100, MNA: 20 },
      expected_stats: { STR: 100, MNA: 20 },
      maximum_stats: { STR: 100, MNA: 20 },
      minimum_cr: 456.7,
      expected_cr: 456.7,
      maximum_cr: 456.7,
      gained_effects: [marker],
      lost_effects: [],
      consequences: [],
    },
    warnings: [],
    recovery: {},
    durability_before: 100,
    durability_after: 100,
    sockets_before: 0,
    sockets_after: 0,
    tradeable_after: true,
    undo_available: true,
    undo_window_seconds: 60,
    cost_explanation: 'Fixture quote cost.',
    ...overrides,
  };
}

async function openForge(page) {
  await fulfillAbyssAPI(page, path => {
    if (path.endsWith('/forge/receipts')) return { ok: true, receipts: [] };
    if (path.endsWith('/transmog')) return { ok: true, owned: 0, total: 0, gold: 10000, appearances: [] };
    return { ok: false, error: 'Unexpected fixture request' };
  });
  await page.goto('/abyss?gear=1');
  await page.locator('.ab-tab[data-tab-key="forge"]').click();
}

async function selectTemper(page) {
  await page.locator('#forgeItemSelect').selectOption('equipped:MainHand');
  await page.locator('#forgePlanOperation').selectOption('temper');
}

test('latest forge item and operation remain selected when older responses arrive last', async ({ page }) => {
  await openForge(page);
  let hold = false;
  const pending = [];
  await page.route('**/api/abyss/forge/quote', async route => {
    const request = route.request().postDataJSON();
    if (hold) {
      await new Promise(resolve => pending.push({ route, request, resolve }));
      return;
    }
    await route.fulfill({ json: { ok: true, quote: forgeQuote(request, 'initial') } });
  });
  await selectTemper(page);
  await expect(page.locator('#forgeQuoteGained')).toContainText('initial');
  hold = true;
  await page.locator('#forgeItemSelect').selectOption('equipped:OffHand');
  await expect.poll(() => pending.length).toBe(1);
  await page.locator('#forgePlanOperation').selectOption('masterwork');
  await expect.poll(() => pending.length).toBe(2);

  const newest = pending[1];
  await newest.route.fulfill({ json: { ok: true, quote: forgeQuote(newest.request, 'latest-offhand-masterwork') } });
  newest.resolve();
  await expect(page.locator('#forgeQuoteGained')).toContainText('latest-offhand-masterwork');
  const oldest = pending[0];
  await oldest.route.fulfill({ json: { ok: true, quote: forgeQuote(oldest.request, 'obsolete-offhand-temper') } });
  oldest.resolve();
  await page.waitForLoadState('networkidle');
  await expect(page.locator('#forgeItemSelect')).toHaveValue('equipped:OffHand');
  await expect(page.locator('#forgePlanOperation')).toHaveValue('masterwork');
  await expect(page.locator('#forgeQuoteGained')).toContainText('latest-offhand-masterwork');
  await expect(page.locator('#forgeQuotePanel')).not.toContainText('obsolete-offhand-temper');
  await expect(page.locator('#forgeQuotePanel')).toHaveAttribute('aria-busy', 'false');
});

test('changing forge parameters clears the old outcome while its replacement is pending', async ({ page }) => {
  await openForge(page);
  let pendingRoute;
  let release;
  let hold = false;
  await page.route('**/api/abyss/forge/quote', async route => {
    if (hold) {
      pendingRoute = route;
      await new Promise(resolve => { release = resolve; });
      return;
    }
    await route.fulfill({ json: { ok: true, quote: forgeQuote(route.request().postDataJSON(), 'old-parameter-outcome') } });
  });
  await page.locator('#forgeItemSelect').selectOption('equipped:MainHand');
  await page.locator('#forgePlanOperation').selectOption('batch_temper');
  await page.getByText('Advanced operation settings', { exact: true }).click();
  await expect(page.locator('#forgeQuoteGained')).toContainText('old-parameter-outcome');
  hold = true;
  try {
    await page.locator('#forgeBatchTarget').fill('2');
    await page.locator('#forgeBatchTarget').blur();
    await expect.poll(() => Boolean(pendingRoute)).toBe(true);
    await expect(page.locator('#forgeQuotePanel')).toHaveAttribute('aria-busy', 'true');
    await expect(page.locator('#forgeQuoteOutcomes .forge-outcome:visible')).toHaveCount(0);
    await expect(page.locator('#forgeQuoteGained')).not.toContainText('old-parameter-outcome');
    expect(pendingRoute.request().postDataJSON().parameters.batch_target).toBe(2);
    await pendingRoute.fulfill({ json: { ok: true, quote: forgeQuote(pendingRoute.request().postDataJSON(), 'new-parameter-outcome') } });
    release();
    release = null;
    await expect(page.locator('#forgeQuoteGained')).toContainText('new-parameter-outcome');
  } finally {
    if (release) {
      await pendingRoute.abort();
      release();
    }
  }
});

for (const failure of ['server rejection', 'network failure']) {
  test(`a ${failure} hides previous forge outcomes and balances`, async ({ page }) => {
    await openForge(page);
    let fail = false;
    await page.route('**/api/abyss/forge/quote', async route => {
      if (!fail) return route.fulfill({ json: { ok: true, quote: forgeQuote(route.request().postDataJSON(), 'stale-success') } });
      if (failure === 'network failure') return route.abort('connectionreset');
      return route.fulfill({ json: { ok: false, error: 'This item changed; refresh its quote.' } });
    });
    await selectTemper(page);
    await expect(page.locator('#forgeQuoteGained')).toContainText('stale-success');
    fail = true;
    await page.locator('#forgeQuoteRefresh').click();
    await expect(page.locator('#forgeQuoteChance')).not.toContainText('100.0% success');
    await expect(page.locator('#forgeQuotePanel')).toHaveAttribute('aria-busy', 'false');
    await expect(page.locator('#forgeQuoteOutcomes .forge-outcome:visible')).toHaveCount(0);
    await expect(page.locator('#forgeQuoteGained')).not.toContainText('stale-success');
    await expect(page.locator('#forgeQuoteBalances')).not.toContainText('9900');
    await expect(page.locator('#forgeAfter')).not.toContainText('456.7');
  });
}

test('a quote countdown expires and removes previously valid forge outcomes', async ({ page }) => {
  await openForge(page);
  const now = Date.now();
  await page.clock.install({ time: now });
  await page.route('**/api/abyss/forge/quote', route => route.fulfill({
    json: { ok: true, quote: forgeQuote(route.request().postDataJSON(), 'expiring-outcome', {
      expires_at: new Date(now + 10000).toISOString(),
    }) },
  }));
  await selectTemper(page);
  await expect(page.locator('#forgeQuoteGained')).toContainText('expiring-outcome');
  await expect(page.locator('#forgeQuoteExpiry')).toBeVisible();
  await expect(page.locator('#forgeQuoteExpiry')).toContainText(/\d/);
  await page.clock.fastForward(11000);
  await expect(page.locator('#forgeQuoteExpiry')).toContainText(/expired|refresh/i);
  await expect(page.locator('#forgeQuoteOutcomes .forge-outcome:visible')).toHaveCount(0);
  await expect(page.locator('#forgeQuoteGained')).not.toContainText('expiring-outcome');
  await expect(page.locator('#forgeAfter')).not.toContainText('456.7');
});

test('compact temper summary uses the canonical current item and guarded server quote', async ({ page }) => {
  const errors = [];
  page.on('pageerror', error => errors.push(error.message));
  await openForge(page);
  await page.route('**/api/abyss/forge/quote', route => route.fulfill({
    json: { ok: true, quote: forgeQuote(route.request().postDataJSON(), 'guarded-temper') },
  }));
  await selectTemper(page);
  await expect(page.locator('#forgeQuoteGained')).toContainText('guarded-temper');
  const outcomes = page.locator('#forgeQuoteOutcomes .forge-outcome');
  await expect(outcomes).toHaveCount(4);
  await expect(outcomes.nth(0)).toContainText('432.1');
  await expect(outcomes.nth(0)).toContainText('STR +99');
  await expect(outcomes.nth(0)).toContainText('MNA +20');
  await expect(outcomes.nth(2)).toContainText('456.7');
  await expect(page.locator('#forgeBefore')).toContainText('432.1');
  await expect(page.locator('#forgeAfter')).toContainText('456.7');
  await expect(page.locator('#forgeAfter')).not.toContainText('≈');
  await expect(page.locator('#forgeTemperChanceLabel')).toContainText(/100(?:\.0)?%/);
  await expect(page.locator('#forgeQuoteChance')).toContainText('100.0%');
  await expect(page.locator('#forgeQuoteChance')).toContainText(/guard/i);
  expect(errors).toEqual([]);
});

test('temper failure and success cards show actual endpoints when a stat is negative', async ({ page }) => {
  await openForge(page);
  await page.route('**/api/abyss/forge/quote', route => {
    const request = route.request().postDataJSON();
    const quote = forgeQuote(request, 'negative-defense-temper', {
      current_item: { ID: 'NEGATIVE_DEF', Name: 'Cursed Test Blade', Slot: request.slot,
        Stats: { STR: 99, DEF: -100 } },
      current_cr: 28.8,
      success_chance: 0.75,
      chance_explanation: '75.0% success.',
      failure_explanation: 'Stats remain unchanged.',
      pity_explanation: 'No guard is active.',
    });
    quote.outcome = {
      // Coordinate-wise extrema mix stats from different outcomes. Neither
      // minimum_stats nor maximum_stats is an actual temper result here.
      minimum_stats: { STR: 99, DEF: -102 }, minimum_cr: 28.2,
      maximum_stats: { STR: 100, DEF: -100 }, maximum_cr: 28.8,
      expected_stats: { STR: 99, DEF: -101 }, expected_cr: 28.35,
      failure_stats: { STR: 99, DEF: -100 }, failure_cr: 28.8,
      success_stats: { STR: 100, DEF: -102 }, success_cr: 28.2,
      gained_effects: ['negative-defense-temper'], lost_effects: [], consequences: [],
    };
    return route.fulfill({ json: { ok: true, quote } });
  });
  await selectTemper(page);
  const cards = page.locator('#forgeQuoteOutcomes .forge-outcome');
  const failure = cards.filter({ has: page.locator('b').filter({ hasText: /^Failure/ }) });
  const success = cards.filter({ has: page.locator('b').filter({ hasText: /^Success/ }) });
  await expect(failure).toHaveCount(1);
  await expect(failure).toContainText('STR +99');
  await expect(failure).toContainText('DEF -100');
  await expect(failure).toContainText('28.8');
  await expect(success).toHaveCount(1);
  await expect(success).toContainText('STR +100');
  await expect(success).toContainText('DEF -102');
  await expect(success).toContainText('28.2');
});

test('mobile forge controls match the operation and a failed quote disables its commit', async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await openForge(page);
  let rejectQuote = false;
  await page.route('**/api/abyss/forge/quote', route => route.fulfill({
    json: rejectQuote ? { ok: false, error: 'Quote rejected after item changed.' } : {
      ok: true, quote: forgeQuote(route.request().postDataJSON(), 'operation-controls-ready'),
    },
  }));
  await page.locator('#forgeItemSelect').selectOption('equipped:MainHand');
  await page.locator('#forgePlanOperation').selectOption('attune');
  await page.locator('.forge-advanced-disclosure > summary').click();
  await expect(page.locator('#forgeQuoteGained')).toContainText('operation-controls-ready');
  await expect(page.locator('#forgeAdvancedControls label:has(#forgeProtection)')).toBeHidden();
  await expect(page.locator('#forgeAdvancedControls label:has(#forgeEventRecipe)')).toBeHidden();
  await expect(page.locator('#forgeRuneFamily')).toBeHidden();
  await expect(page.locator('#forgePlanner button[id$="Commit"]:visible')).toHaveCount(0);

  await page.locator('#forgePlanOperation').selectOption('etch_rune');
  await expect(page.locator('#forgeRuneFamily')).toBeVisible();
  await expect(page.locator('#forgeRuneElement')).toBeVisible();
  await expect(page.locator('#forgeAdvancedControls label:has(#forgeProtection)')).toBeHidden();
  await expect(page.locator('#forgeAdvancedControls label:has(#forgeEventRecipe)')).toBeHidden();
  const visibleCommits = page.locator('#forgePlanner button[id$="Commit"]:visible');
  await expect(visibleCommits).toHaveCount(1);
  await expect(visibleCommits).toHaveAttribute('id', 'forgeEtchCommit');
  await expect(visibleCommits).toBeEnabled();
  rejectQuote = true;
  await page.locator('#forgeQuoteRefresh').click();
  await expect(page.locator('#forgeQuoteChance')).toContainText('Quote rejected after item changed.');
  await expect(page.locator('#forgeEtchCommit')).toBeVisible();
  await expect(page.locator('#forgeEtchCommit')).toBeDisabled();
  await expect(visibleCommits).toHaveCount(1);
});

test('a mocked forge commit retains its actual stat-change receipt after reload', async ({ page }) => {
  const errors = [];
  const commits = [];
  page.on('pageerror', error => errors.push(error.message));
  await openForge(page);
  await page.route('**/api/abyss/forge/quote', route => route.fulfill({
    json: { ok: true, quote: forgeQuote(route.request().postDataJSON(), 'receipt-quote-ready') },
  }));
  await page.route('**/api/abyss/etch_rune', route => {
    commits.push(route.request().postDataJSON());
    return route.fulfill({ json: {
      ok: true, msg: 'Mocked etch complete.',
      stat_changes: [
        { code: 'STR', label: 'STR', before: 50, after: 53, delta: 3 },
        { code: 'DEF', label: 'DEF', before: -100, after: -102, delta: -2 },
      ],
    } });
  });
  await page.locator('#forgeItemSelect').selectOption('equipped:MainHand');
  await page.locator('#forgePlanOperation').selectOption('etch_rune');
  await expect(page.locator('#forgeQuoteGained')).toContainText('receipt-quote-ready');
  await page.locator('#forgeEtchCommit').click();
  await expect(page.locator('#forgeConfirmDialog')).toBeVisible();
  await page.locator('#forgeConfirmPhrase').fill('ETCH RUNE');
  const reloaded = page.waitForEvent('domcontentloaded');
  await page.locator('#forgeConfirmSubmit').click();
  await reloaded;
  await page.locator('.ab-tab[data-tab-key="forge"]').click();
  const receipt = page.locator('#forgeActualReceipts');
  await expect(receipt).toContainText('etch rune');
  await expect(receipt).toContainText('STR 50 → 53 (+3)');
  await expect(receipt).toContainText('DEF -100 → -102 (-2)');
  await expect(receipt.locator('p')).toHaveCount(1);
  expect(commits).toHaveLength(1);
  expect(commits[0]).toMatchObject({ slot: 'MainHand', rune: 'Fire', rune_family: 'offensive' });
  expect(errors).toEqual([]);
});
