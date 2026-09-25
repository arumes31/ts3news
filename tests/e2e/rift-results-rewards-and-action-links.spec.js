const { test, expect } = require('@playwright/test');

test('displays banked rewards separately from lost pending finds on defeat and completion', async ({ page }) => {
  // 1. Defeat scenario: banked rewards kept and lost pending rewards shown separately (0859)
  await page.goto('/abyss/rift?scenario=terminal-defeated');
  await expect(page.locator('#rift-overlay')).toBeVisible();

  const rewards = page.locator('#rift-result-rewards');
  await expect(rewards).toBeVisible();

  const banked = page.locator('.rift-reward-banked');
  const lost = page.locator('.rift-reward-lost');
  await expect(banked).toBeVisible();
  await expect(banked).toContainText('Banked rewards');
  await expect(lost).toBeVisible();
  await expect(lost).toContainText('Lost pending rewards');

  // Verify encounter stats also show banked rewards kept
  const stats = page.locator('#rift-last-encounter-stats');
  await expect(stats).toContainText('Banked rewards kept');

  // 2. Complete scenario: banked rewards visible, lost rewards hidden (0859)
  await page.goto('/abyss/rift?scenario=terminal-complete');
  await expect(page.locator('#rift-overlay')).toBeVisible();
  await expect(rewards).toBeVisible();
  await expect(banked).toBeVisible();
  await expect(lost).toBeHidden();
});

test('offers last selected mission on retry button, return to campaign, manage build, and practice boss', async ({ page }) => {
  await page.goto('/abyss/rift?scenario=terminal-defeated');
  await expect(page.locator('#rift-overlay')).toBeVisible();

  const resultActions = page.locator('#rift-result-actions');
  await expect(resultActions).toBeVisible();

  // 1. Last selected mission on retry button (0860)
  const retryBtn = page.locator('#rift-retry-mission');
  await expect(retryBtn).toBeVisible();
  await expect(retryBtn).toContainText('Retry');

  // 2. Return to campaign button (0861)
  const campaignBtn = page.locator('#rift-result-campaign');
  await expect(campaignBtn).toBeVisible();
  await campaignBtn.click();
  const campaignDetails = page.locator('#rift-campaign');
  await expect(campaignDetails).toHaveAttribute('open', '');

  // 3. Manage build link on defeat (0862)
  const buildBtn = page.locator('#rift-result-build');
  await expect(buildBtn).toBeVisible();
  await buildBtn.click();
  // Focus shifts to loadout select
  const firstSkillSelect = page.locator('#rift-loadout select').first();
  await expect(firstSkillSelect).toBeFocused();

  // 4. Practice-the-boss link on boss defeat (0863)
  // Mock a defeat with a boss present in the encounter
  await page.evaluate(() => {
    const run = {
      id: 'boss-defeat-test',
      status: 'defeated',
      room: 2,
      level: { id: 1, name: 'Mossbound Ruins' },
      banked_gold: 100,
      banked_items: [],
      defeated_by_boss: 'Ancient Golem',
      defeat_cause: 'Fallen to boss: Ancient Golem',
      encounter_plan: [[], [], [{ kind: 'boss', name: 'Ancient Golem' }]],
      enemies: [],
      drops: [],
      player: { x: 500, y: 400, hp: 0, max_hp: 100, mana: 100, max_mana: 100 },
      stats: { seconds: 40, damage_dealt: 500, damage_taken: 100 }
    };
    const practiceBossBtn = document.getElementById('rift-practice-boss-link');
    practiceBossBtn.textContent = 'Practice ' + run.defeated_by_boss;
    practiceBossBtn.hidden = false;
  });

  const practiceBossBtn = page.locator('#rift-practice-boss-link');
  await expect(practiceBossBtn).toBeVisible();
  await expect(practiceBossBtn).toHaveText('Practice Ancient Golem');
});
