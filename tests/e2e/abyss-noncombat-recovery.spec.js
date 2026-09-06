const { test, expect } = require('@playwright/test');

for (const state of ['', 'null', '{}', '{broken']) {
  test(`cleared or invalid event ${JSON.stringify(state)} offers a usable exit`, async ({ page }) => {
    const errors = [];
    page.on('pageerror', error => errors.push(error.message));
    await page.goto('/abyss?active=1&room=forge_floor');
    await page.evaluate(state => {
      floorType = 'event';
      eventState = state;
      renderNonCombat(floorType, eventState);
    }, state);
    await expect(page.locator('#nonCombatPanel')).toBeVisible();
    await expect(page.locator('#ncOptions button')).toHaveCount(0);
    await expect(page.locator('#btnProceed')).toBeVisible();
    await expect(page.locator('#abCurrentObjective')).toContainText('Leave this floor');
    expect(errors).toEqual([]);
  });
}

test('resolved event clears its state and survives a render without offering deferral', async ({ page }) => {
  await page.route('**/api/abyss/noncombat/action', route => route.fulfill({
    contentType: 'application/json', body: JSON.stringify({ ok: true, resolved: true, msg: 'The moment passes.' }),
  }));
  await page.goto('/abyss?active=1&room=forge_floor');
  await page.evaluate(() => {
    floorType = 'event';
    eventState = JSON.stringify({ type: 'echo_floor', echo_reward: 0 });
    renderNonCombat(floorType, eventState);
  });
  await page.getByRole('button', { name: 'Let the Echo Fade' }).click();
  await expect(page.locator('#ncDesc')).toHaveText('The moment passes.');
  expect(await page.evaluate(() => eventState)).toBe('');
  await page.evaluate(() => renderNonCombat(floorType, eventState));
  await expect(page.locator('#ncOptions button')).toHaveCount(0);
  await expect(page.locator('#abCurrentObjective')).toContainText('Leave this floor');
});
