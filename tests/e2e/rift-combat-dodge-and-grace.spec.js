const { test, expect } = require('@playwright/test');

test.describe('Rift Combat Dodge, Grace, and Combat Breakdown (Batch 546)', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/abyss/rift');
    await expect(page.locator('#rift-start')).toBeEnabled();
  });

  test('actionbar includes directional dodge button without title tooltip and with kbd shortcut', async ({ page }) => {
    const dodgeBtn = page.locator('.rift-basics button[data-bind="dodge"]');
    await expect(dodgeBtn).toBeVisible();

    // Check visible action label
    const label = dodgeBtn.locator('.rift-action-label');
    await expect(label).toBeVisible();
    await expect(label).toHaveText('Dodge');

    // Button with visible text must NOT have title attribute
    await expect(dodgeBtn).not.toHaveAttribute('title');

    // Accessible keyshortcut & aria-label
    await expect(dodgeBtn).toHaveAttribute('aria-keyshortcuts', 'C');
    await expect(dodgeBtn).toHaveAttribute('aria-label', 'Dodge');

    // Kbd element aria-hidden
    const kbd = dodgeBtn.locator('kbd');
    await expect(kbd).toBeVisible();
    await expect(kbd).toHaveText('C');
    await expect(kbd).toHaveAttribute('aria-hidden', 'true');

    // Dodge cooldown / ready indicator
    const dodgeReady = page.locator('#rift-dodge-ready');
    await expect(dodgeReady).toBeVisible();
    await expect(dodgeReady).toHaveText('Dodge ready');
  });

  test('controls configuration lists directional dodge with key C', async ({ page }) => {
    await page.locator('#rift-controls-open').click();
    const dialog = page.locator('#rift-controls-dialog');
    await expect(dialog).toBeVisible();

    const dodgeRemap = dialog.locator('#rift-remap-dodge');
    await expect(dodgeRemap).toBeVisible();
    await expect(dodgeRemap).toHaveText('C');

    const refText = dialog.locator('#rift-controls-reference-text');
    await expect(refText).toContainText('Directional dodge:  C');

    await page.locator('#rift-controls-close').click();
    await expect(dialog).toBeHidden();
  });

  test('encounter statistics and result summary render armor-piercing damage, attack chain, combo score, and replay seed', async ({ page }) => {
    const summary = await page.evaluate(() => {
      const mockRun = {
        id: 'seed-test-run-456',
        replay_seed: 9876543210123,
        status: 'defeated',
        room: 0,
        level: {
          id: 1,
          name: 'The Abyssal Gates',
          rooms: [{ name: 'Chamber 1' }]
        },
        player: { hp: 0, max_hp: 100, facing: 1 },
        enemies: [],
        skill_timers: {},
        build: { skills: [], signatures: [], class: 'guardian' },
        stats: {
          seconds: 42.5,
          damage_dealt: 1250,
          damage_taken: 200,
          hits_taken: 5,
          healing: 50,
          guard_blocked: 80,
          armor_piercing_damage: 420.5,
          highest_attack_chain: 14,
          combo_score: 850
        }
      };

      // Call RiftHUD to update DOM
      window.RiftHUD.update({
        ...mockRun,
        attack_chain: 14,
        first_hit_grace: false
      });

      // Call buildResultSummary to verify export
      return window.RiftHUD.buildResultSummary(mockRun);
    });

    // Verify stats in DOM (#rift-statistics dl)
    const statsDl = page.locator('#rift-statistics');
    await expect(statsDl).toContainText('Armor-piercing damage');
    await expect(statsDl).toContainText('421');
    await expect(statsDl).toContainText('Longest uninterrupted attack chain');
    await expect(statsDl).toContainText('14');
    await expect(statsDl).toContainText('Combo score');
    await expect(statsDl).toContainText('850');
    await expect(statsDl).toContainText('Deterministic combat replay seed');
    await expect(statsDl).toContainText('9876543210123');

    // Verify result summary export text
    expect(summary).toContain('Armor-Piercing Damage: 421');
    expect(summary).toContain('Uninterrupted Attack Chain: 14');
    expect(summary).toContain('Combo Score: 850');
    expect(summary).toContain('Replay Seed: 9876543210123');
  });
});
