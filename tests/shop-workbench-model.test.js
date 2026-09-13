const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');
const test = require('node:test');

const context = { window: {} };
vm.createContext(context);
vm.runInContext(fs.readFileSync(path.join(__dirname, '..', 'internal', 'bot', 'webassets', 'shop_workbench.js'), 'utf8'), context);
const model = context.window.ShopWorkbench.model;
const plain = value => JSON.parse(JSON.stringify(value));
const MAX = Number.MAX_SAFE_INTEGER;

test('shop workbench calculations load without document or browser services', () => {
  assert.equal(typeof context.document, 'undefined');
  for (const name of ['integer', 'normalize', 'search', 'budget', 'buffCost', 'maxBuff', 'percentageAmount', 'plan', 'demand', 'compare']) {
    assert.equal(typeof model[name], 'function', name);
  }
});

test('integer accepts nonnegative safe numbers and numeric strings without losing precision', () => {
  for (const [input, expected] of [[0, 0], [42, 42], [MAX, MAX], ['0', 0], ['42', 42], [' 42 ', 42], ['1.0', 1], [String(MAX), MAX]]) {
    assert.equal(model.integer(input, -1), expected, String(input));
  }
});

test('integer rejects malformed, fractional, unsafe, and nonnumeric values', () => {
  for (const input of [-1, 1.5, NaN, Infinity, -Infinity, MAX + 1, '-2', '1.5', 'bad', '', '   ', null, undefined, true, false, [], {}]) {
    assert.equal(model.integer(input), 0, String(input));
    assert.equal(model.integer(input, 17), 17, String(input));
  }
});

test('normalization removes accents and collapses case and whitespace consistently', () => {
  assert.equal(model.normalize('  ÉPÉE\t DU\n FEU  '), 'epee du feu');
  assert.equal(model.normalize('E\u0301pe\u0301e du feu'), 'epee du feu');
  assert.equal(model.normalize('  Ruby   II  '), 'ruby ii');
  assert.equal(model.normalize(''), '');
  const once = model.normalize('  Glacé\n ARMOR  ');
  assert.equal(model.normalize(once), once);
});

test('search combines AND terms, exact phrases, exclusions, and accent normalization', () => {
  const text = 'Épée du Feu — rare sword, fire damage, no poison';
  for (const query of ['', 'epee rare', 'RARE epee', '"du feu" sword', 'sword -ice', '"fire damage" -"cold damage"']) {
    assert.equal(model.search(text, query), true, query);
  }
  for (const query of ['epee legendary', '"feu du"', 'sword -poison', '"fire damage" -"du feu"', '-rare']) {
    assert.equal(model.search(text, query), false, query);
  }
  assert.equal(model.search('fire bright sword', '"fire sword"'), false);
  assert.equal(model.search('fire sword', '"fire sword"'), true);
});

test('budget respects reserves and distinguishes blank limits from an explicit zero', () => {
  for (const limit of [undefined, null, '', '   ']) {
    assert.deepEqual(plain(model.budget(1000, 300, limit)), { available: 700, reserved: 300 });
  }
  assert.deepEqual(plain(model.budget(1000, 300, 500)), { available: 500, reserved: 300 });
  assert.deepEqual(plain(model.budget(1000, 300, 900)), { available: 700, reserved: 300 });
  assert.deepEqual(plain(model.budget(1000, 300, 0)), { available: 0, reserved: 300 });
  assert.deepEqual(plain(model.budget(1000, 300, '0')), { available: 0, reserved: 300 });
  assert.deepEqual(plain(model.budget(100, 300)), { available: 0, reserved: 100 });
  assert.deepEqual(plain(model.budget(0, 0)), { available: 0, reserved: 0 });
});

test('buff cost sums escalating prices exactly across the billion-gold cap boundary', () => {
  for (const [owned, amount, expected] of [
    [0, 1, 1_000_000], [0, 3, 6_000_000], [2, 3, 12_000_000],
    [997, 3, 2_997_000_000], [998, 1, 999_000_000], [998, 2, 1_999_000_000],
    [998, 3, 2_999_000_000], [999, 1, 1_000_000_000], [999, 3, 3_000_000_000],
    [1000, 3, 3_000_000_000], [0, 1000, 500_500_000_000],
  ]) assert.equal(model.buffCost(owned, amount), expected, `owned ${owned}, amount ${amount}`);
});

test('buff cost rejects invalid counts, ownership overflow, and unsafe total prices', () => {
  for (const [owned, amount] of [
    [-1, 1], [0, 0], [0, -1], [0.5, 1], [0, 1.5], [NaN, 1], [0, Infinity],
    [MAX, 1], [MAX - 1, 2], [0, MAX], [1000, 10_000_000],
  ]) assert.equal(model.buffCost(owned, amount), null, `owned ${owned}, amount ${amount}`);
});

test('maximum buff count brackets the wallet exactly including large capped purchases', () => {
  for (const [owned, wallet, expected] of [
    [0, 0, 0], [0, 999_999, 0], [0, 1_000_000, 1], [0, 5_999_999, 2], [0, 6_000_000, 3],
    [998, 1_998_999_999, 1], [998, 1_999_000_000, 2], [999, 3_000_000_000, 3],
    [1000, 9_000_000_000_000_000, 9_000_000],
  ]) {
    const amount = model.maxBuff(owned, wallet);
    assert.equal(amount, expected, `owned ${owned}, wallet ${wallet}`);
    assert.ok(Number.isSafeInteger(amount) && amount >= 0);
    if (amount > 0) assert.ok(model.buffCost(owned, amount) <= wallet);
    const next = model.buffCost(owned, amount + 1);
    assert.ok(next === null || next > wallet, 'the next count must be unaffordable or invalid');
  }
});

test('percentage amounts round down to complete steps and never overspend', () => {
  for (const [balance, percent, step, expected] of [
    [1000, 25, 1, 250], [1050, 50, 100, 500], [99, 100, 100, 0],
    [1099, 100, 100, 1000], [1000, 0, 10, 0], [1000, 150, 10, 1000], [1000, -10, 10, 0],
  ]) assert.equal(model.percentageAmount(balance, percent, step), expected);
  for (const balance of [0, 1, 999, 1_000_001, MAX]) {
    for (const percent of [0, 25, 50, 75, 100]) {
      for (const step of [1, 100, 1_000_000]) {
        const result = model.percentageAmount(balance, percent, step);
        assert.ok(Number.isSafeInteger(result) && result >= 0 && result <= balance, JSON.stringify({ balance, percent, step, result }));
        assert.equal(result % step, 0);
      }
    }
  }
});

test('plans total both currencies separately and leave reserves untouched', () => {
  const result = model.plan([
    { price: 100, currency: 'gold', quantity: 3 }, { price: 50, currency: 'gold', quantity: 2 },
    { price: 7, currency: 'tokens', quantity: 4 },
  ], { gold: 1000, tokens: 100 }, { gold: 200, tokens: 20 });
  for (const [key, value] of Object.entries({ gold: 400, tokens: 28, goldLeft: 400, tokenLeft: 52, goldShortfall: 0, tokenShortfall: 0, valid: true })) {
    assert.equal(result[key], value, key);
  }
  const empty = model.plan([], { gold: 1000, tokens: 100 }, { gold: 200, tokens: 20 });
  assert.equal(empty.valid, true);
  assert.equal(empty.goldLeft, 800);
  assert.equal(empty.tokenLeft, 80);
});

test('over-budget plans remain inspectable and expose separate nonnegative shortfalls', () => {
  const result = model.plan([
    { price: 500, currency: 'gold', quantity: 2 }, { price: 30, currency: 'tokens', quantity: 2 },
  ], { gold: 1000, tokens: 50 }, { gold: 200, tokens: 10 });
  assert.equal(result.valid, true);
  assert.equal(result.goldLeft, 0);
  assert.equal(result.tokenLeft, 0);
  assert.equal(result.goldShortfall, 200);
  assert.equal(result.tokenShortfall, 20);
  const excessReserve = model.plan([{ price: 5, currency: 'gold', quantity: 1 }], { gold: 10, tokens: 0 }, { gold: 100, tokens: 0 });
  assert.equal(excessReserve.valid, true);
  assert.equal(excessReserve.goldLeft, 0);
  assert.equal(excessReserve.goldShortfall, 5);
});

test('plans reject malformed arithmetic and both per-line and accumulated overflow', () => {
  const wallet = { gold: MAX, tokens: MAX };
  const reserves = { gold: 0, tokens: 0 };
  for (const lines of [
    [{ price: -1, currency: 'gold', quantity: 1 }], [{ price: NaN, currency: 'gold', quantity: 1 }],
    [{ price: Infinity, currency: 'tokens', quantity: 1 }], [{ price: 1, currency: 'gold', quantity: -1 }],
    [{ price: 1, currency: 'gold', quantity: 1.5 }], [{ price: 1, currency: 'gems', quantity: 1 }],
    [{ price: MAX, currency: 'gold', quantity: 2 }],
    [{ price: MAX, currency: 'gold', quantity: 1 }, { price: 1, currency: 'gold', quantity: 1 }],
  ]) assert.equal(model.plan(lines, wallet, reserves).valid, false, JSON.stringify(lines));
  for (const invalid of [-1, 0.5, NaN, Infinity, MAX + 1]) {
    assert.equal(model.plan([], { gold: invalid, tokens: 0 }, reserves).valid, false);
    assert.equal(model.plan([], wallet, { gold: 0, tokens: invalid }).valid, false);
  }
});

test('demand distinguishes savings, premiums, and zero-price baselines', () => {
  for (const [base, current, expected] of [
    [100, 75, { saving: 25, premium: 0, percent: -25 }],
    [100, 130, { saving: 0, premium: 30, percent: 30 }],
    [100, 100, { saving: 0, premium: 0, percent: 0 }],
    [0, 50, { saving: 0, premium: 50, percent: 0 }],
    [0, 0, { saving: 0, premium: 0, percent: 0 }],
  ]) assert.deepEqual(plain(model.demand(base, current)), expected);
});

test('comparison includes the union of stats with missing values zero and signed deltas', () => {
  const before = { stats: [{ label: 'STR', value: 10 }, { label: 'HP', value: -20 }, { label: 'CHA', value: 7 }] };
  const after = { stat_details: [{ code: 'STR', value: 15 }, { code: 'HP', value: -10 }, { code: 'MNA', value: 30 }] };
  const rows = plain(model.compare(before, after));
  assert.equal(rows.length, 4);
  const indexed = Object.fromEntries(rows.map(row => [row.code, row]));
  assert.deepEqual(indexed, {
    STR: { code: 'STR', before: 10, after: 15, delta: 5 }, HP: { code: 'HP', before: -20, after: -10, delta: 10 },
    CHA: { code: 'CHA', before: 7, after: 0, delta: -7 }, MNA: { code: 'MNA', before: 0, after: 30, delta: 30 },
  });
  const reverse = Object.fromEntries(plain(model.compare(after, before)).map(row => [row.code, row]));
  for (const row of rows) assert.equal(reverse[row.code].delta, -row.delta);
  assert.deepEqual(plain(model.compare({}, {})), []);
});

test('planning and comparison leave frozen input records and arrays unchanged', () => {
  const lines = Object.freeze([Object.freeze({ price: 10, currency: 'gold', quantity: 2 })]);
  const wallet = Object.freeze({ gold: 100, tokens: 10 });
  const reserves = Object.freeze({ gold: 20, tokens: 2 });
  const a = Object.freeze({ stats: Object.freeze([Object.freeze({ code: 'STR', value: 10 })]) });
  const b = Object.freeze({ stat_details: Object.freeze([Object.freeze({ code: 'STR', value: 20 })]) });
  const snapshot = JSON.stringify({ lines, wallet, reserves, a, b });
  model.plan(lines, wallet, reserves);
  const rows = model.compare(a, b);
  rows[0].after = 999;
  assert.equal(JSON.stringify({ lines, wallet, reserves, a, b }), snapshot);
  assert.equal(model.compare(a, b)[0].after, 20);
});
