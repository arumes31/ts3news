const { test, expect } = require('@playwright/test');

test('core interface groups risk, loot, log, shortcut, and insurance feedback', async ({ page }) => {
  const pageErrors = [];
  page.on('pageerror', error => pageErrors.push(error.message));
  await page.addInitScript(() => {
    class TestNotification {
      static permission = 'granted';
      static requestPermission() { return Promise.resolve('granted'); }
      constructor(title, options) { window.__testNotifications.push({ title, options }); }
    }
    window.__testNotifications = [];
    Object.defineProperty(window, 'Notification', { configurable: true, value: TestNotification });
  });
  await page.goto('/abyss');

  await expect(page.locator('html')).toHaveAttribute('lang', 'en-US');
  await expect(page.locator('html')).toHaveAttribute('dir', 'ltr');
  expect(await page.evaluate(() => {
    const original = document.documentElement.lang;
    document.documentElement.lang = 'de-DE';
    const values = {
      integer: window.AbyssLocale.integer(1234567),
      decimal: window.AbyssLocale.number(1234.5, { minimumFractionDigits: 1 }),
      percent: window.AbyssLocale.percent(.125, 1),
    };
    document.documentElement.lang = original;
    return values;
  })).toEqual({ integer: '1.234.567', decimal: '1.234,5', percent: '12,5 %' });
  await page.evaluate(() => {
    const button = document.createElement('button');
    button.id = 'dynamicEmojiAction';
    button.textContent = '🏦 Secure cache';
    document.querySelector('.abyss-command-page').appendChild(button);
  });
  await expect(page.locator('#dynamicEmojiAction')).toHaveAttribute('aria-label', 'Secure cache');
  await expect(page.locator('#dynamicEmojiAction .ab-emoji-icon')).toHaveAttribute('aria-hidden', 'true');

  await page.evaluate(() => {
    window.inRun = true;
    window.curDepth = 3;
    window.curRisk = 27;
    window.updateThreat();
    const items = [
      { id: 1, depth: 2, item_type: 'gear', title: 'Rare Blade', label: 'Rare Blade', rarity: 'Rare', rarity_rank: 3, estimated_value: 12000 },
      { id: 2, depth: 2, item_type: 'mat', title: 'Void Shards', label: 'Void Shards ×3', estimated_value: 30000 },
      { id: 3, depth: 3, item_type: 'gold', title: 'Gold Cache', label: 'Gold Cache', estimated_value: 5000 },
    ];
    const manifest = document.querySelector('#lootManifest');
    manifest.replaceChildren(...items.map(window.buildAuthoritativeRunLootRow));
    window.updateLootSummary();
  });

  await expect(page.locator('#threatPct')).toHaveText('27%');
  await expect(page.locator('#bossDistance')).toHaveText('👑 2');
  await expect(page.locator('#lootSummary')).toContainText('≈47.0k');

  await page.locator('#lootTypeFilters [data-type="mat"]').click();
  await expect(page.locator('#lootManifest .abyss-side-loot:visible')).toHaveCount(1);
  await expect(page.locator('#lootManifest .abyss-side-loot:visible')).toContainText('Void Shards');
  await expect(page.locator('#lootSummary')).toContainText('≈30.0k');
  expect(await page.evaluate(() => localStorage.getItem('ab_loottype'))).toBe('mat');

  await page.locator('#lootRarityHelp').click();
  await expect(page.locator('#sharedModal')).toHaveClass(/open/);
  await expect(page.locator('.ab-rarity-guide li')).toHaveCount(9);
  await page.locator('#modalOkBtn').click();

  await page.keyboard.press('?');
  await expect(page.locator('#sharedModalCard')).toContainText('Abyss shortcuts');
  await expect(page.locator('#sharedModalCard')).toContainText('Descend when available');
  await page.locator('#modalOkBtn').click();

  await page.locator('#abSettingsBtn').click();
  const timestamps = page.locator('#abSettingsRows .ab-set-row').filter({ hasText: 'Combat log timestamps' });
  await timestamps.locator('input').check();
  const detail = page.locator('#abSettingsRows .ab-set-row').filter({ hasText: 'Combat log detail' });
  await detail.locator('select').selectOption('summary');
  const notifications = page.locator('#abSettingsRows .ab-set-row').filter({ hasText: 'Browser notifications' });
  await notifications.locator('input').check();
  await page.locator('#modalOkBtn').click();
  await page.evaluate(() => {
    const log = document.querySelector('#abyssLog');
    window.addLogHead(log, 'ab-log-divider', 'Timestamp test');
    window.addLogHead(log, '', 'You attack and deal 50 damage.');
    window.addLogHead(log, '', 'Floor 3 survived.');
    window.AbyssQoL.updateFavicon(21, true);
    window.AbyssQoL.notify('bounty', 'Abyss bounty complete', 'Test reward', 'test-day');
    window.AbyssQoL.notify('bounty', 'Abyss bounty complete', 'Duplicate reward', 'test-day');
  });
  await expect(page.locator('body')).toHaveClass(/ab-log-timestamps/);
  await expect(page.locator('#abyssLog .ab-log-line').last()).toHaveAttribute('data-log-time', /\d/);
  expect(await page.evaluate(() => {
    const lines = Array.from(document.querySelectorAll('#abyssLog .ab-log-line'));
    return {
      damage: lines.find(line => line.textContent.includes('deal 50 damage')).style.display,
      summary: lines.find(line => line.textContent.includes('Floor 3 survived')).style.display,
    };
  })).toEqual({ damage: 'none', summary: '' });
  expect(await page.evaluate(() => decodeURIComponent(document.querySelector('link[rel="icon"]').href))).toContain('>21<');
  expect(await page.evaluate(() => window.__testNotifications)).toHaveLength(1);

  await page.evaluate(() => {
    window.curEscrow = Math.max(50000, Math.round((Number(window.escrowSoftCap) || 0) * .5));
    window.curInsured = 0;
    window.__abyssCoreInterface.renderInsuranceNudge();
  });
  await expect(page.locator('#insuranceNudge')).not.toHaveAttribute('hidden', '');
  await expect(page.locator('#insuranceControls')).toHaveClass(/ab-insurance-attention/);
  await expect(page.locator('#insuranceNudge')).toContainText('uninsured');
  await expect(page.locator('#tierPicker .ab-tier-icon')).toHaveCount(4);
  await expect(page.locator('#tierPicker .ab-tier').first().locator('.ab-tier-rate')).toContainText('70% bank rate');

  await page.evaluate(() => {
    const flame = document.querySelector('#dropStreakFlame');
    flame.dataset.streak = '12';
    window.__abyssCoreInterface.renderDropStreakFlame();
    window.applyCombatFrame({ hp: 100, max_hp: 100, mana: 0, max_mana: 0, enemy_hp: 24, enemy_max: 100 });
    window.curDepth = 9;
    window.__abyssCoreInterface.milestoneToast(9);
    window.curDepth = 10;
    window.__abyssCoreInterface.renderDepthPresentation();
    window.__abyssCoreInterface.milestoneToast(10);
    window.achievementBanner('First queued achievement');
    window.achievementBanner('Second queued achievement');
    window.maybeBossCard('💀 BOSS — Gorgoroth');
    document.body.dataset.bossShakeStarted = String(document.querySelector('#abyssStage').classList.contains('ab-boss-spawn-shake'));
    document.querySelector('#abyssStage').classList.add('ab-downed');
  });
  await expect(page.locator('#dropStreakFlame')).toHaveAttribute('data-tier', '3');
  await expect(page.locator('#bossHPOverlay')).toHaveAttribute('data-phase', 'critical');
  await expect(page.locator('#bossHPOverlay')).toHaveCSS('--boss-hp', '24%');
  await expect(page.locator('#abyssStage')).toHaveAttribute('data-depth-band', 'shallow');
  await expect(page.locator('#abyssDepthBackdrop')).toHaveCount(1);
  await expect(page.locator('.ab-milestone-toast')).toContainText('Floor 10');
  await expect(page.locator('#achBanner')).toContainText('First queued achievement');
  await expect(page.locator('#achBanner')).not.toContainText('Second queued achievement');
  await expect(page.locator('body')).toHaveAttribute('data-boss-shake-started', 'true');
  await expect(page.locator('#abyssStage')).toHaveClass(/ab-downed/);
  await expect(page.locator('#escrowVal')).toHaveCSS('color', 'rgb(255, 107, 114)');

  await page.locator('#abCommandCenter').getByRole('button', { name: 'Experience' }).click();
  const fontSize = page.locator('#abExperienceGrid label').filter({ hasText: 'Account font size' }).locator('select');
  await fontSize.selectOption('l');
  await expect(page.locator('body')).toHaveAttribute('data-ab-font-size', 'l');
  expect(await page.evaluate(() => localStorage.getItem('ab_fontsize'))).toBe('l');
  await page.locator('#modalOkBtn').click();

  await page.route('**/api/abyss/loot/manifest', async route => {
    await new Promise(resolve => setTimeout(resolve, 1000));
    await route.continue();
  });
  await page.evaluate(() => { window.__manifestRefresh = window.refreshRunLootManifest(); });
  await expect(page.locator('#lootManifest')).toHaveAttribute('aria-busy', 'true');
  await expect(page.locator('#lootManifest .ab-manifest-skeleton i')).toHaveCount(3);
  await page.evaluate(() => window.__manifestRefresh);
  await expect(page.locator('#lootManifest')).toHaveAttribute('aria-busy', 'false');
  expect(pageErrors).toEqual([]);
});
