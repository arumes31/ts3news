const { test, expect } = require('@playwright/test');

test.use({ hasTouch: true, isMobile: true });

test('combat buttons suppress double-tap zoom and accidental selection while outer text retains selection', async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto('/abyss/rift?practice=skills&subclass=elementalist');
  await expect(page.locator('#rift-start')).toBeEnabled();
  await page.locator('#rift-start').tap();await expect(page.locator('#rift-overlay')).toBeHidden();

  // 1. Verify combat buttons have touch-action: manipulation or none (suppressing double-tap zoom)
  const combatButtons = page.locator('#rift-actionbar button, #rift-signatures button, #rift-pause');
  const count = await combatButtons.count();
  expect(count).toBeGreaterThan(5);

  const buttonStyles = await combatButtons.evaluateAll(nodes =>
    nodes.map(node => {
      const style = window.getComputedStyle(node);
      return {
        id: node.id || node.dataset.bind || node.getAttribute('aria-label'),
        touchAction: style.touchAction,
        userSelect: style.userSelect || style.webkitUserSelect,
      };
    })
  );

  for (const btn of buttonStyles) {
    expect(
      ['manipulation', 'none'].includes(btn.touchAction),
      `Expected touch-action manipulation or none on ${btn.id}, got ${btn.touchAction}`
    ).toBe(true);
    expect(
      btn.userSelect,
      `Expected user-select none on ${btn.id}`
    ).toBe('none');
  }

  // 2. Verify non-combat content preserves normal text selection and touch scrolling
  const contentElements = page.locator('.rift-heading h1, #rift-rules h3, .rift-legend-grid, .rift-fine');
  const contentStyles = await contentElements.evaluateAll(nodes =>
    nodes.map(node => {
      const style = window.getComputedStyle(node);
      return {
        tag: node.tagName,
        userSelect: style.userSelect || style.webkitUserSelect,
        touchAction: style.touchAction,
      };
    })
  );

  expect(contentStyles.length).toBeGreaterThan(0);
  for (const elem of contentStyles) {
    expect(elem.userSelect).not.toBe('none');
    expect(elem.touchAction).not.toBe('none');
  }

  // 3. Rapid double tap on combat button does not zoom the viewport
  const attackBtn = page.locator('#rift-actionbar button[data-bind="attack"]');
  await attackBtn.tap();
  await attackBtn.tap();

  const scale = await page.evaluate(() => window.visualViewport ? window.visualViewport.scale : 1);
  expect(scale).toBe(1);

  await page.locator('#rift-pause').tap();
});
