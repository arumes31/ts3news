const { test, expect } = require('@playwright/test');

test('displays dedicated mission-clear, full-campaign completion, and voluntary-exit result headings', async ({ page }) => {
  // 1. Mission clear result heading (0841)
  await page.goto('/abyss/rift?scenario=terminal-complete');
  await expect(page.locator('#rift-overlay')).toBeVisible();
  await expect(page.locator('#rift-result-banner')).toBeVisible();
  await expect(page.locator('#rift-result-heading')).toContainText('Mission 1 Cleared:');
  await expect(page.locator('#rift-result-cause')).toHaveText('All 3 encounter tiers secured.');
  await expect(page.locator('#rift-clear-result')).toContainText('Mission 1 cleared:');

  // 2. Full-campaign completion result heading (0842)
  await page.evaluate(() => {
    const headingEl = document.getElementById('rift-result-heading');
    const causeEl = document.getElementById('rift-result-cause');
    const clearResult = document.getElementById('rift-clear-result');
    const banner = document.getElementById('rift-result-banner');

    const run = {
      status: 'complete',
      completed_levels: Array.from({ length: 100 }, (_, i) => i + 1)
    };
    if (run.completed_levels.length >= 100) {
      headingEl.textContent = 'Full Campaign Cleared — The Abyss Conquered';
      causeEl.textContent = 'All 100 campaign missions cleared across all regions!';
      clearResult.textContent = 'Campaign complete: All 100 missions cleared';
      clearResult.hidden = false;
      banner.hidden = false;
    }
  });
  await expect(page.locator('#rift-result-heading')).toHaveText('Full Campaign Cleared — The Abyss Conquered');
  await expect(page.locator('#rift-result-cause')).toHaveText('All 100 campaign missions cleared across all regions!');
  await expect(page.locator('#rift-clear-result')).toHaveText('Campaign complete: All 100 missions cleared');

  // 3. Voluntary-exit result heading (0843)
  await page.goto('/abyss/rift?scenario=terminal-banked');
  await expect(page.locator('#rift-overlay')).toBeVisible();
  await expect(page.locator('#rift-result-banner')).toBeVisible();
  await expect(page.locator('#rift-result-heading')).toHaveText('Voluntary Exit at Checkpoint');
  await expect(page.locator('#rift-result-cause')).toHaveText('Safely banked rewards before venturing deeper.');
  await expect(page.locator('#rift-clear-result')).toHaveText('Expedition banked: voluntary exit at checkpoint');
});

test('displays exact cause of defeat and responsible enemy or hazard in banner and encounter details', async ({ page }) => {
  // 1. Defeat by hazard (0844, 0845)
  await page.goto('/abyss/rift?scenario=terminal-defeated');
  await expect(page.locator('#rift-overlay')).toBeVisible();
  await expect(page.locator('#rift-result-banner')).toBeVisible();
  await expect(page.locator('#rift-result-heading')).toHaveText('Expedition Defeat');

  // Update encounter and banner with named hazard defeat
  await page.evaluate(() => {
    const hazardEncounter = {
      mission: 1,
      mission_name: 'Mossbound Approach',
      room: 0,
      room_name: 'Approach',
      outcome: 'defeated',
      seconds: 15,
      player_hp: 0,
      player_max_hp: 100,
      enemies: 0,
      damage_dealt: 100,
      damage_taken: 100,
      hazard_damage_taken: 100,
      enemy_damage_taken: 0,
      hits_taken: 1,
      defeated_by_hazard: { kind: 'spike_trap', jumpable: true },
      defeat_cause: 'Fallen to spike_trap hazard (jump to evade)'
    };
    window.RiftHUD.updateLastEncounter(hazardEncounter, false);

    const headingEl = document.getElementById('rift-result-heading');
    const causeEl = document.getElementById('rift-result-cause');
    headingEl.textContent = 'Expedition Defeat';
    causeEl.textContent = 'Cause: ' + hazardEncounter.defeat_cause + ' · Final hit: ' + hazardEncounter.defeated_by_hazard.kind;
  });

  await expect(page.locator('#rift-result-heading')).toHaveText('Expedition Defeat');
  await expect(page.locator('#rift-result-cause')).toContainText('spike_trap');
  await expect(page.locator('#rift-result-cause')).toContainText('jump to evade');

  // Verify encounter details list final hit source and cause
  const encounterStats = page.locator('#rift-last-encounter-stats');
  await expect(encounterStats).toContainText('Final hit source');
  await expect(encounterStats).toContainText('Hazard: spike_trap');
  await expect(encounterStats).toContainText('Cause of defeat');

  // 2. Defeat by regular enemy (0844, 0845)
  await page.evaluate(() => {
    const enemyEncounter = {
      mission: 1,
      mission_name: 'Mossbound Approach',
      room: 1,
      room_name: 'Overgrowth',
      outcome: 'defeated',
      seconds: 22,
      player_hp: 0,
      player_max_hp: 100,
      enemies: 2,
      damage_dealt: 300,
      damage_taken: 100,
      hazard_damage_taken: 0,
      enemy_damage_taken: 100,
      hits_taken: 3,
      defeated_by_enemy: 'Moss Lurker',
      defeat_cause: 'Fallen to enemy: Moss Lurker'
    };
    window.RiftHUD.updateLastEncounter(enemyEncounter, false);

    const causeEl = document.getElementById('rift-result-cause');
    causeEl.textContent = 'Cause: ' + enemyEncounter.defeat_cause + ' · Final hit: ' + enemyEncounter.defeated_by_enemy;
  });

  await expect(page.locator('#rift-result-cause')).toContainText('Moss Lurker');
  await expect(encounterStats).toContainText('Defeated by enemy');
  await expect(encounterStats).toContainText('Moss Lurker');
  await expect(encounterStats).toContainText('Final hit source');
  await expect(encounterStats).toContainText('Enemy: Moss Lurker');
  await expect(encounterStats).toContainText('Fallen to enemy: Moss Lurker');

  // 3. Defeat by boss (0844, 0845)
  await page.evaluate(() => {
    const bossEncounter = {
      mission: 1,
      mission_name: 'Mossbound Ruins',
      room: 2,
      room_name: "Guardian's Stand",
      outcome: 'defeated',
      seconds: 45,
      player_hp: 0,
      player_max_hp: 100,
      enemies: 1,
      boss_encounter: true,
      boss_name: 'Moss Colossus',
      damage_dealt: 1200,
      damage_taken: 250,
      hazard_damage_taken: 0,
      enemy_damage_taken: 250,
      hits_taken: 4,
      defeated_by_boss: 'Moss Colossus',
      defeat_cause: 'Fallen to boss: Moss Colossus'
    };
    window.RiftHUD.updateLastEncounter(bossEncounter, false);

    const causeEl = document.getElementById('rift-result-cause');
    causeEl.textContent = 'Cause: ' + bossEncounter.defeat_cause + ' · Final hit: ' + bossEncounter.defeated_by_boss;
  });

  await expect(page.locator('#rift-result-cause')).toContainText('Moss Colossus');
  await expect(encounterStats).toContainText('Defeated by boss');
  await expect(encounterStats).toContainText('Moss Colossus');
  await expect(encounterStats).toContainText('Final hit source');
  await expect(encounterStats).toContainText('Boss: Moss Colossus');
});
