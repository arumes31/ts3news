const { test, expect } = require('@playwright/test');

test.describe('Rift Results Summary Export and Statistics Preferences', () => {
  test('offers copyable result summary free of session tokens and formats large totals', async ({ page }) => {
    await page.goto('/abyss/rift?scenario=terminal-defeated');
    await expect(page.locator('#rift-overlay')).toBeVisible();

    const copyBtn = page.locator('#rift-copy-result-summary');
    await expect(copyBtn).toBeVisible();

    // Mock clipboard write to verify both button interaction and copied payload
    await page.evaluate(() => {
      window.__copiedText = '';
      navigator.clipboard.writeText = async (text) => {
        window.__copiedText = text;
      };
    });

    const summaryText = await page.evaluate(() => {
      const run = {
        id: 'secret-session-token-abc-987',
        status: 'defeated',
        room: 1,
        level: {
          id: 5,
          name: 'Sunken Citadel',
          rooms: [
            { name: 'Outer Moat' },
            { name: 'Sunken Hall' }
          ]
        },
        banked_gold: 150000,
        banked_items: ['Rare Helm', 'Epic Blade'],
        gold: 25000,
        drops: [{ id: 1, banked: false }, { id: 2, banked: false }],
        defeated_by_enemy: 'Abyssal Warden',
        defeat_cause: 'Lethal blow from Abyssal Warden',
        player: { hp: 0, max_hp: 2500 },
        stats: {
          seconds: 75.4,
          damage_dealt: 1250000,
          damage_taken: 84500,
          hits_taken: 32,
          healing: 12000,
          guard_blocked: 45000
        }
      };
      return window.RiftHUD.buildResultSummary(run);
    });

    // 0864: Offer copyable result summary
    expect(summaryText).toContain('Abyss Rift Brawl — Result Summary');
    expect(summaryText).toContain('Mission: Sunken Citadel');
    expect(summaryText).toContain('Tier 2: Sunken Hall');
    expect(summaryText).toContain('75.4s combat');

    // 0865: Keep copied summaries free of session tokens
    expect(summaryText).not.toContain('secret-session-token');
    expect(summaryText).not.toContain('abc-987');
    expect(summaryText).not.toContain('token');
    expect(summaryText).not.toContain('session');

    // 0868: Readable locale formatting for large totals
    expect(summaryText).toMatch(/1[, ]?250[, ]?000/);
    expect(summaryText).toMatch(/150[, ]?000/);

    // Test copy button trigger
    await copyBtn.click();
    const copyStatus = page.locator('#rift-result-copy-status');
    await expect(copyStatus).toBeVisible();
    await expect(copyStatus).toContainText('Summary copied');

    // Verify copied text via mock
    const copiedFromButton = await page.evaluate(() => window.__copiedText);
    expect(copiedFromButton).toContain('Abyss Rift Brawl — Result Summary');
    expect(copiedFromButton).not.toContain('session');
  });

  test('allows hiding detailed statistics by default and remembers detail expansion preference', async ({ page }) => {
    await page.goto('/abyss/rift?scenario=checkpoint');
    await expect(page.locator('#rift-start')).toBeEnabled();

    const toggleBtn = page.locator('#rift-toggle-encounter-stats');
    const statsDl = page.locator('#rift-last-encounter-stats');
    const prefCheckbox = page.locator('#rift-hide-detailed-stats');

    // Initially details are shown
    await expect(toggleBtn).toBeVisible();
    await expect(toggleBtn).toHaveText('Hide details');
    await expect(toggleBtn).toHaveAttribute('aria-expanded', 'true');
    await expect(statsDl).toBeVisible();

    // 0867: Click toggle button to collapse details
    await toggleBtn.click();
    await expect(toggleBtn).toHaveText('Show details');
    await expect(toggleBtn).toHaveAttribute('aria-expanded', 'false');
    await expect(statsDl).toBeHidden();

    // Verify preference is remembered in localStorage
    const savedPref = await page.evaluate(() => localStorage.getItem('riftHideDetailedStats'));
    expect(savedPref).toBe('true');

    // Open settings panel to view checkbox
    await page.locator('.rift-settings > summary').click();
    // Preference checkbox in settings is also in sync
    await expect(prefCheckbox).toBeChecked();

    // Reload page to verify persistence
    await page.reload();
    await expect(page.locator('#rift-toggle-encounter-stats')).toHaveText('Show details');
    await expect(page.locator('#rift-toggle-encounter-stats')).toHaveAttribute('aria-expanded', 'false');
    await expect(page.locator('#rift-last-encounter-stats')).toBeHidden();

    // 0866: Toggle checkbox back in settings
    await page.locator('.rift-settings > summary').click();
    await prefCheckbox.uncheck();
    await expect(page.locator('#rift-toggle-encounter-stats')).toHaveText('Hide details');
    await expect(page.locator('#rift-last-encounter-stats')).toBeVisible();
  });

  test('avoids empty statistics rows for unused mechanics in encounter and run stats', async ({ page }) => {
    await page.goto('/abyss/rift?scenario=checkpoint');
    await expect(page.locator('#rift-start')).toBeEnabled();

    // Fetch genuine run structure from server and update stats
    const { run } = await (await page.request.get('/api/abyss/rift')).json();
    await page.evaluate(({ run }) => {
      run.stats.damage_dealt = 500;
      run.stats.damage_taken = 50;
      run.stats.hazard_damage_taken = 0;
      run.stats.enemy_damage_taken = 50;
      run.stats.guard_blocked = 20;
      run.stats.barrier_blocked = 0; // unused mechanic
      run.stats.charged_finishers = 0; // unused mechanic
      run.stats.empty_finishers = 0;
      run.stats.charges_spent = 0;
      run.stats.armor_blocked = 0; // unused mechanic
      run.stats.treasure_goblins = 0; // unused mechanic
      run.last_encounter = {
        mission: 1,
        mission_name: 'Mossbound Ruins',
        room: 0,
        room_name: 'Approach',
        outcome: 'cleared',
        seconds: 20,
        player_hp: 100,
        player_max_hp: 100,
        enemies: 3,
        damage_dealt: 500,
        damage_taken: 50,
        hazard_damage_taken: 0,
        enemy_damage_taken: 50,
        guard_blocked: 20,
        barrier_blocked: 0 // unused mechanic
      };
      window.RiftHUD.updateLastEncounter(run);
      window.RiftHUD.update(run, false, true);
    }, { run });

    const encounterStats = page.locator('#rift-last-encounter-stats');
    // Guarded damage should be visible (20)
    await expect(encounterStats.locator('dt').filter({ hasText: 'Guarded damage' })).toBeVisible();
    // Barrier absorbed must NOT be present when 0
    await expect(encounterStats.locator('dt').filter({ hasText: 'Barrier absorbed' })).toHaveCount(0);

    // Open run statistics
    await page.locator('.rift-run-statistics > summary').click();
    const runStats = page.locator('#rift-statistics');

    // 0869: Unused charge finishers, barrier prevented, armor prevented must NOT be rendered
    await expect(runStats.locator('dt').filter({ hasText: 'Charged finishers' })).toHaveCount(0);
    await expect(runStats.locator('dt').filter({ hasText: 'Barrier prevented' })).toHaveCount(0);
    await expect(runStats.locator('dt').filter({ hasText: 'Armor prevented' })).toHaveCount(0);
    await expect(runStats.locator('dt').filter({ hasText: 'Treasure goblins defeated' })).toHaveCount(0);

    // Unused healing or absorbed per-skill rows for basic skills must NOT be rendered
    await expect(runStats.locator('dt').filter({ hasText: 'Healing · Slash (optional)' })).toHaveCount(0);
    await expect(runStats.locator('dt').filter({ hasText: 'Absorbed · Slash (optional)' })).toHaveCount(0);
  });
});
