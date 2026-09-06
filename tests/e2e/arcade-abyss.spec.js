const { test, expect } = require('@playwright/test');

const outcomes = {
  slots: { symbols: ['🍒', '🍋', '🔔', '⭐', '💎'], detail: 'No match', payout: 0, net: -100, win: false },
  dice: { roll: 4, detail: 'Rolled 4 — push', payout: 100, net: 0, win: false },
  coinflip: { side: 'tails', detail: 'tails — you win ×1.95', payout: 195, net: 95, win: true },
  wheel: { segment: 11, mult: 5, detail: 'Won ×5', payout: 500, net: 400, win: true },
  highlow: { card: 3, detail: 'Drew 3 — win ×2', payout: 200, net: 100, win: true },
};

test.use({ reducedMotion: 'reduce' });

test('all five Abyss games show the server outcome and keep usable controls', async ({ page }) => {
  const errors = [];
  page.on('pageerror', error => errors.push(error.message));
  const requests = [];
  await page.route('**/api/arcade/play', async route => {
    const request = route.request().postDataJSON();
    requests.push(request);
    await route.fulfill({ json: { ok: true, game: request.game, bet: 100, gold: 24900, ...outcomes[request.game] } });
  });
  await page.goto('/arcade');
  for (const [game, button] of [['slots', 'Spin relics'], ['dice', 'Roll dice'], ['coinflip', 'Tails'], ['wheel', 'Spin wheel'], ['highlow', 'Low <7']]) {
    const card = page.locator(`[data-game="${game}"]`);
    await card.getByRole('button', { name: button, exact: true }).click();
    await expect(card.locator('.game-result')).toContainText(outcomes[game].detail);
    await expect(card.getByRole('button').first()).toBeEnabled();
  }
  await expect(page.locator('[data-game="dice"] .game-result')).toContainText('Bet returned');
  await expect(page.locator('[data-game="dice"]')).not.toHaveClass(/loss-shake/);
  await expect(page.locator('#cardVal')).toHaveText('3');
  await expect(page.locator('#coin')).toHaveAttribute('aria-label', 'Coin: tails');
  await expect(page.locator('#reels')).toHaveAttribute('aria-label', /Health potion, Sapphire, Ring, Rune, Sword/);
  expect(requests.map(request => request.game)).toEqual(Object.keys(outcomes));
  expect(requests[2].choice).toBe('tails');
  expect(requests[4].choice).toBe('low');
  expect(errors).toEqual([]);
});

test('failed requests unlock games and stop auto-bet', async ({ page }) => {
  await page.route('**/api/arcade/play', route => route.abort());
  await page.goto('/arcade');
  await page.locator('#autoBet').check();
  await page.getByRole('button', { name: 'Roll dice', exact: true }).click();
  await expect(page.locator('#arcadeMsg')).toContainText('Could not confirm');
  await expect(page.locator('#autoBet')).not.toBeChecked();
  await expect(page.locator('#btn-dice')).toBeEnabled();
  await expect(page.locator('#btn-coin-tails')).toBeEnabled();
});

test('unchecking auto-bet cancels the queued wager', async ({ page }) => {
  let plays = 0;
  await page.route('**/api/arcade/play', async route => {
    plays++;
    await route.fulfill({ json: { ok: true, game: 'dice', bet: 100, gold: 25000, ...outcomes.dice } });
  });
  await page.goto('/arcade');
  await page.locator('#autoBet').check();
  await page.locator('#btn-dice').click();
  await expect(page.locator('[data-game="dice"] .game-result')).toContainText('Bet returned');
  await page.locator('#autoBet').uncheck();
  await page.waitForTimeout(1200);
  expect(plays).toBe(1);
});

test('Abyss art loads and arcade fits desktop and narrow screens', async ({ page }) => {
  for (const width of [1440, 768, 390, 320]) {
    await page.setViewportSize({ width, height: 1000 });
    await page.goto('/arcade');
    await expect(page.locator('body')).toHaveClass(/delver-shell/);
    await expect(page.locator('.game-card')).toHaveCount(8);
    await expect(page.getByRole('button', { name: 'Low <7', exact: true })).toBeVisible();
    expect(await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth)).toBeLessThanOrEqual(1);
    const atlas = await page.locator('.arcade-art').first().evaluate(node => getComputedStyle(node).backgroundImage);
    expect(atlas).toContain('abyss_icon_atlas.png');
    const response = await page.request.get(atlas.match(/url\("?([^"\)]+)/)[1]);
    expect(response.ok()).toBe(true);
  }
});

test('treasure and expedition choices reveal the authoritative outcomes', async ({ page }) => {
  const requests = [];
  await page.route('**/api/arcade/play', async route => {
    const body = route.request().postDataJSON();
    requests.push(body);
    await route.fulfill({ json: {
      ok: true, game: body.game, bet: 100, gold: 26000, win: true, net: 185, payout: 285,
      ...(body.game === 'vault'
        ? { chest: 2, detail: 'Treasure found in chest 2' }
        : { roll: 24, chance: 24, mult: 4, net: 300, payout: 400, detail: 'Expedition survived' }),
    } });
  });
  await page.goto('/arcade');
  await page.getByRole('button', { name: 'Open chest 2', exact: true }).click();
  await expect(page.locator('.treasure-chest.treasure')).toHaveCount(1);
  await expect(page.locator('.treasure-chest').nth(1)).toHaveClass(/treasure.*opened|opened.*treasure/);
  await page.getByRole('radio', { name: 'Abyss 24% · ×4' }).check();
  await expect(page.locator('#expeditionHint')).toContainText('1–24');
  await page.locator('#btn-expedition').click();
  await expect(page.locator('[data-game="expedition"] .game-result')).toContainText('Expedition survived');
  await expect(page.locator('.expedition-meter')).toHaveAttribute('aria-label', 'Rolled 24; survival threshold 24');
  expect(requests.map(({ game, choice }) => ({ game, choice }))).toEqual([
    { game: 'vault', choice: '2' }, { game: 'expedition', choice: 'abyss' },
  ]);
});

test('rune memory mismatches, matches, completes, and restarts without wagering', async ({ page }) => {
  let wagers = 0;
  page.on('request', request => { if (request.url().includes('/api/arcade/')) wagers++; });
  await page.goto('/arcade');
  const balance = await page.locator('#goldHere').textContent();
  await page.locator('#btn-memory').click();
  const tiles = page.locator('.memory-tile');
  await expect(tiles).toHaveCount(16);
  await expect(tiles.first()).toBeFocused();
  const runes = await tiles.evaluateAll(nodes => nodes.map(node => node.dataset.rune));
  const other = runes.findIndex(rune => rune !== runes[0]);
  await tiles.nth(0).click();
  await tiles.nth(other).click();
  await expect(tiles.nth(0)).not.toHaveClass(/revealed/);
  await expect(page.locator('#memoryMoves')).toHaveText('1');
  for (const rune of new Set(runes)) {
    const indices = runes.map((value, index) => value === rune ? index : -1).filter(index => index >= 0);
    await tiles.nth(indices[0]).click();
    await tiles.nth(indices[1]).click();
    await expect(tiles.nth(indices[0])).toHaveClass(/matched/);
  }
  await expect(page.locator('#memoryPairs')).toHaveText('8 / 8');
  await expect(page.locator('[data-game="memory"] .game-result')).toContainText('Archive restored');
  await expect(page.locator('#memoryBest')).toContainText('9 moves');
  await page.locator('#btn-memory').click();
  await expect(page.locator('#memoryPairs')).toHaveText('0 / 8');
  await expect(page.locator('#memoryMoves')).toHaveText('0');
  await expect(page.locator('#goldHere')).toHaveText(balance);
  expect(wagers).toBe(0);
});

test('daily tribute claims once and updates the balance', async ({ page }) => {
  await page.route('**/api/arcade/daily-spin', route => route.fulfill({ json: { ok: true, reward: 'Looted 500 gold!', new_gold: 25500 } }));
  await page.goto('/arcade');
  await page.locator('#btn-daily').click();
  await expect(page.locator('.arcade-daily')).toHaveCount(0);
  await expect(page.locator('#goldHere')).toHaveText('25,500');
  await expect(page.locator('#arcadeMsg')).toContainText('Looted 500 gold');
});

test('an in-flight wager locks all wager controls but leaves the free game available', async ({ page }) => {
  let release;
  const pending = new Promise(resolve => { release = resolve; });
  const requests = [];
  await page.route('**/api/arcade/play', async route => {
    requests.push(route.request().postDataJSON());
    await pending;
    await route.fulfill({ json: { ok: true, game: 'coinflip', bet: 100, gold: 25095, ...outcomes.coinflip } });
  });
  await page.goto('/arcade');
  await page.locator('#btn-coin-tails').click();
  try {
    await expect(page.locator('#btn-coin')).toBeDisabled();
    await expect(page.locator('#btn-dice')).toBeDisabled();
    await expect(page.locator('#bet')).toBeDisabled();
    await expect(page.locator('#btn-daily')).toBeDisabled();
    await expect(page.locator('#btn-expedition')).toBeDisabled();
    await page.locator('#btn-memory').click();
    await expect(page.locator('.memory-tile')).toHaveCount(16);
  } finally { release(); }
  await expect(page.locator('[data-game="coinflip"] .game-result')).toContainText('tails');
  await expect(page.locator('#btn-dice')).toBeEnabled();
  expect(requests).toHaveLength(1);
});

test('invalid wagers never call the API and rejected rounds remain playable', async ({ page }) => {
  let requests = 0;
  await page.route('**/api/arcade/play', async route => {
    requests++;
    await route.fulfill({ json: { ok: false, error: 'not enough gold' } });
  });
  await page.goto('/arcade');
  for (const value of ['0', '100000001', '1.5', '']) {
    await page.locator('#bet').fill(value);
    await page.locator('#btn-dice').click();
    await expect(page.locator('#arcadeMsg')).toContainText('whole-gold wager');
  }
  expect(requests).toBe(0);
  await page.locator('#bet').fill('100');
  await page.locator('#btn-dice').click();
  await expect(page.locator('#arcadeMsg')).toHaveText('not enough gold');
  await expect(page.locator('#btn-dice')).toBeEnabled();
  expect(requests).toBe(1);
});

test('animated reels and repeated wheel spins settle on server results', async ({ page }) => {
  await page.emulateMedia({ reducedMotion: 'no-preference' });
  const segments = [0, 11];
  await page.route('**/api/arcade/play', async route => {
    const body = route.request().postDataJSON();
    await route.fulfill({ json: {
      ok: true, game: body.game, bet: 100, gold: 30000, win: true, net: 8700, payout: 8800,
      ...(body.game === 'slots'
        ? { symbols: Array(5).fill('7️⃣'), detail: 'JACKPOT! 5 of a kind ×88' }
        : { segment: segments.shift(), detail: 'Wheel settled', payout: 500, net: 400 }),
    } });
  });
  await page.goto('/arcade');
  await page.locator('#btn-slots').click();
  await expect(page.locator('.reel.matched')).toHaveCount(5);
  for (const index of [0, 4]) {
    const reel = page.locator('.reel').nth(index);
    const art = reel.locator('.sym').last();
    const reelBox = await reel.boundingBox();
    const artBox = await art.boundingBox();
    expect(Math.abs(reelBox.y - artBox.y)).toBeLessThan(1);
  }
  for (const label of ['Wheel result: blank', 'Wheel result: ×5']) {
    await page.locator('#btn-wheel').click();
    await expect(page.locator('#wheel')).toHaveAttribute('aria-label', label);
    await expect(page.locator('#btn-wheel')).toBeEnabled();
  }
});
