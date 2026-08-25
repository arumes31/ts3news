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
  let entered = false;
  await fulfillAbyssAPI(page, path => {
    if (path.endsWith('/enter')) {
      entered = true;
      return { ok: true, free_entry: true };
    }
    return { ok: false, error: 'unexpected e2e request' };
  });
  await page.goto('/abyss');
  await page.evaluate(() => { window.reduceMotion = true; });
  await page.locator('#btnEnter').click();
  await expect.poll(() => entered).toBe(true);
});

test('HUD normalizes an interest rate from a rolling deployment', async ({ page }) => {
  await page.goto('/abyss?active=1');
  await page.evaluate(() => {
    window.interestRatePct = '0.500';
    window.renderHudChipsNow();
  });
  await expect(page.locator('#hudChips')).toContainText('+0.5%/floor');
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

test('a victorious descend can preview and commit bank', async ({ page }) => {
  let committed = false;
  await fulfillAbyssAPI(page, (path, body) => {
    if (path.endsWith('/descend')) return {
      ok: true, victory: true, depth: 13, risk: 18, hp: 900, max_hp: 1000,
      gold: 5000, tokens: 12, bonus: 750, escrow: 3750, logs: [], loot: [],
      dura: [], timeline: [], consumables: [], run_floors_cleared: 3,
    };
    if (path.endsWith('/bank') && body.preview) return {
      ok: true, escrow: 3750, depth_bonus_pct: 13, depth_bonus: 488,
      streak_bonus_pct: 0, streak_bonus: 0, payout: 4238, tokens_grant: 2,
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
  await page.locator('#btnBank').click();
  await page.locator('#modalOkBtn').click();
  await expect.poll(() => committed).toBe(true);
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
    gold: 5000, tokens: 12, bonus: 900, escrow: 4650, logs: [], loot: [],
    dura: [], timeline: [], consumables: [], run_floors_cleared: 5,
    floor_results: [13, 14, 15].map((depth, index) => ({
      depth, victory: true, hp: 920 - index * 80, max_hp: 1000,
      logs: [`Floor ${depth} test combat`], loot: [], dura: [], timeline: [],
    })),
  } : { ok: false, error: 'unexpected e2e request' });
  await page.goto('/abyss?active=1');
  await page.evaluate(() => {
    window.reduceMotion = true;
    window.__batchFloors = [];
    document.addEventListener('abyss:batch-floor', event => window.__batchFloors.push(event.detail.depth));
  });
  await page.locator('#btnDescendMulti').click();
  await expect.poll(() => page.evaluate(() => window.__batchFloors)).toEqual([13, 14, 15]);
  await expect(page.locator('#abStatus')).toContainText('survived');
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
  await page.goto('/abyss?active=1');
  await page.evaluate(() => {
    window.reduceMotion = true;
    const enemies = Array.from({ length: 7 }, (_, index) => ({
      id: `enemy:${index}`, name: `Raider ${index}`, hp: 100, max_hp: 100,
      role: 'common', speed: 10 + index, effects: [],
    }));
    window.renderLiveCombat({
      ok: true, session_id: 'e2e-live', phase: 'planning', round: 1,
      deadline: new Date(Date.now() + 60000).toISOString(), tactic: 'balanced',
      policy: {}, allies: [{ id: 'ally:e2e', name: 'Tester', hp: 900, max_hp: 1000, mana: 100, max_mana: 100, is_self: true, is_player: true }],
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
  const enemySide = await page.locator('#livePixelEnemies').boundingBox();
  const allySide = await page.locator('#livePixelAllies').boundingBox();
  expect(enemySide.x).toBeLessThan(allySide.x);
  await expect(ordinary.locator('.ab-catalog-actor')).toHaveCSS('background-image', /abyss_atlas_creatures/);
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
});
