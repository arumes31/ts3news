const {test, expect} = require('@playwright/test');
const path = require('path');
const fs = require('fs');

for (const count of [2, 12]) test(`fit preserves ${count} actors while resizing labels in a compact stage`, async ({page}) => {
  await page.setViewportSize({width: 1440, height: 1000});
  await page.emulateMedia({reducedMotion: 'reduce'});
  await page.route('**/static/abyss_fight_visuals.js*', route => route.fulfill({
    path: path.resolve(process.env.ABYSS_FIT_SOURCE || 'internal/bot/webassets/abyss_fight_visuals.js'), contentType: 'application/javascript',
  }));
  await page.route('**/api/abyss/combat/state', route => route.fulfill({json: {ok: false}}));
  await page.goto('/abyss?active=1');
  await page.evaluate(count => {
    const units = Array.from({length: count / 2}, (_, index) => ({id: `ally:${index}`, entity_id: `ally:${index}`,
      name: `Guardian ${index}`, hp: 700, max_hp: 1000, is_player: true, is_self: index === 0, position: 'frontline'}));
    const enemies = units.map((unit, index) => ({id: `enemy:${index}`, entity_id: `enemy:stable-${index}`,
      name: `Warden ${index}`, hp: 700, max_hp: 1000, effects: []}));
    window.connectLiveCombat = function () {};
    window.startLiveCombat({state: {ok: true, session_id: 'fit-e2e', phase: 'planning', round: 1, version: 1,
      deadline: new Date(Date.now() + 600_000).toISOString(), tactic: 'balanced', pause_mode: 'adaptive', policy: {}, social: {},
      allies: units, enemies, options: [{kind: 'attack', id: '', name: 'Attack', target: 'enemy'}],
      recent_logs: [], log_cursor: 0, initiative: [], enemy_intents: [], presentation_events: [], presentation_cursor: 0}}, 700, 1000);
    // Exercise real sprite resizing below the 110px cap. The compact stage is
    // representative of limited cockpit space; viewport stays fixed for comparison.
    Object.assign(document.getElementById('livePixelStage').style, {height: '160px', minHeight: '160px', maxHeight: '160px', flex: '0 0 160px'});
  }, count);
  await expect(page.locator('#liveCombat')).toBeVisible();
  const cdp = await page.context().newCDPSession(page);
  await cdp.send('Performance.enable');
  const before = await cdp.send('Performance.getMetrics');
  const result = await page.evaluate(() => {
    const stage = document.getElementById('livePixelStage');
    const units = [...stage.querySelectorAll('.ab-pixel-unit')];
    const sprites = units.map(unit => unit.querySelector('.ab-actor-sprite'));
    const identities = sprites.map(sprite => sprite.style.backgroundImage);
    const geometry = () => ({stage: stage.getBoundingClientRect().toJSON(), actors: sprites.map((sprite, index) => ({
      entity: units[index].dataset.entityId, size: units[index].style.getPropertyValue('--fight-sprite-height'), bounds: sprite.getBoundingClientRect().toJSON(),
    }))});
    window.AbyssFightVisuals.fit();
    const initial = geometry(), samples = [], start = performance.now();
    // Controlled label-height changes model metadata updates; this is a fit
    // microbenchmark, not a combat frame-rate measurement.
    for (let iteration = 0; iteration < 30; iteration++) {
      units.forEach(unit => { unit.querySelector('.ab-combat-unit-info').style.paddingBottom = iteration % 2 ? '0px' : '2px'; });
      const tick = performance.now(); window.AbyssFightVisuals.fit(); samples.push(performance.now() - tick);
      stage.getBoundingClientRect();
    }
    const changingMs = performance.now() - start;
    const settled = geometry(), unchangedStart = performance.now();
    for (let iteration = 0; iteration < 30; iteration++) window.AbyssFightVisuals.fit();
    const unchangedMs = performance.now() - unchangedStart;
    return {initial, settled, samples, changingMs, unchangedMs,
      identitiesPreserved: sprites.every((sprite, index) => sprite === units[index].querySelector('.ab-actor-sprite') && sprite.style.backgroundImage === identities[index]),
      final: geometry()};
  });
  const after = await cdp.send('Performance.getMetrics');
  const metric = (response, name) => response.metrics.find(item => item.name === name).value;
  result.layoutCount = metric(after, 'LayoutCount') - metric(before, 'LayoutCount');
  result.layoutMs = (metric(after, 'LayoutDuration') - metric(before, 'LayoutDuration')) * 1000;
  // Allow three layouts per resize plus setup; adding actors must not cause
  // another forced layout for each inherited sprite-size write.
  expect(result.layoutCount).toBeLessThanOrEqual(100);
  expect(result.identitiesPreserved).toBe(true);
  expect(result.final).toEqual(result.initial);
  expect(result.final.actors).toHaveLength(count);
  for (const actor of result.final.actors) {
    expect(actor.bounds.width).toBeGreaterThan(0); expect(actor.bounds.height).toBeGreaterThan(0);
    expect(actor.bounds.left).toBeGreaterThanOrEqual(result.final.stage.left - 1);
    expect(actor.bounds.right).toBeLessThanOrEqual(result.final.stage.right + 1);
    expect(actor.bounds.top).toBeGreaterThanOrEqual(result.final.stage.top - 1);
    expect(actor.bounds.bottom).toBeLessThanOrEqual(result.final.stage.bottom + 1);
  }
  console.log(JSON.stringify({actors: count, layoutCount: result.layoutCount, layoutMs: result.layoutMs,
    fitMs: result.samples.reduce((total, value) => total + value, 0), changingMs: result.changingMs, unchangedMs: result.unchangedMs}));
  const artifact = test.info().outputPath(`fit-${count}.json`);
  fs.writeFileSync(artifact, JSON.stringify(result, null, 2));
  await test.info().attach(`fit-${count}.json`, {path: artifact, contentType: 'application/json'});
});
