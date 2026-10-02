const { test, expect } = require('@playwright/test');

function expectColorClose(actual, expectedR, expectedG, expectedB) {
  const match = actual.match(/rgba?\((\d+),\s*(\d+),\s*(\d+)/);
  expect(match).not.toBeNull();
  const [r, g, b] = [Number(match[1]), Number(match[2]), Number(match[3])];
  expect(Math.abs(r - expectedR)).toBeLessThanOrEqual(3);
  expect(Math.abs(g - expectedG)).toBeLessThanOrEqual(3);
  expect(Math.abs(b - expectedB)).toBeLessThanOrEqual(3);
}

test('maintains high-contrast visible focus on dark mission thumbnails across states and navigation', async ({ page }) => {
  await page.goto('/abyss/rift');
  await expect(page.locator('#rift-start')).toBeEnabled();

  const grid = page.locator('#rift-levels');
  await expect(grid).toBeVisible();

  const cards = page.locator('#rift-levels button[data-level]');
  await expect(cards.first()).toBeVisible();

  // 1. Verify container padding accommodates focus indicator without clipping
  const gridPadding = await grid.evaluate(el => {
    const style = window.getComputedStyle(el);
    return {
      paddingTop: parseFloat(style.paddingTop),
      paddingLeft: parseFloat(style.paddingLeft),
      paddingRight: parseFloat(style.paddingRight),
      paddingBottom: parseFloat(style.paddingBottom),
    };
  });
  expect(gridPadding.paddingTop).toBeGreaterThanOrEqual(6);
  expect(gridPadding.paddingLeft).toBeGreaterThanOrEqual(6);

  // 2. Test selected & focused state on initial selected mission (Mission 1)
  const card1 = page.locator('#rift-levels button[data-level="1"]');
  await expect(card1).toHaveAttribute('aria-pressed', 'true');
  await card1.focus();
  await expect(card1).toBeFocused();

  const selectedFocusStyles = await card1.evaluate(el => {
    const style = window.getComputedStyle(el);
    return {
      outlineStyle: style.outlineStyle,
      outlineColor: style.outlineColor,
      outlineOffset: style.outlineOffset,
      borderColor: style.borderColor,
      boxShadow: style.boxShadow,
      zIndex: style.zIndex,
    };
  });

  expect(selectedFocusStyles.outlineStyle).toBe('solid');
  expectColorClose(selectedFocusStyles.outlineColor, 255, 232, 133);
  expect(selectedFocusStyles.outlineOffset).toBe('2px');
  expect(selectedFocusStyles.boxShadow).toContain('inset');
  expect(selectedFocusStyles.boxShadow).toMatch(/rgba?\(25[45],\s*25[45],\s*25[2-5]/);
  expect(selectedFocusStyles.boxShadow).toMatch(/rgba?\(7,\s*21,\s*16/);
  expect(selectedFocusStyles.zIndex).toBe('3');

  // 3. Test unselected & focused state on Mission 2
  const card2 = page.locator('#rift-levels button[data-level="2"]');
  await expect(card2).toHaveAttribute('aria-pressed', 'false');
  await card2.focus();
  await expect(card2).toBeFocused();

  const unselectedFocusStyles = await card2.evaluate(el => {
    const style = window.getComputedStyle(el);
    return {
      outlineStyle: style.outlineStyle,
      outlineColor: style.outlineColor,
      outlineOffset: style.outlineOffset,
      borderColor: style.borderColor,
      boxShadow: style.boxShadow,
      zIndex: style.zIndex,
    };
  });

  expect(unselectedFocusStyles.outlineStyle).toBe('solid');
  expectColorClose(unselectedFocusStyles.outlineColor, 255, 232, 133);
  expect(unselectedFocusStyles.outlineOffset).toBe('2px');
  expect(unselectedFocusStyles.boxShadow).toMatch(/rgba?\(7,\s*21,\s*16/);
  expect(unselectedFocusStyles.boxShadow).toMatch(/rgba?\(25[45],\s*232,\s*13[3-5]/);
  expect(unselectedFocusStyles.zIndex).toBe('3');

  // Distinct contrast between unselected focused border vs selected focused border
  expect(unselectedFocusStyles.borderColor).not.toBe('rgb(255, 255, 255)');

  // 4. Keyboard navigation (Arrow keys / Home / End) retains focus and auto-scrolls
  await card1.focus();
  await page.keyboard.press('ArrowRight');
  await expect(card2).toBeFocused();

  await page.keyboard.press('End');
  const lastVisibleCard = page.locator('#rift-levels button[data-level]:not([hidden])').last();
  await expect(lastVisibleCard).toBeFocused();

  // Assert last card is scrolled into view
  const isLastInView = await lastVisibleCard.evaluate((el) => {
    const rect = el.getBoundingClientRect();
    const container = el.closest('#rift-levels').getBoundingClientRect();
    return rect.bottom <= container.bottom + 10 && rect.top >= container.top - 10;
  });
  expect(isLastInView).toBe(true);

  // Return to first card
  await page.keyboard.press('Home');
  await expect(card1).toBeFocused();

  // 5. Test compact view maintains visible focus
  const compactCheck = page.locator('#rift-compact');
  await compactCheck.check();
  await expect(grid).toHaveClass(/compact/);

  await card2.focus();
  await expect(card2).toBeFocused();
  const compactFocusOutline = await card2.evaluate(el => window.getComputedStyle(el).outlineColor);
  expectColorClose(compactFocusOutline, 255, 232, 133);
});
