const {test, expect} = require('@playwright/test');
const fs = require('fs');
const path = require('path');

function state(intents) {
  return {
    ok: true, session_id: 'targeting-e2e', phase: 'planning', round: 1, version: 1,
    deadline: new Date(Date.now() + 600_000).toISOString(), tactic: 'balanced', pause_mode: 'adaptive', policy: {}, social: {},
    allies: [{id: 'ally:self', entity_id: 'ally:self', name: 'Guardian', hp: 800, max_hp: 1000, is_self: true, is_player: true}],
    enemies: [0, 1].map(index => ({id: `enemy:${index}`, entity_id: `enemy:stable-${index}`, name: 'Echo', hp: 500, max_hp: 500})),
    options: [{kind: 'attack', id: '', name: 'Basic Attack', target: 'enemy', cooldown: 0}],
    recent_logs: [], log_cursor: 0, initiative: [], presentation_events: [], presentation_cursor: 0, enemy_intents: intents,
  };
}

async function openCombat(page, initial) {
  // The fixture embeds templates. Replace only the function under test with its
  // working source, keeping the actual portal, controls, and renderer intact.
  const source = fs.readFileSync(path.join(__dirname, '../../internal/bot/webassets/abyss_live.html'), 'utf8');
  const picker = /function renderLiveTargetPicker\(state\)\{[\s\S]*?(?=function syncLiveRecoveryStatus\()/;
  await page.route('**/abyss?active=1', async route => {
    const response = await route.fetch();
    await route.fulfill({response, body: (await response.text()).replace(picker, source.match(picker)[0])});
  });
  await page.route('**/api/abyss/combat/state', route => route.fulfill({json: {ok: false}}));
  await page.goto('/abyss?active=1');
  await page.evaluate(snapshot => {
    window.connectLiveCombat = function () {};
    window.startLiveCombat({state: snapshot}, 800, 1000);
  }, initial);
  await expect(page.locator('#liveCombat')).toBeVisible();
}

test('same-name enemies show their own intent and never borrow another identity', async ({page}) => {
  const initial = state([
    {enemy_id: 'enemy:0', enemy_name: 'Echo', ability: 'Heavy Strike', kind: 'attack'},
    {enemy_id: 'enemy:1', enemy_name: 'Echo', ability: 'Healing Wave', kind: 'heal'},
  ]);
  await openCombat(page, initial);
  await page.locator('#liveTargetSelect').selectOption('enemy:1');
  await expect(page.locator('#liveSelectedIntent')).toHaveText('Intent: Echo · Healing Wave');
  await page.locator('#liveTargetSelect').selectOption('enemy:0');
  await expect(page.locator('#liveSelectedIntent')).toHaveText('Intent: Echo · Heavy Strike');
  const next = structuredClone(initial); next.version++;
  next.enemy_intents = [{enemy_id: 'enemy:absent', enemy_name: 'Echo', ability: 'Wrong Target', kind: 'attack'}];
  await page.evaluate(snapshot => window.renderLiveCombat(snapshot), next);
  await expect(page.locator('#liveSelectedIntent')).toHaveText('Intent: not revealed');
});

test('exact target IDs outrank legacy names while ID-free intents remain compatible', async ({page}) => {
  const initial = state([
    {enemy_name: 'Echo', ability: 'Legacy Strike', kind: 'attack'},
    {enemy_id: 'enemy:1', enemy_name: 'Echo', ability: 'Exact Strike', kind: 'attack'},
  ]);
  await openCombat(page, initial);
  await page.locator('#liveTargetSelect').selectOption('enemy:1');
  await expect(page.locator('#liveSelectedIntent')).toHaveText('Intent: Echo · Exact Strike');
  const next = structuredClone(initial); next.version++;
  next.enemy_intents = [initial.enemy_intents[0]];
  await page.evaluate(snapshot => window.renderLiveCombat(snapshot), next);
  await expect(page.locator('#liveSelectedIntent')).toHaveText('Intent: Echo · Legacy Strike');
});
