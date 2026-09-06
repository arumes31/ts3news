const { test, expect } = require('@playwright/test');

const SELF = 'ally:animation-tester';
const BOSS = 'enemy:warden';
const ADD = 'enemy:echo';

function planningState(overrides = {}) {
  return {
    ok: true,
    session_id: 'animation-e2e',
    phase: 'planning',
    round: 3,
    version: 10,
    deadline: new Date(Date.now() + 60_000).toISOString(),
    tactic: 'balanced',
    pause_mode: 'adaptive',
    policy: {},
    action_budget: { limit: 64, remaining: 64 },
    allies: [{
      id: SELF, entity_id: SELF, name: 'Echo', hp: 800, max_hp: 1000,
      mana: 100, max_mana: 100, shield: 100, max_shield: 100,
      is_self: true, is_player: true, position: 'frontline',
    }],
    enemies: [
      { id: 'enemy:0', entity_id: BOSS, name: 'Echo', hp: 900, max_hp: 1000, role: 'boss', effects: [] },
      { id: 'enemy:1', entity_id: ADD, name: 'Echo', hp: 500, max_hp: 500, effects: [] },
    ],
    options: [
      { kind: 'attack', id: '', name: 'Basic Attack', target: 'enemy', cooldown: 0 },
      { kind: 'defend', id: '', name: 'Defend', target: 'self', cooldown: 0 },
      { kind: 'skill', id: 'arc-bolt', name: 'Arc Bolt', target: 'enemy', mana: 15, cooldown: 0 },
    ],
    recent_logs: [],
    log_cursor: 0,
    initiative: [],
    enemy_intents: [],
    social: {},
    presentation_events: [],
    presentation_cursor: 0,
    ...overrides,
  };
}

function combatEvent(seq, targets, overrides = {}) {
  return {
    seq, round: 3, kind: 'attack', actor_id: SELF,
    ability_id: '', ability_name: 'Basic Attack', element: 'physical',
    targets,
    ...overrides,
  };
}

async function openCombat(page, initial = planningState()) {
  // The fixture does not simulate the event stream. Stub only transport, then
  // feed snapshots through the same renderer used by SSE and polling.
  await page.route('**/api/abyss/combat/state', route => route.fulfill({
    status: 200, contentType: 'application/json', body: JSON.stringify({ ok: false }),
  }));
  await page.goto('/abyss?active=1');
  await page.evaluate(state => {
    window.connectLiveCombat = function noTransportForAnimationTest() {};
    window.startLiveCombat({ state }, 800, 1000);
  }, initial);
  await expect(page.locator('#liveCombat')).toBeVisible();
}

async function render(page, state) {
  await page.evaluate(snapshot => window.renderLiveCombat(snapshot), state);
}

function unit(page, entityID) {
  return page.locator(`.ab-pixel-unit[data-entity-id="${entityID}"]`);
}

async function observePresentation(page) {
  await page.evaluate(() => {
    window.__presentationObserver?.disconnect();
    window.__presentationRecords = [];
    const seen = new WeakSet();
    function record(element) {
      if (!(element instanceof Element) || seen.has(element)) return;
      if (element.matches('.ab-combat-effect, .ab-combat-number, .ab-combat-announcement')) {
        seen.add(element);
        window.__presentationRecords.push({
          kind: element.matches('.ab-combat-number') ? 'number' : element.matches('.ab-combat-announcement') ? 'announcement' : 'effect',
          seq: Number(element.dataset.eventSeq),
          actor: element.dataset.actorId || '',
          target: element.dataset.targetId || '',
          text: element.textContent,
          time: performance.now(),
        });
      }
      element.querySelectorAll('.ab-combat-effect, .ab-combat-number, .ab-combat-announcement').forEach(record);
    }
    window.__presentationObserver = new MutationObserver(records => {
      records.forEach(mutation => mutation.addedNodes.forEach(record));
    });
    window.__presentationObserver.observe(document.getElementById('livePixelStage'), { childList: true, subtree: true });
  });
}

async function records(page) {
  return page.evaluate(() => window.__presentationRecords);
}

async function consumed(page, sequence) {
  await expect(page.locator('#livePixelStage')).toHaveAttribute('data-last-event-seq', String(sequence));
  await expect(page.locator('#livePixelStage')).toHaveAttribute('data-presentation-state', /^(idle|catchup)$/);
}

test('structured events animate the named actor and targets in sequence despite identical display names', async ({ page }) => {
  const initial = planningState();
  await openCombat(page, initial);
  await observePresentation(page);
  const next = structuredClone(initial);
  next.version += 1;
  next.presentation_cursor = 2;
  next.presentation_events = [
    combatEvent(1, [{ target_id: BOSS, damage: 80 }]),
    combatEvent(2, [{ target_id: SELF, damage: 30 }], { actor_id: ADD }),
  ];
  // Deliberately ambiguous text must not choose the animation actor or target.
  next.recent_logs = ['Echo heals Echo. Echo casts a spell on Echo.'];
  next.enemies[0].hp -= 80;
  next.allies[0].hp -= 30;
  await render(page, next);
  await consumed(page, 2);
  const presentation = await records(page);
  const effects = presentation.filter(item => item.kind === 'effect');
  expect(effects).toEqual(expect.arrayContaining([
    expect.objectContaining({ seq: 1, actor: SELF, target: BOSS }),
    expect.objectContaining({ seq: 2, actor: ADD, target: SELF }),
  ]));
  expect(effects.findIndex(item => item.seq === 1)).toBeLessThan(effects.findIndex(item => item.seq === 2));
  expect(effects.every(item => item.seq === 1 ? item.actor === SELF && item.target === BOSS : item.actor === ADD && item.target === SELF)).toBe(true);
  expect(presentation.filter(item => item.kind === 'number')).toEqual(expect.arrayContaining([
    expect.objectContaining({ seq: 1, target: BOSS, text: expect.stringContaining('80') }),
    expect.objectContaining({ seq: 2, target: SELF, text: expect.stringContaining('30') }),
  ]));
});

test('damage, healing, absorption, block, and dodge remain visible distinct outcomes', async ({ page }) => {
  const initial = planningState();
  await openCombat(page, initial);
  await observePresentation(page);
  const next = structuredClone(initial);
  next.version += 1;
  next.presentation_cursor = 5;
  next.presentation_events = [
    combatEvent(1, [{ target_id: BOSS, damage: 125, critical: true }]),
    combatEvent(2, [{ target_id: SELF, healing: 40 }], { kind: 'skill', ability_id: 'mend', ability_name: 'Mend' }),
    combatEvent(3, [{ target_id: SELF, absorbed: 25 }], { actor_id: BOSS }),
    combatEvent(4, [{ target_id: SELF, blocked: true }], { actor_id: BOSS }),
    combatEvent(5, [{ target_id: SELF, dodged: true }], { actor_id: ADD }),
  ];
  next.enemies[0].hp -= 125;
  next.allies[0].hp += 40;
  next.allies[0].shield -= 25;
  await render(page, next);
  await consumed(page, 5);
  const numbers = (await records(page)).filter(item => item.kind === 'number');
  for (const [seq, target, text] of [
    [1, BOSS, /125/], [2, SELF, /\+40/], [3, SELF, /25/],
    [4, SELF, /block/i], [5, SELF, /dodg/i],
  ]) {
    expect(numbers.some(item => item.seq === seq && item.target === target && text.test(item.text)),
      `visible outcome for event ${seq}: ${JSON.stringify(numbers)}`).toBe(true);
  }
  await expect(unit(page, SELF)).toHaveAttribute('aria-label', /84 percent health/);
});

test('duplicate and stale snapshots cannot replay effects or roll back health', async ({ page }) => {
  const initial = planningState();
  await openCombat(page, initial);
  await observePresentation(page);
  const next = structuredClone(initial);
  next.version += 1;
  next.presentation_cursor = 1;
  next.presentation_events = [combatEvent(1, [{ target_id: BOSS, damage: 80 }])];
  next.enemies[0].hp = 820;
  await render(page, next);
  await consumed(page, 1);
  const firstRecords = await records(page);
  expect(firstRecords.some(item => item.seq === 1)).toBe(true);

  await render(page, structuredClone(next));
  await render(page, { ...next, version: next.version + 1 });
  await render(page, { ...initial, presentation_cursor: 2, presentation_events: [combatEvent(2, [{ target_id: BOSS, damage: 700 }])] });
  await consumed(page, 1);
  expect(await records(page)).toEqual(firstRecords);
  await expect(unit(page, BOSS)).toHaveAttribute('aria-label', /82 percent health/);
});

test('initial history and reconnect catch-up establish a baseline without replaying old combat', async ({ page }) => {
  const initial = planningState({
    presentation_cursor: 40,
    presentation_events: Array.from({ length: 40 }, (_, index) => combatEvent(index + 1, [{ target_id: BOSS, damage: 1 }])),
  });
  await openCombat(page, initial);
  await consumed(page, 40);
  await expect(page.locator('#livePixelStage .ab-combat-effect')).toHaveCount(0);
  await observePresentation(page);
  await page.evaluate(() => {
    window.setLiveConnection('reconnecting', 'RECONNECTING');
    window.setLiveConnection('synced', 'LIVE');
  });
  const catchup = { ...initial, version: 11, presentation_cursor: 42, presentation_events: [
    combatEvent(41, [{ target_id: BOSS, damage: 40 }]),
    combatEvent(42, [{ target_id: SELF, damage: 20 }], { actor_id: BOSS }),
  ] };
  await render(page, catchup);
  await consumed(page, 42);
  expect((await records(page)).filter(item => item.kind === 'effect')).toEqual([]);

  await render(page, { ...catchup, version: 12, presentation_cursor: 43, presentation_events: [
    ...catchup.presentation_events, combatEvent(43, [{ target_id: BOSS, damage: 30 }]),
  ] });
  await consumed(page, 43);
  expect([...new Set((await records(page)).filter(item => item.kind === 'effect').map(item => item.seq))]).toEqual([43]);
});

test('a new session cancels old work and accepts event numbers reused by the new fight', async ({ page }) => {
  const initial = planningState();
  await openCombat(page, initial);
  await observePresentation(page);
  const old = { ...initial, version: 11, presentation_cursor: 3, presentation_events: [
    combatEvent(1, [{ target_id: BOSS, damage: 10 }]),
    combatEvent(2, [{ target_id: BOSS, damage: 20 }]),
    combatEvent(3, [{ target_id: BOSS, damage: 30 }]),
  ] };
  await render(page, old);
  const fresh = planningState({ session_id: 'animation-new-session', version: 1 });
  await page.evaluate(state => window.startLiveCombat({ state }, 800, 1000), fresh);
  await consumed(page, 0);
  await observePresentation(page);
  await render(page, { ...old, version: 99 });
  await render(page, { ...fresh, version: 2, presentation_cursor: 1, presentation_events: [
    combatEvent(1, [{ target_id: ADD, damage: 17 }]),
  ] });
  await consumed(page, 1);
  const effects = (await records(page)).filter(item => item.kind === 'effect');
  expect(effects.length).toBeGreaterThan(0);
  expect(effects.every(item => item.seq === 1 && item.actor === SELF && item.target === ADD)).toBe(true);
  expect(await page.evaluate(() => window.liveCombatState.session_id)).toBe(fresh.session_id);
});

test('hidden-tab catch-up updates authoritative health without an animation backlog', async ({ page }) => {
  const initial = planningState();
  await openCombat(page, initial);
  await observePresentation(page);
  // Simulate the browser visibility boundary; no game state or animation
  // implementation is mocked.
  await page.evaluate(() => {
    Object.defineProperty(document, 'hidden', { configurable: true, get: () => true });
    Object.defineProperty(document, 'visibilityState', { configurable: true, get: () => 'hidden' });
    document.dispatchEvent(new Event('visibilitychange'));
  });
  const next = structuredClone(initial);
  next.version += 1;
  next.enemies[0].hp = 650;
  next.presentation_cursor = 20;
  next.presentation_events = Array.from({ length: 20 }, (_, index) => combatEvent(index + 1, [{ target_id: BOSS, damage: 10 }]));
  await render(page, next);
  await page.evaluate(() => {
    delete document.hidden;
    delete document.visibilityState;
    document.dispatchEvent(new Event('visibilitychange'));
  });
  await render(page, next);
  await consumed(page, 20);
  expect((await records(page)).filter(item => item.kind === 'effect')).toEqual([]);
  await expect(unit(page, BOSS)).toHaveAttribute('aria-label', /65 percent health/);
});

test('concealed enemy health never produces an inferred numeric damage label', async ({ page }) => {
  const initial = planningState();
  initial.enemies[0].hp_hidden = true;
  await openCombat(page, initial);
  await observePresentation(page);
  const next = structuredClone(initial);
  next.version += 1;
  next.enemies[0].hp = 100;
  next.presentation_cursor = 1;
  // The authoritative event deliberately withholds numeric damage.
  next.presentation_events = [combatEvent(1, [{ target_id: BOSS, blocked: true }])];
  next.recent_logs = ['Echo strikes Echo for 800 damage.'];
  await render(page, next);
  await consumed(page, 1);
  await expect(unit(page, BOSS)).toHaveAttribute('aria-label', /health concealed/);
  const output = (await records(page)).filter(item => item.target === BOSS && item.kind === 'number');
  expect(output.some(item => /block/i.test(item.text))).toBe(true);
  expect(output.some(item => /\d/.test(item.text))).toBe(false);
  await expect(unit(page, BOSS).locator('.ab-pixel-float')).toHaveCount(0);
});

test('boss phase and defeat presentation preserve the boss identity and authoritative outcome', async ({ page }) => {
  const initial = planningState();
  await openCombat(page, initial);
  await observePresentation(page);
  await unit(page, BOSS).evaluate(element => { window.__phaseBoss = element; });
  const enraged = structuredClone(initial);
  enraged.version += 1;
  enraged.presentation_cursor = 1;
  enraged.presentation_events = [combatEvent(1, [{ target_id: BOSS, status: 'enrage' }], {
    kind: 'phase', actor_id: BOSS, ability_id: 'enrage', ability_name: 'Enrage',
  })];
  enraged.enemies[0].effects = [{ name: 'Enrage', remaining: 3 }];
  await render(page, enraged);
  await consumed(page, 1);
  expect((await records(page)).some(item => item.seq === 1 && /enrage|phase/i.test(item.text))).toBe(true);
  expect(await unit(page, BOSS).evaluate(element => element === window.__phaseBoss)).toBe(true);

  const defeated = structuredClone(enraged);
  defeated.version += 1;
  defeated.enemies[0].hp = 0;
  defeated.presentation_cursor = 2;
  defeated.presentation_events.push(combatEvent(2, [{ target_id: BOSS, damage: 900, defeated: true }]));
  await render(page, defeated);
  await consumed(page, 2);
  await expect(unit(page, BOSS)).toHaveClass(/defeated/);
  await expect(unit(page, BOSS)).toBeDisabled();
  await expect(unit(page, ADD)).toBeEnabled();
  expect(await unit(page, BOSS).evaluate(element => element === window.__phaseBoss)).toBe(true);
});

test('a defeated enemy omitted by the next server snapshot remains visible for its final outcome', async ({ page }) => {
  const initial = planningState();
  await openCombat(page, initial);
  await observePresentation(page);
  await page.evaluate(({ boss, add }) => {
    window.__removedBoss = document.querySelector(`[data-entity-id="${boss}"]`);
    window.__survivingAdd = document.querySelector(`[data-entity-id="${add}"]`);
    const bounds = window.__survivingAdd.getBoundingClientRect();
    window.__survivingAddPosition = { x: bounds.x, y: bounds.y };
  }, { boss: BOSS, add: ADD });

  const next = structuredClone(initial);
  next.version += 1;
  next.round += 1;
  // prepareRound removes the dead boss and renumbers live action targets.
  // Presentation events still name the stable identity of the removed actor.
  next.enemies = [{ ...initial.enemies[1], id: 'enemy:0' }];
  next.presentation_cursor = 1;
  next.presentation_events = [combatEvent(1, [{ target_id: BOSS, damage: 900, defeated: true }])];
  await render(page, next);
  await expect(unit(page, BOSS)).toBeVisible();
  await expect(unit(page, BOSS)).toBeDisabled();
  await consumed(page, 1);
  await expect(unit(page, BOSS)).toBeVisible();
  await expect(unit(page, BOSS)).toHaveClass(/defeated/);
  const numbers = (await records(page)).filter(item => item.kind === 'number' && item.target === BOSS);
  expect(numbers.some(item => /900/.test(item.text))).toBe(true);
  expect(numbers.some(item => /defeated/i.test(item.text))).toBe(true);
  expect(await page.evaluate(({ boss, add }) => {
    const currentBoss = document.querySelector(`[data-entity-id="${boss}"]`);
    const currentAdd = document.querySelector(`[data-entity-id="${add}"]`);
    const bounds = currentAdd.getBoundingClientRect();
    return {
      sameBoss: currentBoss === window.__removedBoss,
      sameAdd: currentAdd === window.__survivingAdd,
      movement: Math.max(Math.abs(bounds.x - window.__survivingAddPosition.x), Math.abs(bounds.y - window.__survivingAddPosition.y)),
    };
  }, { boss: BOSS, add: ADD })).toMatchObject({ sameBoss: true, sameAdd: true, movement: 0 });
  await unit(page, ADD).click();
  expect(await page.evaluate(() => window.liveSelectedTarget)).toBe('enemy:0');
});

test('terminal event batches present all outcomes before the real combat teardown', async ({ page }) => {
  const initial = planningState();
  await openCombat(page, initial);
  await observePresentation(page);
  // Isolate the run-summary/economy boundary. Keep finishLiveCombat and its
  // actual dismissal timer, which must not cut off the presentation queue.
  await page.evaluate(() => { window.processDescendResult = async function noRunSummaryForAnimationTest() {}; });
  const complete = structuredClone(initial);
  complete.version += 1;
  complete.phase = 'complete';
  complete.result = { ok: true, victory: true };
  complete.presentation_cursor = 8;
  complete.presentation_events = Array.from({ length: 8 }, (_, index) => combatEvent(index + 1, [{
    target_id: BOSS, damage: 10 + index, defeated: index === 7,
  }]));
  complete.enemies[0].hp = 0;
  await render(page, complete);
  await expect(page.locator('#liveCombat')).toBeHidden();
  const numbers = (await records(page)).filter(item => item.kind === 'number');
  expect([...new Set(numbers.map(item => item.seq))].sort((a, b) => a - b)).toEqual([1, 2, 3, 4, 5, 6, 7, 8]);
  expect(numbers.some(item => item.seq === 8 && /defeated/i.test(item.text))).toBe(true);
});

async function selectPresentationPreference(page, selector, value) {
  const control = page.locator(selector);
  if (!await control.isVisible()) await page.locator('#liveSettingsDrawer > summary').click();
  await control.selectOption(value);
}

test('fast playback stays bounded, preserves every outcome, and persists without changing combat time', async ({ page }) => {
  const initial = planningState();
  await openCombat(page, initial);
  await selectPresentationPreference(page, '#liveAnimationSpeed', 'fast');
  await observePresentation(page);
  const next = structuredClone(initial);
  next.version += 1;
  next.presentation_cursor = 8;
  next.presentation_events = Array.from({ length: 8 }, (_, index) => combatEvent(index + 1, [{ target_id: BOSS, damage: index + 10 }]));
  // Measure playback inside the page. Separate driver calls and locator polling
  // otherwise count transport/screenshot overhead as animation time on slow hosts.
  const elapsed = await page.evaluate(snapshot => new Promise((resolve, reject) => {
    const stage = document.getElementById('livePixelStage');
    const start = performance.now();
    const timeout = setTimeout(() => { observer.disconnect(); reject(new Error('Playback did not finish')); }, 10_000);
    const observer = new MutationObserver(() => {
      if (stage.dataset.lastEventSeq !== '8' || !/^(idle|catchup)$/.test(stage.dataset.presentationState)) return;
      observer.disconnect(); clearTimeout(timeout); resolve(performance.now() - start);
    });
    observer.observe(stage, { attributes: true, attributeFilter: ['data-last-event-seq', 'data-presentation-state'] });
    window.renderLiveCombat(snapshot);
  }), next);
  await consumed(page, 8);
  expect(elapsed).toBeLessThan(2500);
  const numbers = (await records(page)).filter(item => item.kind === 'number');
  expect([...new Set(numbers.map(item => item.seq))].sort((a, b) => a - b)).toEqual([1, 2, 3, 4, 5, 6, 7, 8]);
  expect(await page.evaluate(() => window.liveCombatState.deadline)).toBe(initial.deadline);

  await page.reload();
  await page.evaluate(state => {
    window.connectLiveCombat = function noTransportForAnimationTest() {};
    window.startLiveCombat({ state }, 800, 1000);
  }, initial);
  await expect(page.locator('#liveAnimationSpeed')).toHaveValue('fast');
});

for (const preference of ['reduced effects', 'system reduced motion']) {
  test(`${preference} keeps outcomes readable without projectile or actor movement`, async ({ page }) => {
    if (preference === 'system reduced motion') await page.emulateMedia({ reducedMotion: 'reduce' });
    const initial = planningState();
    await openCombat(page, initial);
    if (preference === 'reduced effects') await selectPresentationPreference(page, '#liveAnimationEffects', 'reduced');
    await observePresentation(page);
    await page.evaluate(() => {
      window.__observedMovingAnimations = [];
      window.__observeAnimationMovement = setInterval(() => {
        document.getElementById('livePixelStage').getAnimations({ subtree: true }).forEach(animation => {
          if (animation.playState !== 'running' || !animation.effect) return;
          const frames = animation.effect.getKeyframes();
          if (frames.some(frame => frame.transform && frame.transform !== 'none'
            || frame.translate && frame.translate !== 'none')) {
            window.__observedMovingAnimations.push(animation.animationName || 'motion');
          }
        });
      }, 16);
    });
    const next = { ...initial, version: 11, presentation_cursor: 2, presentation_events: [
      combatEvent(1, [{ target_id: BOSS, damage: 60 }], { kind: 'skill', ability_id: 'arc-bolt', ability_name: 'Arc Bolt' }),
      combatEvent(2, [{ target_id: SELF, healing: 40 }], { kind: 'skill', ability_id: 'mend', ability_name: 'Mend' }),
    ] };
    await render(page, next);
    await consumed(page, 2);
    const motion = await page.evaluate(() => {
      clearInterval(window.__observeAnimationMovement);
      return window.__observedMovingAnimations;
    });
    expect(motion).toEqual([]);
    const numbers = (await records(page)).filter(item => item.kind === 'number');
    expect(numbers).toEqual(expect.arrayContaining([
      expect.objectContaining({ seq: 1, target: BOSS, text: expect.stringContaining('60') }),
      expect.objectContaining({ seq: 2, target: SELF, text: expect.stringContaining('40') }),
    ]));
  });
}

test('snapshot refresh retains combatant DOM, formation, and action focus', async ({ page }) => {
  const initial = planningState();
  await openCombat(page, initial);
  const attack = page.locator('#liveActionBar [data-action-key="attack:basic"]');
  await attack.focus();
  await page.evaluate(() => {
    window.__animationOriginalUnits = [...document.querySelectorAll('.ab-pixel-unit')];
    window.__animationOriginalAction = document.activeElement;
    window.__animationOriginalPositions = window.__animationOriginalUnits.map(element => {
      const bounds = element.getBoundingClientRect();
      return { x: bounds.x, y: bounds.y };
    });
  });

  const next = structuredClone(initial);
  next.version += 1;
  next.allies[0].hp = 700;
  next.enemies[0].hp = 750;
  await render(page, next);

  expect(await page.evaluate(() => {
    const current = [...document.querySelectorAll('.ab-pixel-unit')];
    return {
      sameUnits: current.every((element, index) => element === window.__animationOriginalUnits[index]),
      sameAction: document.activeElement === window.__animationOriginalAction,
      connectedAction: window.__animationOriginalAction.isConnected,
      movement: current.map((element, index) => {
        const bounds = element.getBoundingClientRect();
        const before = window.__animationOriginalPositions[index];
        return Math.max(Math.abs(bounds.x - before.x), Math.abs(bounds.y - before.y));
      }),
    };
  })).toMatchObject({ sameUnits: true, sameAction: true, connectedAction: true, movement: [0, 0, 0] });
  await expect(unit(page, SELF)).toHaveAttribute('aria-label', /70 percent health/);
  await expect(unit(page, BOSS)).toHaveAttribute('aria-label', /75 percent health/);
});

test('stable entity identity survives a changed action-routing index', async ({ page }) => {
  const initial = planningState();
  await openCombat(page, initial);
  await unit(page, ADD).evaluate(element => { window.__survivingCombatant = element; });

  const next = structuredClone(initial);
  next.version += 1;
  next.enemies = [{ ...initial.enemies[1], id: 'enemy:0' }];
  await render(page, next);

  await expect(page.locator('#livePixelEnemies .ab-pixel-unit:visible')).toHaveCount(1);
  await expect(unit(page, BOSS)).toBeHidden();
  expect(await unit(page, ADD).evaluate(element => element === window.__survivingCombatant)).toBe(true);
  await unit(page, ADD).click();
  expect(await page.evaluate(() => window.liveSelectedTarget)).toBe('enemy:0');
});

for (const selected of [ADD, BOSS]) {
  test(`target selection ${selected === ADD ? 'follows the surviving identity' : 'clears the removed identity'} when the server renumbers enemies`, async ({ page }) => {
    const initial = planningState();
    await openCombat(page, initial);
    await unit(page, selected).click();
    await page.locator('#liveSettingsDrawer > summary').click();
    await page.locator('#liveTargetLock').click();
    expect(await page.evaluate(() => window.liveTargetLocked)).toBe(true);

    const next = structuredClone(initial);
    next.version += 1;
    next.enemies = [{ ...initial.enemies[1], id: 'enemy:0' }];
    await render(page, next);

    expect(await page.evaluate(() => ({ target: window.liveSelectedTarget, locked: window.liveTargetLocked })))
      .toEqual(selected === ADD ? { target: 'enemy:0', locked: true } : { target: '', locked: false });
    await expect(unit(page, ADD)).toHaveAttribute('aria-pressed', selected === ADD ? 'true' : 'false');
    await expect(page.locator('#liveTargetLock')).toHaveAttribute('aria-pressed', selected === ADD ? 'true' : 'false');
  });
}

test('a death followed by revival in the same batch restores the living combatant', async ({ page }) => {
  const initial = planningState();
  await openCombat(page, initial);
  await observePresentation(page);
  await unit(page, SELF).evaluate(element => { window.__revivingCombatant = element; });
  const next = structuredClone(initial);
  next.version += 1;
  next.allies[0].hp = 200;
  next.presentation_cursor = 2;
  next.presentation_events = [
    combatEvent(1, [{ target_id: SELF, damage: 800, defeated: true }], { actor_id: BOSS }),
    combatEvent(2, [{ target_id: SELF, healing: 200, status: 'revived' }], {
      kind: 'status', actor_id: SELF, ability_id: 'phoenix', ability_name: 'Phoenix Revival',
    }),
  ];
  await render(page, next);
  await consumed(page, 2);
  await expect(unit(page, SELF)).not.toHaveClass(/ab-defeated/);
  await expect(unit(page, SELF)).toHaveAttribute('data-pose', 'idle');
  await expect(unit(page, SELF)).toBeEnabled();
  expect(await unit(page, SELF).evaluate(element => element === window.__revivingCombatant)).toBe(true);
  const numbers = (await records(page)).filter(item => item.kind === 'number' && item.target === SELF);
  expect(numbers.some(item => item.seq === 1 && /defeated/i.test(item.text))).toBe(true);
  expect(numbers.some(item => item.seq === 2 && /200/.test(item.text))).toBe(true);
  await unit(page, SELF).click();
  expect(await page.evaluate(() => window.liveSelectedTarget)).toBe(SELF);
});

for (const viewport of [{ width: 1440, height: 1000 }, { width: 1280, height: 720 }]) {
  test(`crowded structured combat keeps every actor and command inside the stage at ${viewport.width}x${viewport.height}`, async ({ page }) => {
    await page.setViewportSize(viewport);
    const initial = planningState();
    initial.allies.push(...Array.from({ length: 5 }, (_, index) => ({
      id: `ally:helper-${index}`, entity_id: `ally:helper-${index}`, name: `Helper ${index + 1}`,
      hp: 700, max_hp: 900, mana: 60, max_mana: 80,
      is_self: false, is_player: true, position: index % 2 ? 'backline' : 'frontline',
    })));
    initial.enemies.push(...Array.from({ length: 4 }, (_, index) => ({
      id: `enemy:${index + 2}`, entity_id: `enemy:crowded-${index}`, name: `Enemy ${index + 3}`,
      hp: 350, max_hp: 500, effects: [],
    })));
    initial.options.push(...Array.from({ length: 5 }, (_, index) => ({
      kind: 'skill', id: `crowded-skill-${index}`, name: `Crowded Skill ${index + 1}`,
      target: 'enemy', mana: 10, cooldown: 0,
    })));
    initial.queued = { kind: 'skill', ability_id: 'arc-bolt', target_id: 'enemy:0', round: initial.round };
    await openCombat(page, initial);
    await observePresentation(page);
    await expect(page.locator('#livePixelAllies .ab-pixel-unit:visible')).toHaveCount(6);
    await expect(page.locator('#livePixelEnemies .ab-pixel-unit:visible')).toHaveCount(6);
    const next = structuredClone(initial);
    next.version += 1;
    next.presentation_cursor = 1;
    next.presentation_events = [combatEvent(1, initial.enemies.map(enemy => ({ target_id: enemy.entity_id, damage: 30 })), {
      kind: 'skill', ability_id: 'arc-bolt', ability_name: 'Arc Bolt', element: 'storm',
    })];
    next.enemies.forEach(enemy => { enemy.hp -= 30; });
    await render(page, next);
    await consumed(page, 1);

    const geometry = await page.evaluate(() => {
      const stage = document.getElementById('livePixelStage').getBoundingClientRect();
      const actions = document.getElementById('liveActionBar').getBoundingClientRect();
      return {
        stage: stage.toJSON(), actions: actions.toJSON(),
        actors: [...document.querySelectorAll('.ab-pixel-unit:not(.ab-departed)')].map(element => {
          const sprite = element.querySelector('.ab-actor-sprite').getBoundingClientRect();
          return { entity: element.dataset.entityId, sprite: sprite.toJSON() };
        }),
      };
    });
    expect(geometry.actions.top).toBeGreaterThanOrEqual(geometry.stage.bottom - 1);
    expect(geometry.actions.bottom).toBeLessThanOrEqual(viewport.height + 1);
    for (const actor of geometry.actors) {
      expect(actor.sprite.width, actor.entity).toBeGreaterThan(0);
      expect(actor.sprite.height, actor.entity).toBeGreaterThan(0);
      expect(actor.sprite.left, actor.entity).toBeGreaterThanOrEqual(geometry.stage.left - 1);
      expect(actor.sprite.right, actor.entity).toBeLessThanOrEqual(geometry.stage.right + 1);
      expect(actor.sprite.top, actor.entity).toBeGreaterThanOrEqual(geometry.stage.top - 1);
      expect(actor.sprite.bottom, actor.entity).toBeLessThanOrEqual(geometry.stage.bottom + 1);
    }
    const numbers = (await records(page)).filter(item => item.kind === 'number');
    expect(new Set(numbers.filter(item => /30/.test(item.text)).map(item => item.target)))
      .toEqual(new Set(initial.enemies.map(enemy => enemy.entity_id)));
    await expect(page.locator('#liveQueue')).toContainText('Arc Bolt');
  });
}

for (const viewport of [
  { name: 'desktop', width: 1440, height: 1000 },
  { name: 'mobile', width: 390, height: 844 },
]) {
  test(`animations leave commands below the stage and usable on ${viewport.name}`, async ({ page }) => {
    await page.setViewportSize(viewport);
    const initial = planningState();
    await openCombat(page, initial);
    const requests = [];
    await page.route('**/api/abyss/combat/action', async route => {
      const action = route.request().postDataJSON();
      requests.push(action);
      await route.fulfill({
        status: 200, contentType: 'application/json',
        body: JSON.stringify({ ...next, version: 12, queued: action }),
      });
    });
    const next = structuredClone(initial);
    next.version = 11;
    next.presentation_cursor = 1;
    next.presentation_events = [combatEvent(1, [{ target_id: BOSS, damage: 80 }])];
    next.enemies[0].hp -= 80;
    await render(page, next);

    const geometry = await page.evaluate(() => {
      const stage = document.getElementById('livePixelStage').getBoundingClientRect();
      const actions = document.getElementById('liveActionBar').getBoundingClientRect();
      const button = document.querySelector('#liveActionBar [data-action-key="defend:basic"]');
      const bounds = button.getBoundingClientRect();
      const hit = document.elementFromPoint(bounds.x + bounds.width / 2, bounds.y + bounds.height / 2);
      return { stageBottom: stage.bottom, actionTop: actions.top, reachable: button === hit || button.contains(hit) };
    });
    expect(geometry.actionTop).toBeGreaterThanOrEqual(geometry.stageBottom - 1);
    expect(geometry.reachable).toBe(true);
    await page.locator('#liveActionBar [data-action-key="defend:basic"]').click();
    await expect.poll(() => requests.length).toBe(1);
    expect(requests[0]).toMatchObject({ kind: 'defend', round: initial.round });
    await expect.poll(() => page.evaluate(() => window.liveCombatState.queued?.kind)).toBe('defend');
    expect(await page.evaluate(() => window.liveCombatState.deadline)).toBe(initial.deadline);
  });
}

for (const discard of ['dense', 'gapped']) {
  test(`${discard} catch-up discards an omitted boss without preserving stale health or a pending corpse`, async ({ page }) => {
    const initial = planningState();
    await openCombat(page, initial);
    await observePresentation(page);
    const next = structuredClone(initial);
    next.version += 1;
    next.allies[0].hp = 400;
    next.enemies = [{ ...initial.enemies[1], id: 'enemy:0', hp: 250 }];
    next.presentation_events = discard === 'dense'
      ? Array.from({ length: 49 }, (_, index) => combatEvent(index + 1, [{ target_id: BOSS, damage: 10, defeated: index === 48 }]))
      : [combatEvent(5, [{ target_id: BOSS, damage: 900, defeated: true }])];
    next.presentation_cursor = discard === 'dense' ? 49 : 5;
    await render(page, next);
    await consumed(page, next.presentation_cursor);
    await expect(page.locator('#livePixelStage')).toHaveAttribute('data-presentation-state', 'catchup');
    await expect(unit(page, BOSS)).toBeHidden();
    await expect(unit(page, BOSS)).toBeDisabled();
    expect(await unit(page, BOSS).evaluate(element => !!element._pendingDefeat)).toBe(false);
    await expect(unit(page, ADD)).toHaveAttribute('aria-label', /50 percent health/);
    await expect(unit(page, SELF)).toHaveAttribute('aria-label', /40 percent health/);
    await expect(unit(page, ADD)).toHaveAttribute('data-pose', 'idle');
    expect(await records(page)).toEqual([]);
  });
}

test('concealed regeneration uses the public healing outcome without numbers or a false hit label', async ({ page }) => {
  const initial = planningState();
  initial.enemies[0].hp_hidden = true;
  await openCombat(page, initial);
  await observePresentation(page);
  const next = structuredClone(initial);
  next.version += 1;
  next.presentation_cursor = 3;
  next.presentation_events = [
    combatEvent(1, [{ target_id: BOSS, healed: true }], { kind: 'status', actor_id: BOSS, ability_name: 'Regeneration' }),
    combatEvent(2, [{ target_id: BOSS }], { kind: 'status', actor_id: BOSS, ability_name: 'Ward' }),
    combatEvent(3, [{ target_id: BOSS, damaged: true }], { kind: 'status', ability_name: 'Poison' }),
  ];
  await render(page, next);
  await consumed(page, 3);
  const numbers = (await records(page)).filter(item => item.kind === 'number' && item.target === BOSS);
  expect(numbers.filter(item => item.seq === 1).map(item => item.text)).toEqual(['HEAL']);
  expect(numbers.filter(item => item.seq === 2)).toEqual([]);
  expect(numbers.filter(item => item.seq === 3).map(item => item.text)).toEqual(['HIT']);
  expect(numbers.some(item => /\d/.test(item.text))).toBe(false);
});

test('an unavailable event target does not redirect its effect to the attacker', async ({ page }) => {
  const initial = planningState();
  await openCombat(page, initial);
  await observePresentation(page);
  const next = structuredClone(initial);
  next.version += 1;
  next.presentation_cursor = 1;
  next.presentation_events = [combatEvent(1, [
    { target_id: 'enemy:never-rendered', damage: 100 },
    { target_id: ADD, damage: 30 },
  ])];
  await render(page, next);
  await consumed(page, 1);
  const output = await records(page);
  expect(output.filter(item => item.target === 'enemy:never-rendered')).toEqual([]);
  expect(output.some(item => item.kind === 'effect' && item.actor === SELF && item.target === ADD)).toBe(true);
  expect(output.some(item => item.kind === 'number' && item.target === ADD && /30/.test(item.text))).toBe(true);
});


test('empowered subclass sprites persist through authoritative combat refresh and casting', async ({page}) => {
  const subs=['vanguard','berserker','marksman','beastmaster','elementalist','chronomancer','oracle','geomancer','bloodblade','voidwalker','runesmith','alchemist'];
  let state=planningState();
  state.allies[0]={...state.allies[0],class:'warrior',subclass:subs[0],weapon_type:'bow'};
  await openCombat(page,state);
  for (const [index,subclass] of subs.entries()) {
    state={...state,version:20+index*2,presentation_cursor:index+1,
      allies:[{...state.allies[0],subclass,weapon_type:'staff'}],
      presentation_events:[combatEvent(index+1,[{target_id:BOSS,damage:10}],{kind:'skill',ability_id:'CLASS_'+subclass+'_finish',ability_name:'Signature finish'})]};
    await render(page,state);
    await expect(unit(page,SELF).locator('.ab-actor-sprite')).toHaveAttribute('data-rig',subclass);
    await expect(unit(page,SELF).locator('.ab-actor-sprite')).toHaveCSS('background-image',/abyss_subclasses_/);
    await render(page,{...state,version:21+index*2,presentation_events:[]});
    await expect(unit(page,SELF).locator('.ab-actor-sprite')).toHaveAttribute('data-rig',subclass);
    state.version=21+index*2;
  }
});


test('live snapshots refresh overhead, player, boss and owned companion health without replaying events', async ({ page }) => {
  const state = planningState();
  state.allies.push({ id: 'pet:animation-tester:0', entity_id: 'pet:owned', name: 'Ember', hp: 400, max_hp: 500, role: 'Mind-controlled ally' });
  await openCombat(page, state);
  await expect(page.locator('#petHPBar')).toHaveAttribute('aria-valuenow', '400');
  const next = structuredClone(state);
  next.version += 1;
  next.allies[0].hp = 360;
  next.allies[1].hp = 120;
  next.enemies[0].hp = 250;
  await render(page, next);
  await expect(unit(page, SELF).locator('.ab-overhead-hp em')).toHaveText('36%');
  await expect(unit(page, BOSS).locator('.ab-overhead-hp em')).toHaveText('25%');
  await expect(page.locator('#hpBar')).toHaveAttribute('aria-valuenow', '360');
  await expect(page.locator('#petHPBar')).toHaveAttribute('aria-valuenow', '120');
  await expect(page.locator('#bossHPOverlay')).toContainText('250 / 1,000');
  await render(page, state);
  await expect(page.locator('#hpBar')).toHaveAttribute('aria-valuenow', '360');
  await expect(page.locator('#petHPBar')).toHaveAttribute('aria-valuenow', '120');
  await expect(page.locator('#bossHPOverlay')).toContainText('250 / 1,000');
  const noPet = structuredClone(next);
  noPet.version += 1;
  noPet.allies.pop();
  await render(page, noPet);
  await expect(page.locator('#petBar')).toBeHidden();
  await page.evaluate(() => { floorType = 'rest'; eventState = {}; renderState(); });
  await expect(page.locator('#bossHPOverlay')).toBeHidden();
});
