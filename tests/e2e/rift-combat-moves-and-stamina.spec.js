const { test, expect } = require('@playwright/test');

test.describe('Rift Combat Moves, Guard Stamina, and Juggle Limits (Batch 547)', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/abyss/rift');
    await expect(page.locator('#rift-start')).toBeEnabled();
  });

  test('displays guard stamina meter and reacts to guard-break recovery state', async ({ page }) => {
    const staminaNode = page.locator('#rift-guard-stamina');
    await expect(staminaNode).toBeVisible();
    await expect(staminaNode).toHaveText('Stamina: 100%');

    const guardNode = page.locator('#rift-guard-reduction');
    await expect(guardNode).toBeVisible();
    await expect(guardNode).toHaveText('Guard inactive');

    const guardBtn = page.locator('.rift-basics button[data-bind="guard"]');
    await expect(guardBtn).toBeVisible();
    await expect(guardBtn).not.toHaveAttribute('aria-disabled');

    // Simulate active guard with stamina
    await page.evaluate(() => {
      window.RiftHUD.update({
        id: 'stamina-test',
        status: 'fighting',
        room: 0,
        player: { hp: 100, max_hp: 100, facing: 1, guard: true, guard_stamina: 85 },
        enemies: [],
        skill_timers: {},
        build: { skills: [], signatures: [], class: 'guardian' },
        stats: {}
      });
    });

    await expect(staminaNode).toContainText('Stamina: 85%');
    await expect(guardNode).toContainText('Stamina 85%');

    // Simulate guard-broken state
    await page.evaluate(() => {
      window.RiftHUD.update({
        id: 'stamina-test',
        status: 'fighting',
        room: 0,
        player: { hp: 100, max_hp: 100, facing: 1, guard: false, guard_stamina: 0 },
        enemies: [],
        skill_timers: { guard_break_recovery: 1.2 },
        build: { skills: [], signatures: [], class: 'guardian' },
        stats: { guard_breaks: 1 }
      });
    });

    // Verify broken state presentation
    await expect(staminaNode).toContainText('Stamina: Broken (1.2s)');
    await expect(guardNode).toContainText('Guard broken (1.2s)');
    await expect(guardBtn).toHaveAttribute('aria-disabled', 'true');
    await expect(guardBtn).toHaveAttribute('data-broken', 'true');
  });

  test('encounter statistics and result summary export track combat move varieties', async ({ page }) => {
    const summary = await page.evaluate(() => {
      const mockRun = {
        id: 'moves-test-run-123',
        replay_seed: 5432109876543,
        status: 'defeated',
        room: 0,
        level: {
          id: 1,
          name: 'The Abyssal Gates',
          rooms: [{ name: 'Chamber 1' }]
        },
        player: { hp: 0, max_hp: 100, facing: 1, guard_stamina: 80 },
        enemies: [],
        skill_timers: {},
        build: { skills: [], signatures: [], class: 'berserker' },
        stats: {
          seconds: 35.0,
          damage_dealt: 2500,
          damage_taken: 150,
          hits_taken: 4,
          healing: 0,
          guard_blocked: 50,
          guard_breaks: 2,
          heavy_attacks: 8,
          aerial_attacks: 5,
          sweep_attacks: 4,
          launchers: 6,
          downed_followups: 3
        }
      };

      window.RiftHUD.update(mockRun);
      return window.RiftHUD.buildResultSummary(mockRun);
    });

    // Check DOM stats
    const statsDl = page.locator('#rift-statistics');
    await expect(statsDl).toContainText('Guard breaks suffered');
    await expect(statsDl).toContainText('2');
    await expect(statsDl).toContainText('Heavy basic attacks');
    await expect(statsDl).toContainText('8');
    await expect(statsDl).toContainText('Aerial basic attacks');
    await expect(statsDl).toContainText('5');
    await expect(statsDl).toContainText('Grounded sweep attacks');
    await expect(statsDl).toContainText('4');
    await expect(statsDl).toContainText('Launchers against small enemies');
    await expect(statsDl).toContainText('6');
    await expect(statsDl).toContainText('Downed enemy follow-ups');
    await expect(statsDl).toContainText('3');

    // Check result summary export text
    expect(summary).toContain('Guard Breaks: 2');
    expect(summary).toContain('Heavy Basic Attacks: 8');
    expect(summary).toContain('Aerial Basic Attacks: 5');
    expect(summary).toContain('Grounded Sweeps: 4');
    expect(summary).toContain('Launchers: 6');
    expect(summary).toContain('Downed Enemy Follow-ups: 3');
  });
});
