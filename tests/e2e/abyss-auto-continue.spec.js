const { test, expect } = require('@playwright/test');
const { fulfillAbyssAPI } = require('./helpers/abyss');

function victory(count, extra = {}) {
  return { ok: true, victory: true, depth: 12 + count, hp: 750, max_hp: 1000,
    gold: 5000, tokens: 12, escrow: 3500 + count * 100, bonus: 100, risk: 20,
    logs: [], loot: [], dura: [], timeline: [], consumables: [], ...extra };
}

async function enable(page, count = '3') {
  await page.goto('/abyss?active=1');
  await page.evaluate(() => { window.reduceMotion = true; });
  await page.locator('#autoContinueCount').fill(count);
  await page.locator('#autoContinueEnabled').check();
}

test('auto-continue performs the selected normal interactive descents and then stops', async ({ page }) => {
  const requests = [];
  await fulfillAbyssAPI(page, (path, body) => {
    if (!path.endsWith('/descend')) return { ok: false };
    requests.push(body); return victory(requests.length);
  });
  await enable(page, '2');
  await page.locator('#btnDescend').click();
  await expect(page.locator('#autoContinueStatus')).toContainText('Complete', { timeout: 15000 });
  expect(requests).toEqual([{ interactive: true }, { interactive: true }]);
  await expect(page.locator('#autoContinueEnabled')).not.toBeChecked();
  await expect(page.locator('#depthNum')).toHaveText('14');
});

for (const [name, result] of [
  ['defeat', { victory: false, hp: 0, can_revive: true }],
  ['low health', { hp: 200 }],
  ['error', { ok: false, error: 'Encounter unavailable' }],
]) {
  test(`auto-continue stops for ${name}`, async ({ page }) => {
    let requests = 0;
    await fulfillAbyssAPI(page, path => {
      if (!path.endsWith('/descend')) return { ok: false };
      requests += 1; return victory(requests, result);
    });
    await enable(page);
    await page.locator('#btnDescend').click();
    await expect(page.locator('#autoContinueEnabled')).not.toBeChecked({ timeout: 15000 });
    expect(requests).toBe(1);
    expect(await page.evaluate(() => abyssAutoContinue.remaining)).toBe(0);
  });
}

const draft = { pending: true, depth: 13, options: [{ id: 1, name: 'Deep Well', icon: '♥', effect: '+10% maximum HP per stack' }] };

test('boons pause the batch through dismissal and a failed choice, then resume with the original count', async ({ page }) => {
  let descents = 0, choices = 0;
  await fulfillAbyssAPI(page, path => {
    if (path.endsWith('/descend')) return victory(++descents, descents === 1 ? { boon_draft: draft } : {});
    if (path.endsWith('/boon')) return ++choices === 1 ? { ok: false, error: 'Try the choice again' } : { ok: true, name: 'Deep Well', stacks: 1, run_identity: { active: true, draft: { pending: false } } };
    return { ok: false };
  });
  await enable(page, '2');
  await page.clock.install();
  await page.locator('#btnDescend').click();
  const dialog = page.getByRole('dialog');
  await expect(dialog).toContainText('Your remaining descents are saved');
  await dialog.getByRole('button', { name: 'Decide later' }).click();
  await page.clock.fastForward(8000);
  expect(descents).toBe(1);
  await expect(page.locator('#autoContinueEnabled')).toBeChecked();
  expect(await page.evaluate(() => abyssAutoContinue.remaining)).toBe(1);
  await page.locator('.ab-boon-draft-trigger').click();
  await dialog.getByRole('button', { name: /Deep Well/ }).click();
  await expect(page.locator('#abToastHost')).toContainText('Try the choice again');
  await page.clock.fastForward(5000);
  expect(descents).toBe(1);
  await dialog.getByRole('button', { name: /Deep Well/ }).click();
  await expect(dialog).toBeHidden();
  await page.clock.fastForward(2500);
  await expect(page.locator('#autoContinueStatus')).toContainText('Complete');
  expect(descents).toBe(2);
});

test('a normal popup waits without cancelling the next descent', async ({ page }) => {
  let descents = 0;
  await fulfillAbyssAPI(page, path => path.endsWith('/descend') ? victory(++descents) : { ok: false });
  await enable(page, '2');
  await page.clock.install();
  await page.locator('#btnDescend').click();
  await expect(page.locator('#autoContinueStatus')).toContainText('Next descent');
  await page.evaluate(() => openModal('<h3>Run information</h3><button onclick="closeModal()">Close</button>'));
  await page.clock.fastForward(5000);
  expect(descents).toBe(1);
  await expect(page.locator('#autoContinueEnabled')).toBeChecked();
  await page.getByRole('dialog').getByRole('button', { name: 'Close' }).click();
  await page.clock.fastForward(2500);
  await expect(page.locator('#autoContinueStatus')).toContainText('Complete');
  expect(descents).toBe(2);
});

test('choosing a path completes the same descent without consuming an extra slot', async ({ page }) => {
  let descents = 0, paths = 0;
  await fulfillAbyssAPI(page, path => {
    if (path.endsWith('/descend')) return ++descents === 1 ? { ok: true, choose_floor: true, depth: 13, options: [{ index: 0, label: 'Combat' }] } : victory(descents);
    if (path.endsWith('/choose_floor')) { paths++; return victory(1); }
    return { ok: false };
  });
  await enable(page, '2');
  await page.clock.install();
  await page.locator('#btnDescend').click();
  await expect(page.getByRole('dialog')).toBeVisible();
  await page.clock.fastForward(5000);
  expect(descents).toBe(1);
  await page.getByRole('dialog').getByRole('button', { name: 'Combat' }).click();
  await expect(page.locator('#autoContinueStatus')).toContainText('Next descent');
  await page.clock.fastForward(2500);
  await expect(page.locator('#autoContinueStatus')).toContainText('Complete');
  expect({ descents, paths }).toEqual({ descents: 2, paths: 1 });
});

for (const room of ['event', 'rest']) {
  test(`auto-continue survives leaving a ${room} floor and its required refresh`, async ({ page }) => {
    let descents = 0, proceeds = 0, depth = 12;
    await page.route('**/abyss?active=1', async route => {
      const response = await route.fetch();
      await route.fulfill({ response, body: (await response.text()).replace(/var curDepth\s*=\s*\d+\s*;/, `var curDepth = ${depth};`) });
    });
    await fulfillAbyssAPI(page, path => {
      if (path.endsWith('/descend')) {
        depth++;
        return victory(++descents, descents === 1 ? { depth, noncombat: true, floor_type: room, event_state: JSON.stringify({ type: 'statue' }) } : { depth });
      }
      if (path.endsWith('/noncombat/action')) return { ok: true, resolved: true, msg: 'Statue resolved.' };
      if (path.endsWith('/noncombat/proceed')) { proceeds++; return { ok: true, resolved: true, depth, bonus: 100, escrow: 3600 }; }
      return { ok: false };
    });
    await enable(page, '2');
    await page.locator('#btnDescend').click();
    await expect(page.locator('#nonCombatPanel')).toBeVisible();
    await expect(page.locator('#autoContinueEnabled')).toBeChecked();
    expect(await page.evaluate(() => abyssAutoContinue.remaining)).toBe(1);
    if (room === 'event') await page.getByRole('button', { name: /Touch the Statue/ }).click();
    else await page.locator('#btnProceed').click();
    await expect(page.locator('#autoContinueStatus')).toContainText('Complete', { timeout: 15000 });
    expect({ descents, proceeds }).toEqual({ descents: 2, proceeds: 1 });
    expect(await page.evaluate(() => sessionStorage.getItem('abyssAutoEventResume'))).toBeNull();
    await page.reload();
    await expect(page.locator('#autoContinueEnabled')).not.toBeChecked();
  });
}

test('invalid event handoffs are consumed without re-arming a batch', async ({ page }) => {
  await page.goto('/abyss?active=1');
  for (const invalid of ['expired', 'other run', 'other floor', 'too many', 'missing time']) {
    await page.evaluate(kind => {
      const saved = { run: abyssAutoRunKey, depth: curDepth, remaining: 1, count: 2, at: Date.now() };
      if (kind === 'expired') saved.at -= 60000;
      if (kind === 'other run') saved.run = 'another expedition';
      if (kind === 'other floor') saved.depth++;
      if (kind === 'too many') saved.remaining = 30;
      if (kind === 'missing time') delete saved.at;
      sessionStorage.setItem('abyssAutoEventResume', JSON.stringify(saved));
    }, invalid);
    await page.reload();
    await expect(page.locator('#autoContinueEnabled')).not.toBeChecked();
    expect(await page.evaluate(() => sessionStorage.getItem('abyssAutoEventResume'))).toBeNull();
  }
});

test('stopping while a boon is pending prevents its choice from restarting the batch', async ({ page }) => {
  let descents = 0;
  await fulfillAbyssAPI(page, path => path.endsWith('/descend') ? victory(++descents, { boon_draft: draft }) : { ok: true, name: 'Deep Well', stacks: 1, run_identity: { active: true, draft: { pending: false } } });
  await enable(page, '2');
  await page.clock.install();
  await page.locator('#btnDescend').click();
  await page.getByRole('dialog').getByRole('button', { name: 'Decide later' }).click();
  await page.locator('#autoContinueStop').click();
  await page.locator('.ab-boon-draft-trigger').click();
  await page.getByRole('dialog').getByRole('button', { name: /Deep Well/ }).click();
  await expect(page.getByRole('dialog')).toBeHidden();
  await page.clock.fastForward(10000);
  expect(descents).toBe(1);
  await expect(page.locator('#autoContinueEnabled')).not.toBeChecked();
});

test('Stop cancels a pending continuation and reload never re-arms it', async ({ page }) => {
  let requests = 0;
  await fulfillAbyssAPI(page, path => path.endsWith('/descend') ? victory(++requests) : { ok: false });
  await enable(page);
  await page.locator('#btnDescend').click();
  await expect(page.locator('#autoContinueStatus')).toContainText('Next descent');
  await page.locator('#autoContinueStop').click();
  await page.clock.install();
  await page.clock.fastForward(5000);
  expect(requests).toBe(1);
  await page.reload();
  await expect(page.locator('#autoContinueEnabled')).not.toBeChecked();
});

test('an oversized count performs at most 30 descents', async ({ page }) => {
  let requests = 0;
  await fulfillAbyssAPI(page, path => path.endsWith('/descend') ? victory(++requests) : { ok: false });
  await enable(page, '99');
  await page.clock.install();
  await page.locator('#btnDescend').click();
  for (let count = 1; count <= 30; count += 1) {
    await expect.poll(() => requests).toBe(count);
    await expect.poll(() => page.evaluate(() => busy)).toBe(false);
    if (count < 30) await page.clock.fastForward(2500);
  }
  await expect(page.locator('#autoContinueStatus')).toContainText('Complete');
  await page.clock.fastForward(10000);
  expect(requests).toBe(30);
});
