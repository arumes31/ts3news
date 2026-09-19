const { test, expect } = require('@playwright/test');

test.describe('Optional directional captions for off-screen threats (Proposal 0145)', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/abyss/rift');
    await expect(page.locator('#rift-start')).toBeEnabled();
  });

  test('setting is present in Sound & display settings and persists across reload', async ({ page }) => {
    const settings = page.locator('.rift-settings');
    await settings.locator('> summary').click();

    const dirCheck = page.locator('#rift-directional-threat-captions');
    await expect(dirCheck).toBeAttached();
    expect(await dirCheck.isChecked()).toBe(false);

    const dirLabel = page.locator('label[for="rift-directional-threat-captions"]');
    await expect(dirLabel).toHaveText('Directional captions for off-screen threats');

    // Toggle on
    await dirCheck.check();
    expect(await dirCheck.isChecked()).toBe(true);

    // Reload and check persistence
    await page.reload();
    await expect(page.locator('#rift-start')).toBeEnabled();
    await settings.locator('> summary').click();
    expect(await page.locator('#rift-directional-threat-captions').isChecked()).toBe(true);
  });

  test('directional captions append arrows and positions for off-screen events', async ({ page }) => {
    const settings = page.locator('.rift-settings');
    await settings.locator('> summary').click();
    await page.locator('#rift-combat-captions').check();
    await page.locator('#rift-directional-threat-captions').check();

    // Start expedition and pause to set up simulation
    await page.locator('#rift-start').click();
    await page.keyboard.press('Escape');

    // Dispatch simulated events: one off-screen left, one off-screen right, one on-screen
    // Player is at x: 350, so cam is 0, screen spans [0, 960].
    // Left off-screen is x < 0; right off-screen is x > 960.
    await page.evaluate(() => {
      const run = {
        id: 'test-run',
        level: { id: 1 },
        room: 0,
        status: 'fighting',
        paused: false,
        counter: 10,
        clock: 5,
        player: { x: 350, hp: 100, max_hp: 100, mana: 100, cooldown: 0, knockdown: 0 },
        build: { skills: [], signatures: [] },
        skill_timers: {},
        enemies: [],
        events: []
      };
      window.RiftFeedback.update(run, true, true);
      run.events = [
        { id: 11, kind: 'boss_roar', x: -100, y: 300, value: 0 },
        { id: 12, kind: 'slam', x: 1200, y: 300, value: 0 }
      ];
      window.RiftFeedback.update(run, false, true);
    });

    const captionItems = page.locator('#rift-captions li');
    await expect(captionItems).toHaveCount(2);
    expect(await captionItems.nth(0).textContent()).toBe('← Boss roar (off-screen left)');
    expect(await captionItems.nth(1).textContent()).toBe('Ground slam (off-screen right) →');

    // Check screen reader live region
    const liveText = await page.locator('#rift-captions [role="status"]').textContent();
    expect(liveText).toContain('← Boss roar (off-screen left)');
    expect(liveText).toContain('Ground slam (off-screen right) →');
  });

  test('directional captions announce off-screen living threats', async ({ page }) => {
    const settings = page.locator('.rift-settings');
    await settings.locator('> summary').click();
    await page.locator('#rift-directional-threat-captions').check();

    await page.locator('#rift-start').click();
    await page.keyboard.press('Escape');

    // Simulate run with enemies off-screen left and right
    await page.evaluate(() => {
      const run = {
        id: 'test-threats',
        level: { id: 1 },
        room: 0,
        status: 'fighting',
        paused: false,
        counter: 1,
        clock: 2,
        player: { x: 350, hp: 100, max_hp: 100, mana: 100, cooldown: 0, knockdown: 0 },
        build: { skills: [], signatures: [] },
        skill_timers: {},
        enemies: [],
        events: []
      };
      window.RiftFeedback.update(run, true, true);
      run.enemies = [
        { id: 1, hp: 50, x: -50, y: 200, kind: 'goblin' },
        { id: 2, hp: 50, x: -80, y: 220, kind: 'goblin' },
        { id: 3, hp: 500, x: 1300, y: 300, kind: 'boss' }
      ];
      window.RiftFeedback.update(run, false, true);
    });

    const captionItems = page.locator('#rift-captions li');
    await expect(captionItems).toHaveCount(2);
    expect(await captionItems.nth(0).textContent()).toBe('← 2 threats (off-screen left)');
    expect(await captionItems.nth(1).textContent()).toBe('Boss threat (off-screen right) →');
  });

  test('without directional captions enabled, sound cues remain non-directional', async ({ page }) => {
    const settings = page.locator('.rift-settings');
    await settings.locator('> summary').click();
    await page.locator('#rift-combat-captions').check();
    await page.locator('#rift-directional-threat-captions').uncheck();

    await page.locator('#rift-start').click();
    await page.keyboard.press('Escape');

    await page.evaluate(() => {
      const run = {
        id: 'test-run-nodir',
        level: { id: 1 },
        room: 0,
        status: 'fighting',
        paused: false,
        counter: 20,
        clock: 5,
        player: { x: 350, hp: 100, max_hp: 100, mana: 100, cooldown: 0, knockdown: 0 },
        build: { skills: [], signatures: [] },
        skill_timers: {},
        enemies: [{ id: 1, hp: 50, x: -100, y: 200, kind: 'goblin' }],
        events: []
      };
      window.RiftFeedback.update(run, true, true);
      run.events = [{ id: 21, kind: 'boss_roar', x: -100, y: 300, value: 0 }];
      window.RiftFeedback.update(run, false, true);
    });

    const captionItems = page.locator('#rift-captions li');
    await expect(captionItems).toHaveCount(1);
    expect(await captionItems.first().textContent()).toBe('Boss roar');
  });

  test('master reset restores directional captions setting to unchecked default', async ({ page }) => {
    const settings = page.locator('.rift-settings');
    await settings.locator('> summary').click();
    const dirCheck = page.locator('#rift-directional-threat-captions');
    await dirCheck.check();
    expect(await dirCheck.isChecked()).toBe(true);

    const masterReset = page.locator('#rift-reset-preferences');
    await masterReset.click();
    expect(await dirCheck.isChecked()).toBe(false);
  });
});
