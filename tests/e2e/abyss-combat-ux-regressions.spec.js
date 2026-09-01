const { test, expect } = require('@playwright/test');

function planningState(options = []) {
  return {
    ok: true,
    session_id: 'combat-ux-e2e',
    phase: 'planning',
    round: 3,
    version: 7,
    deadline: new Date(Date.now() + 60_000).toISOString(),
    tactic: 'balanced',
    pause_mode: 'adaptive',
    policy: {},
    action_budget: { limit: 64, remaining: 64 },
    recommended: {
      action: { kind: 'attack', target_id: 'enemy:0', round: 3 },
      reason: 'Strike the exposed target.',
    },
    allies: [{
      id: 'ally:e2e', name: 'Tester', hp: 900, max_hp: 1000,
      mana: 100, max_mana: 100, is_self: true, is_player: true,
    }],
    enemies: [{ id: 'enemy:0', name: 'Rate Warden', hp: 500, max_hp: 1000, effects: [] }],
    options: options.length ? options : [
      { kind: 'attack', id: '', name: 'Basic Attack', target: 'enemy', cooldown: 0 },
    ],
    recent_logs: [],
    initiative: [],
    enemy_intents: [],
    social: {},
  };
}

async function startPlanningCombat(page, state = planningState()) {
  await page.goto('/abyss?active=1');
  await page.evaluate(liveState => {
    window.connectLiveCombat = function noopLiveConnectionForTest() {};
    window.startLiveCombat({ state: liveState }, 900, 1000);
  }, state);
  await expect(page.locator('#liveCombat')).toBeVisible();
}

async function installMockGamepad(page) {
  await page.addInitScript(() => {
    const buttons = Array.from({ length: 16 }, () => ({ pressed: false, touched: false, value: 0 }));
    window.__abyssE2EPad = {
      axes: [0, 0, 0, 0],
      buttons,
      connected: true,
      id: 'Abyss E2E Pad',
      index: 0,
      mapping: 'standard',
      timestamp: 0,
    };
    window.__abyssE2EPadEnabled = false;
    Object.defineProperty(navigator, 'getGamepads', {
      configurable: true,
      value: () => window.__abyssE2EPadEnabled ? [window.__abyssE2EPad] : [],
    });
  });
}

test('mobile live combat leads with the action deck and retires the run-action proxy', async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await page.emulateMedia({ reducedMotion: 'reduce' });
  await startPlanningCombat(page);
  await page.waitForTimeout(250);

  const layout = await page.evaluate(() => {
    const live = document.getElementById('liveCombat').getBoundingClientRect();
    const action = document.querySelector('#liveActionBar .ab-actionslot').getBoundingClientRect();
    const stageElement = document.querySelector('#abyssCombatCockpit > .abyss-stage-row');
    const stage = stageElement.getBoundingClientRect();
    const mobile = document.getElementById('abyssMobileActions');
    const stageVisible = stageElement.getClientRects().length > 0;
    return {
      liveBeforeStage: !stageVisible || live.top < stage.top,
      actionBeforeStage: !stageVisible || action.top < stage.top,
      actionInViewport: action.bottom > 0 && action.top < innerHeight,
      mobileHidden: mobile.hidden || getComputedStyle(mobile).display === 'none',
      mobileInert: mobile.inert,
      focusedDecision: document.activeElement.matches('#liveActionBar .ab-actionslot:not(:disabled)')
        && document.activeElement.getClientRects().length > 0,
    };
  });

  expect(layout.liveBeforeStage).toBe(true);
  expect(layout.actionBeforeStage).toBe(true);
  expect(layout.actionInViewport).toBe(true);
  expect(layout.mobileHidden).toBe(true);
  expect(layout.mobileInert).toBe(true);
  expect(layout.focusedDecision).toBe(true);
});

test('desktop live combat keeps vitals, battlefield, spell queue, actions, and log fully visible', async ({ page }) => {
  const viewports = [
    { width: 901, height: 768 },
    { width: 901, height: 1000 },
    { width: 1024, height: 768 },
    { width: 1280, height: 720 },
    { width: 1280, height: 1000 },
    { width: 1440, height: 900 },
    { width: 1439, height: 1000 },
    { width: 1440, height: 1000 },
    { width: 1440, height: 1080 },
    { width: 1920, height: 1080 },
  ];

  for (const viewport of viewports) {
    await page.setViewportSize(viewport);
    await page.emulateMedia({ reducedMotion: 'reduce' });
    const state = planningState([
      { kind: 'attack', id: '', name: 'Basic Attack', target: 'enemy', cooldown: 0 },
      { kind: 'skill', id: 'arc-bolt', name: 'Arc Bolt', target: 'enemy', mana: 15, cooldown: 0 },
      ...Array.from({ length: 6 }, (_, index) => ({
        kind: 'skill', id: `crowd-skill-${index}`, name: `Crowd Skill ${index + 1}`,
        target: 'enemy', mana: 10 + index, cooldown: 0,
      })),
    ]);
    state.queued = { kind: 'skill', ability_id: 'arc-bolt', target_id: 'enemy:0', round: 3 };
    state.enrage_round = 5;
    state.can_configure_pause = true;
    state.pause_reason = 'Waiting for all party members to confirm the synchronized combat action before the timer resumes.';
    state.round_recap = 'The Rate Warden shifts into an exposed stance.';
    state.encounter_warning = 'Reinforcements arrive after the next exchange.';
    state.hazard_telegraph = 'Arc storm targets the back line.';
    state.initiative = [
      { side: 'ally', name: 'Tester', speed: 120 },
      { side: 'enemy', name: 'Rate Warden', speed: 95 },
    ];
    state.enemy_intents = [{ enemy_name: 'Rate Warden', kind: 'skill', ability: 'Arc Storm', target: 'Tester' }];
    state.allies.push(...Array.from({ length: 5 }, (_, index) => ({
      id: `ally:${index + 1}`, name: `Ally ${index + 1}`, hp: 700, max_hp: 900,
      mana: 60, max_mana: 80, is_self: false, is_player: false,
    })));
    state.enemies.push(...Array.from({ length: 5 }, (_, index) => ({
      id: `enemy:${index + 1}`, name: `Enemy ${index + 1}`, hp: 350, max_hp: 500, effects: [],
    })));
    state.enemies[0].weakpoints = [
      { id: 'core', name: 'Core', description: 'Break guard' },
      { id: 'focus', name: 'Focus', description: 'Disrupt casting' },
    ];
    await startPlanningCombat(page, state);
    await page.evaluate(() => {
      window.liveSelectedTarget = 'enemy:0';
      window.renderLiveCombat(window.liveCombatState);
    });
    await expect(page.locator('#liveQueue')).toContainText('Arc Bolt');
    await expect(page.locator('#liveWeakpoints button')).toHaveCount(2);
    await expect(page.locator('#liveEncounterWarningText')).toHaveAttribute('title', state.encounter_warning);
    await expect(page.locator('#liveHazardTelegraphText')).toHaveAttribute('title', state.hazard_telegraph);
    const compactExpected = viewport.width <= 1439 || viewport.height <= 1050;
    if (compactExpected) await expect(page.locator('#liveCombat > .ab-live-details')).toBeHidden();
    else await expect(page.locator('#liveCombat > .ab-live-details')).toBeVisible();
    await page.waitForTimeout(250);
    expect(await page.evaluate(() => scrollY), `combat focus should not scroll at ${viewport.width}x${viewport.height}`).toBeLessThanOrEqual(1);

    const layout = await page.evaluate(() => {
      const selectors = {
        health: '#hpBar',
        mana: '#mpBar',
        tactic: '#liveTactic',
        pauses: '#livePauseMode',
        battlefield: '#livePixelStage',
        queue: '#liveQueue',
        actions: '#liveActionBar',
        policy: '#liveCriticalTactic',
        encounterWarning: '#liveEncounterWarning',
        hazardWarning: '#liveHazardTelegraph',
        liveFeed: '#liveFeed',
        log: '#logWrap',
        logFeed: '#abyssLog',
      };
      return Object.fromEntries(Object.entries(selectors).map(([name, selector]) => {
        const element = document.querySelector(selector);
        const rect = element.getBoundingClientRect();
        const clip = { left: 0, top: 0, right: innerWidth, bottom: innerHeight };
        const clippers = [];
        for (let ancestor = element.parentElement; ancestor; ancestor = ancestor.parentElement) {
          const style = getComputedStyle(ancestor);
          const bounds = ancestor.getBoundingClientRect();
          if (/(auto|clip|hidden|scroll)/.test(style.overflowX)) {
            clip.left = Math.max(clip.left, bounds.left);
            clip.right = Math.min(clip.right, bounds.right);
            clippers.push({ axis: 'x', className: ancestor.className, id: ancestor.id, left: bounds.left, right: bounds.right });
          }
          if (/(auto|clip|hidden|scroll)/.test(style.overflowY)) {
            clip.top = Math.max(clip.top, bounds.top);
            clip.bottom = Math.min(clip.bottom, bounds.bottom);
            clippers.push({ axis: 'y', bottom: bounds.bottom, className: ancestor.className, id: ancestor.id, top: bounds.top });
          }
        }
        const hit = document.elementFromPoint(
          Math.max(0, Math.min(innerWidth - 1, rect.left + rect.width / 2)),
          Math.max(0, Math.min(innerHeight - 1, rect.top + rect.height / 2)),
        );
        return [name, {
          bottom: rect.bottom,
          height: rect.height,
          left: rect.left,
          right: rect.right,
          top: rect.top,
          width: rect.width,
          clip,
          clippers,
          visible: element.getClientRects().length > 0 && getComputedStyle(element).visibility !== 'hidden',
          fullyUnclipped: clip.left <= rect.left + 1 && clip.top <= rect.top + 1
            && clip.right >= rect.right - 1 && clip.bottom >= rect.bottom - 1,
          uncovered: Boolean(hit && (hit === element || element.contains(hit))),
        }];
      }));
    });

    for (const [name, rect] of Object.entries(layout)) {
      expect(rect.visible, `${name} should render at ${viewport.width}x${viewport.height}`).toBe(true);
      expect(rect.width, `${name} should have width at ${viewport.width}x${viewport.height}`).toBeGreaterThan(0);
      expect(rect.height, `${name} should have height at ${viewport.width}x${viewport.height}`).toBeGreaterThan(0);
      expect(rect.left, `${name} should not clip left at ${viewport.width}x${viewport.height}`).toBeGreaterThanOrEqual(0);
      expect(rect.top, `${name} should not clip above at ${viewport.width}x${viewport.height}`).toBeGreaterThanOrEqual(0);
      expect(rect.right, `${name} should not clip right at ${viewport.width}x${viewport.height}`).toBeLessThanOrEqual(viewport.width + 1);
      expect(rect.bottom, `${name} should not clip below at ${viewport.width}x${viewport.height}`).toBeLessThanOrEqual(viewport.height + 1);
      expect(rect.fullyUnclipped, `${name} should not be clipped by an ancestor at ${viewport.width}x${viewport.height}: ${JSON.stringify(rect)}`).toBe(true);
      expect(rect.uncovered, `${name} should not be covered at ${viewport.width}x${viewport.height}`).toBe(true);
    }
  }
});

test('live snapshots keep the visible health and mana bars synchronized', async ({ page }) => {
  const state = planningState();
  state.allies[0].hp = 675;
  state.allies[0].mana = 42;
  await startPlanningCombat(page, state);

  await expect(page.locator('#hpBar')).toHaveAttribute('aria-valuenow', '675');
  await expect(page.locator('#hpBar')).toHaveAttribute('aria-valuemax', '1000');
  await expect(page.locator('#hpText')).toHaveText('675 / 1,000');
  await expect(page.locator('#mpBar')).toHaveAttribute('aria-valuenow', '42');
  await expect(page.locator('#mpBar')).toHaveAttribute('aria-valuemax', '100');
  await expect(page.locator('#mpText')).toContainText('42 / 100 Mana');
});

test('animated battlefield signals hits and honors reduced motion', async ({ page }) => {
  await page.emulateMedia({ reducedMotion: 'no-preference' });
  await startPlanningCombat(page);
  const enemy = page.locator('#livePixelEnemies .ab-pixel-unit').first();
  const sprite = enemy.locator('.ab-actor-sprite');
  await expect(sprite).toHaveCSS('animation-name', 'ab-pixel-idle');

  await page.evaluate(() => {
    const next = structuredClone(window.liveCombatState);
    next.version += 1;
    next.enemies[0].hp -= 100;
    next.recent_logs = ['Tester strikes Rate Warden for 100 damage.'];
    window.renderLiveCombat(next);
  });
  await expect(enemy).toHaveClass(/taking-hit/);
  await expect(sprite).toHaveCSS('animation-name', /ab-pixel-(hit|lunge-left)/);

  await page.emulateMedia({ reducedMotion: 'reduce' });
  await expect(sprite).toHaveCSS('animation-name', 'none');
});

test('live combat streams logs into both views with accessible controls', async ({ page }) => {
  await page.setViewportSize({ width: 1440, height: 900 });
  const state = planningState();
  state.recent_logs = [
    '<span style="color:#efc46d"><b>Tester</b> casts &#34;Arc Bolt&#34; for 125 damage.</span>',
    'Round 3 planning begins.',
  ];
  await startPlanningCombat(page, state);

  await expect(page.locator('#liveFeed > div')).toHaveCount(2);
  await expect(page.locator('#abyssLog > .ab-live-mirror')).toHaveCount(2);
  await expect(page.locator('#abyssLog')).toContainText('Arc Bolt');
  await expect(page.locator('#abyssLog > .ab-live-mirror').first()).toContainText('casts "Arc Bolt"');
  await expect(page.locator('#abyssLog > .ab-live-mirror').first()).toHaveAttribute('data-log-time', /\d/);
  await expect(page.locator('#abyssLog')).not.toContainText('<span');
  await expect(page.locator('#logRoundsBtn')).toHaveAttribute('aria-label', 'Toggle combat log round numbers');
  await expect(page.locator('#logRoundsBtn')).toBeDisabled();
  await expect(page.locator('#logRoundsBtn')).toHaveAttribute('title', /available after combat/i);
  await expect(page.locator('#logMonoBtn')).toHaveAttribute('aria-label', 'Toggle combat log monospace font');

  await page.evaluate(() => window.setCombatLogCategory('damage'));
  await expect(page.locator('#abyssLog > .ab-live-mirror').first()).toBeVisible();
  await expect(page.locator('#abyssLog > .ab-live-mirror').last()).toBeHidden();

  const toggle = page.locator('#liveLogToggle');
  await expect(toggle).toHaveAttribute('aria-expanded', 'false');
  await toggle.click();
  await expect(toggle).toHaveAttribute('aria-expanded', 'true');
  await expect(toggle).toHaveText('COLLAPSE LOG');
});

test('live snapshots keep the newest server version and name the queued target', async ({ page }) => {
  const state = planningState();
  state.version = 12;
  state.queued = { kind: 'attack', ability_id: '', target_id: 'enemy:0', round: 3 };
  await startPlanningCombat(page, state);

  await expect(page.locator('#liveQueue')).toContainText('Rate Warden');
  await expect(page.locator('#liveQueue')).toHaveAttribute('data-state', 'queued');
  await expect(page.locator('#liveQueue')).toHaveAttribute('aria-label', 'Queued combat action');
  await page.evaluate(() => {
    const stale = structuredClone(window.liveCombatState);
    stale.version = 11;
    stale.round = 2;
    stale.queued = { kind: 'defend', ability_id: '', target_id: '', round: 2 };
    window.renderLiveCombat(stale);
  });

  await expect(page.locator('#liveRound')).toContainText('ROUND 3');
  await expect(page.locator('#liveQueue')).toContainText('Rate Warden');
  expect(await page.evaluate(() => window.liveCombatState.version)).toBe(12);
});

test('server recommendation stays visible and names its exact target across loadout pages', async ({ page }) => {
  const options = [
    { kind: 'attack', id: '', name: 'Basic Attack', target: 'enemy', cooldown: 0 },
    ...Array.from({ length: 6 }, (_, index) => ({
      kind: 'skill', id: `skill-${index + 1}`, name: `Skill ${index + 1}`,
      target: 'enemy', mana: 5, cooldown: 0,
    })),
  ];
  const state = planningState(options);
  state.recommended = {
    action: { kind: 'skill', ability_id: 'skill-6', target_id: 'enemy:0', round: 3 },
    reason: 'Exploit the exposed target.',
  };
  await startPlanningCombat(page, state);

  await expect(page.locator('#liveActionBar .ab-actionslot.recommended')).toHaveCount(1);
  await expect(page.locator('#liveActionBar .ab-actionslot.recommended')).toContainText('Skill 6');
  await expect(page.locator('#liveAutoPreview')).toContainText('Rate Warden');
  await expect(page.locator('#liveActionBar .ab-actionslot.recommended')).toBeFocused();
});

test('live keyboard shortcuts stay behind an open modal and arrows stay on combat decisions', async ({ page }) => {
  let actionRequests = 0;
  await page.route('**/api/abyss/combat/action', async route => {
    actionRequests++;
    await route.fulfill({ status: 409, contentType: 'application/json', body: JSON.stringify({ ok: false, error: 'test guard' }) });
  });
  await startPlanningCombat(page);
  await page.evaluate(() => {
    window.selectLiveTarget('enemy:0');
    window.openModal('<h3>Keep this modal open</h3><div class="modal-actions"><button id="modalCancelBtn">Cancel</button></div>');
  });

  await page.keyboard.press('a');
  await page.waitForTimeout(180);
  expect(actionRequests).toBe(0);
  expect(await page.evaluate(() => document.getElementById('sharedModalCard').contains(document.activeElement))).toBe(true);

  await page.evaluate(() => {
    window.closeModal();
    document.querySelector('.nav a').focus();
  });
  await page.keyboard.press('ArrowDown');
  expect(await page.evaluate(() => document.activeElement === document.querySelector('.nav a'))).toBe(true);

  await page.evaluate(() => {
    document.querySelector('#liveActionBar .ab-actionslot:not(:disabled)').focus();
  });
  await page.keyboard.press('ArrowLeft');
  const keyboardFocus = await page.evaluate(() => {
    const active = document.activeElement;
    const allowed = '#liveActionBar .ab-actionslot:not(:disabled), #liveCombat [data-target]:not(:disabled), #liveReady:not(:disabled)';
    return active.matches(allowed) && active.getClientRects().length > 0;
  });
  expect(keyboardFocus).toBe(true);
});

test('live gamepad stays behind an open modal and traverses only visible combat decisions', async ({ page }) => {
  await installMockGamepad(page);
  let actionRequests = 0;
  await page.route('**/api/abyss/combat/action', async route => {
    actionRequests++;
    await route.fulfill({ status: 409, contentType: 'application/json', body: JSON.stringify({ ok: false, error: 'test guard' }) });
  });
  await startPlanningCombat(page);
  await page.evaluate(() => {
    window.selectLiveTarget('enemy:0');
    const live = document.getElementById('liveCombat');
    live.tabIndex = -1;
    live.focus();
    const controls = Array.from(document.querySelectorAll('#liveCombat button:not(:disabled)'));
    window.liveGamepadIndex = controls.indexOf(document.querySelector('#liveActionBar .kind-attack'));
    window.openModal('<h3>Keep this modal open</h3><div class="modal-actions"><button id="modalCancelBtn">Cancel</button></div>');
    window.__abyssE2EPadEnabled = true;
    window.__abyssE2EPad.buttons[0].pressed = true;
  });
  await page.waitForTimeout(240);
  await page.evaluate(() => { window.__abyssE2EPad.buttons[0].pressed = false; });
  await page.waitForTimeout(200);

  expect(actionRequests).toBe(0);
  expect(await page.evaluate(() => document.getElementById('sharedModalCard').contains(document.activeElement))).toBe(true);

  await page.evaluate(() => {
    window.closeModal();
    const live = document.getElementById('liveCombat');
    live.focus();
    window.liveGamepadIndex = -1;
    window.__abyssE2EPad.buttons[15].pressed = true;
  });
  await page.waitForTimeout(240);
  await page.evaluate(() => { window.__abyssE2EPad.buttons[15].pressed = false; });
  await page.waitForTimeout(200);
  const gamepadFocus = await page.evaluate(() => {
    const active = document.activeElement;
    const allowed = '#liveActionBar .ab-actionslot:not(:disabled), #liveCombat [data-target]:not(:disabled), #liveReady:not(:disabled)';
    return active.matches(allowed) && active.getClientRects().length > 0;
  });
  expect(gamepadFocus).toBe(true);
});

test('last-item confirmation names the safe and destructive choices explicitly', async ({ page }) => {
  await startPlanningCombat(page, planningState([
    { kind: 'attack', id: '', name: 'Basic Attack', target: 'enemy', cooldown: 0 },
    { kind: 'item', id: 'last-tonic', name: 'Last Tonic', target: 'self', count: 1, cooldown: 0 },
  ]));

  await page.locator('#liveActionBar .kind-item').click();
  await expect(page.locator('#sharedModal')).toHaveClass(/open/);
  await expect(page.locator('#modalOkBtn')).toHaveText('Use item');
  await expect(page.locator('#modalCancelBtn')).toHaveText('Keep it');
});

test('rapid live-action taps submit one intent while the first request is in flight', async ({ page }) => {
  let actionRequests = 0;
  await page.route('**/api/abyss/combat/action', async route => {
    actionRequests++;
    await new Promise(resolve => setTimeout(resolve, 350));
    await route.fulfill({ status: 409, contentType: 'application/json', body: JSON.stringify({ ok: false, error: 'test hold' }) });
  });
  await startPlanningCombat(page);
  await page.evaluate(() => {
    window.selectLiveTarget('enemy:0');
    const action = document.querySelector('#liveActionBar .kind-attack');
    action.click();
    action.click();
  });
  await page.waitForTimeout(160);

  expect(actionRequests).toBe(1);
});

test('late action responses cannot overwrite or unlock a newer combat session', async ({ page }) => {
  const first = planningState();
  first.session_id = 'combat-session-a';
  first.version = 3;
  await startPlanningCombat(page, first);

  await page.evaluate(() => {
    window.__combatPostResolvers = [];
    window.abPost = (url, payload) => new Promise(resolve => {
      window.__combatPostResolvers.push({ url, payload, resolve });
    });
    window.selectLiveTarget('enemy:0');
    document.querySelector('#liveActionBar .kind-attack').click();
  });
  await expect.poll(() => page.evaluate(() => window.__combatPostResolvers.length)).toBe(1);

  const second = planningState();
  second.session_id = 'combat-session-b';
  second.version = 1;
  await page.evaluate(state => {
    window.startLiveCombat({ state }, 900, 1000);
    window.selectLiveTarget('enemy:0');
    document.querySelector('#liveActionBar .kind-attack').click();
  }, second);
  await expect.poll(() => page.evaluate(() => window.__combatPostResolvers.length)).toBe(2);

  const lateFirst = structuredClone(first);
  lateFirst.version = 99;
  await page.evaluate(state => window.__combatPostResolvers[0].resolve(state), lateFirst);
  await page.waitForTimeout(120);
  await page.evaluate(() => document.querySelector('#liveActionBar .kind-attack').click());

  const protectedState = await page.evaluate(() => ({
    requests: window.__combatPostResolvers.length,
    session: window.liveCombatState.session_id,
    guarded: Boolean(window.liveActionInFlight),
  }));
  expect(protectedState).toEqual({ requests: 2, session: 'combat-session-b', guarded: true });

  await page.evaluate(state => window.__combatPostResolvers[1].resolve(state), second);
});

test('completed-session teardown cannot dismiss its replacement combat', async ({ page }) => {
  const first = planningState();
  first.session_id = 'completed-session-a';
  await startPlanningCombat(page, first);

  const second = planningState();
  second.session_id = 'replacement-session-b';
  await page.evaluate(state => {
    setTimeout(() => window.dismissFinishedLiveCombat('completed-session-a'), 80);
    window.startLiveCombat({ state }, 900, 1000);
  }, second);
  await page.waitForTimeout(160);

  await expect(page.locator('#liveCombat')).toBeVisible();
  expect(await page.evaluate(() => ({
    rendered: window.liveCombatState.session_id,
    owned: window.liveCombatSessionID,
  }))).toEqual({ rendered: 'replacement-session-b', owned: 'replacement-session-b' });
});

test('Ready and Time Bank stay busy across live snapshots and submit once', async ({ page }) => {
  const state = planningState();
  state.queued = { kind: 'attack', ability_id: '', target_id: 'enemy:0', round: 3 };
  state.time_bank_ms = 5_000;
  await startPlanningCombat(page, state);

  await page.evaluate(() => {
    window.__combatControlPosts = [];
    window.abPost = (url, payload) => new Promise(resolve => {
      window.__combatControlPosts.push({ url, payload, resolve });
    });
    window.setLiveReady();
    window.setLiveReady();
    window.spendLiveTimeBank();
    window.spendLiveTimeBank();
  });
  await expect.poll(() => page.evaluate(() => window.__combatControlPosts.length)).toBe(2);

  await page.evaluate(() => {
    const snapshot = structuredClone(window.liveCombatState);
    snapshot.version += 1;
    window.renderLiveCombat(snapshot);
  });
  await expect(page.locator('#liveReady')).toBeDisabled();
  await expect(page.locator('#liveReady')).toHaveAttribute('aria-busy', 'true');
  await expect(page.locator('#liveTimeBank')).toBeDisabled();
  await expect(page.locator('#liveTimeBank')).toHaveAttribute('aria-busy', 'true');

  await page.evaluate(() => {
    const authoritative = structuredClone(window.liveCombatState);
    authoritative.version += 1;
    authoritative.time_bank_ms = 0;
    window.__combatControlPosts.forEach(post => post.resolve(post.url.endsWith('/time')
      ? { ok: false, error: 'round advanced', state: authoritative }
      : { ok: false, error: 'round advanced' }));
  });
  await expect.poll(() => page.evaluate(() => window.liveCombatState.time_bank_ms)).toBe(0);
  await expect(page.locator('#liveReady')).toHaveAttribute('aria-busy', 'false');
  await expect(page.locator('#liveTimeBank')).toHaveAttribute('aria-busy', 'false');
});
