const { test, expect } = require('@playwright/test');

const SELF = 'ally:visual-tester';
const BOSS = 'enemy:visual-warden';
const ADD = 'enemy:visual-echo';

function planningState(overrides = {}) {
  return {
    ok: true, session_id: 'visual-quickwins-e2e', phase: 'planning', round: 3, version: 10,
    deadline: new Date(Date.now() + 60000).toISOString(), tactic: 'balanced', pause_mode: 'adaptive',
    policy: {}, action_budget: { limit: 64, remaining: 64 },
    allies: [{ id: SELF, entity_id: SELF, name: 'Visual Tester', hp: 800, max_hp: 1000,
      mana: 100, max_mana: 100, is_self: true, is_player: true, position: 'frontline' }],
    enemies: [
      { id: 'enemy:0', entity_id: BOSS, name: 'Visual Warden', hp: 900, max_hp: 1000, role: 'boss', effects: [] },
      { id: 'enemy:1', entity_id: ADD, name: 'Visual Echo', hp: 500, max_hp: 500, effects: [] },
    ],
    options: [
      { kind: 'attack', id: '', name: 'Basic Attack', target: 'enemy', cooldown: 0 },
      { kind: 'defend', id: '', name: 'Defend', target: 'self', cooldown: 0 },
      { kind: 'skill', id: 'arc-bolt', name: 'Arc Bolt', target: 'enemy', mana: 15, cooldown: 0 },
    ],
    recent_logs: [], log_cursor: 0, initiative: [], enemy_intents: [], social: {},
    presentation_events: [], presentation_cursor: 0,
    ...overrides,
  };
}

function combatEvent(seq, targets, overrides = {}) {
  return { seq, round: 3, kind: 'skill', actor_id: SELF, ability_id: 'arc-bolt',
    ability_name: 'Arc Bolt', element: 'storm', targets, ...overrides };
}

async function startCombat(page, state) {
  await page.evaluate(snapshot => {
    window.connectLiveCombat = function noTransportForVisualTest() {};
    window.startLiveCombat({ state: snapshot }, snapshot.allies[0].hp, snapshot.allies[0].max_hp);
  }, state);
  await expect(page.locator('#liveCombat')).toBeVisible();
}

async function openCombat(page, state = planningState()) {
  await page.route('**/api/abyss/combat/state', route => route.fulfill({ json: { ok: false } }));
  await page.goto('/abyss?active=1');
  await startCombat(page, state);
}

async function render(page, state) {
  await page.evaluate(snapshot => window.renderLiveCombat(snapshot), state);
}

async function consumed(page, sequence) {
  await expect(page.locator('#livePixelStage')).toHaveAttribute('data-last-event-seq', String(sequence));
  await expect(page.locator('#livePixelStage')).toHaveAttribute('data-presentation-state', /^(idle|catchup)$/);
}

function unit(page, id) {
  return page.locator(`.ab-pixel-unit[data-entity-id="${id}"]`);
}

async function settings(page) {
  const drawer = page.locator('#liveSettingsDrawer');
  if (!await drawer.evaluate(node => node.open)) await drawer.locator(':scope > summary').click();
}

async function readSettings(page) {
  return page.evaluate(() => ({
    quality: document.getElementById('liveVisualQuality').value,
    contrast: document.getElementById('liveVisualContrast').value,
    numbers: document.getElementById('liveVisualNumbers').value,
    particles: document.getElementById('liveVisualParticles').checked,
    backdrop: document.getElementById('liveVisualBackdrop').value,
  }));
}

test('visual preferences update the stage and survive a reload', async ({ page }) => {
  const state = planningState();
  await openCombat(page, state);
  await expect.poll(() => page.evaluate(() => typeof window.AbyssFightVisuals)).toBe('object');
  await settings(page);
  await page.locator('#liveVisualQuality').selectOption('low');
  await page.locator('#liveVisualContrast').selectOption('high');
  await page.locator('#liveVisualNumbers').selectOption('large');
  await page.locator('#liveVisualParticles').uncheck();
  await page.locator('#liveVisualBackdrop').focus();
  await page.locator('#liveVisualBackdrop').press('Home');
  await page.locator('#liveVisualBackdrop').press('ArrowRight');
  const saved = await readSettings(page);
  await expect(page.locator('#livePixelStage')).toHaveAttribute('data-visual-quality', 'low');
  await expect(page.locator('#livePixelStage')).toHaveAttribute('data-visual-contrast', 'high');
  await expect(page.locator('#livePixelStage')).toHaveAttribute('data-visual-numbers', 'large');
  await page.reload();
  await startCombat(page, state);
  expect(await readSettings(page)).toEqual(saved);
  await expect(page.locator('#livePixelStage')).toHaveAttribute('data-visual-quality', 'low');
  await expect(page.locator('#livePixelStage')).toHaveAttribute('data-visual-contrast', 'high');
  await expect(page.locator('#livePixelStage')).toHaveAttribute('data-visual-numbers', 'large');
});

test('reset restores visual defaults without changing authoritative combat state', async ({ page }) => {
  const initial = planningState();
  await openCombat(page, initial);
  await settings(page);
  await expect(page.locator('#liveVisualReset')).toBeVisible();
  const defaults = await readSettings(page);
  await page.locator('#liveVisualQuality').selectOption('full');
  await page.locator('#liveVisualContrast').selectOption('high');
  await page.locator('#liveVisualNumbers').selectOption('large');
  await page.locator('#liveVisualParticles').uncheck();
  await page.locator('#liveVisualBackdrop').focus();
  await page.locator('#liveVisualBackdrop').press('End');
  await page.locator('#liveVisualReset').click();
  expect(await readSettings(page)).toEqual(defaults);
  expect(await page.evaluate(() => ({ version: liveCombatState.version, deadline: liveCombatState.deadline,
    hp: liveCombatState.allies[0].hp, mana: liveCombatState.allies[0].mana })))
    .toEqual({ version: initial.version, deadline: initial.deadline, hp: 800, mana: 100 });
});

test('skip consumes the cosmetic queue immediately while preserving server health', async ({ page }) => {
  const initial = planningState();
  await openCombat(page, initial);
  await expect(page.locator('#liveAnimationSkip')).toBeAttached();
  const next = structuredClone(initial);
  next.version++;
  next.allies[0].hp = 720;
  next.enemies[0].hp = 640;
  next.presentation_cursor = 20;
  next.presentation_events = Array.from({ length: 20 }, (_, index) =>
    combatEvent(index + 1, [{ target_id: BOSS, damage: 13 }]));
  const skipped = await page.evaluate(snapshot => {
    window.renderLiveCombat(snapshot);
    const status = document.getElementById('liveVisualPlayback').textContent;
    document.getElementById('liveAnimationSkip').click();
    const stage = document.getElementById('livePixelStage');
    return { status, sequence: stage.dataset.lastEventSeq, playback: stage.dataset.presentationState };
  }, next);
  expect(skipped.status.trim()).not.toBe('');
  expect(skipped.sequence).toBe('20');
  expect(skipped.playback).toMatch(/^(idle|catchup)$/);
  await consumed(page, 20);
  await expect(page.locator('#hpBar')).toHaveAttribute('aria-valuenow', '720');
  await expect(unit(page, BOSS)).toHaveAttribute('aria-label', /64 percent health/);
  expect(await page.evaluate(() => liveCombatState.deadline)).toBe(initial.deadline);
});

test('history retains only the latest twelve events and the current ability caption', async ({ page }) => {
  const initial = planningState();
  await openCombat(page, initial);
  const next = { ...initial, version: 11, presentation_cursor: 16,
    presentation_events: Array.from({ length: 16 }, (_, index) => combatEvent(index + 1,
      [{ target_id: BOSS, damage: 1 }], { ability_name: 'VisualPulse-' + String(index + 1).padStart(2, '0') })) };
  await render(page, next);
  await consumed(page, 16);
  const history = page.locator('#liveVisualHistory');
  await expect(history.locator('li')).toHaveCount(12);
  await expect(history).not.toContainText('VisualPulse-01');
  await expect(history).toContainText('VisualPulse-05');
  await expect(history).toContainText('VisualPulse-16');
  await expect(page.locator('#liveVisualCaption')).toContainText('VisualPulse-16');
  await render(page, next);
  await expect(history.locator('li')).toHaveCount(12);
});

test('hidden-health targets never disclose numeric damage in visual history', async ({ page }) => {
  const initial = planningState();
  initial.enemies[0].hp_hidden = true;
  await openCombat(page, initial);
  const next = { ...initial, version: 11, presentation_cursor: 1,
    presentation_events: [combatEvent(1, [{ target_id: BOSS, damage: 987654, damaged: true }],
      { ability_name: 'Veiled strike' })] };
  await render(page, next);
  await consumed(page, 1);
  await expect(page.locator('#liveVisualHistory li')).toHaveCount(1);
  await expect(page.locator('#liveVisualHistory')).toContainText('Veiled strike');
  await expect(page.locator('#liveVisualHistory')).not.toContainText(/987[,.\s]?654/);
  await expect(unit(page, BOSS)).toHaveAttribute('aria-label', /health concealed/);
});

test('critical health labels use the twenty percent boundary without exposing hidden health', async ({ page }) => {
  const initial = planningState();
  initial.allies[0].hp = 200;
  initial.enemies[0].hp = 1;
  initial.enemies[0].hp_hidden = true;
  await openCombat(page, initial);
  await expect(unit(page, SELF)).toHaveAttribute('data-health', 'critical');
  await expect(unit(page, SELF).locator('.ab-health-state')).toContainText(/critical/i);
  await expect(unit(page, BOSS).locator('.ab-health-state').filter({ hasText: /critical|1%/i })).toHaveCount(0);
  const next = structuredClone(initial);
  next.version++;
  next.allies[0].hp = 201;
  await render(page, next);
  await expect(unit(page, SELF)).not.toHaveAttribute('data-health', 'critical');
  await expect(unit(page, SELF).locator('.ab-health-state').filter({ hasText: /critical/i })).toHaveCount(0);
});

test('full-quality flourishes are bounded decorative elements that cannot intercept targets', async ({ page }) => {
  const initial = planningState();
  await openCombat(page, initial);
  await settings(page);
  await page.locator('#liveVisualQuality').selectOption('full');
  await page.locator('#liveVisualParticles').check();
  await page.locator('#liveSettingsDrawer [data-close-live-drawer]').click();
  await page.evaluate(() => {
    window.__visualDecorationAudit = { count: 0, maximum: 0, invalid: [] };
    const audit = window.__visualDecorationAudit;
    new MutationObserver(records => {
      audit.maximum = Math.max(audit.maximum, document.querySelectorAll('.ab-fight-flourish').length);
      for (const record of records) for (const node of record.addedNodes) {
        if (!(node instanceof Element)) continue;
        const decorations = [...node.querySelectorAll('.ab-fight-flourish')];
        if (node.matches('.ab-fight-flourish')) decorations.push(node);
        for (const decoration of decorations) {
          audit.count++;
          if (decoration.getAttribute('aria-hidden') !== 'true' || getComputedStyle(decoration).pointerEvents !== 'none') {
            audit.invalid.push(decoration.outerHTML);
          }
        }
      }
    }).observe(document.getElementById('livePixelStage'), { childList: true, subtree: true });
  });
  const next = { ...initial, version: 11, presentation_cursor: 8,
    presentation_events: Array.from({ length: 8 }, (_, index) => combatEvent(index + 1,
      [{ target_id: BOSS, damage: 10, critical: true }, { target_id: ADD, damage: 10 }])) };
  await render(page, next);
  await consumed(page, 8);
  const audit = await page.evaluate(() => window.__visualDecorationAudit);
  expect(audit.count).toBeGreaterThan(0);
  expect(audit.maximum).toBeLessThanOrEqual(36);
  expect(audit.invalid).toEqual([]);
  await unit(page, BOSS).locator('.ab-actor-sprite').click();
  await expect(page.locator('#liveTargetSelect')).toHaveValue('enemy:0');
});

test('reduced motion keeps portraits usable and a new session clears visual history', async ({ page }) => {
  await page.emulateMedia({ reducedMotion: 'reduce' });
  const initial = planningState();
  await openCombat(page, initial);
  await page.evaluate(() => {
    window.__movingVisualDecorations = [];
    new MutationObserver(() => {
      document.querySelectorAll('.ab-fight-flourish').forEach(node => node.getAnimations().forEach(animation => {
        if (animation.playState === 'running' && animation.effect.getKeyframes().some(frame =>
          frame.transform && frame.transform !== 'none')) window.__movingVisualDecorations.push(node.className);
      }));
    }).observe(document.getElementById('livePixelStage'), { childList: true, subtree: true });
  });
  await render(page, { ...initial, version: 11, presentation_cursor: 1,
    presentation_events: [combatEvent(1, [{ target_id: BOSS, damage: 10 }], { ability_name: 'Motion-safe bolt' })] });
  await consumed(page, 1);
  await expect(page.locator('#liveVisualHistory li')).toHaveCount(1);
  expect(await page.evaluate(() => window.__movingVisualDecorations)).toEqual([]);
  await unit(page, BOSS).locator('.ab-actor-sprite').click();
  await expect(page.locator('#liveTargetSelect')).toHaveValue('enemy:0');
  const fresh = planningState({ session_id: 'visual-quickwins-new-session', version: 1 });
  await startCombat(page, fresh);
  await expect(page.locator('#liveVisualHistory li')).toHaveCount(0);
  await expect(page.locator('#liveVisualCaption')).not.toContainText('Motion-safe bolt');
});

test('known positive health has a visible proportional bar fill', async ({ page }) => {
  await page.setViewportSize({ width: 1280, height: 720 });
  await openCombat(page);
  for (const [id, expected] of [[SELF, 0.8], [BOSS, 0.9], [ADD, 1]]) {
    const bar = unit(page, id).locator('.ab-overhead-hp');
    await expect.poll(() => bar.evaluate(node => {
      const fill = node.querySelector('i').getBoundingClientRect();
      return fill.width / node.clientWidth;
    })).toBeGreaterThan(expected - 0.03);
    const fill = await bar.evaluate(node => {
      const rect = node.querySelector('i').getBoundingClientRect();
      return { ratio: rect.width / node.clientWidth, height: rect.height };
    });
    expect(fill.ratio).toBeLessThanOrEqual(expected + 0.03);
    expect(fill.height).toBeGreaterThan(0);
  }
});

test('actor names and sprites fit below the caption through height-only viewport resizes', async ({ page }) => {
  await page.setViewportSize({ width: 1280, height: 720 });
  await openCombat(page);
  for (const height of [720, 900, 720]) {
    await page.setViewportSize({ width: 1280, height });
    await expect.poll(() => page.evaluate(() => {
      const stage = document.getElementById('livePixelStage').getBoundingClientRect();
      const caption = document.querySelector('#livePixelStage .ab-fight-presentation').getBoundingClientRect();
      const violations = [];
      document.querySelectorAll('#livePixelStage .ab-pixel-unit:not(.ab-departed)').forEach(actor => {
        for (const selector of ['.ab-pixel-name', '.ab-actor-sprite']) {
          const rect = actor.querySelector(selector).getBoundingClientRect();
          if (rect.width <= 0 || rect.height <= 0 || rect.top < caption.bottom - 1 ||
              rect.bottom > stage.bottom + 1 || rect.left < stage.left - 1 || rect.right > stage.right + 1) {
            violations.push({ actor: actor.dataset.entityId, selector, rect: rect.toJSON(),
              captionBottom: caption.bottom, stage: stage.toJSON() });
          }
        }
      });
      return violations;
    }), { message: 'Actor geometry at 1280x' + height }).toEqual([]);
  }
});

test('fire and frost accents retain distinct computed corner shapes', async ({ page }) => {
  const initial = planningState();
  await openCombat(page, initial);
  await page.evaluate(() => {
    window.__elementAccentRadii = {};
    new MutationObserver(records => {
      for (const record of records) for (const node of record.addedNodes) {
        if (!(node instanceof Element) || !node.matches('.ab-combat-effect')) continue;
        if (node.dataset.element === 'fire' || node.dataset.element === 'frost') {
          window.__elementAccentRadii[node.dataset.element] = getComputedStyle(node, '::after').borderRadius;
        }
      }
    }).observe(document.getElementById('livePixelStage'), { childList: true, subtree: true });
  });
  await render(page, { ...initial, version: 11, presentation_cursor: 2, presentation_events: [
    combatEvent(1, [{ target_id: BOSS, damage: 10 }], { ability_id: 'visual-fire', ability_name: 'Fire bolt', element: 'fire' }),
    combatEvent(2, [{ target_id: BOSS, damage: 10 }], { ability_id: 'visual-frost', ability_name: 'Frost bolt', element: 'frost' }),
  ] });
  await consumed(page, 2);
  const radii = await page.evaluate(() => window.__elementAccentRadii);
  expect(radii.fire).toBeTruthy();
  expect(radii.frost).toBeTruthy();
  expect(radii.fire).not.toBe(radii.frost);
});

test('malformed presentation sequences are ignored on initial and subsequent snapshots', async ({ page }) => {
  const errors = [];
  page.on('pageerror', error => errors.push(error.message));
  const malformed = [null, ...[0, 0.5, Infinity, -1].map(seq =>
    combatEvent(seq, [{ target_id: BOSS, damage: 10 }], { ability_name: 'Malformed event' }))];
  const initial = planningState({ presentation_events: malformed });
  await openCombat(page, initial);
  await consumed(page, 0);
  await expect(page.locator('#liveVisualHistory li')).toHaveCount(0);
  await render(page, { ...initial, version: 11, presentation_cursor: 1,
    presentation_events: [...malformed, combatEvent(1, [{ target_id: BOSS, damage: 10 }],
      { ability_name: 'Valid first event' })] });
  await consumed(page, 1);
  await expect(page.locator('#liveVisualHistory li')).toHaveCount(1);
  await expect(page.locator('#liveVisualHistory')).toContainText('Valid first event');
  await expect(page.locator('#liveVisualHistory')).not.toContainText('Malformed event');
  expect(errors).toEqual([]);
});

test('one newly accepted combat event immediately reports one pending effect', async ({ page }) => {
  const initial = planningState();
  await openCombat(page, initial);
  const pending = await page.evaluate(snapshot => {
    window.renderLiveCombat(snapshot);
    return document.getElementById('liveVisualPlayback').textContent;
  }, { ...initial, version: 11, presentation_cursor: 1,
    presentation_events: [combatEvent(1, [{ target_id: BOSS, damage: 10 }])] });
  expect(pending).toBe('1 effect pending');
  await consumed(page, 1);
});
