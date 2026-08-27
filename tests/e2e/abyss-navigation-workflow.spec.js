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

test('successful shop and forge actions return focus to Descend', async ({ page }) => {
  await page.emulateMedia({ reducedMotion: 'reduce' });
  await page.goto('/abyss?active=1');

  const result = await page.evaluate(async () => {
    window.inRun = true;
    window.reduceMotion = true;
    window.abRawPost = async () => ({ ok: true });
    const controls = document.getElementById('abyssControls');
    const descend = document.getElementById('btnDescend');
    descend.style.display = '';
    let scrollCalls = 0;
    controls.scrollIntoView = () => { scrollCalls += 1; };

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
    };
  });

  expect(result).toEqual({ shopFocused: true, forgeFocused: true, scrollCalls: 2 });
});
