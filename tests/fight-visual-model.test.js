const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');
const test = require('node:test');

const context = { window: {} };
vm.createContext(context);
vm.runInContext(fs.readFileSync(path.join(__dirname, '..', 'internal', 'bot', 'webassets', 'abyss_fight_visuals.js'), 'utf8'), context);
const model = context.window.AbyssFightVisuals.model;

// Cross-realm records have different prototypes; compare their public values.
const plain = value => JSON.parse(JSON.stringify(value));
const defaults = { quality: 'auto', contrast: 'normal', numbers: 'normal', particles: true, backdrop: 45, linger: 900 };

test('the fight visual model loads without document or browser services', () => {
  assert.equal(typeof context.document, 'undefined');
  for (const name of ['preferences', 'health', 'impact', 'position', 'sequence']) assert.equal(typeof model[name], 'function', name);
});

test('preferences return safe defaults for missing and invalid settings', () => {
  for (const input of [undefined, null, {}, false, 'invalid', 42]) {
    assert.deepEqual(plain(model.preferences(input)), defaults);
  }
  assert.deepEqual(plain(model.preferences({
    quality: 'ultra', contrast: 'rainbow', numbers: 'tiny', particles: 'false', backdrop: NaN, linger: Infinity,
  })), defaults);
  assert.deepEqual(plain(model.preferences({ backdrop: Infinity, linger: NaN })), defaults);
});

test('preferences preserve valid choices and bound finite numeric settings', () => {
  for (const quality of ['auto', 'low', 'full']) {
    for (const contrast of ['normal', 'high']) {
      for (const numbers of ['normal', 'large']) {
        for (const particles of [false, true]) {
          const input = { quality, contrast, numbers, particles, backdrop: 60, linger: 1200 };
          assert.deepEqual(plain(model.preferences(input)), input);
        }
      }
    }
  }
  assert.deepEqual(plain(model.preferences({ backdrop: -20, linger: 1 })), { ...defaults, backdrop: 0, linger: 600 });
  assert.deepEqual(plain(model.preferences({ backdrop: 200, linger: 9000 })), { ...defaults, backdrop: 100, linger: 1800 });
  for (const [backdrop, linger] of [[0, 600], [100, 1800]]) {
    const result = model.preferences({ backdrop, linger });
    assert.equal(result.backdrop, backdrop);
    assert.equal(result.linger, linger);
  }
});

test('concealed health takes priority and never exposes numeric health in its label', () => {
  for (const unit of [
    { hp_hidden: true, hp: 7341, max_hp: 9812 },
    { hp_hidden: true, hp: 0, max_hp: 100 },
    { hp_hidden: true, hp: -5, max_hp: -1 },
    { hp_hidden: true, hp: Infinity, max_hp: Infinity },
  ]) {
    const result = model.health(unit);
    assert.equal(result.kind, 'concealed');
    assert.equal(result.percent, null);
    assert.equal(typeof result.label, 'string');
    assert.ok(result.label.length > 0);
    assert.doesNotMatch(result.label, /[0-9]/, 'concealed labels must not leak current, maximum, or percentage HP');
  }
});

test('invalid maximum health is concealed instead of producing misleading percentages', () => {
  for (const max_hp of [undefined, null, NaN, Infinity, -Infinity, 0, -100]) {
    const result = model.health({ hp: 10, max_hp });
    assert.equal(result.kind, 'concealed', String(max_hp));
    assert.equal(result.percent, null, String(max_hp));
    assert.doesNotMatch(result.label, /[0-9]/);
  }
});

test('health classifications obey exact thresholds and clamp out-of-range HP', () => {
  for (const [hp, kind, percent] of [
    [-100, 'defeated', 0], [0, 'defeated', 0], [1, 'critical', 1], [20, 'critical', 20],
    [21, 'hurt', 21], [50, 'hurt', 50], [51, 'healthy', 51], [100, 'healthy', 100], [200, 'healthy', 100],
  ]) {
    const result = model.health({ hp, max_hp: 100 });
    assert.equal(result.kind, kind, String(hp));
    assert.equal(result.percent, percent, String(hp));
    assert.equal(typeof result.label, 'string');
    assert.ok(result.label.length > 0);
  }
  for (const hp of [NaN, Infinity, -Infinity, undefined]) {
    const result = model.health({ hp, max_hp: 100 });
    assert.ok(result.percent === null || Number.isFinite(result.percent) && result.percent >= 0 && result.percent <= 100);
    assert.doesNotMatch(result.label, /NaN|Infinity/);
  }
});

test('impact classifies single outcomes and uses explicit priority for mixed flags', () => {
  const mixed = { status: 'revived', dodged: true, blocked: true, absorbed: 20, healing: 10, critical: true, damage: 50 };
  assert.equal(model.impact(mixed), 'revive');
  const stages = [
    ['status', 'dodge'], ['dodged', 'block'], ['blocked', 'absorb'], ['absorbed', 'heal'],
    ['healing', 'critical'], ['critical', 'damage'], ['damage', 'none'],
  ];
  const remaining = { ...mixed };
  for (const [remove, expected] of stages) {
    delete remaining[remove];
    assert.equal(model.impact(remaining), expected, `after removing ${remove}`);
  }
  for (const [target, expected] of [
    [{ status: 'stunned' }, 'status'], [{ damage: 1, status: 'stunned' }, 'damage'],
    [{ critical: true, damage: 1, status: 'stunned' }, 'critical'],
    [{ healing: 1 }, 'heal'], [{ absorbed: 1 }, 'absorb'], [{ blocked: true }, 'block'], [{ dodged: true }, 'dodge'],
    [{ damage: 0, healing: 0, absorbed: 0 }, 'none'], [{ damage: -1, healing: -1, absorbed: -1 }, 'none'],
    [{ damage: NaN, healing: NaN, absorbed: NaN }, 'none'], [{}, 'none'],
  ]) assert.equal(model.impact(target), expected, JSON.stringify(target));
});

test('position keeps every lane inside the viewport with available safety margins', () => {
  const dimensions = [[1280, 720], [390, 844], [96, 24], [70, 20], [1, 1]];
  const points = [{ x: 0, y: 0 }, { x: -1000, y: -1000 }, { x: 1e9, y: 1e9 }, { x: NaN, y: Infinity }, {}];
  for (const [width, height] of dimensions) {
    for (const point of points) {
      for (const lane of [-5, 0, 1, 2, 99]) {
        const result = model.position(point, width, height, lane);
        assert.ok(Number.isFinite(result.x) && Number.isFinite(result.y), JSON.stringify({ point, width, height, lane, result }));
        const xMargin = Math.min(48, width / 2);
        const yMargin = Math.min(12, height / 2);
        assert.ok(result.x >= xMargin && result.x <= width - xMargin, `x margin violated: ${JSON.stringify({ width, lane, result })}`);
        assert.ok(result.y >= yMargin && result.y <= height - yMargin, `y margin violated: ${JSON.stringify({ height, lane, result })}`);
      }
    }
  }
});

test('position remains finite when viewport dimensions or lanes are invalid', () => {
  for (const [width, height] of [[0, 0], [-1, -10], [NaN, Infinity], [undefined, undefined]]) {
    for (const lane of [NaN, Infinity, -Infinity, undefined]) {
      const result = model.position({ x: NaN, y: Infinity }, width, height, lane);
      assert.ok(Number.isFinite(result.x) && Number.isFinite(result.y), JSON.stringify(result));
    }
  }
});

test('sequence accepts only positive safe integers', () => {
  for (const value of [1, 2, 12345, Number.MAX_SAFE_INTEGER]) assert.equal(model.sequence(value), value);
  for (const value of [0, -0, -1, -12345, 0.5, 1.5, NaN, Infinity, -Infinity, Number.MAX_SAFE_INTEGER + 1, undefined, null]) {
    assert.equal(model.sequence(value), 0, String(value));
  }
});

test('model operations do not mutate their inputs or share mutable output state', () => {
  const preferences = Object.freeze({ quality: 'low', contrast: 'high', numbers: 'large', particles: false, backdrop: 66, linger: 1500 });
  const unit = Object.freeze({ hp: 20, max_hp: 100, hp_hidden: false });
  const target = Object.freeze({ status: 'revived', damage: 55, critical: true });
  const point = Object.freeze({ x: -10, y: 1200 });
  const before = JSON.stringify({ preferences, unit, target, point });
  const result = model.preferences(preferences);
  model.health(unit);
  model.impact(target);
  model.position(point, 1280, 720, 2);
  assert.equal(JSON.stringify({ preferences, unit, target, point }), before);
  assert.notEqual(result, preferences);
  result.quality = 'full';
  assert.equal(model.preferences(preferences).quality, 'low');
});
