const {test,expect}=require('@playwright/test');

test.describe('Proposal 0157: Accessible summary of the last encounter', () => {
  test('initial state before encounter completion presents accessible landmark and empty notice', async ({page}) => {
    await page.goto('/abyss/rift');
    const container = page.locator('#rift-last-encounter');
    await expect(container).toBeVisible();
    await expect(container).toHaveAttribute('aria-labelledby', 'rift-last-encounter-title');
    await expect(page.locator('#rift-last-encounter-title')).toHaveText('Last encounter summary');
    
    const badge = page.locator('#rift-last-encounter-badge');
    await expect(badge).toHaveText('No encounter');
    await expect(badge).toHaveAttribute('data-outcome', 'none');

    await expect(page.locator('#rift-last-encounter-empty')).toBeVisible();
    await expect(page.locator('#rift-last-encounter-empty')).toHaveText('No encounter completed yet this expedition.');
    await expect(page.locator('#rift-last-encounter-details')).toBeHidden();
  });

  test('cleared room presents rich structured encounter summary with polite live announcement', async ({page}) => {
    await page.goto('/abyss/rift?scenario=checkpoint');
    await expect(page.locator('#rift-start')).toBeEnabled();

    const container = page.locator('#rift-last-encounter');
    await expect(container).toBeVisible();
    await expect(page.locator('#rift-last-encounter-empty')).toBeHidden();
    await expect(page.locator('#rift-last-encounter-details')).toBeVisible();

    const badge = page.locator('#rift-last-encounter-badge');
    await expect(badge).toHaveText('Room secured');
    await expect(badge).toHaveAttribute('data-outcome', 'cleared');

    const headline = page.locator('#rift-last-encounter-headline');
    await expect(headline).toHaveAttribute('role', 'status');
    await expect(headline).toHaveAttribute('aria-live', 'polite');
    await expect(headline).toContainText('secured in');
    await expect(headline).toContainText('combat');

    const stats = page.locator('#rift-last-encounter-stats');
    await expect(stats).toBeVisible();
    await expect(stats.locator('dt').filter({hasText: 'Outcome'})).toBeVisible();
    await expect(stats.locator('dd').filter({hasText: 'Room secured'})).toBeVisible();
    await expect(stats.locator('dt').filter({hasText: 'Tier & location'})).toBeVisible();
    await expect(stats.locator('dt').filter({hasText: 'Combat duration'})).toBeVisible();
    await expect(stats.locator('dt').filter({hasText: 'Ending health'})).toBeVisible();
    await expect(stats.locator('dt').filter({hasText: 'Damage dealt'})).toBeVisible();
    await expect(stats.locator('dt').filter({hasText: 'Damage taken'})).toBeVisible();
    await expect(stats.locator('dt').filter({hasText: 'Guarded damage'})).toBeVisible();

    // Verify reload persistence
    const savedHeadline = await headline.textContent();
    await page.reload();
    await expect(page.locator('#rift-last-encounter-headline')).toHaveText(savedHeadline);
  });

  test('defeat state displays lost loot, critical health and focuses encounter on review button click', async ({page}) => {
    await page.route('**/api/abyss/rift', async route => {
      const response = await route.fetch();
      const data = await response.json();
      if (data.run) {
        data.run.status = 'defeated';
        data.run.player.hp = 0;
        data.run.last_encounter = {
          mission: 1,
          mission_name: 'Mossbound Ruins',
          room: 1,
          room_name: 'Inner Court',
          outcome: 'defeated',
          seconds: 9.8,
          player_hp: 0,
          player_max_hp: 200,
          enemies: 4,
          boss_encounter: false,
          damage_dealt: 110,
          damage_taken: 200,
          hits_taken: 5,
          guard_blocked: 30,
          barrier_blocked: 0,
          healing: 0,
          gold_gained: 40,
          loot_items: 0
        };
      }
      await route.fulfill({response, json: data});
    });

    await page.goto('/abyss/rift?scenario=checkpoint');
    const badge = page.locator('#rift-last-encounter-badge');
    await expect(badge).toHaveText('Defeated');
    await expect(badge).toHaveAttribute('data-outcome', 'defeated');

    const headline = page.locator('#rift-last-encounter-headline');
    await expect(headline).toContainText('Defeated in Mossbound Ruins · Tier 2: Inner Court');
    await expect(headline).toContainText('9.8s combat');

    const stats = page.locator('#rift-last-encounter-stats');
    await expect(stats.locator('dt').filter({hasText: 'Unbanked loot lost'})).toBeVisible();
    await expect(stats.locator('dd').filter({hasText: '40 gold'})).toBeVisible();

    // Clicking "Review this fight" opens run stats and focuses #rift-last-encounter
    await page.locator('#rift-defeat-review').click();
    await expect(page.locator('.rift-run-statistics')).toHaveAttribute('open', '');
    await expect(page.locator('#rift-last-encounter')).toBeFocused();
  });

  test('result action button jumps to and focuses last encounter summary', async ({page}) => {
    await page.goto('/abyss/rift?scenario=checkpoint');
    await page.evaluate(() => {
      const actions = document.getElementById('rift-result-actions');
      if (actions) actions.hidden = false;
    });

    const encounterBtn = page.locator('#rift-result-encounter');
    await expect(encounterBtn).toBeVisible();
    await expect(encounterBtn).toHaveText('Encounter summary');
    // Button with visible text must NOT have title attribute
    await expect(encounterBtn).not.toHaveAttribute('title');

    await encounterBtn.click();
    await expect(page.locator('#rift-last-encounter')).toBeFocused();
  });

  test('maintains accessible non-overflowing layout on mobile viewports and forced-colors borders', async ({browser}) => {
    const context = await browser.newContext({
      viewport: {width: 390, height: 844},
      forcedColors: 'active'
    });
    const page = await context.newPage();
    await page.goto('/abyss/rift?scenario=checkpoint');

    const container = page.locator('#rift-last-encounter');
    await expect(container).toBeVisible();

    // Verify no horizontal overflow
    const noOverflow = await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth);
    expect(noOverflow).toBe(true);

    // Verify forced-colors mode border
    const borderStyle = await container.evaluate(el => getComputedStyle(el).borderTopStyle);
    expect(borderStyle).toBe('solid');

    await context.close();
  });
});


test('escaped treasure is a separate persistent encounter outcome',async({page})=>{
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
 const run=(await(await page.request.get('/api/abyss/rift')).json()).run;
 await page.evaluate(run=>{run.last_encounter={mission:1,mission_name:'Test ruins',room:0,room_name:'Gate',outcome:'cleared',seconds:12,player_hp:100,player_max_hp:100,enemies:1,treasure_escaped:2};window.RiftHUD.updateLastEncounter(run);},run);
 const stats=page.locator('#rift-last-encounter-stats');
 await expect(stats.locator('dt').filter({hasText:'Treasure goblins escaped'})).toBeVisible();
 await expect(stats).toContainText('2 · No loot or defeat credit');
 await expect(stats.locator('dt').filter({hasText:'Enemies defeated'}).locator('xpath=following-sibling::dd[1]')).toHaveText('1');
 await page.evaluate(run=>{run.last_encounter={mission:1,mission_name:'Test ruins',room:0,room_name:'Gate',outcome:'cleared',seconds:12,player_hp:100,player_max_hp:100,enemies:1};window.RiftHUD.updateLastEncounter(run);},run);
 await expect(stats.locator('dt').filter({hasText:'Treasure goblins escaped'})).toHaveCount(0);
});


test('defeat recap identifies the fatal boss rather than the room boss',async({page})=>{
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
 const run=(await(await page.request.get('/api/abyss/rift')).json()).run;
 await page.evaluate(run=>{run.last_encounter={mission:1,mission_name:'Ruins',room:0,room_name:'Gate',outcome:'defeated',seconds:12,player_hp:0,player_max_hp:100,enemies:0,boss_name:'Moss King',defeated_by_boss:'Void Queen'};window.RiftHUD.updateLastEncounter(run);},run);
 const stats=page.locator('#rift-last-encounter-stats');
 await expect(stats.locator('dt').filter({hasText:'Defeated by boss'}).locator('xpath=following-sibling::dd[1]')).toHaveText('Void Queen');
 await page.evaluate(run=>{run.last_encounter={mission:1,mission_name:'Ruins',room:0,room_name:'Gate',outcome:'defeated',seconds:12,player_hp:0,player_max_hp:100,enemies:0,boss_name:'Moss King'};window.RiftHUD.updateLastEncounter(run);},run);
 await expect(stats.locator('dt').filter({hasText:'Defeated by boss'})).toHaveCount(0);
});
