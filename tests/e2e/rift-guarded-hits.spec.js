const { test, expect } = require('@playwright/test');

test.describe('Make guarded hits distinguishable without audio (Proposal 0150)', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/abyss/rift');
    await expect(page.locator('#rift-start')).toBeEnabled();
  });

  test('floating combat text distinguishes guarded hits with a shield prefix and distinct color', async ({ page }) => {
    const properties = await page.evaluate(() => {
      const renderer = window.RiftRenderer;
      return {
        guardedDamage: renderer.combatText({ kind: 'block', value: 3.2 }),
        guardedFullBlock: renderer.combatText({ kind: 'block', value: 0 }),
        unguardedHurt: renderer.combatText({ kind: 'hurt', value: 25 }),
        enemyDamage: renderer.combatText({ kind: 'hit', value: 40 }),
        heal: renderer.combatText({ kind: 'heal', value: 15 }),
        barrier: renderer.combatText({ kind: 'barrier', value: 20 })
      };
    });

    // Guarded hit with remaining damage has distinct cyan/shield color and shield + label
    expect(properties.guardedDamage.color).toBe('#7fd7ff');
    expect(properties.guardedDamage.label).toBe('🛡️ Guarded -3');

    // Zero-damage full guard also produces visible label with shield
    expect(properties.guardedFullBlock.color).toBe('#7fd7ff');
    expect(properties.guardedFullBlock.label).toBe('🛡️ Guarded');

    // Unguarded hurt is salmon/reddish and has no shield symbol
    expect(properties.unguardedHurt.color).toBe('#ffb2a0');
    expect(properties.unguardedHurt.label).toBe('25');
    expect(properties.unguardedHurt.label).not.toContain('🛡️');

    // Colors must all be distinct
    const colors = new Set([
      properties.guardedDamage.color,
      properties.unguardedHurt.color,
      properties.enemyDamage.color,
      properties.heal.color,
      properties.barrier.color
    ]);
    expect(colors.size).toBe(5);
  });

  test('sound cue captions provide accessible text and screen-reader announcements for guarded hits while muted', async ({ page }) => {
    // Open settings and enable combat captions
    await page.locator('.rift-settings > summary').click();
    const captionCheckbox = page.locator('#rift-combat-captions');
    await captionCheckbox.check();

    // Mute audio completely to verify non-audio accessibility
    await page.evaluate(() => {
      window.RiftAudio.set('muted', true);
    });

    // Start expedition and pause to inject combat simulation
    await page.locator('#rift-start').click();
    await page.keyboard.press('Escape');

    // Trigger a block event in RiftFeedback
    await page.evaluate(() => {
      const run = {
        id: 'test-guard-run',
        level: { id: 1 },
        room: 0,
        status: 'fighting',
        paused: false,
        counter: 5,
        clock: 2,
        player: { x: 350, hp: 100, max_hp: 100, mana: 100, cooldown: 0, knockdown: 0 },
        build: { skills: [], signatures: [] },
        skill_timers: {},
        enemies: [],
        events: []
      };
      window.RiftFeedback.update(run, true, true);
      run.counter++;
      run.events = [{ id: run.counter, kind: 'block', x: 350, y: 380, value: 4 }];
      window.RiftFeedback.update(run, false, true);
    });

    // Captions list should display 'Attack guarded'
    const captions = page.locator('#rift-captions');
    await expect(captions).toBeVisible();
    await expect(captions.locator('li')).toContainText(['Attack guarded']);

    // Live status region should announce 'Attack guarded.' for screen readers
    const liveRegion = captions.locator('[role="status"]');
    await expect(liveRegion).toContainText('Attack guarded.');
  });

  test('HUD guard reduction indicator and button reflect recent guarded hits with visual confirmation', async ({ page }) => {
    await page.locator('#rift-start').click();
    await page.keyboard.press('Escape');

    const indicator = page.locator('#rift-guard-reduction');
    const guardBtn = page.locator('.rift-basics button[data-bind="guard"]');

    // Initially inactive, no guarded hit attribute
    await expect(indicator).not.toHaveAttribute('data-guarded-hit');

    // Simulate HUD update with a block event
    await page.evaluate(() => {
      const run = {
        id: 'test-hud-run',
        room: 0,
        status: 'fighting',
        paused: false,
        counter: 10,
        player: { x: 350, y: 380, facing: 1, guard: true, hp: 100, max_hp: 100 },
        build: { skills: [], signatures: [] },
        skill_timers: {},
        enemies: [],
        stats: { guards: 1 },
        events: [{ id: 10, kind: 'block', x: 350, y: 380, value: 2 }]
      };
      window.RiftHUD.update(run, true, false);
    });

    // Indicator and button should have data-guarded-hit
    await expect(indicator).toHaveAttribute('data-guarded-hit');
    await expect(guardBtn).toHaveAttribute('data-guarded-hit');

    // When the event is aged out, data-guarded-hit is removed
    await page.evaluate(() => {
      const run = {
        id: 'test-hud-run',
        room: 0,
        status: 'fighting',
        paused: false,
        counter: 20,
        player: { x: 350, y: 380, facing: 1, guard: true, hp: 100, max_hp: 100 },
        build: { skills: [], signatures: [] },
        skill_timers: {},
        enemies: [],
        stats: { guards: 1 },
        events: [{ id: 10, kind: 'block', x: 350, y: 380, value: 2 }]
      };
      window.RiftHUD.update(run, true, false);
    });

    await expect(indicator).not.toHaveAttribute('data-guarded-hit');
    await expect(guardBtn).not.toHaveAttribute('data-guarded-hit');
  });

  test('guard practice drill visually confirms successful guards without audio', async ({ page }) => {
    // Enable captions and mute audio
    await page.locator('.rift-settings > summary').click();
    await page.locator('#rift-combat-captions').check();
    await page.evaluate(() => {
      window.RiftAudio.set('muted', true);
    });

    await page.goto('/abyss/rift?practice=guard');
    await expect(page.locator('#rift-start')).toBeEnabled();
    await page.locator('#rift-start').click();
    await expect(page.locator('#rift-overlay')).toBeHidden();

    // Turn toward the training attacker (press KeyD) and hold guard (KeyL)
    await page.keyboard.press('KeyD');
    await page.keyboard.down('KeyL');

    // Wait for at least one guard to succeed
    const saved = async () => (await (await page.request.get('/api/abyss/rift?practice=guard')).json()).run;
    await expect.poll(async () => (await saved()).stats.guards, { timeout: 15000 }).toBeGreaterThan(0);

    // Caption should confirm the guarded attack even while audio is muted
    await expect(page.locator('#rift-captions')).toBeVisible();
    await expect(page.locator('#rift-captions')).toContainText('Attack guarded');

    await page.keyboard.up('KeyL');
  });
});
