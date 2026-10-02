const { test, expect } = require('@playwright/test');
const path = require('path');

const SELF = 'ally:reconciliation';
const ENEMY = 'enemy:reconciliation';
function snapshot(version = 1, cursor = 0, sequences = []) {
  return {
    ok: true, session_id: 'reconciliation-e2e', phase: 'planning', round: 1, version,
    deadline: new Date(Date.now() + 60_000).toISOString(), tactic: 'balanced', pause_mode: 'adaptive',
    policy: {}, action_budget: {limit: 64, remaining: 64},
    allies: [{id: SELF, entity_id: SELF, name: 'Guardian', hp: 800, max_hp: 1000, is_self: true, is_player: true}],
    enemies: [{id: 'enemy:0', entity_id: ENEMY, name: 'Warden', hp: 900 - cursor * 10, max_hp: 1000}],
    options: [{kind: 'attack', id: '', name: 'Basic Attack', target: 'enemy', cooldown: 0}],
    recent_logs: [], log_cursor: 0, initiative: [], enemy_intents: [], social: {},
    presentation_cursor: cursor,
    presentation_events: sequences.map(seq => ({seq, round: 1, kind: 'attack', actor_id: SELF,
      ability_name: 'Basic Attack', element: 'physical', targets: [{target_id: ENEMY, damage: 10}]})),
  };
}

async function openCombat(page) {
  // Exercise the actual renderer with deterministic snapshots; this is mocked transport.
  // Read the working asset so an already-running embedded fixture can test edits.
  for (const asset of ['abyss_combat_animation.js', 'abyss_fight_visuals.js']) {
    await page.route(`**/static/${asset}*`, route => route.fulfill({
      path: path.resolve(__dirname, '../../internal/bot/webassets', asset),
      contentType: 'application/javascript',
    }));
  }
  await page.route('**/api/abyss/combat/state', route => route.fulfill({json: {ok: false}}));
  await page.goto('/abyss?active=1');
  await page.evaluate(state => {
    window.connectLiveCombat = function () {};
    window.startLiveCombat({state}, 800, 1000);
    window.__reconciliationNumbers = [];
    new MutationObserver(records => records.forEach(record => record.addedNodes.forEach(node => {
      if (node instanceof Element && node.matches('.ab-combat-number')) {
        window.__reconciliationNumbers.push({seq: Number(node.dataset.eventSeq), text: node.textContent});
      }
    }))).observe(document.getElementById('livePixelStage'), {childList: true});
  }, snapshot());
  await expect(page.locator('#liveCombat')).toBeVisible();
}

for (const [name, sequences] of [['internal gap', [1, 3]], ['missing suffix', [1]], ['cursor only', []]]) {
  test(`${name} reconciles to the snapshot without replaying incomplete outcomes`, async ({ page }) => {
    await openCombat(page);
    await page.evaluate(state => window.renderLiveCombat(state), snapshot(2, 3, sequences));
    await expect(page.locator('#livePixelStage')).toHaveAttribute('data-presentation-state', 'catchup');
    await expect(page.locator('#livePixelStage')).toHaveAttribute('data-last-event-seq', '3');
    await expect(page.locator(`[data-entity-id="${ENEMY}"] .ab-overhead-hp em`)).toHaveText('87%');
    await expect(page.locator('#liveAnimationSkip')).toBeDisabled();
    expect(await page.evaluate(() => window.__reconciliationNumbers)).toEqual([]);
    // Late gap contents must not replay; the next contiguous event still must play.
    await page.evaluate(state => window.renderLiveCombat(state), snapshot(3, 4, [3, 2, 1, 4]));
    await expect(page.locator('#livePixelStage')).toHaveAttribute('data-last-event-seq', '4');
    expect(await page.evaluate(() => window.__reconciliationNumbers)).toEqual([{seq: 4, text: '−10'}]);
  });
}

test('a gap cancels active travel and its callbacks before accepting the next event', async ({ page }) => {
  await openCombat(page);
  const casting = snapshot(2, 1, [1]);
  Object.assign(casting.presentation_events[0], {kind: 'skill', ability_id: 'fireball', ability_name: 'Fireball'});
  const flight = await page.evaluate(states => new Promise(resolve => {
    const stage = document.getElementById('livePixelStage');
    const observer = new MutationObserver(records => {
      for (const record of records) for (const node of record.addedNodes) {
        if (!(node instanceof Element) || !node.matches('.ab-effect-travel')) continue;
        const animations = node.getAnimations(), bounds = node.getBoundingClientRect();
        observer.disconnect();
        // Reconcile at the actual launch, before slow-runner scheduling can
        // turn a separate Playwright round trip into an already-finished hit.
        window.renderLiveCombat(states[1]);
        resolve({visible: bounds.width > 0 && bounds.height > 0,
          animated: animations.length > 0, cancelled: animations.every(animation => animation.playState === 'idle')});
      }
    });
    observer.observe(stage, {childList: true});
    window.renderLiveCombat(states[0]);
  }), [casting, snapshot(3, 4, [2, 4])]);
  expect(flight).toEqual({visible: true, animated: true, cancelled: true});
  await expect(page.locator('#livePixelStage')).toHaveAttribute('data-presentation-state', 'catchup');
  await expect(page.locator('#livePixelStage')).toHaveAttribute('data-last-event-seq', '4');
  await expect(page.locator('.ab-combat-effect,.ab-combat-number,.ab-performing,.ab-reacting')).toHaveCount(0);
  await page.evaluate(state => window.renderLiveCombat(state), snapshot(4, 5, [5]));
  await expect(page.locator('#livePixelStage')).toHaveAttribute('data-last-event-seq', '5');
  expect(await page.evaluate(() => window.__reconciliationNumbers)).toEqual([{seq: 5, text: '−10'}]);
});

test('a complete unordered batch plays each confirmed outcome once in sequence', async ({ page }) => {
  await openCombat(page);
  await page.evaluate(state => window.renderLiveCombat(state), snapshot(2, 3, [3, 1, 2, 2]));
  await expect(page.locator('#livePixelStage')).toHaveAttribute('data-last-event-seq', '3');
  await expect(page.locator('#livePixelStage')).toHaveAttribute('data-presentation-state', 'idle');
  expect(await page.evaluate(() => window.__reconciliationNumbers.map(record => record.seq))).toEqual([1, 2, 3]);
});
