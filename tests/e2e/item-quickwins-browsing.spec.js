const { test, expect } = require('@playwright/test');

async function openAdvancedFilters(page) {
  const filters = page.locator('#shopAdvancedFilters');
  if (!(await filters.evaluate(element => element.open))) await filters.locator('summary').click();
  return filters;
}

async function showEveryMatch(page) {
  const more = page.locator('#shopMore');
  for (let batch = 0; batch < 10 && await more.isVisible(); batch++) await more.click();
  await expect(more).toBeHidden();
}

async function openPinnedComparison(page) {
  await page.locator('#shopSlot').selectOption('Head');
  const cards = page.locator('.shop-card:visible');
  expect(await cards.count()).toBeGreaterThanOrEqual(2);
  const first = JSON.parse(await cards.nth(0).locator('[data-item-inspect]').getAttribute('data-item-inspect'));
  const second = JSON.parse(await cards.nth(1).locator('[data-item-inspect]').getAttribute('data-item-inspect'));
  await cards.nth(0).locator('.shop-pin-comparison').click();
  await expect(page.locator('#shopPinnedComparison')).toContainText(first.name);
  await cards.nth(1).locator('.shop-compare-offer').click();
  const tools = page.locator('#shopPinnedComparison .item-offer-comparison details.item-tools');
  await expect(tools).toBeVisible();
  if (!(await tools.evaluate(element => element.open))) await tools.locator('summary').click();
  return { tools, first, second };
}

test('advanced shop filters persist and each active chip can remove its filter', async ({ page }) => {
  await page.goto('/shop');
  await openAdvancedFilters(page);
  await page.locator('#shopStatCode').selectOption('STR');
  const strength = await page.evaluate(() => shopCards.map(card => {
    const item = JSON.parse(card.querySelector('[data-item-inspect]').dataset.itemInspect);
    return (item.stat_details || item.stats).find(stat => (stat.code || stat.label) === 'STR')?.value || 0;
  }));
  const minimum = Math.max(...strength);
  expect(minimum).toBeGreaterThan(0);
  expect(strength.some(value => value < minimum)).toBe(true);
  await page.locator('#shopMinStat').fill(String(minimum));
  await expect(page.locator('#shopActiveFilters button')).toHaveCount(1);
  const cards = await page.locator('.shop-card:visible').evaluateAll(elements => elements.map(card => {
    const item = JSON.parse(card.querySelector('[data-item-inspect]').dataset.itemInspect);
    return (item.stat_details || item.stats).find(stat => (stat.code || stat.label) === 'STR')?.value || 0;
  }));
  expect(cards.length).toBeGreaterThan(0);
  expect(cards.every(str => str >= minimum)).toBe(true);
  await page.reload();
  await openAdvancedFilters(page);
  await expect(page.locator('#shopStatCode')).toHaveValue('STR');
  await expect(page.locator('#shopMinStat')).toHaveValue(String(minimum));
  await page.locator('#shopActiveFilters button').click();
  await expect(page.locator('#shopMinStat')).toHaveValue('');
  await expect(page.locator('#shopActiveFilters button')).toHaveCount(0);

  const values = {
    shopMinStat: '10', shopMinXP: '0.001', shopMinRegen: '0.1', shopMinSockets: '1',
  };
  for (const [id, value] of Object.entries(values)) await page.locator(`#${id}`).fill(value);
  await page.locator('#shopPreserveStat').selectOption('MNA');
  for (const id of ['shopSetFilter', 'shopEffectFilter']) {
    const option = await page.locator(`#${id} option`).evaluateAll(options => options.find(option => option.value)?.value);
    if (option) {
      await page.locator(`#${id}`).selectOption(option);
      values[id] = option;
    }
  }
  await page.reload();
  await openAdvancedFilters(page);
  for (const [id, value] of Object.entries(values)) await expect(page.locator(`#${id}`)).toHaveValue(value);
  await expect(page.locator('#shopPreserveStat')).toHaveValue('MNA');
  await page.locator('#shopReset').click();
  for (const id of [...Object.keys(values), 'shopPreserveStat']) await expect(page.locator(`#${id}`)).toHaveValue('');
  await expect(page.locator('#shopActiveFilters button')).toHaveCount(0);
  await expect(page.locator('.shop-card:visible')).toHaveCount(12);
});

test('impossible minimums exclude offers and preserve-stat choices cover every attribute', async ({ page }) => {
  await page.goto('/shop');
  await openAdvancedFilters(page);
  const codes = await page.locator('#shopPreserveStat option').evaluateAll(options => options.map(option => option.value));
  expect(codes).toEqual(['', 'HP', 'MNA', 'STR', 'DEF', 'SPD', 'CRT', 'DGE', 'LCK', 'INT', 'STA', 'CHA', 'STN', 'SHN', 'HGR']);
  for (const id of ['shopMinStat', 'shopMinXP', 'shopMinRegen', 'shopMinSockets']) {
    await page.locator(`#${id}`).fill('1000000000');
    await expect(page.locator('.shop-card:visible')).toHaveCount(0);
    await expect(page.locator('#shopEmpty')).toBeVisible();
    await page.locator(`#${id}`).fill('');
    await expect(page.locator('.shop-card:visible')).toHaveCount(12);
  }
});

test('value sorts rank all matching offers and gain value excludes nonpositive gains', async ({ page }) => {
  await page.goto('/shop');
  const initial = await page.evaluate(() => shopCards.map(card => Number(card.dataset.gain)));
  expect(initial.some(gain => gain <= 0)).toBe(true);
  expect(initial.some(gain => gain > 0)).toBe(true);
  for (const [sort, metric] of [['value_power', 'power'], ['value_gain', 'gain']]) {
    await page.locator('#shopSort').selectOption(sort);
    await showEveryMatch(page);
    const cards = await page.locator('.shop-card:visible').evaluateAll((elements, key) => elements.map(card => ({
      metric: Number(card.dataset[key]), price: Number(card.dataset.price),
    })), metric);
    expect(cards.length).toBeGreaterThan(1);
    expect(cards.every(card => card.price > 0)).toBe(true);
    if (sort === 'value_gain') {
      expect(cards.every(card => card.metric > 0)).toBe(true);
      expect(cards.length).toBe(initial.filter(gain => gain > 0).length);
    }
    const values = cards.map(card => card.metric / card.price);
    expect(values).toEqual([...values].sort((a, b) => b - a));
  }
});

test('pinned offers compare their own numbers and expose all fourteen stat rows', async ({ page }) => {
  const errors = [];
  page.on('pageerror', error => errors.push(error.message));
  await page.goto('/shop');
  const { tools, first, second } = await openPinnedComparison(page);
  await tools.getByLabel('Show unchanged', { exact: true }).check();
  await tools.getByLabel('Show flavour', { exact: true }).check();
  const table = tools.locator('table.item-stat-table');
  await expect(table.locator('thead')).toContainText('Current');
  await expect(table.locator('thead')).toContainText('Candidate');
  await expect(table.locator('thead')).toContainText('Difference');
  await expect(table.locator('tbody tr')).toHaveCount(14);
  const rows = await table.locator('tbody tr').evaluateAll(elements => elements.map(row => ({
    code: row.querySelector('th').textContent.replace(' (flavour)', '').replace('%', ''),
    values: [...row.querySelectorAll('td')].slice(0, 3).map(cell => Number(cell.querySelector('[title]').title)),
  })));
  const stat = (item, code) => (item.stat_details || item.stats).find(row => (row.code || row.label.replace('%', '')) === code)?.value || 0;
  for (const row of rows) {
    expect(row.values, row.code).toEqual([stat(first, row.code), stat(second, row.code), stat(second, row.code) - stat(first, row.code)]);
  }
  await tools.getByLabel('Show flavour', { exact: true }).uncheck();
  await expect(table.locator('tbody tr')).toHaveCount(10);
  await tools.getByLabel('Show unchanged', { exact: true }).uncheck();
  const deltas = await table.locator('tbody tr:has(th)').evaluateAll(elements => elements.map(row => Number(row.querySelectorAll('td')[2].querySelector('[title]').title)));
  expect(deltas.every(delta => delta !== 0)).toBe(true);
  expect(errors).toEqual([]);
});

test('comparison diagnostic downloads useful JSON without account or authorization fields', async ({ page }) => {
  await page.goto('/shop');
  const { tools, first, second } = await openPinnedComparison(page);
  const downloadPromise = page.waitForEvent('download');
  await tools.getByRole('button', { name: 'Export diagnostic', exact: true }).click();
  const download = await downloadPromise;
  expect(download.suggestedFilename()).toMatch(/\.json$/);
  const stream = await download.createReadStream();
  const chunks = [];
  for await (const chunk of stream) chunks.push(chunk);
  const payload = JSON.parse(Buffer.concat(chunks).toString('utf8'));
  expect(payload.version).toBe('item-contributions-v2');
  expect(payload.status).toBeTruthy();
  expect(payload.all_stats).toHaveLength(14);
  expect(payload.inputs.map(item => item.name)).toEqual([first.name, second.name]);
  for (const row of payload.all_stats) {
    const value = item => (item.stat_details || item.stats).find(stat => (stat.code || stat.label.replace('%', '')) === row.code)?.value || 0;
    expect([row.before, row.after, row.delta], `exported ${row.code}`).toEqual([value(first), value(second), value(second) - value(first)]);
  }
  const forbidden = [];
  function inspect(value, path = '$') {
    if (!value || typeof value !== 'object') return;
    for (const [key, child] of Object.entries(value)) {
      if (/(?:token|(?:^|_)uid(?:$|_)|session|authorization|cookie)/i.test(key)) forbidden.push(`${path}.${key}`);
      inspect(child, `${path}.${key}`);
    }
  }
  inspect(payload);
  expect(forbidden).toEqual([]);
});

test('rejected clipboard access exposes focused manual comparison text', async ({ page }) => {
  await page.addInitScript(() => {
    Object.defineProperty(navigator, 'clipboard', {
      configurable: true,
      value: { writeText: () => Promise.reject(new DOMException('Clipboard denied', 'NotAllowedError')) },
    });
  });
  await page.goto('/shop');
  const { tools } = await openPinnedComparison(page);
  await tools.getByRole('button', { name: 'Copy comparison', exact: true }).click();
  const fallback = tools.getByRole('textbox', { name: 'Copy this comparison manually' });
  await expect(fallback).toBeVisible();
  await expect(fallback).toBeFocused();
  await expect(fallback).toHaveAttribute('readonly', '');
  const text = await fallback.inputValue();
  expect(text).toContain('tradeoff');
  expect(text).toContain('→');
  expect(text).not.toMatch(/undefined|NaN/);
  expect(await fallback.evaluate(element => element.selectionEnd - element.selectionStart)).toBe(text.length);
  await expect(page.locator('#itemToolsStatus')).toContainText('Select and copy');
});

for (const source of [
  { name: 'inventory', path: '/inventory', trigger: '.inv-card[data-id="1"] .inv-inspect', reference: 'inv:1', itemName: 'Lucky Test Charm' },
  { name: 'equipped', path: '/armory-fixture', trigger: '.gear-cell.item-inspect-trigger', reference: 'equipped:MainHand' },
]) {
  test(`${source.name} inspector opens forge with the selected instance`, async ({ page }) => {
    await page.goto(source.path);
    await page.locator(source.trigger).first().click();
    const inspector = page.locator('.item-inspector');
    await expect(inspector).toBeVisible();
    const opened = page.waitForURL('**/abyss?gear=1&forge=1');
    await inspector.getByRole('button', { name: 'Open in forge', exact: true }).click();
    await opened;
    await expect(page.locator('#forgeItemSelect')).toBeVisible();
    await expect(page.locator('#forgeItemSelect')).toHaveValue(source.reference);
    if (source.itemName) await expect(page.locator('#forgeItemSelect option:checked')).toContainText(source.itemName);
    expect(await page.evaluate(() => sessionStorage.getItem('itemForgeSelection'))).toBeNull();
  });
}

test('confirmed shop auto-equip refreshes comparisons without losing scroll or filters', async ({ page }) => {
  let navigationCount = 0;
  let comparisonRefreshes = 0;
  page.on('request', request => {
    if (request.isNavigationRequest() && new URL(request.url()).pathname === '/shop') navigationCount++;
  });
  await page.route('**/shop', async route => {
    if (route.request().headers()['x-requested-with'] !== 'item-comparison-refresh') return route.continue();
    comparisonRefreshes++;
    const response = await route.fetch();
    const original = await response.text();
    expect(original).toContain('Compared with');
    await route.fulfill({ response, body: original.replaceAll('Compared with', 'Compared with refreshed equipment:') });
  });
  await page.route('**/api/shop/buy', route => route.fulfill({ json: {
    ok: true, bought: 'Fixture purchase', gold: 20_000_000, equipped: true,
  } }));
  await page.goto('/shop');
  await openAdvancedFilters(page);
  await page.locator('#shopSlot').selectOption('Head');
  await page.locator('#shopSort').selectOption('price');
  await page.locator('#shopMinStat').fill('1');
  const buy = page.locator('.shop-card:visible .shop-buy-action').last();
  await buy.scrollIntoViewIfNeeded();
  const position = await page.evaluate(() => window.scrollY);
  expect(position).toBeGreaterThan(100);
  await buy.click();
  await page.getByRole('dialog').getByRole('button', { name: /^Buy for/ }).click();
  await expect(page.locator('#itemToolsStatus')).toContainText('Equipment comparisons updated after purchase');
  expect(comparisonRefreshes).toBe(1);
  expect(navigationCount).toBe(1);
  await expect(page.locator('#shopSlot')).toHaveValue('Head');
  await expect(page.locator('#shopSort')).toHaveValue('price');
  await expect(page.locator('#shopMinStat')).toHaveValue('1');
  const cards = page.locator('.shop-card:visible');
  expect(await cards.count()).toBeGreaterThan(0);
  for (const comparison of await cards.locator('.shop-comparison').all()) await expect(comparison).toContainText('Compared with refreshed equipment:');
  expect(Math.abs(await page.evaluate(() => window.scrollY) - position)).toBeLessThanOrEqual(2);
  await expect(cards.first().locator('.shop-buy-action')).toBeEnabled();
});
