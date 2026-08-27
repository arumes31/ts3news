const { test, expect } = require('@playwright/test');

const replay = {
  ok: true,
  replay: {
    version: 2,
    run_id: 71,
    depth: 12,
    victory: true,
    tier: 'nightmare',
    audit_hash: '2e525d56e58e6efc72b321c38f47ff1219e8662a93177f554941698c015de59a',
    run_seed: ['18446744073709551615', '9223372036854775809'],
    choices: [{ depth: 4, kind: 'encounter', value: 'silent_anvil' }],
    floors: [{
      depth: 4,
      biome: 'Silent Anvil',
      victory: true,
      hp: 812,
      max_hp: 1000,
      seed: ['14931284292190084211', '13788917118546029120'],
      logs: ['Player <img data-evil src=x onerror="window.__replayXSS=1"> dealt 188 damage.'],
    }],
  },
};

async function mockObservatoryAPI(page) {
  let configured = false;
  await page.route('**/api/abyss/api-token', async route => {
    const method = route.request().method();
    if (method === 'POST') {
      configured = true;
      await route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({ ok: true, token: 'abp_test_secret_once', prefix: 'abp_test_sec' }),
      });
      return;
    }
    if (method === 'DELETE') configured = false;
    await route.fulfill({
      contentType: 'application/json',
      body: JSON.stringify(method === 'DELETE'
        ? { ok: true, revoked: true }
        : { ok: true, token: { configured, prefix: configured ? 'abp_test_sec' : '' } }),
    });
  });
  await page.route('**/api/abyss/run/replay**', route => route.fulfill({
    contentType: 'application/json',
    body: JSON.stringify(replay),
  }));
}

test('Run Observatory deep-links, manages its key, and safely renders signed replay data', async ({ page }) => {
  const pageErrors = [];
  page.on('pageerror', error => pageErrors.push(error.message));
  await mockObservatoryAPI(page);
  await page.goto('/abyss#observatory');

  const panel = page.locator('#abyssObservatory');
  await expect(panel).toBeVisible();
  await expect(page.locator('#abyss-tab-observatory')).toHaveAttribute('aria-selected', 'true');
  await expect(panel).toHaveCSS('background-color', 'rgb(9, 15, 24)');

  await page.evaluate(() => { window.confirmModal = async () => true; });
  await panel.getByRole('button', { name: 'Generate / rotate' }).click();
  await expect(page.locator('#abyssAPISecret')).toBeVisible();
  await expect(page.locator('#abyssAPISecret code')).toHaveText('abp_test_secret_once');

  await page.evaluate(() => window.viewArchivedAbyssRunReplay(71));
  const archive = page.locator('.ab-archive-modal');
  await expect(archive).toBeVisible();
  await expect(archive).toContainText('18446744073709551615');
  await expect(archive).toContainText('<img data-evil');
  await expect(archive.locator('img[data-evil]')).toHaveCount(0);
  expect(await page.evaluate(() => window.__replayXSS)).toBeUndefined();
  await page.locator('#modalOkBtn').click();

  await panel.getByRole('button', { name: 'Revoke' }).click();
  await expect(page.locator('#abyssAPIStatus')).toHaveText('No integration key configured.');
  await expect(page.locator('#abyssAPISecret')).toBeHidden();
  expect(pageErrors).toEqual([]);
});

test('Run Observatory remains opaque and contained on a mobile viewport', async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await mockObservatoryAPI(page);
  await page.goto('/abyss#observatory');

  const panel = page.locator('#abyssObservatory');
  await expect(panel).toBeVisible();
  await expect(panel).toHaveCSS('background-color', 'rgb(9, 15, 24)');
  expect(await panel.evaluate(element => ({
    contained: element.scrollWidth <= element.clientWidth + 1,
    columns: getComputedStyle(element.querySelector('.ab-observatory-grid')).gridTemplateColumns.split(' ').length,
  }))).toEqual({ contained: true, columns: 1 });
});
