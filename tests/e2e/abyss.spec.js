const { test, expect } = require('@playwright/test');

async function fulfillAbyssAPI(page, handler) {
  await page.route('**/api/abyss/**', async route => {
    const request = route.request();
    const body = request.postDataJSON?.() || {};
    const response = handler(new URL(request.url()).pathname, body);
    await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(response) });
  });
}

test('enter sends the selected run setup', async ({ page }) => {
  let enteredBody = null;
  await fulfillAbyssAPI(page, (path, body) => {
    if (path.endsWith('/enter')) {
      enteredBody = body;
      return { ok: true, free_entry: true };
    }
    return { ok: false, error: 'unexpected e2e request' };
  });
  await page.goto('/abyss');
  await page.evaluate(() => { window.reduceMotion = true; });
  await page.locator('[data-entry-step="build"]').click();
  await expect(page.locator('#combatPosition')).toHaveAccessibleDescription('Locked for this run · shown on every live combatant card.');
  await page.locator('#combatPosition').selectOption('backline');
  await expect(page.locator('#abyssEntrySummaryLine')).toContainText('Backline');
  await page.locator('#btnEnter').click();
  await expect.poll(() => enteredBody).not.toBeNull();
  expect(enteredBody.position).toBe('backline');
});

test('companion command is accessible and persists from the Build step', async ({ page }) => {
  let savedBody = null;
  await fulfillAbyssAPI(page, (path, body) => {
    if (path.endsWith('/combat/settings') && Object.keys(body).length === 0) {
      return { ok: true, hold_mana: false, pet_command: 'free' };
    }
    if (path.endsWith('/combat/settings')) {
      savedBody = body;
      return { ok: true, hold_mana: body.hold_mana, pet_command: body.pet_command };
    }
    return { ok: false, error: 'unexpected e2e request' };
  });

  await page.goto('/abyss');
  await page.locator('[data-entry-step="build"]').click();
  const command = page.locator('#petCommandSetting');
  await expect(command).toHaveAccessibleDescription('Free-for-All · companions choose independent targets.');
  await command.selectOption('guard');
  await expect.poll(() => savedBody).toEqual({ hold_mana: false, pet_command: 'guard' });
  await expect(page.locator('#petCommandHint')).toContainText('intercepts 15%');
  await expect(page.locator('#abToastHost')).toContainText('Combat preferences saved.');
});

test('custom stakes preview, submit, and remain adjustable between floors', async ({ page }) => {
  let enteredBody = null;
  let dialBody = null;
  await fulfillAbyssAPI(page, (path, body) => {
    if (path.endsWith('/enter')) {
      enteredBody = body;
      return { ok: true, token_ante: body.token_ante, risk_dial_pct: body.risk_dial_pct, tokens: 32 };
    }
    if (path.endsWith('/risk_dial')) {
      dialBody = body;
      return { ok: true, percent: body.percent, risk: 42, msg: 'Next-floor danger and cache reward adjusted together.' };
    }
    return { ok: false, error: 'unexpected e2e request' };
  });

  await page.setViewportSize({ width: 480, height: 900 });
  await page.goto('/abyss');
  await page.evaluate(() => { window.reduceMotion = true; });
  await page.locator('[data-entry-step="pacts"]').click();
  await page.locator('#tokenAnte').selectOption('10');
  await page.locator('#entryRiskDial').fill('30');
  await expect(page.locator('#abyssEntrySummaryLine')).toContainText('🜲10 ante');
  await expect(page.locator('#abyssEntrySummaryLine')).toContainText('+30% risk');
  expect(await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth)).toBeLessThanOrEqual(1);
  await page.locator('#btnEnter').click();
  await expect(page.locator('#sharedModal')).toHaveClass(/open/);
  await expect(page.locator('#sharedModalCard')).toContainText('Token ante');
  const enteredNavigation = page.waitForEvent('framenavigated', frame => frame === page.mainFrame());
  await page.locator('#modalOkBtn').click();
  await expect.poll(() => enteredBody).not.toBeNull();
  expect(enteredBody.token_ante).toBe(10);
  expect(enteredBody.risk_dial_pct).toBe(30);
  await enteredNavigation;
  await page.waitForLoadState('load');

  await page.goto('/abyss?active=1');
  await page.locator('#runRiskDial').fill('40');
  await expect(page.locator('#runRiskDialOut')).toHaveText('+40%');
  await page.locator('#saveRunRiskDial').click();
  await expect.poll(() => dialBody).toEqual({ percent: 40 });
  await expect(page.locator('#abyssRiskChips')).toContainText('Risk dial +40%');
});

test('HUD normalizes an interest rate from a rolling deployment', async ({ page }) => {
  await page.goto('/abyss?active=1');
  await page.evaluate(() => {
    window.interestRatePct = '0.500';
    window.renderHudChipsNow();
  });
  await expect(page.locator('#hudChips')).toContainText('+0.5%/floor');
});

test('Silent Anvil guides a free action through the dedicated Forge tab', async ({ page }) => {
  await page.setViewportSize({ width: 1280, height: 900 });
  await page.goto('/abyss?active=1&room=forge_floor');
  await page.evaluate(() => { window.reduceMotion = true; });

  const room = page.locator('#nonCombatPanel');
  await expect(room).toBeVisible();
  await expect(room).toContainText('The Silent Anvil');
  await expect(room).toContainText('Free Temper');
  await room.getByRole('button', { name: /Choose Gear · Free Temper/ }).click();

  await expect(page.locator('.ab-tab[data-tab-key="forge"]')).toHaveAttribute('aria-selected', 'true');
  await expect(page.locator('#forgeFloorBanner')).toBeVisible();
  await expect(page.locator('#forgeFloorBanner')).toContainText('socket punch, or full repair is free');
  await expect(page.locator('#btnForgeTemper')).toHaveClass(/ab-forge-floor-free/);
  await expect(page.locator('#btnForgePunchSocket')).toHaveClass(/ab-forge-floor-free/);
  await expect(page.locator('#btnForgeRepairAll')).toHaveClass(/ab-forge-floor-free/);
  await expect(page.locator('#forgeItemSelect')).toBeFocused();
  expect(await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth)).toBeLessThanOrEqual(1);
});

test('Triune Sigil Hunt binds a ten-floor quest and renders its chest completion', async ({ page }) => {
  let acceptedAction = '';
  await fulfillAbyssAPI(page, (path, body) => {
    if (path.endsWith('/noncombat/action')) {
      acceptedAction = body.action;
      return {
        ok: true, resolved: true, msg: 'Triune Hunt bound.',
        event_chain: { active: true, sigils: 0, required: 3, deadline: 22, floors_left: 10, next_depth: 14, chains: 0 },
      };
    }
    return { ok: false, error: 'unexpected e2e request' };
  });
  await page.setViewportSize({ width: 480, height: 900 });
  await page.goto('/abyss?active=1&room=sigil_chain');
  await page.evaluate(() => { window.reduceMotion = true; });

  const room = page.locator('#nonCombatPanel');
  await expect(room).toContainText('The Triune Sigil Hunt');
  await room.getByRole('button', { name: /Bind the 10-floor Hunt/ }).click();
  await expect.poll(() => acceptedAction).toBe('sigil_chain_accept');
  await expect(page.locator('#eventChainRibbon')).toHaveClass(/is-active/);
  await expect(page.locator('#eventChainStatus')).toHaveText('0/3 · next F14 · 10 left');

  await page.evaluate(() => window.renderAbyssEventChain({
    active: false, sigils: 3, required: 3, chains: 1, collected: true, completed: true, chest_reward: 9000,
  }));
  await expect(page.locator('#eventChainRibbon')).toHaveClass(/is-complete/);
  await expect(page.locator('#eventChainRibbon .ab-sigil-marks .is-found')).toHaveCount(3);
  await expect(page.locator('#eventChainStatus')).toContainText('chest opened');
  expect(await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth)).toBeLessThanOrEqual(1);
});

test('Lost Cartographer sells and advances an authoritative five-floor chart', async ({ page }) => {
  let acceptedAction = '';
  const chart = [13, 14, 15, 16, 17].map((depth, index) => ({
    depth,
    type: index === 1 ? 'event' : (index === 3 ? 'rest' : 'combat'),
    label: index === 1 ? 'Investigate a strange presence' : (index === 3 ? 'Rest at a sanctuary' : 'Press onward'),
    icon: index === 1 ? '❔' : (index === 3 ? '🕊️' : '⚔️'),
  }));
  await fulfillAbyssAPI(page, (path, body) => {
    if (path.endsWith('/noncombat/action')) {
      acceptedAction = body.action;
      return {
        ok: true, resolved: true, msg: 'Route purchased.', gold: 121881,
        map_route: { active: true, remaining: 5, floors: chart },
      };
    }
    return { ok: false, error: 'unexpected e2e request' };
  });
  await page.setViewportSize({ width: 480, height: 900 });
  await page.goto('/abyss?active=1&room=lost_cartographer');
  await page.evaluate(() => { window.reduceMotion = true; });

  const room = page.locator('#nonCombatPanel');
  await expect(room).toContainText('The Lost Cartographer');
  await room.getByRole('button', { name: /Buy Five-Floor Chart/ }).click();
  await expect.poll(() => acceptedAction).toBe('cartographer_buy');
  await expect(page.locator('#cartographerRoute')).toBeVisible();
  await expect(page.locator('#cartographerRouteFloors li')).toHaveCount(5);
  await expect(page.locator('#cartographerRouteFloors')).toContainText('F17');

  await page.evaluate(() => window.renderAbyssCartographerRoute({
    active: true, remaining: 4, floors: window.cartographerRouteState.floors.slice(1),
  }));
  await expect(page.locator('#cartographerRouteFloors li')).toHaveCount(4);
  await expect(page.locator('#cartographerRouteFloors')).not.toContainText('F13');
  expect(await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth)).toBeLessThanOrEqual(1);
});

test('known boss preview shows exact elemental matchup and advances with depth', async ({ page }) => {
  await page.setViewportSize({ width: 480, height: 900 });
  await page.emulateMedia({ reducedMotion: 'reduce' });
  await page.goto('/abyss?active=1');

  const preview = page.locator('#abyssElementalPreview');
  await expect(preview).toBeVisible();
  await expect(page.locator('#abElementOutcome')).toContainText('ADVANTAGE');
  await expect(page.locator('.ab-element-rule.is-strong')).toContainText('2×');
  await expect(page.locator('.ab-element-rule.is-resisted')).toContainText('½×');
  await expect(page.locator('#abElementTarget')).toHaveText('F15');

  await page.evaluate(() => window.setDepth(15));
  await expect(page.locator('#abElementTarget')).toHaveText('F20');
  await expect(page.locator('#abAffinityTarget')).toHaveText('F20');
  expect(await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth)).toBeLessThanOrEqual(1);
});

test('next combat signal renders safely and refreshes after a floor', async ({ page }) => {
  await page.setViewportSize({ width: 480, height: 900 });
  const styles = await page.request.get('/static/abyss_enemy_forecast.css');
  expect(styles.status()).toBe(200);
  expect(await styles.text()).toContain('.ab-enemy-forecast');
  await page.goto('/abyss?active=1');

  const forecast = page.locator('#abyssEnemyForecast');
  await expect(forecast).toBeVisible();
  await expect(page.locator('#abEnemyForecastKicker')).toContainText('F13');
  await expect(page.locator('#abEnemyForecastCounter')).not.toBeEmpty();

  await page.evaluate(() => window.renderAbyssEnemyForecast({
    active: true,
    depth: 14,
    key: 'summoner',
    icon: '🔮',
    name: '<img src=x onerror=alert(1)>',
    signal: 'On defeat, calls reinforcements.',
    counter: 'Leave it for last.',
  }));
  await expect(page.locator('#abEnemyForecastKicker')).toHaveText('NEXT COMBAT SIGNAL · F14');
  await expect(page.locator('#abEnemyForecastName')).toHaveText('<img src=x onerror=alert(1)>');
  await expect(forecast.locator('img')).toHaveCount(0);

  await page.evaluate(() => window.renderAbyssEnemyForecast({active: true, concealed: true, depth: 15}));
  await expect(forecast).toHaveClass(/is-concealed/);
  await expect(page.locator('#abEnemyForecastKicker')).toHaveText('SIGNAL JAMMED · F15');
  expect(await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth)).toBeLessThanOrEqual(1);
});

test('shadow scout confirms its token price and renders an isolated 100-fight report', async ({ page }) => {
  let simulationBody = null;
  await fulfillAbyssAPI(page, (path, body) => {
    if (path.endsWith('/simulate')) {
      simulationBody = body;
      return {
        ok: true,
        message: '100 shadow fights complete: 73% observed wins.',
        simulation: {
          depth: 13,
          encounter: '<img src=x onerror=alert(1)> · Ancient Dragon',
          trials: 100,
          wins: 73,
          losses: 27,
          win_pct: 73,
          confidence_low_pct: 64,
          confidence_high_pct: 81,
          median_win_hp_pct: 42,
          cost: 2,
          tokens: 38,
        },
      };
    }
    return { ok: false, error: 'unexpected e2e request' };
  });

  await page.setViewportSize({ width: 480, height: 900 });
  const styles = await page.request.get('/static/abyss_shadow_simulation.css');
  expect(styles.status()).toBe(200);
  expect(await styles.text()).toContain('.ab-shadow-report');
  await page.goto('/abyss?active=1');
  await page.evaluate(() => {
    window.reduceMotion = true;
    window.lastTokens = 40;
    const tokenPill = document.createElement('span');
    tokenPill.id = 'tokenPill';
    tokenPill.textContent = '🜲 40';
    document.body.appendChild(tokenPill);
  });

  await page.locator('#btnSimulate').click();
  await expect(page.locator('#sharedModalCard')).toContainText('Run 100 fights (🜲 2)');
  await page.locator('#modalOkBtn').click();
  await expect.poll(() => simulationBody).toEqual({});

  const report = page.locator('#abyssShadowReport');
  await expect(report).toBeVisible();
  await expect(page.locator('#abyssShadowWinPct')).toHaveText('73%');
  await expect(page.locator('#abyssShadowRecord')).toHaveText('73 W · 27 L');
  await expect(page.locator('#abyssShadowInterval')).toHaveText('64–81%');
  await expect(page.locator('#abyssShadowHP')).toHaveText('42%');
  await expect(page.locator('#abyssShadowEncounter')).toHaveText('<img src=x onerror=alert(1)> · Ancient Dragon');
  await expect(report.locator('img')).toHaveCount(0);
  await expect(page.locator('#tokenPill')).toContainText('38');
  await expect(report).toContainText('no HP, consumables, cooldowns, loot, pity, or run progress changed');
  expect(await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth)).toBeLessThanOrEqual(1);

  await page.evaluate(() => window.setDepth(13));
  await expect(report).toBeHidden();
});

test('live skill variety meter tracks distinct casts and unlocks its XP bonus', async ({ page }) => {
  await page.setViewportSize({ width: 480, height: 900 });
  const styles = await page.request.get('/static/abyss_skill_variety.css');
  expect(styles.status()).toBe(200);
  expect(await styles.text()).toContain('.ab-skill-variety');
  await page.goto('/abyss?active=1');

  await page.evaluate(() => {
    document.getElementById('liveCombat').style.display = 'block';
    window.renderLiveSkillVariety({distinct: 2, target: 3, bonus_pct: 5, unlocked: false});
  });
  const meter = page.locator('#liveSkillVariety');
  await expect(meter).toBeVisible();
  await expect(page.locator('#liveSkillVarietyCount')).toHaveText('2/3');
  await expect(meter).not.toHaveClass(/unlocked/);
  await expect(meter).toHaveAttribute('title', /1 more distinct skill/);

  await page.evaluate(() => window.renderLiveSkillVariety({distinct: 3, target: 3, bonus_pct: 5, unlocked: true}));
  await expect(page.locator('#liveSkillVarietyCount')).toHaveText('3/3');
  await expect(meter).toHaveClass(/unlocked/);
  await expect(meter).toHaveAttribute('title', /\+5% floor XP/);
  expect(await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth)).toBeLessThanOrEqual(1);
});

test('SPD-scaled first strike is prominent and safely rendered in both combat logs', async ({ page }) => {
  await page.setViewportSize({ width: 480, height: 900 });
  const styles = await page.request.get('/static/abyss_first_strike.css');
  expect(styles.status()).toBe(200);
  expect(await styles.text()).toContain('.ab-log-line.ab-log-first-strike');
  await page.goto('/abyss?active=1');

  await page.evaluate(() => {
    const log = document.getElementById('abyssLog');
    window.appendLogLine(log, '⚡ FIRST STRIKE · &lt;img src=x onerror=alert(1)&gt; outruns Rat (SPD 150 vs 100) — opener +10%.', 1);
    const live = document.getElementById('liveCombat');
    live.style.display = 'block';
    window.appendLiveLogLine(document.getElementById('liveFeed'), '⚡ FIRST STRIKE · Runner outruns Rat (SPD 150 vs 100) — opener +10%.');
  });

  const mainLine = page.locator('#abyssLog .ab-log-first-strike').last();
  await expect(mainLine).toContainText('SPD 150 vs 100');
  await expect(mainLine.locator('img')).toHaveCount(0);
  const liveLine = page.locator('#liveFeed .ab-live-first-strike');
  await expect(liveLine).toContainText('opener +10%');
  expect(await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth)).toBeLessThanOrEqual(1);
});

test('gear damage channels explain contribution and support keyboard disclosure', async ({ page }) => {
  await page.setViewportSize({ width: 480, height: 900 });
  const styles = await page.request.get('/static/abyss_gear_damage.css');
  expect(styles.status()).toBe(200);
  expect(await styles.text()).toContain('.ab-gear-damage[hidden]');
  await page.goto('/abyss?active=1&gear=1');

  const mainHand = page.locator('.abyss-side-gear[data-slot="MainHand"]');
  const offHand = page.locator('.abyss-side-gear[data-slot="OffHand"]');
  const mainPanel = page.locator('#abGearDamage-MainHand');
  const offPanel = page.locator('#abGearDamage-OffHand');

  await expect(mainHand).toHaveAttribute('role', 'button');
  await expect(mainHand).toHaveAttribute('aria-expanded', 'false');
  await expect(mainPanel).toHaveCount(1);
  await expect(offPanel).toHaveCount(1);
  await expect(mainPanel).toBeHidden();
  await mainHand.hover();
  await expect(page.locator('.ab-hovertip.show')).toContainText('Damage contribution: +60 STR (60%) · +20 INT (25%) · Fire');
  await expect(page.locator('.ab-hovertip.show')).toContainText('press Enter to expand');

  await mainHand.focus();
  await mainHand.press('Enter');
  await expect(mainHand).toHaveAttribute('aria-expanded', 'true');
  await expect(mainPanel).toBeVisible();
  await expect(mainPanel).toContainText('Base power');
  await expect(mainPanel).toContainText('+60 STR');
  await expect(mainPanel).toContainText('60% of gear STR');
  await expect(mainPanel).toContainText('Spell scaling');
  await expect(mainPanel).toContainText('+20 INT');
  await expect(mainPanel).toContainText('25% of gear INT');
  await expect(mainPanel).toContainText('Fire');
  await expect(mainPanel).toContainText('Final damage also depends on skills');

  await offHand.click();
  await expect(mainHand).toHaveAttribute('aria-expanded', 'false');
  await expect(mainPanel).toBeHidden();
  await expect(offHand).toHaveAttribute('aria-expanded', 'true');
  await expect(offPanel).toBeVisible();
  await offHand.focus();
  await offHand.press(' ');
  await expect(offPanel).toBeHidden();
  expect(await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth)).toBeLessThanOrEqual(1);
});

test('skill priority reorders by drag and keyboard then persists to the server', async ({ page }) => {
  let savedOrder = null;
  await fulfillAbyssAPI(page, (path, body) => {
    if (path.endsWith('/combat/skill_priority')) {
      savedOrder = body.skill_priority;
      return { ok: true, skill_priority: body.skill_priority, msg: 'Automatic skill priority saved.' };
    }
    return { ok: false, error: 'unexpected e2e request' };
  });

  await page.setViewportSize({ width: 480, height: 900 });
  await page.emulateMedia({ reducedMotion: 'reduce' });
  await page.goto('/abyss');
  await page.locator('[data-entry-step="build"]').click();
  const panel = page.locator('#abyssSkillPriority');
  await panel.evaluate(element => { element.open = true; });
  const items = page.locator('#abyssSkillPriorityList > li');
  await expect(items).toHaveCount(3);
  const originalOrder = await items.evaluateAll(nodes => nodes.map(node => node.dataset.skillId));

  await items.nth(2).dragTo(items.nth(0));
  await expect(items.first()).toHaveAttribute('data-skill-id', originalOrder[2]);
  await items.nth(2).focus();
  await page.keyboard.press('Alt+ArrowUp');
  const editedOrder = await items.evaluateAll(nodes => nodes.map(node => node.dataset.skillId));
  expect(editedOrder).not.toEqual(originalOrder);
  await page.locator('#abyssSkillPrioritySave').click();
  await expect.poll(() => savedOrder).toEqual(editedOrder);
  await expect(page.locator('#abyssSkillPriorityReceipt')).toContainText('Server order saved');

  await page.locator('#abyssSkillPriorityReset').click();
  await expect.poll(() => items.evaluateAll(nodes => nodes.map(node => node.dataset.skillId))).toEqual(originalOrder);
  expect(await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth)).toBeLessThanOrEqual(1);
});

test('desktop Abyss keeps its dark canvas and aligned stage in light system mode', async ({ page }) => {
  await page.setViewportSize({ width: 1600, height: 1000 });
  await page.emulateMedia({ colorScheme: 'light', reducedMotion: 'reduce' });
  await page.goto('/abyss?active=1');

  const layout = await page.evaluate(() => {
    const box = selector => document.querySelector(selector).getBoundingClientRect();
    const bodyStyle = getComputedStyle(document.body);
    const panelStyle = getComputedStyle(document.querySelector('.abyss-stage'));
    const objective = box('#abCurrentObjective');
    const elevator = box('.ab-elevator');
    const dial = box('.abyss-dial');
    const controls = box('.abyss-controls');
    return {
      bodyBackground: bodyStyle.backgroundColor,
      panelBackground: panelStyle.backgroundColor,
      objectiveBottom: objective.bottom,
      elevatorTop: elevator.top,
      dialTop: dial.top,
      controlsTop: controls.top,
      controlsWidth: controls.width,
      overflow: Array.from(document.querySelectorAll('body *')).map(element => {
        const rect = element.getBoundingClientRect();
        return {
          element: element.id || element.className || element.tagName,
          right: rect.right,
          visible: getComputedStyle(element).visibility !== 'hidden',
        };
      }).filter(item => item.visible && item.right > window.innerWidth + 1).slice(0, 12),
    };
  });

  expect(layout.bodyBackground).toBe('rgb(8, 11, 17)');
  expect(layout.panelBackground).not.toBe('rgb(255, 253, 248)');
  expect(layout.objectiveBottom).toBeLessThanOrEqual(layout.elevatorTop + 1);
  expect(Math.abs(layout.elevatorTop - layout.dialTop)).toBeLessThan(2);
  expect(Math.abs(layout.dialTop - layout.controlsTop)).toBeLessThan(2);
  expect(layout.controlsWidth).toBeGreaterThan(300);
  expect(layout.overflow).toEqual([]);
});

test('command palette ranks sections, remembers recents, and explains locked actions', async ({ page }) => {
  await page.goto('/abyss?active=1');
  await page.evaluate(() => {
    window.reduceMotion = true;
    localStorage.removeItem('ab_command_recent_v2');
    const locked = document.createElement('button');
    locked.id = 'e2eLockedCommand';
    locked.type = 'button';
    locked.textContent = 'Use locked test action';
    locked.disabled = true;
    locked.dataset.commandDisabledReason = 'Needs an E2E key.';
    document.querySelector('#abyssControls').appendChild(locked);
  });

  await page.keyboard.press('Control+k');
  const search = page.locator('#abCommandSearch');
  await expect(search).toBeVisible();
  await expect(search).toHaveAttribute('role', 'combobox');
  await expect(search).toHaveAttribute('aria-controls', 'abCommandResults');
  await search.fill('frg');
  await expect(page.locator('#abCommandResults [role="option"]').first()).toContainText('Forge');
  await search.press('Enter');
  await expect(page.locator('.ab-tab[data-tab-key="forge"]')).toHaveClass(/active/);

  await page.keyboard.press('Control+k');
  await expect(page.locator('#abCommandResults [role="option"]').first()).toHaveAttribute('data-command-id', 'section:forge');
  await search.fill('no-such-abyss-command-zzzz');
  await expect(page.locator('.ab-command-empty')).toBeVisible();
  await expect.poll(() => search.getAttribute('aria-activedescendant')).toBeNull();
  await search.fill('locked test');
  const lockedResult = page.locator('[data-command-id="control:e2eLockedCommand"]');
  await expect(lockedResult).toHaveAttribute('aria-disabled', 'true');
  await search.press('Enter');
  await expect(page.locator('#sharedModal')).toHaveClass(/open/);
  await expect(page.locator('#abCommandStatus')).toContainText('Needs an E2E key.');

  await page.keyboard.press('Escape');
  await page.locator('body').press('/');
  await expect(page.locator('#abCommandSearch')).toBeVisible();
});

test('season journey has responsive free and token-premium reward lanes', async ({ page }) => {
  let claimedWeek = 0;
  let premiumUnlocked = false;
  let premiumClaimedWeek = 0;
  await fulfillAbyssAPI(page, (path, body) => {
    if (path.endsWith('/season/premium/unlock')) {
      premiumUnlocked = true;
      return { ok: true, unlocked: true, already_unlocked: false, tokens: 2 };
    }
    if (path.endsWith('/season/premium/claim')) {
      premiumClaimedWeek = body.week;
      return { ok: true, week: body.week, name: 'Ember Gilded Scout Sigil', claimed: true, already_owned: false };
    }
    if (path.endsWith('/season/claim')) {
      claimedWeek = body.week;
      return { ok: true, week: body.week, name: 'Ember Scout Sigil', claimed: true, already_owned: false };
    }
    return { ok: false, error: 'unexpected e2e request' };
  });
  await page.setViewportSize({ width: 1280, height: 900 });
  await page.goto('/abyss');
  await page.locator('.ab-tab[data-tab-key="season"]').click();

  await expect(page.locator('#abyssSeasonJourney')).toBeVisible();
  await expect(page.locator('#abyssSeasonTitle')).toHaveText('Ember Descent');
  await expect(page.locator('body')).toHaveAttribute('data-ab-season', 'ember');
  await expect(page.locator('.ab-season-week')).toHaveCount(10);
  await expect(page.locator('[data-season-lane="free"]')).toHaveCount(10);
  await expect(page.locator('[data-season-lane="premium"]')).toHaveCount(10);
  await expect(page.locator('.ab-season-affinity')).toContainText('fire ×3');
  await page.getByRole('button', { name: 'Claim free reward' }).click();
  await expect.poll(() => claimedWeek).toBe(1);
  await expect(page.locator('#abyssSeasonWeek1 [data-season-lane="free"]')).toHaveClass(/claimed/);
  await expect(page.locator('#abyssSeasonWeek1 [data-season-lane="free"] button')).toHaveText('✓ In collection');
  await expect(page.locator('#seasonFreeClaimed')).toHaveText('1');

  await page.getByRole('button', { name: 'Unlock for' }).click();
  await expect.poll(() => premiumUnlocked).toBe(true);
  await expect(page.locator('#abyssSeasonPremiumPass')).toHaveClass(/unlocked/);
  await page.getByRole('button', { name: 'Claim premium reward' }).click();
  await expect.poll(() => premiumClaimedWeek).toBe(1);
  await expect(page.locator('#abyssSeasonWeek1 [data-season-lane="premium"]')).toHaveClass(/claimed/);
  await expect(page.locator('#seasonPremiumClaimed')).toHaveText('1');

  await page.setViewportSize({ width: 480, height: 900 });
  const overflow = await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth);
  expect(overflow).toBeLessThanOrEqual(1);
});

test('dedicated special-item pixel atlases are served', async ({ request }) => {
  const families = [
    'relics', 'ranged', 'artifacts', 'souls', 'auras', 'charms', 'mounts',
    'companions', 'pets', 'emblems', 'banners', 'totems', 'offhands',
  ];
  await Promise.all(families.map(async family => {
    const response = await request.head(`/static/abyss_atlas_${family}.png`);
    expect(response.ok(), `${family} atlas should be served`).toBe(true);
    expect(response.headers()['content-type']).toContain('image/png');
  }));
});

test('operator dashboard charts balance data and applies runtime flags', async ({ page }) => {
  let socialEnabled = true;
  const snapshot = () => ({
    ok: true,
    registry: { active: 3, stale: 0, orphan: 0 },
    latency_ms: { request_avg: 18, request_max: 44 },
    actions: { automatic: 80, manual: 20, automatic_rate: 0.8 },
    anomalies: { total: 1, reward: 1, damage: 0, economy: 0 },
    features: {
      live_actions: true, social: socialEnabled, tree_enhancements: true,
      forge_workbench: true, rollout_percent: 100,
      reward_experiment_enabled: true, reward_experiment_rollout_percent: 50,
      reward_treatment_bonus_bps: 500, revision: 4, reward_experiment_revision: 2,
    },
    reward_experiment: {
      enabled: true, revision: 2, status: 'collecting',
      cohorts: { control: { floors: 8, average_reward: 1200, death_rate: 0.125, anomaly_rate: 0 } },
    },
    balance: {
      available: true, window_days: 30,
      days: [
        { date: '2026-08-23', death_rate: 0.2, drops_per_floor: 0.6 },
        { date: '2026-08-24', death_rate: 0.1, drops_per_floor: 0.8 },
      ],
    },
  });
  await page.route('**/api/abyss/ops', async route => {
    const request = route.request();
    expect(request.headers().authorization).toBe('Bearer e2e-operator');
    if (request.method() === 'POST') {
      const body = request.postDataJSON();
      if (body.feature === 'social') socialEnabled = body.enabled;
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ ok: true, features: snapshot().features, reward_experiment: snapshot().reward_experiment }) });
      return;
    }
    await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(snapshot()) });
  });
  await page.goto('/abyss/ops');
  await page.locator('#opsToken').fill('e2e-operator');
  await page.getByRole('button', { name: 'Connect' }).click();
  await expect(page.locator('#opsActive')).toHaveText('3');
  await expect(page.locator('#opsDeathChart path.line')).toHaveCount(1);
  await expect(page.locator('#opsDropChart circle')).toHaveCount(2);
  await page.locator('[data-ops-feature="social"]').uncheck();
  await expect.poll(() => socialEnabled).toBe(false);
  await expect(page.locator('#opsStatus')).toContainText('updated');
});

test('owned combat replay renders server logs as escaped text', async ({ page }) => {
  await page.route('**/api/abyss/replay/code', async route => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        ok: true,
        replay: {
          version: 1,
          session_id: 'e2e-replay',
          archived_at: '2026-08-25T12:00:00Z',
          random_seed: [31, 41],
          truncated: false,
          total_events: 1,
          frames: [{
            event_id: 7,
            at: '2026-08-25T12:00:01Z',
            round: 3,
            phase: 'complete',
            allies: { alive: 1, units: 1, hp: 80, max_hp: 100 },
            enemies: { alive: 0, units: 1, hp: 0, max_hp: 100 },
            logs: ['<img id="replay-xss" src=x> Victory confirmed.'],
          }],
        },
      }),
    });
  });
  await page.goto('/abyss');
  await page.evaluate(() => sessionStorage.setItem('abyssLastReplaySession', 'e2e-replay'));
  await page.getByRole('button', { name: 'View last replay' }).click();

  await expect(page.locator('#sharedModalCard')).toContainText('<img id="replay-xss" src=x> Victory confirmed.');
  await expect(page.locator('#sharedModalCard #replay-xss')).toHaveCount(0);
  await expect(page.locator('.ab-replay-frame')).toContainText('ROUND 3');
});

test('a victorious descend can preview and commit bank', async ({ page }) => {
  let committed = false;
  await fulfillAbyssAPI(page, (path, body) => {
    if (path.endsWith('/descend')) return {
      ok: true, victory: true, depth: 13, risk: 18, hp: 900, max_hp: 1000,
      gold: 5000, tokens: 12, bonus: 750, escrow: 3750, logs: [], loot: [],
      dura: [], timeline: [], consumables: [], run_floors_cleared: 3,
      overkill_damage: 240, overkill_gold: 24,
    };
    if (path.endsWith('/bank') && body.preview) return {
      ok: true, escrow: 3750, source_escrow: 13750, depth_bonus_pct: 13, depth_bonus: 488,
      streak_bonus_pct: 0, streak_bonus: 0, payout: 4238, tokens_grant: 2,
      base_tokens_grant: 1, pact_tokens_grant: 0, overcap_gold_converted: 100000,
      overcap_tokens_grant: 1, next_bank_streak: 3, free_insurance_earned: true,
      loot_count: 0, capped: false, partial: false,
    };
    if (path.endsWith('/bank')) {
      committed = true;
      return { ok: true, banked: 4238, depth: 13, mult: 1.13, gold: 9238, tokens: 14 };
    }
    return { ok: false, error: 'unexpected e2e request' };
  });
  await page.goto('/abyss?active=1');
  await page.evaluate(() => { window.reduceMotion = true; });
  await page.locator('#btnDescend').click();
  await expect(page.locator('#abStatus')).toContainText('survived');
  await expect(page.locator('.ab-log-overkill-reward')).toContainText('240 excess damage → +24g cache');
  await expect(page.locator('#abyssBanner')).toContainText('Final overkill');
  await page.locator('#btnBank').click();
  await expect(page.locator('#sharedModalCard')).toContainText('Over-cap conversion');
  await expect(page.locator('#sharedModalCard')).toContainText('Next insurance is free');
  await page.locator('#modalOkBtn').click();
  await expect.poll(() => committed).toBe(true);
});

test('bank-streak insurance is presented and consumed as free cover', async ({ page }) => {
  let insuredBody = null;
  await fulfillAbyssAPI(page, (path, body) => {
    if (path.endsWith('/insure')) {
      insuredBody = body;
      return { ok: true, insured: body.pct, cost: 0, gold: 5000, free_insurance_used: true };
    }
    return { ok: false, error: 'unexpected e2e request' };
  });
  await page.setViewportSize({ width: 480, height: 900 });
  await page.goto('/abyss?active=1');
  await page.evaluate(() => {
    window.reduceMotion = true;
    window.__abyssCoreRisk.state.free_insurance_ready = true;
    window.__abyssCoreRisk.render();
  });
  await expect(page.locator('#btnInsureSelected')).toHaveText('Use free cover');
  expect(await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth)).toBeLessThanOrEqual(1);
  await page.locator('#btnInsureSelected').click();
  await expect(page.locator('#sharedModalCard')).toContainText('for free');
  await page.locator('#modalOkBtn').click();
  await expect.poll(() => insuredBody).toEqual({ pct: 25 });
  await expect(page.locator('#btnInsureSelected')).toHaveText('Buy cover');
});

test('a fatal descend exposes the revive and concede decision', async ({ page }) => {
  await fulfillAbyssAPI(page, path => path.endsWith('/descend') ? {
    ok: true, victory: false, depth: 13, risk: 72, hp: 0, max_hp: 1000,
    gold: 5000, tokens: 12, escrow: 3750, logs: ['A fatal test blow lands.'],
    loot: [], dura: [], timeline: [], can_revive: true, can_last_stand: true,
    revive_chance_pct: 48, revive_streak: 0, last_stand_cost: 8,
  } : { ok: false, error: 'unexpected e2e request' });
  await page.goto('/abyss?active=1');
  await page.evaluate(() => { window.reduceMotion = true; });
  await page.locator('#btnDescend').click();
  await expect(page.locator('#abStatus')).toContainText('Choose');
  await expect(page.locator('#btnRevive')).toBeVisible();
  await expect(page.locator('#btnConcede')).toBeVisible();
});

test('planned multi-floor descent presents every combat floor in order', async ({ page }) => {
  await fulfillAbyssAPI(page, path => path.endsWith('/descend_multi') ? {
    ok: true, victory: true, depth: 15, risk: 20, hp: 760, max_hp: 1000,
    gold: 5000, tokens: 12, bonus: 900, escrow: 13650, logs: [], loot: [],
    dura: [], timeline: [], consumables: [], run_floors_cleared: 5,
    floor_results: [13, 14, 15].map((depth, index) => ({
      depth, victory: true, hp: 920 - index * 80, max_hp: 1000,
      logs: [`Floor ${depth} test combat`], loot: [], dura: [], timeline: [],
      overkill_damage: 100 + index * 10, overkill_gold: 10 + index,
      event_chain: index < 2
        ? { active: true, sigils: index + 1, required: 3, floors_left: 8 - index, next_depth: depth + 1, collected: true }
        : { active: false, sigils: 3, required: 3, chains: 1, collected: true, completed: true, chest_reward: 9000 },
      map_route: {
        active: true, remaining: 4 - index,
        floors: Array.from({ length: 4 - index }, (_, offset) => ({
          depth: depth + 1 + offset, type: 'combat', label: 'Press onward', icon: '⚔️',
        })),
      },
    })),
    event_chain: { active: false, sigils: 3, required: 3, chains: 1, collected: true, completed: true, chest_reward: 9000 },
    map_route: { active: true, remaining: 2, floors: [
      { depth: 16, type: 'combat', label: 'Press onward', icon: '⚔️' },
      { depth: 17, type: 'combat', label: 'Press onward', icon: '⚔️' },
    ] },
  } : { ok: false, error: 'unexpected e2e request' });
  await page.goto('/abyss?active=1');
  await page.evaluate(() => {
    window.reduceMotion = true;
    window.__batchFloors = [];
    window.__batchOverkillReceipts = [];
    document.addEventListener('abyss:batch-floor', event => {
      window.__batchFloors.push(event.detail.depth);
      const receipt = document.querySelector('.ab-log-overkill-reward');
      if (receipt) window.__batchOverkillReceipts.push(receipt.textContent);
    });
  });
  await page.locator('#btnDescendMulti').click();
  await expect.poll(() => page.evaluate(() => window.__batchFloors)).toEqual([13, 14, 15]);
  await expect.poll(() => page.evaluate(() => window.__batchOverkillReceipts)).toEqual([
    '💰 FINAL OVERKILL · 100 excess damage → +10g cache',
    '💰 FINAL OVERKILL · 110 excess damage → +11g cache',
  ]);
  await expect(page.locator('.ab-log-overkill-reward')).toContainText('120 excess damage → +12g cache');
  await expect(page.locator('#abStatus')).toContainText('survived');
  await expect(page.locator('#eventChainRibbon')).toHaveClass(/is-complete/);
  await expect(page.locator('#eventChainRibbon .ab-sigil-marks .is-found')).toHaveCount(3);
  await expect(page.locator('#cartographerRouteFloors li')).toHaveCount(2);
  await expect(page.locator('#cartographerRouteFloors')).toContainText('F17');
});

test('auto-descend submits safeguards and stops after settled floor playback', async ({ page }) => {
  let submittedRules = null;
  await fulfillAbyssAPI(page, (path, body) => {
    if (!path.endsWith('/descend_multi')) return { ok: false, error: 'unexpected e2e request' };
    submittedRules = body.stop_rules;
    return {
      ok: true, victory: true, auto_stopped: true, stop_reason: 'legendary',
      depth: 14, risk: 24, hp: 720, max_hp: 1000, gold: 5000, tokens: 12,
      bonus: 800, escrow: 12800, logs: [], loot: [], dura: [], timeline: [],
      consumables: [], run_floors_cleared: 4,
      floor_results: [13, 14].map((depth, index) => ({
        depth, victory: true, hp: 860 - index * 140, max_hp: 1000,
        legendary_drop: index === 1,
        logs: [`Floor ${depth} safety test`], loot: [], dura: [], timeline: [],
      })),
    };
  });
  await page.setViewportSize({ width: 480, height: 900 });
  await page.goto('/abyss?active=1');
  await page.evaluate(() => { window.reduceMotion = true; window.__batchFloors = []; document.addEventListener('abyss:batch-floor', event => window.__batchFloors.push(event.detail.depth)); });
  await expect(page.locator('#autoDescendRules')).toBeVisible();
  await page.locator('#autoStopDepth').fill('14');
  await page.locator('#btnDescendMulti').click();
  await expect.poll(() => submittedRules).toEqual({ hp_below_pct: 50, target_depth: 14, stop_on_legendary: true });
  await expect.poll(() => page.evaluate(() => window.__batchFloors)).toEqual([13, 14]);
  await expect(page.locator('#abStatus')).toContainText('Legendary+ drop secured');
  await expect(page.locator('#abyssBanner')).toContainText('AUTO-DESCEND STOPPED');
  expect(await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth)).toBeLessThanOrEqual(1);
  const css = await page.request.get('/static/abyss_auto_descend.css');
  expect(css.status()).toBe(200);
});

test('mob affixes explain mechanics safely in tactical and pixel combat views', async ({ page }) => {
  await page.setViewportSize({ width: 480, height: 900 });
  await page.goto('/abyss?active=1');
  await page.evaluate(() => {
    window.reduceMotion = true;
    document.getElementById('liveCombat').style.display = 'block';
    document.querySelector('.ab-live-details').open = true;
    const enemy = {
      id: 'enemy:0', name: '<img src=x onerror=alert(1)>', hp: 700, max_hp: 1000,
      role: 'elite', element: 'Fire', weak_to: 'Water', speed: 80,
      effects: [
        { name: 'Armored', key: 'armored', icon: '🛡️', description: 'Carries reinforced defenses.', tone: 'defense', affix: true, duration: 'encounter' },
        { name: 'Fleet-foot', key: 'fleet', icon: '💨', description: 'Carries heightened Speed.', tone: 'speed', affix: true, duration: 'encounter' },
        { name: 'Vampiric', key: 'vampiric', icon: '🩸', description: '<b>Heals 15%</b>', tone: 'sustain', affix: true, duration: 'encounter' },
      ],
    };
    renderLiveTargets('liveEnemies', [enemy], true);
    resetLivePixelState('affix-test');
    renderLivePixelStage({ session_id: 'affix-test', version: 1, recent_logs: [], allies: [], enemies: [enemy] });
  });
  await expect(page.locator('#liveEnemies .ab-mob-affix')).toHaveCount(3);
  await expect(page.locator('#liveEnemies .ab-mob-affix').first()).toContainText('Armored');
  await expect(page.locator('#livePixelEnemies .ab-mob-affix')).toHaveCount(3);
  await expect(page.locator('#liveEnemies button')).toHaveAttribute('aria-label', /Affixes: Armored.*Vampiric/);
  await expect(page.locator('#liveEnemies .ab-mob-affix').last()).toHaveAttribute('title', 'Vampiric — <b>Heals 15%</b>');
  await expect(page.locator('#liveEnemies img')).toHaveCount(0);
  expect(await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth)).toBeLessThanOrEqual(1);
  const css = await page.request.get('/static/abyss_mob_affixes.css');
  expect(css.status()).toBe(200);
});

test('critical fumble drama stays escaped, legible, and explicitly cosmetic', async ({ page }) => {
  await page.setViewportSize({ width: 360, height: 780 });
  await page.emulateMedia({ reducedMotion: 'reduce' });
  await page.goto('/abyss?active=1');
  await page.evaluate(() => {
    const feed = document.getElementById('liveFeed');
    feed.innerHTML = '';
    appendLiveLogLine(
      feed,
      '🎭 CRITICAL FUMBLE · <img src=x onerror=alert(1)> trips, recovers, and hits Warden anyway. (No combat effect)',
    );
  });
  const line = page.locator('#liveFeed .ab-live-critical-fumble');
  await expect(line).toHaveCount(1);
  await expect(line).toContainText('<img src=x onerror=alert(1)>');
  await expect(line).toContainText('No combat effect');
  await expect(page.locator('#liveFeed img')).toHaveCount(0);
  expect(await line.evaluate((node) => getComputedStyle(node).animationName)).toBe('none');
  expect(await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth)).toBeLessThanOrEqual(1);
  const css = await page.request.get('/static/abyss_critical_fumble.css');
  expect(css.status()).toBe(200);
  expect(await css.text()).toContain('DAMAGE INTACT');
});

test('crowded live combat can target an ordinary enemy', async ({ page }) => {
  let submittedTarget = '';
  await fulfillAbyssAPI(page, (path, body) => {
    if (path.endsWith('/combat/action')) {
      submittedTarget = body.target_id;
      return { ok: false, error: 'e2e receipt complete' };
    }
    return { ok: false, error: 'unexpected e2e request' };
  });
  const shieldStyles = await page.request.get('/static/abyss_shield.css');
  expect(shieldStyles.status()).toBe(200);
  expect(await shieldStyles.text()).toContain('.ab-overhead-shield');
  const weaknessStyles = await page.request.get('/static/abyss_weakness_window.css');
  expect(weaknessStyles.status()).toBe(200);
  const weaknessCSS = await weaknessStyles.text();
  expect(weaknessCSS).toContain('.ab-pixel-weakness');
  expect(weaknessCSS).toContain('prefers-reduced-motion');
  await page.goto('/abyss?active=1');
  await page.evaluate(() => {
    window.reduceMotion = true;
    const enemies = Array.from({ length: 7 }, (_, index) => ({
      id: `enemy:${index}`, name: `Raider ${index}`,
      hp: index === 4 ? 31 : index === 5 ? 29 : index === 6 ? 10 : 100,
      max_hp: 100, hp_hidden: index === 6,
      role: 'common', speed: 10 + index, effects: [],
    }));
    enemies[5].weakness_ready = true;
    enemies[5].effects = [{ name: 'Weakness Window', duration: 'Next player hit · guaranteed critical' }];
    window.renderLiveCombat({
      ok: true, session_id: 'e2e-live', phase: 'planning', round: 1,
      deadline: new Date(Date.now() + 60000).toISOString(), tactic: 'balanced',
      policy: {}, allies: [{ id: 'ally:e2e', name: 'Tester', hp: 900, max_hp: 1000, shield: 150, max_shield: 200, mana: 100, max_mana: 100, is_self: true, is_player: true }],
      enemies, options: [
        { kind: 'attack', id: '', name: 'Basic Attack', target: 'enemy', cooldown: 0 },
        { kind: 'skill', id: 'S_E2E_A', name: 'Fiery Blast', target: 'enemy', mana: 5, cooldown: 0 },
        { kind: 'skill', id: 'S_E2E_B', name: 'Fiery Blast', target: 'enemy', mana: 5, cooldown: 0 },
      ],
      recent_logs: [], initiative: [], enemy_intents: [], social: {},
    });
    document.querySelector('#lootManifest').innerHTML = '<div class="abyss-side-loot" data-loot-id="42" data-gear-id="ABYSS_TEST" data-slot="MainHand" data-tip="Test Blade"><span class="ab-loot-main"><span>Test Blade</span></span></div><div class="abyss-side-loot" data-loot-id="43" data-gear-id="ABYSS_RELIC" data-slot="Relic" data-tip="Test Relic"><span class="ab-loot-main"><span>Test Relic</span></span></div>';
    window.updateLootRewardPresentation();
    const petCard = document.createElement('div');
    petCard.className = 'ab-pet-card';
    petCard.dataset.petArtKey = 'pet:e2e:frost-lich';
    petCard.innerHTML = '<span class="ab-pixel-icon ab-pet-pixel"></span>';
    document.querySelector('.abyss-command-page').appendChild(petCard);
    window.decorateAbyssPetCards();
  });
  const ordinary = page.locator('#livePixelEnemies [data-target="enemy:5"]');
  await expect(ordinary).toBeVisible();
  const tacticalShield = page.locator('#liveAllies [data-target="ally:e2e"] .ab-combatant-shield');
  const pixelAlly = page.locator('#livePixelAllies [data-target="ally:e2e"]');
  await expect(tacticalShield).toContainText('150 / 200');
  await expect(tacticalShield).toHaveClass(/active/);
  await expect(pixelAlly).toHaveClass(/shielded/);
  await expect(pixelAlly.locator('.ab-overhead-shield')).toContainText('150 / 200');
  await expect(pixelAlly).toHaveAttribute('aria-label', /150 of 200 shield/);
  const closedThreshold = page.locator('#liveEnemies [data-target="enemy:4"]');
  const openThreshold = page.locator('#liveEnemies [data-target="enemy:5"]');
  const concealedThreshold = page.locator('#liveEnemies [data-target="enemy:6"]');
  await expect(closedThreshold).not.toHaveClass(/execute-ready/);
  await expect(closedThreshold.locator('.execute-track')).toHaveCount(1);
  await expect(openThreshold).toHaveClass(/execute-ready/);
  await expect(openThreshold).toHaveClass(/weakness-ready/);
  await expect(openThreshold.locator('.ab-combatant-weakness')).toContainText('NEXT PLAYER HIT CRITS');
  await expect(openThreshold).toHaveAttribute('aria-label', /next direct player hit is a guaranteed critical/);
  await expect(openThreshold).toContainText('EXECUTE');
  await expect(concealedThreshold).not.toHaveClass(/execute-ready/);
  await expect(concealedThreshold).toContainText('HP CONCEALED');
  await expect(concealedThreshold.locator('.execute-track')).toHaveCount(0);
  const enemySide = await page.locator('#livePixelEnemies').boundingBox();
  const allySide = await page.locator('#livePixelAllies').boundingBox();
  expect(enemySide.x).toBeLessThan(allySide.x);
  await expect(ordinary.locator('.ab-catalog-actor')).toHaveCSS('background-image', /abyss_atlas_creatures/);
  await expect(ordinary).toHaveClass(/weakness-ready/);
  await expect(ordinary).toHaveClass(/weakness-open/);
  await expect(ordinary.locator('.ab-pixel-weakness')).toContainText('EXPOSED');
  await expect(ordinary).toHaveAttribute('aria-label', /next direct player hit is a guaranteed critical/);
  await expect(ordinary.locator('.ab-catalog-actor')).toHaveAttribute('data-art-sheet', 'creatures');
  const monsterSignatures = await page.locator('#livePixelEnemies .ab-actor-sigil').evaluateAll(nodes => nodes.map(node => node.dataset.artSignature));
  expect(new Set(monsterSignatures).size).toBe(7);
  await ordinary.click();
  await expect(page.locator('#liveQueue')).toContainText('TARGET · Raider 5');
  const attackIcon = page.locator('#liveActionBar .kind-attack .ab-catalog-icon');
  await expect(attackIcon).toHaveCSS('background-image', /abyss_atlas_skills/);
  await expect(attackIcon).toHaveAttribute('data-art-sheet', 'skills');
  const skillSignatures = await page.locator('#liveActionBar .kind-skill .ab-art-unique').evaluateAll(nodes => nodes.map(node => node.dataset.artSignature));
  expect(new Set(skillSignatures).size).toBe(2);
  const skillComposites = await page.locator('#liveActionBar .kind-skill .ab-art-unique').evaluateAll(nodes => nodes.map(node => node.getAttribute('style')));
  expect(new Set(skillComposites).size).toBe(2);
  const skillMotif = await page.locator('#liveActionBar .kind-skill .ab-art-unique').first().evaluate(node => getComputedStyle(node, '::before').backgroundImage);
  expect(skillMotif).toContain('abyss_atlas_skills');
  const lootIcons = page.locator('#lootManifest .ab-loot-pixel.ab-art-unique');
  await expect(lootIcons).toHaveCount(2);
  await expect(lootIcons.nth(0)).toHaveCSS('background-image', /abyss_atlas_items/);
  await expect(lootIcons.nth(1)).toHaveCSS('background-image', /abyss_atlas_relics/);
  await expect(lootIcons.nth(1)).toHaveCSS('background-size', '1300% 1200%');
  await expect(lootIcons.nth(1)).toHaveAttribute('data-art-sheet', 'relics');
  const petIcon = page.locator('.ab-pet-card[data-pet-art-key="pet:e2e:frost-lich"] .ab-pet-pixel');
  await expect(petIcon).toHaveCSS('background-image', /abyss_atlas_pets/);
  await expect(petIcon).toHaveCSS('width', '48px');
  await expect(petIcon).toHaveCSS('height', '48px');
  await expect(petIcon).toHaveAttribute('data-art-sheet', 'pets');
  const specialFamilies = await page.evaluate(() => ({
    relic: abGearArtFamily('Relic'), ranged: abGearArtFamily('Ranged'), artifact: abGearArtFamily('Artifact'),
    soul: abGearArtFamily('Soul'), aura: abGearArtFamily('Aura'), charm: abGearArtFamily('Charm'),
    mount: abGearArtFamily('Mount'), companion: abGearArtFamily('Companion'), pet1: abGearArtFamily('Pet1'),
    pet2: abGearArtFamily('Pet2'), emblem1: abGearArtFamily('Emblem1'), emblem2: abGearArtFamily('Emblem2'),
    banner: abGearArtFamily('Banner'), totem: abGearArtFamily('Totem'), offhand: abGearArtFamily('OffHand'),
  }));
  expect(specialFamilies).toEqual({
    relic: 'relics', ranged: 'ranged', artifact: 'artifacts', soul: 'souls', aura: 'auras',
    charm: 'charms', mount: 'mounts', companion: 'companions', pet1: 'pets', pet2: 'pets',
    emblem1: 'emblems', emblem2: 'emblems', banner: 'banners', totem: 'totems', offhand: 'offhands',
  });
  await page.locator('#liveActionBar .kind-attack').click();
  await expect.poll(() => submittedTarget).toBe('enemy:5');
  await page.evaluate(() => {
    const next = structuredClone(window.liveCombatState);
    next.version = Number(next.version || 0) + 1;
    delete next.allies[0].shield;
    delete next.enemies[5].weakness_ready;
    next.enemies[5].effects = [];
    next.recent_logs = [
      '💥 WEAKNESS CRITICAL! Tester exploits Raider 5\'s opening for 2× damage.',
      '🛡️ AEGIS · Tester absorbs 150 damage — barrier broken.',
    ];
    window.renderLiveCombat(next);
  });
  await expect(tacticalShield).toHaveClass(/broken/);
  await expect(tacticalShield).toContainText('BROKEN');
  await expect(pixelAlly).toHaveClass(/shield-break/);
  await expect(pixelAlly.locator('.ab-pixel-float.shield')).toContainText('🛡 -150');
  await expect(openThreshold).not.toHaveClass(/weakness-ready/);
  await expect(openThreshold.locator('.ab-combatant-weakness')).toHaveCount(0);
  await expect(ordinary).not.toHaveClass(/weakness-ready/);
  await expect(ordinary.locator('.ab-pixel-weakness')).toHaveCount(0);
});

test('live combat exposes and enforces the remaining action-change budget', async ({ page }) => {
  await page.goto('/abyss?active=1');
  await page.evaluate(() => {
    window.reduceMotion = true;
    window.renderLiveCombat({
      ok: true, session_id: 'e2e-budget', phase: 'planning', round: 4,
      deadline: new Date(Date.now() + 60000).toISOString(), tactic: 'balanced',
      policy: {}, action_budget: { limit: 64, remaining: 0 },
      queued: { kind: 'attack', ability_id: '', target_id: 'enemy:0', round: 4 },
      allies: [{ id: 'ally:e2e', name: 'Tester', hp: 900, max_hp: 1000, is_self: true, is_player: true }],
      enemies: [{ id: 'enemy:0', name: 'Rate Warden', hp: 100, max_hp: 100 }],
      options: [{ kind: 'attack', id: '', name: 'Basic Attack', target: 'enemy', cooldown: 0 }],
      recent_logs: [], initiative: [], enemy_intents: [], social: {},
    });
  });

  await expect(page.locator('#liveActionBudget')).toHaveText('0 / 64 CHANGES');
  await expect(page.locator('#liveActionBudget')).toHaveClass(/exhausted/);
  await expect(page.locator('#liveActionBar .kind-attack')).toBeDisabled();
  await expect(page.locator('#liveQueue')).toContainText('QUEUED · ATTACK');
  await expect(page.locator('#liveActionBudget')).toHaveAttribute('title', /queued action or timeout fallback/);
});
