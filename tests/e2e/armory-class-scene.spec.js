const { test, expect } = require('@playwright/test');

const classes = {
  warrior: ['vanguard', 'berserker'],
  ranger: ['marksman', 'beastmaster'],
  arcanist: ['elementalist', 'chronomancer'],
  warden: ['oracle', 'geomancer'],
  reaver: ['bloodblade', 'voidwalker'],
  artificer: ['runesmith', 'alchemist'],
};

test('every active class and subclass renders its own cached background', async ({ page }) => {
  const errors = [];
  page.on('pageerror', error => errors.push(error.message));
  for (const [parent, subclasses] of Object.entries(classes)) {
    for (const style of [parent, ...subclasses]) {
      await page.goto(`/armory-fixture?class=${parent}&subclass=${style === parent ? '' : style}`);
      const scene = page.locator('.armory-scene');
      const asset = style === 'warrior' ? 'hall' : style;
      await expect(scene).toHaveAttribute('src', new RegExp(`/static/armory_${asset}\\.webp\\?v=[a-f0-9]+$`));
      await expect.poll(() => scene.evaluate(image => image.naturalWidth)).toBeGreaterThan(1000);
      await expect(page.locator('.armory-portrait')).toHaveAccessibleName(new RegExp(style, 'i'));
      await expect(page.locator('.armory-equipment .gear-cell')).toHaveCount(30);
    }
  }
  expect(errors).toEqual([]);
});

test('switching from subclass to foundation restores the class scene; unset uses default', async ({ page }) => {
  for (const [query, asset] of [
    ['class=reaver&subclass=voidwalker', 'voidwalker'],
    ['class=reaver', 'reaver'],
    ['class=ranger', 'ranger'],
    ['', 'hall'],
    ['class=unknown&subclass=unknown', 'hall'],
  ]) {
    await page.goto(`/armory-fixture?${query}`);
    await expect(page.locator('.armory-scene')).toHaveAttribute('src', new RegExp(`armory_${asset}\\.webp`));
  }
});

test('subclass backgrounds preserve equipment access on desktop and mobile', async ({ page }, testInfo) => {
  for (const width of [1440, 390]) {
    await page.setViewportSize({ width, height: 1000 });
    await page.goto('/armory-fixture?class=arcanist&subclass=chronomancer');
    await expect(page.locator('.armory-scene')).toHaveJSProperty('complete', true);
    await expect(page.locator('.armory-portrait')).toBeVisible();
    expect(await page.evaluate(() => document.documentElement.scrollWidth - innerWidth)).toBeLessThanOrEqual(1);
    await page.screenshot({ path: testInfo.outputPath(`chronomancer-${width}.png`), fullPage: true });
  }
});
