const { test, expect } = require('@playwright/test');
test.use({ reducedMotion: 'reduce' });

test('a stalled request times out, stops auto-bet, and releases the controls', async ({ page }) => {
  await page.clock.install();
  await page.route('**/api/arcade/play', () => {});
  await page.goto('/arcade');
  await page.locator('#autoBet').check();
  const request = page.waitForRequest('**/api/arcade/play');
  await page.locator('#btn-dice').click();
  await request;
  await page.clock.fastForward(16000);
  await expect(page.locator('#arcadeMsg')).toContainText('Could not confirm');
  await expect(page.locator('#btn-dice')).toBeEnabled();
  await expect(page.locator('#autoBet')).not.toBeChecked();
  await expect(page.locator('#die')).not.toHaveClass(/rolling/);
});

const games = [
  ['slots', [''], '#btn-slots'],
  ['dice', [''], '#btn-dice'],
  ['coinflip', ['heads', 'tails'], null],
  ['wheel', [''], '#btn-wheel'],
  ['highlow', ['low', 'high'], null],
  ['vault', ['1', '2', '3'], null],
  ['expedition', ['scout', 'delve', 'abyss'], '#btn-expedition'],
];

test('all wager games accept 100 million and quick bets respect the new limit', async ({ page }) => {
  const requests = [];
  await page.route('**/api/arcade/play', async route => {
    requests.push(route.request().postDataJSON());
    await route.fulfill({ json: { ok: false, error: 'not enough gold' } });
  });
  await page.goto('/arcade');
  const wager = page.locator('#bet');
  await expect(wager).toHaveAttribute('max', '100000000');
  await wager.fill('100000');
  await page.getByRole('button', { name: '×2 — Double wager', exact: true }).click();
  await expect(wager).toHaveValue('200000');
  await wager.fill('75000000');
  await page.getByRole('button', { name: '×2 — Double wager', exact: true }).click();
  await expect(wager).toHaveValue('100000000');
  for (const [game] of games) {
    const card = page.locator('#game-' + game);
    await expect(card.locator('.game-wager')).toContainText('100,000,000 gold');
    await card.locator('button').first().click();
    await expect(card.locator('.game-result')).toHaveText('not enough gold');
  }
  expect(requests.map(request => request.game)).toEqual(games.map(([game]) => game));
  expect(requests.every(request => request.bet === 100000000)).toBe(true);
});

for (const [game, choices, selector] of games) {
  test(game + ': repeated rounds use real server outcomes and conserve test gold', async ({ page }, testInfo) => {
    const errors = [];
    const rounds = [];
    page.on('pageerror', error => errors.push(error.message));
    await page.goto('/arcade');
    let gold = 25000;
    for (let i = 0; i < 12; i++) {
      const choice = choices[i % choices.length];
      if (game === 'expedition') await page.locator('input[value="' + choice + '"]').check();
      const button = game === 'coinflip' ? page.locator(choice === 'heads' ? '#btn-coin' : '#btn-coin-tails')
        : game === 'highlow' ? page.locator(choice === 'low' ? '#btn-low' : '#btn-high')
        : game === 'vault' ? page.locator('.treasure-chest').nth(Number(choice) - 1)
        : page.locator(selector);
      const response = page.waitForResponse(response => response.url().endsWith('/api/arcade/play'));
      await button.click();
      const result = await (await response).json();
      expect(result.ok).toBe(true);
      gold += result.payout - result.bet;
      expect(result.gold).toBe(gold);
      const card = page.locator('[data-game="' + game + '"]');
      await expect(card.locator('.game-result')).toContainText(result.detail);
      await expect(card).toHaveAttribute('data-result', result.net === 0 ? 'push' : result.win ? 'win' : 'loss');
      await expect(page.locator('#goldHere')).toHaveText(gold.toLocaleString('en-US'));
      await expect(button).toBeEnabled();
      rounds.push(result);
    }
    expect(errors).toEqual([]);
    await testInfo.attach(game + '-actual-outcomes', { body: JSON.stringify(rounds, null, 2), contentType: 'application/json' });
  });
}

test('each wager game previews the current stake beside its controls', async ({ page }) => {
  await page.goto('/arcade');
  await page.locator('#bet').fill('125');
  for (const [game] of games) await expect(page.locator('#game-' + game + ' .game-wager')).toContainText('125 gold');
  await expect(page.locator('#game-coinflip .game-wager')).toContainText('243');
  await page.locator('input[value="abyss"]').check();
  await expect(page.locator('#game-expedition .game-wager')).toContainText('500');
});

test('vault closes and clears the old outcome immediately while the next request is pending', async ({ page }) => {
  await page.goto('/arcade');
  await page.locator('.treasure-chest').first().click();
  await expect(page.locator('.treasure-chest.opened')).toHaveCount(3);
  let release;
  const pending = new Promise(resolve => { release = resolve; });
  await page.route('**/api/arcade/play', async route => { await pending; await route.continue(); });
  await page.locator('.treasure-chest').nth(1).click();
  try {
    await expect(page.locator('.treasure-chest.opened')).toHaveCount(0);
    await expect(page.locator('.treasure-chest').nth(1)).toHaveClass(/chosen/);
    await expect(page.locator('.chest-caption').first()).toHaveText('Chest 1');
  } finally { release(); }
  await expect(page.locator('.treasure-chest.opened')).toHaveCount(3);
});

test('oracle exposes only the visible card face and highlights the drawn range', async ({ page }) => {
  await page.goto('/arcade');
  await expect(page.locator('.pcard-back')).toHaveAttribute('aria-hidden', 'true');
  await page.locator('#btn-high').click();
  await expect(page.locator('.pcard-front')).toHaveAttribute('aria-hidden', 'true');
  await expect(page.locator('.pcard-back')).toHaveAttribute('aria-hidden', 'false');
  await expect(page.locator('.oracle-range .selected')).toHaveCount(1);
  await expect(page.locator('#btn-high')).toHaveClass(/chosen/);
});

test('memory supports arrow navigation, pausing, and a complete game using revealed tiles', async ({ page }) => {
  await page.goto('/arcade');
  await page.locator('#btn-memory').click();
  const tiles = page.locator('.memory-tile');
  await page.keyboard.press('ArrowRight');
  await expect(tiles.nth(1)).toBeFocused();
  await page.keyboard.press('Enter');
  await page.locator('#btn-memory-pause').click();
  await expect(page.locator('#memoryBoard')).toHaveAttribute('aria-hidden', 'true');
  const frozenTime = await page.locator('#memoryTime').textContent();
  await page.waitForTimeout(1100);
  await expect(page.locator('#memoryTime')).toHaveText(frozenTime);
  await page.locator('#btn-memory-pause').click();
  await expect(page.locator('#memoryBoard')).toHaveAttribute('aria-hidden', 'false');
  await page.locator('#btn-memory').click();
  const known = new Map();
  const matched = new Set();
  let turns = 0;
  while (matched.size < 16 && turns++ < 40) {
    const unmatched = Array.from({ length: 16 }, (_, i) => i).filter(i => !matched.has(i));
    const first = unmatched.find(i => !known.has(i)) ?? unmatched[0];
    await tiles.nth(first).click();
    const name = (await tiles.nth(first).getAttribute('aria-label')).split(': ')[1];
    known.set(first, name);
    const second = unmatched.find(i => i !== first && known.get(i) === name)
      ?? unmatched.find(i => i !== first && !known.has(i))
      ?? unmatched.find(i => i !== first);
    await tiles.nth(second).click();
    const other = (await tiles.nth(second).getAttribute('aria-label')).split(': ')[1].replace(', matched', '');
    known.set(second, other);
    if (name === other) { matched.add(first); matched.add(second); }
    else await expect(tiles.nth(first)).not.toHaveClass(/revealed/);
  }
  expect(matched.size).toBe(16);
  await expect(page.locator('#game-memory .game-result')).toContainText('Archive restored');
});
