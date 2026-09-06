const { test, expect } = require('@playwright/test');
const { expectVisibleSurface } = require('./helpers/abyss-surface');

for (const overflow of [0, 1]) {
  test(`surface within the ${overflow}px overflow tolerance avoids unrelated geometry reads`, async ({ page }) => {
    await page.setContent(`
      <style>body { margin: 0; } #unrelated { width: calc(100vw + ${overflow}px); }</style>
      <main id="surface"><button>Descend</button></main>
      <div id="unrelated">${'<span>Decoration</span> '.repeat(200)}</div>
    `);
    expect(await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth)).toBe(overflow);
    await page.evaluate(() => {
      window.unrelatedGeometryReads = 0;
      const original = Element.prototype.getBoundingClientRect;
      Element.prototype.getBoundingClientRect = function (...args) {
        if (this.closest('#unrelated')) window.unrelatedGeometryReads += 1;
        return original.apply(this, args);
      };
    });

    await expectVisibleSurface(page, '#surface', 'Combat');

    expect(await page.evaluate(() => window.unrelatedGeometryReads)).toBe(0);
  });
}

test('surface overflow still fails with the offending element and its bounds', async ({ page }) => {
  await page.setContent(`
    <style>body { margin: 0; } #wide { width: calc(100vw + 20px); }</style>
    <main id="surface"><button>Descend</button></main>
    <div id="wide">Overflow outside the inspected surface</div>
  `);

  await expect(expectVisibleSurface(page, '#surface', 'Combat'))
    .rejects.toThrow(/Combat horizontal overflow: #wide \[0, \d+\]/);
});
