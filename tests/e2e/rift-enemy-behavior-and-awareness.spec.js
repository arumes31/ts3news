const { test, expect } = require('@playwright/test');

test.describe('Rift Enemy Behavior, Flanking, and Awareness (Batch 548)', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/abyss/rift?scenario=checkpoint');
    await expect(page.locator('#rift-start')).toBeEnabled();
  });

  test('encounter table and result summary track coordinated pack attacks, summon punishes, flanking, and rear strikes', async ({ page }) => {
    const summary = await page.evaluate(() => {
      const mockRun = {
        id: 'enemy-behavior-test-run',
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
        build: { skills: [], signatures: [], class: 'delver' },
        stats: {
          seconds: 22.4,
          damage_dealt: 650,
          damage_taken: 100,
          hits_taken: 4,
          pack_attacks: 3,
          summon_punishes: 2,
          flank_attempts: 5,
          rear_strikes: 4
        }
      };

      return window.RiftHUD.buildResultSummary(mockRun);
    });

    expect(summary).toContain('Coordinated Pack Attacks: 3');
    expect(summary).toContain('Summon Arrival Punishes: 2');
    expect(summary).toContain('Enemy Flank Maneuvers: 5');
    expect(summary).toContain('Unshielded Rear Strikes: 4');

    // Test encounter table rendering
    await page.evaluate(() => {
      window.RiftHUD.updateLastEncounter({
        room_name: 'Chamber 1',
        outcome: 'cleared',
        seconds: 15.2,
        damage_dealt: 450,
        damage_taken: 20,
        hits_taken: 1,
        pack_attacks: 2,
        summon_punishes: 1,
        flank_attempts: 3,
        rear_strikes: 2
      });
    });

    const encounterStats = page.locator('#rift-last-encounter-stats');
    await expect(encounterStats).toBeVisible();
    await expect(encounterStats).toContainText('Coordinated pack attacks');
    await expect(encounterStats).toContainText('Summon arrival punishes');
    await expect(encounterStats).toContainText('Enemy flank maneuvers');
    await expect(encounterStats).toContainText('Unshielded rear strikes');
  });

  test('actors protocol accepts optional patrol, awareness, and behavior fields', async ({ page }) => {
    const isValid = await page.evaluate(async () => {
      const data = await (await fetch('/api/abyss/rift')).json();
      data.run.enemies.push({
        id: 'patrol-mob',
        name: 'Sentry',
        kind: 'goblin',
        x: 500,
        y: 410,
        hp: 60,
        max_hp: 60,
        facing: -1,
        patrol: true,
        patrol_origin_x: 500,
        patrol_dir: -1,
        alerted: false,
        awareness_timer: 0.4,
        pack: true
      });
      data.run.enemies.push({
        id: 'summon-mob',
        name: 'Summoned Imp',
        kind: 'goblin',
        x: 400,
        y: 410,
        hp: 40,
        max_hp: 40,
        facing: -1,
        summoned: true,
        arrival_vulnerability: 0.85
      });

      try {
        window.RiftProtocol.validate(data, 'GET');
        return true;
      } catch (err) {
        return false;
      }
    });

    expect(isValid).toBe(true);
  });
});
