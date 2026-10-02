const { test, expect } = require('@playwright/test');
const fs = require('node:fs');
test.use({ video: 'on', viewport: { width: 1440, height: 900 } });

// No route.fulfill: combat snapshots, submissions, ready signals and SSE are
// served by production Go handlers and the seeded production combat engine.
// Entry/bank/revive are fixture orchestration; SQL is a committed-state double.
async function fixtureState(page) {
  return (await page.request.get('/api/abyss/transport/control')).json();
}

async function enter(page) {
  await page.locator('#btnEnter').click();
  await expect(page.locator('#btnDescend')).toBeVisible();
}

async function finishFight(page) {
  const deadline = Date.now() + 60_000;
  while (Date.now() < deadline) {
    const state = await (await page.request.get('/api/abyss/combat/state')).json();
    if (state.phase === 'complete' || state.phase === 'failed') {
      await expect.poll(() => page.evaluate(() => window.busy)).toBe(false);
      return state;
    }
    const attack = page.locator('#liveActionBar .kind-attack');
    if (state.phase === 'planning' && !state.queued && await attack.isVisible() && await attack.isEnabled()) {
      const enemy = state.enemies.find(unit => unit.hp > 0);
      const picker = page.locator('#liveTargetSelect');
      try {
        if (enemy && await picker.inputValue() !== enemy.id) await picker.selectOption(enemy.id, { timeout: 1500 });
        await attack.click({ timeout: 1500 });
      } catch (error) {
        const next = await (await page.request.get('/api/abyss/combat/state')).json();
        if (next.round === state.round && next.phase === 'planning' && !next.queued) throw error;
      }
    }
    const ready = page.locator('#liveReady');
    if (await ready.isVisible() && await ready.isEnabled()) {
      try { await ready.click({ timeout: 1500 }); }
      catch (error) {
        const next = await (await page.request.get('/api/abyss/combat/state')).json();
        if (next.round === state.round && next.phase === 'planning') throw error;
      }
    }
    await page.waitForTimeout(100);
  }
  throw new Error('Production combat did not complete within 60 seconds');
}

test('real engine and SSE survive refresh, persist completion, bank, defeat and recover', async ({ page, browser }) => {
  test.setTimeout(120_000);
  const errors = [];
  const consoleErrors = [];
  page.on('pageerror', error => errors.push(error.message));
  page.on('console', message => { if (message.type() === 'error') consoleErrors.push({ text: message.text(), location: message.location() }); });
  await page.goto('/abyss/transport');
  await enter(page);
  await page.locator('#btnDescend').click();
  await expect(page.locator('#liveCombat')).toBeVisible();
  await expect(page.locator('#liveActionBar .kind-attack')).toBeEnabled();
  await page.locator('#liveTargetSelect').selectOption('enemy:0');
  const submitted = page.waitForResponse(response => response.url().endsWith('/api/abyss/combat/action'), { timeout: 15_000 });
  await page.locator('#liveActionBar .kind-attack').click();
  const accepted = await (await submitted).json();
  expect(accepted.queued.kind).toBe('attack');

  // Replay the identical action over real HTTP. It must retain the same version.
  const duplicate = await (await page.request.post('/api/abyss/combat/action', { data: accepted.queued })).json();
  expect(duplicate.version).toBe(accepted.version);
  const invalid = await (await page.request.post('/api/abyss/combat/action', { data: {
    ...accepted.queued, target_id: 'enemy:999', idempotency_key: 'invalid-target',
  } })).json();
  expect(invalid.ok).toBe(false);

  // A second isolated browser context uses the same participant cookie. Refresh
  // the first while the second continues; both use actual resume requests.
  const secondContext = await browser.newContext({ baseURL: test.info().project.use.baseURL });
  await secondContext.addCookies(await page.context().cookies());
  const second = await secondContext.newPage();
  await second.goto('/abyss/transport');
  await expect(second.locator('#liveCombat')).toBeVisible();
  const resumes = [];
  page.on('request', request => { if (request.url().includes('/combat/events?since=')) resumes.push(request.url()); });
  await page.reload();
  await expect(page.locator('#liveCombat')).toBeVisible();
  const terminal = await finishFight(second);
  expect(terminal.result.victory).toBe(true);
  await expect.poll(() => page.evaluate(() => window.busy)).toBe(false);
  expect(resumes.length).toBeGreaterThan(0);
  await secondContext.close();

  const persisted = await fixtureState(page);
  const archived = JSON.parse(persisted.session_states[terminal.session_id]);
  expect(archived.snapshot.phase).toBe('complete');
  expect(archived.snapshot.result.victory).toBe(true);
  expect(archived.events.map(event => event.id)).toEqual([...new Set(archived.events.map(event => event.id))].sort((a,b) => a-b));
  expect(archived.events.some(event => event.snapshots[persistedKey(archived)].presentation_events?.length)).toBe(true);
  expect(persisted.commits).toBeGreaterThan(2);
  await page.screenshot({ path: test.info().outputPath('real-transport-victory.png') });

  await page.locator('#btnBank').click();
  await expect(page.locator('#sharedModalCard')).toBeVisible();
  await page.locator('#modalOkBtn').focus();
  await page.locator('#modalOkBtn').press('Enter');
  await expect.poll(async () => (await fixtureState(page)).active).toBe(false);
  await page.reload();
  await enter(page);
  await page.request.post('/api/abyss/transport/control', { data: { defeat: true } });
  await page.locator('#btnDescend').click();
  const defeated = await finishFight(page);
  expect(defeated.result.victory).toBe(false);
  await expect(page.locator('#btnRevive')).toBeVisible();
  await page.getByRole('button', { name: 'I understand', exact: true }).click();
  await page.locator('#btnRevive').focus();
  await page.locator('#btnRevive').click();
  await expect.poll(async () => (await fixtureState(page)).hp).toBe(1000);
  await page.reload();
  await expect(page.locator('#btnDescend')).toBeVisible();
  await page.screenshot({ path: test.info().outputPath('real-transport-recovered.png') });
  const consolePath = test.info().outputPath('real-transport-console.json');
  fs.writeFileSync(consolePath, JSON.stringify({ pageErrors: errors, consoleErrors }, null, 2));
  await test.info().attach('real-transport-console.json', { path: consolePath, contentType: 'application/json' });
  expect(errors).toEqual([]);
});

function persistedKey(archive) {
  return Object.keys(archive.events.find(event => Object.keys(event.snapshots).length).snapshots)[0];
}

for (const mode of ['stream unavailable', 'EventSource unavailable']) {
  test(`real combat converges through polling when ${mode}`, async ({ page }) => {
    test.setTimeout(90_000);
    if (mode === 'EventSource unavailable') await page.addInitScript(() => { window.EventSource = undefined; });
    await page.goto('/abyss/transport');
    await enter(page);
    if (mode === 'stream unavailable') await page.request.post('/api/abyss/transport/control', { data: { stream_failures: 100 } });
    await page.locator('#btnDescend').click();
    await expect(page.locator('#liveConnection')).toContainText('POLL');
    // Let one deadline expire: automatic combat uses the same authoritative clock.
    await expect.poll(async () => (await (await page.request.get('/api/abyss/combat/state')).json()).round).toBeGreaterThan(1);
    const terminal = await finishFight(page);
    expect(terminal.result.victory).toBe(true);
    const state = await fixtureState(page);
    expect(state.polls).toBeGreaterThan(2);
    expect(state.stream_requests).toBe(mode === 'stream unavailable' ? 1 : 0);
    expect(state.escrow).toBe(100);
  });
}

test('a delayed real poll cannot restart polling after same-session SSE reconnect', async ({ page }) => {
  test.setTimeout(90_000);
  await page.goto('/abyss/transport');
  await enter(page);
  await page.locator('#btnDescend').click();
  await expect(page.locator('#liveActionBar .kind-attack')).toBeEnabled();
  let releasePoll;
  const stalledPoll = new Promise(resolve => { releasePoll = resolve; });
  let pollRequests = 0;
  await page.route('**/api/abyss/combat/state', async route => {
    pollRequests++;
    if (pollRequests === 1) await stalledPoll;
    await route.continue(); // Delay only: the real Go handler provides the response.
  });
  await page.evaluate(() => {
    window.savedTransportEventSource = window.EventSource;
    window.EventSource = undefined;
    window.connectLiveCombat();
  });
  await expect.poll(() => pollRequests).toBe(1);
  await page.evaluate(() => {
    window.EventSource = window.savedTransportEventSource;
    window.connectLiveCombat();
  });
  releasePoll();
  await expect(page.locator('#liveConnection')).toContainText('LIVE');
  await page.waitForTimeout(1800); // More than the 900ms poll interval plus transport slack.
  expect(pollRequests).toBe(1);
  await expect(page.locator('#liveConnection')).toContainText('LIVE');
  await page.unroute('**/api/abyss/combat/state');
  expect((await finishFight(page)).result.victory).toBe(true);
});
