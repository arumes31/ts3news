const { test, expect } = require('@playwright/test');

test('core interface groups risk, loot, log, shortcut, and insurance feedback', async ({ page }) => {
  const pageErrors = [];
  page.on('pageerror', error => pageErrors.push(error.message));
  await page.goto('/abyss');

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
  await page.locator('#modalOkBtn').click();
  await page.evaluate(() => window.addLogHead(document.querySelector('#abyssLog'), 'ab-log-divider', 'Timestamp test'));
  await expect(page.locator('body')).toHaveClass(/ab-log-timestamps/);
  await expect(page.locator('#abyssLog .ab-log-line').last()).toHaveAttribute('data-log-time', /\d/);

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
    await new Promise(resolve => setTimeout(resolve, 300));
    await route.continue();
  });
  await page.evaluate(() => { window.__manifestRefresh = window.refreshRunLootManifest(); });
  await expect(page.locator('#lootManifest')).toHaveAttribute('aria-busy', 'true');
  await expect(page.locator('#lootManifest .ab-manifest-skeleton i')).toHaveCount(3);
  await page.evaluate(() => window.__manifestRefresh);
  await expect(page.locator('#lootManifest')).toHaveAttribute('aria-busy', 'false');
  expect(pageErrors).toEqual([]);
});
