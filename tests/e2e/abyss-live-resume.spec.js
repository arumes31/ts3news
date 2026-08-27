const { test, expect } = require('@playwright/test');

function liveState() {
  const history = Array.from({ length: 40 }, (_, index) => `Combat event ${index + 1}`);
  return {
    ok: true,
    session_id: 'resume-e2e',
    phase: 'planning',
    round: 5,
    version: 12,
    deadline: new Date(Date.now() + 60_000).toISOString(),
    tactic: 'balanced',
    pause_mode: 'adaptive',
    policy: {},
    allies: [{ id: 'ally:e2e', name: 'Tester', hp: 900, max_hp: 1000, mana: 100, max_mana: 100, is_self: true, is_player: true }],
    enemies: [{ id: 'enemy:0', name: 'Resume Warden', hp: 500, max_hp: 1000, role: 'boss', effects: [] }],
    options: [{ kind: 'attack', id: '', name: 'Basic Attack', target: 'enemy', cooldown: 0 }],
    recent_logs: history.slice(-2),
    log_start: 38,
    log_cursor: 40,
    log_history: history,
    initiative: [],
    enemy_intents: [],
    social: {},
  };
}

test('mid-fight refresh restores the complete feed and exact scroll position', async ({ page }) => {
  await page.route('**/api/abyss/combat/state', route => route.fulfill({
    status: 200,
    contentType: 'application/json',
    body: JSON.stringify(liveState()),
  }));
  await page.goto('/abyss?active=1');

  const feed = page.locator('#liveFeed');
  await expect(page.locator('#liveCombat')).toBeVisible();
  await expect(feed.locator(':scope > div')).toHaveCount(40);
  await expect(feed.locator(':scope > div').first()).toHaveText('Combat event 1');
  await expect(feed.locator(':scope > div').last()).toHaveText('Combat event 40');

  const savedTop = await feed.evaluate(element => {
    element.scrollTop = Math.min(72, element.scrollHeight - element.clientHeight - 20);
    element.dispatchEvent(new Event('scroll'));
    return element.scrollTop;
  });
  expect(savedTop).toBeGreaterThan(0);
  await page.waitForTimeout(150);
  await page.reload();

  await expect(feed.locator(':scope > div')).toHaveCount(40);
  await expect.poll(() => feed.evaluate(element => element.scrollTop)).toBeCloseTo(savedTop, 0);
  const position = await page.evaluate(() => JSON.parse(sessionStorage.getItem('abyss_live_log_position_v1')));
  expect(position).toMatchObject({ session_id: 'resume-e2e', cursor: 40, at_bottom: false });

  // Polling fallback returns a complete snapshot repeatedly. A later poll must
  // not snap a reader away from the position restored above.
  await page.evaluate(() => window.renderLiveCombat(structuredClone(window.liveCombatState)));
  await expect.poll(() => feed.evaluate(element => element.scrollTop)).toBeCloseTo(savedTop, 0);

  await page.evaluate(() => {
    const next = structuredClone(window.liveCombatState);
    next.version = 13;
    next.round = 6;
    next.log_start = 40;
    next.log_cursor = 42;
    next.recent_logs = ['Combat event 41', 'Combat event 42'];
    delete next.log_history;
    window.renderLiveCombat(next);
  });
  await expect(feed.locator(':scope > div')).toHaveCount(42);
  await expect(feed.locator(':scope > div').last()).toHaveText('Combat event 42');
  await expect.poll(() => feed.evaluate(element => element.scrollTop)).toBeCloseTo(savedTop, 0);
});
