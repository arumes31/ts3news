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

test('live snapshots keep the newest server version and name the queued target', async ({ page }) => {
  const state = planningState();
  state.version = 12;
  state.queued = { kind: 'attack', ability_id: '', target_id: 'enemy:0', round: 3 };
  await startPlanningCombat(page, state);

  await expect(page.locator('#liveQueue')).toContainText('Rate Warden');
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
