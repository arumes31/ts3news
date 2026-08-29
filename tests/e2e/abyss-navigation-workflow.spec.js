const { test, expect } = require('@playwright/test');

test('floor planner caps at 20 while its add control stays pinned', async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await page.emulateMedia({ reducedMotion: 'reduce' });
  await page.goto('/abyss?active=1');

  await page.evaluate(() => {
    window.curDepth = 7;
    window.busy = false;
    buildPathQueuePlanner();
  });
  const add = page.locator('#btnQueueMore');
  const initialX = await add.evaluate(node => node.getBoundingClientRect().x);

  await page.evaluate(() => {
    for (let index = 0; index < 30; index += 1) addPathToQueue();
  });

  await expect(page.locator('#pathQueueContainer .ab-path-select')).toHaveCount(20);
  await expect(add).toBeDisabled();
  await expect(add).toHaveCSS('position', 'sticky');
  expect(await add.evaluate(node => node.getBoundingClientRect().x)).toBe(initialX);
  expect(await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth)).toBeLessThanOrEqual(1);
});

test('successful shop and forge actions return focus without changing scroll position', async ({ page }) => {
  await page.emulateMedia({ reducedMotion: 'reduce' });
  await page.goto('/abyss?active=1');

  const result = await page.evaluate(async () => {
    window.inRun = true;
    window.reduceMotion = true;
    window.abRawPost = async () => ({ ok: true });
    const cockpit = document.getElementById('abyssCombatCockpit');
    const descend = document.getElementById('btnDescend');
    descend.style.display = '';
    window.scrollTo(0, 320);
    const scrollBefore = window.scrollY;
    let scrollCalls = 0;
    cockpit.scrollIntoView = () => { scrollCalls += 1; };

    const run = async section => {
      const button = document.createElement('button');
      document.querySelector(`[data-abyss-section="${section}"]`).appendChild(button);
      window.lastActionBtn = button;
      await abPost('/api/abyss/e2e-focus', {});
      await new Promise(resolve => setTimeout(resolve, 240));
      window.lastActionBtn = null;
      button.remove();
      return document.activeElement === descend;
    };

    return {
      shopFocused: await run('shop'),
      forgeFocused: await run('forge'),
      scrollCalls,
      scrollBefore,
      scrollAfter: window.scrollY,
    };
  });

  expect(result.shopFocused).toBe(true);
  expect(result.forgeFocused).toBe(true);
  expect(result.scrollCalls).toBe(0);
  expect(result.scrollAfter).toBe(result.scrollBefore);
});

test('active combat cockpit keeps stage, actions, and log inside one desktop viewport', async ({ page }) => {
  await page.setViewportSize({ width: 1440, height: 1000 });
  await page.emulateMedia({ reducedMotion: 'reduce' });
  await page.goto('/abyss?active=1');
  await page.evaluate(() => focusAbyssDescend());

  const initial = await page.evaluate(() => {
    const box = id => document.getElementById(id).getBoundingClientRect();
    return { cockpit: box('abyssCombatCockpit'), stage: box('abyssStage'), descend: box('btnDescend'), log: box('logWrap'), viewport: innerHeight };
  });
  expect(initial.cockpit.height).toBeLessThanOrEqual(initial.viewport - 73);
  expect(initial.cockpit.top).toBeGreaterThanOrEqual(0);
  expect(initial.cockpit.bottom).toBeLessThanOrEqual(initial.viewport + 1);
  expect(initial.stage.bottom).toBeLessThanOrEqual(initial.viewport + 1);
  expect(initial.descend.bottom).toBeLessThanOrEqual(initial.viewport + 1);
  expect(initial.log.bottom).toBeLessThanOrEqual(initial.viewport + 1);

  await page.evaluate(() => {
    document.getElementById('liveCombat').style.display = 'block';
    updateAbyssCombatCockpit(true);
  });
  const live = await page.evaluate(() => {
    const box = id => document.getElementById(id).getBoundingClientRect();
    return { stage: box('abyssStage'), combat: box('liveCombat'), log: box('logWrap'), viewport: innerHeight };
  });
  expect(live.stage.bottom).toBeLessThanOrEqual(live.viewport + 1);
  expect(live.combat.bottom).toBeLessThanOrEqual(live.viewport + 1);
  expect(live.log.bottom).toBeLessThanOrEqual(live.viewport + 1);
});

test('entry reload marker returns keyboard focus to Descend', async ({ page }) => {
  await page.emulateMedia({ reducedMotion: 'reduce' });
  await page.addInitScript(() => sessionStorage.setItem('ab_focus_cockpit', '1'));
  await page.goto('/abyss?active=1');
  await expect(page.locator('#btnDescend')).toBeFocused();
  expect(await page.evaluate(() => window.scrollY)).toBe(0);
});

test('wide cockpit keeps the Armoury fixed in the left viewport gutter', async ({ page }) => {
  await page.setViewportSize({ width: 1900, height: 1100 });
  await page.goto('/abyss?active=1');

  const layout = await page.evaluate(() => {
    const armoury = document.querySelector('.abyss-side-left').getBoundingClientRect();
    const stage = document.getElementById('abyssStage').getBoundingClientRect();
    return { position: getComputedStyle(document.querySelector('.abyss-side-left')).position, armoury, stage };
  });
  expect(layout.position).toBe('fixed');
  expect(layout.armoury.left).toBeLessThanOrEqual(12);
  expect(layout.armoury.right).toBeLessThan(layout.stage.left);
});

test('ultrawide lobby gives the command deck the available center lane', async ({ page }) => {
  await page.setViewportSize({ width: 2048, height: 1048 });
  await page.goto('/abyss');

  const layout = await page.evaluate(() => {
    const box = selector => document.querySelector(selector).getBoundingClientRect();
    return {
      cockpit: box('#abyssCombatCockpit'),
      row: box('.abyss-stage-row'),
      stage: box('#abyssStage'),
      controls: box('#abyssControls'),
      objective: box('#abCurrentObjective'),
      enter: box('#btnEnter'),
      loot: box('.abyss-side-right'),
    };
  });

  expect(layout.row.width).toBeGreaterThanOrEqual(layout.cockpit.width - 1);
  expect(layout.stage.width).toBeGreaterThan(700);
  expect(layout.controls.width).toBeGreaterThan(420);
  expect(layout.objective.width).toBeGreaterThan(600);
  expect(layout.enter.width).toBeGreaterThan(110);
  expect(layout.enter.right).toBeLessThanOrEqual(layout.stage.right);
  expect(layout.stage.right).toBeLessThanOrEqual(layout.loot.left - 8);
});
