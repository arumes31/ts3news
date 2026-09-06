const { test, expect } = require('@playwright/test');

function combatState(id = 'usability') {
  return { ok: true, session_id: id, phase: 'planning', round: 3, version: 7,
    deadline: new Date(Date.now() + 600_000).toISOString(), tactic: 'balanced',
    pause_mode: 'adaptive', can_configure_pause: true, policy: {}, social: {},
    allies: [{ id: 'ally:self', name: 'Tester', hp: 900, max_hp: 1000, mana: 100, max_mana: 100, is_self: true, is_player: true }],
    enemies: [{ id: 'enemy:0', name: 'Warden', hp: 500, max_hp: 1000, effects: [] }, { id: 'enemy:1', name: 'Archer', hp: 300, max_hp: 500, effects: [] }],
    options: [{ kind: 'attack', id: '', name: 'Basic Attack', target: 'enemy', cooldown: 0 }, { kind: 'skill', id: 'arc', name: 'Arc Bolt', target: 'enemy', mana: 15, cooldown: 0 }],
    recommended: { action: { kind: 'skill', ability_id: 'arc', target_id: 'enemy:0', round: 3 }, reason: 'Exploit weakness.' },
    enemy_intents: [{ enemy_name: 'Warden', ability: 'Arc Storm', target: 'Tester', kind: 'skill' }],
    initiative: [{ name: 'Tester', side: 'ally', speed: 120 }], recent_logs: [] };
}
async function start(page, width = 1440) {
  await page.setViewportSize({ width, height: width === 390 ? 844 : 900 });
  await page.emulateMedia({ reducedMotion: 'reduce' });
  await page.goto('/abyss?active=1');
  await page.evaluate(state => { connectLiveCombat = () => {}; startLiveCombat({ state }, 900, 1000); setLiveConnection('synced', 'LIVE · SYNCED'); }, combatState());
}

test('desktop tactical information and party controls remain accessible', async ({ page }) => {
  await start(page);
  await page.locator('#liveIntelDrawer > summary').click();
  await expect(page.locator('#liveIntents')).toContainText('Arc Storm');
  await expect(page.locator('#liveEnemies')).toBeVisible();
  await expect(page.locator('#liveEnemies')).toContainText('500');
  await expect(page.locator('#liveSocialShell')).toBeVisible();
  await page.locator('#liveSocialShell summary').click();
  await expect(page.locator('#livePreferredRole')).toBeVisible();
  await page.locator('#liveIntelDrawer [data-close-live-drawer]').click();
  await expect(page.locator('#liveIntelDrawer > summary')).toBeFocused();
  await expect(page.locator('#liveSelectedIntent')).toContainText('Arc Storm');
  await page.screenshot({ path: '.impeccable/critique/runtime/combat-fixed-desktop.png' });
});

test('mobile timer and adjacent targeting survive combat focus without navigation overlap', async ({ page }) => {
  await start(page, 390);
  await expect(page.locator('#abyssWorkspaceSelect')).toBeHidden();
  await expect(page.locator('#liveTargetSelect')).toBeVisible();
  await page.locator('#liveTargetSelect').selectOption('enemy:1');
  expect(await page.evaluate(() => liveSelectedTarget)).toBe('enemy:1');
  const layout = await page.evaluate(() => {
    const box = id => document.getElementById(id).getBoundingClientRect();
    const clock = box('liveClock'), picker = box('liveTargetSelect'), actions = box('liveActionBar');
    const hit = document.elementFromPoint(clock.x + clock.width / 2, clock.y + clock.height / 2);
    return { clockVisible: clock.y >= 0 && clock.bottom <= innerHeight && document.getElementById('liveClock').contains(hit),
      gap: actions.top - picker.bottom, font: parseFloat(getComputedStyle(document.getElementById('liveQueue')).fontSize), overflow: document.documentElement.scrollWidth - innerWidth };
  });
  expect(layout.clockVisible).toBe(true);
  expect(layout.gap).toBeLessThan(150);
  expect(layout.font).toBeGreaterThanOrEqual(13);
  expect(layout.overflow).toBeLessThanOrEqual(1);
  await page.screenshot({ path: '.impeccable/critique/runtime/combat-fixed-mobile.png' });
  await page.locator('#abCombatMenuToggle').click();
  await expect(page.locator('#abyssWorkspaceSelect')).toBeVisible();
  await page.locator('#liveIntelDrawer > summary').click();
  await page.evaluate(() => scrollTo(0, 0));
  await expect.poll(() => page.evaluate(() => {
    const close = document.querySelector('#liveIntelDrawer [data-close-live-drawer]'), rect = close.getBoundingClientRect();
    return close.contains(document.elementFromPoint(rect.x + rect.width / 2, rect.y + rect.height / 2));
  })).toBe(true);
  const panelOverflow = await page.locator('#liveIntelDrawer .ab-combat-drawer-body').evaluate(el => el.scrollWidth - el.clientWidth);
  expect(panelOverflow).toBeLessThanOrEqual(1);
  await page.locator('#liveIntelDrawer [data-close-live-drawer]').click();
});

test('rejected preferences roll back, stay reported and cannot alter a newer session', async ({ page }) => {
  await start(page);
  await page.locator('#liveSettingsDrawer > summary').click();
  await page.evaluate(() => { abPost = () => new Promise(resolve => { window.rejectPreference = resolve; }); });
  await page.locator('#liveTactic').selectOption('aggressive');
  await expect(page.locator('#liveTactic')).toBeDisabled();
  await expect(page.locator('#livePreferenceStatus')).toContainText('Saving');
  await page.evaluate(() => rejectPreference({ ok: false, error: 'Round advanced' }));
  await expect(page.locator('#liveTactic')).toHaveValue('balanced');
  await expect(page.locator('#livePreferenceStatus')).toContainText('Round advanced');
  for (const [id, rejected, confirmed] of [['livePauseMode', 'fast', 'adaptive'], ['liveAttackPriority', 'highest_hp', 'lowest_hp']]) {
    await page.locator('#' + id).selectOption(rejected);
    await page.evaluate(() => rejectPreference({ ok: false, error: 'Preference rejected' }));
    await expect(page.locator('#' + id)).toHaveValue(confirmed);
    await expect(page.locator('#livePreferenceStatus')).toContainText('Preference rejected');
  }
  await page.evaluate(() => renderLiveCombat(structuredClone(liveCombatState)));
  await expect(page.locator('#livePreferenceStatus')).toContainText('Preference rejected');
  await page.locator('#liveTactic').selectOption('aggressive');
  await page.evaluate(state => startLiveCombat({ state }, 900, 1000), combatState('replacement'));
  await page.evaluate(() => rejectPreference({ ...liveCombatState, session_id: 'usability', version: 99, tactic: 'aggressive' }));
  await expect(page.locator('#liveTactic')).toHaveValue('balanced');
  expect(await page.evaluate(() => liveCombatState.session_id)).toBe('replacement');
});

test('timeout and reconnect feedback stay precise until a newer state confirms recovery', async ({ page }) => {
  await start(page);
  await expect(page.locator('#liveAutoPreview')).toContainText('Arc Bolt');
  await page.evaluate(async () => { setLiveConnection('reconnecting', 'RECONNECT'); abPost = async () => ({ ok: false, error: 'Round advanced' }); selectLiveTarget('enemy:0'); await submitLiveOption(liveCombatState.options[0]); });
  await expect(page.locator('#abStatus')).toContainText('Reconnecting');
  await expect(page.locator('#liveRecoveryStatus')).toContainText('Round advanced');
  await page.waitForTimeout(2300);
  await expect(page.locator('#liveRecoveryStatus')).toContainText('Round advanced');
  await page.evaluate(() => { setLiveConnection('synced', 'LIVE'); renderLiveCombat({ ...liveCombatState, version: 8 }); });
  await expect(page.locator('#liveRecoveryStatus')).toBeHidden();
  await page.evaluate(async () => { abPost = async () => { throw new Error('network'); }; await submitLiveOption(liveCombatState.options[0]); });
  await expect(page.locator('#liveRecoveryStatus')).toContainText('not confirmed');
  await expect(page.locator('#liveTargetSelect')).toBeEnabled();
});
