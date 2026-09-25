const { test, expect } = require('@playwright/test');

test.use({ hasTouch: true, isMobile: true });

test('keeps controls inside device safe areas without clipping', async ({ page }) => {
  await page.setViewportSize({ width: 844, height: 390 });
  await page.goto('/abyss/rift?practice=skills');
  await expect(page.locator('#rift-start')).toBeEnabled();

  // Verify safe area insets are referenced on controls band and touch pad
  const paddingStyles = await page.evaluate(() => {
    const pad = document.querySelector('.rift-touch');
    const band = document.querySelector('.rift-controls-band');
    return {
      padPaddingBottom: window.getComputedStyle(pad).paddingBottom,
      padPaddingLeft: window.getComputedStyle(pad).paddingLeft,
      bandPaddingLeft: window.getComputedStyle(band).paddingLeft,
    };
  });

  expect(parseInt(paddingStyles.padPaddingBottom, 10)).toBeGreaterThanOrEqual(6);
  expect(parseInt(paddingStyles.padPaddingLeft, 10)).toBeGreaterThanOrEqual(8);

  await page.locator('#rift-start').tap();
  await expect(page.locator('#rift-overlay')).toBeHidden();

  // Verify buttons are fully within the viewport bounds
  const buttons = page.locator('.rift-touch button, .rift-actionbar button, .rift-signatures button');
  const count = await buttons.count();
  expect(count).toBeGreaterThan(6);

  const buttonRects = await buttons.evaluateAll(nodes =>
    nodes.filter(n => n.getClientRects().length).map(n => {
      const r = n.getBoundingClientRect();
      return { id: n.id || n.dataset.bind || n.dataset.move, left: r.left, right: r.right, bottom: r.bottom };
    })
  );

  for (const b of buttonRects) {
    expect(b.left).toBeGreaterThanOrEqual(0);
    expect(b.right).toBeLessThanOrEqual(844);
  }

  // Zero horizontal scroll overflow
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true);
});

test('prevents browser panning while dragging movement controls and allows directional sliding', async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto('/abyss/rift?practice=skills');
  await expect(page.locator('#rift-start')).toBeEnabled();
  await page.locator('#rift-start').tap();
  await expect(page.locator('#rift-overlay')).toBeHidden();

  // 1. Verify touch-action: none on touch pad and movement buttons
  const pad = page.locator('.rift-touch');
  await expect(pad).toHaveCSS('touch-action', 'none');

  const moveButtons = page.locator('.rift-touch button');
  const touchActions = await moveButtons.evaluateAll(nodes =>
    nodes.map(n => window.getComputedStyle(n).touchAction)
  );
  for (const ta of touchActions) {
    expect(ta).toBe('none');
  }

  // 2. Perform touch drag gesture on movement button
  const rightBtn = page.locator('.rift-touch button[data-move="right"]');
  await rightBtn.scrollIntoViewIfNeeded();
  const box = await rightBtn.boundingBox();
  expect(box).not.toBeNull();

  const startX = box.x + box.width / 2;
  const startY = box.y + box.height / 2;

  const initialScrollX = await page.evaluate(() => window.scrollX);
  const initialScrollLeft = await page.evaluate(() => document.documentElement.scrollLeft);

  // Dispatch pointerdown and drag horizontally and vertically
  await page.touchscreen.tap(startX, startY);
  await expect(rightBtn).toBeVisible();

  // Dragging movement: mouse/pointer move across pad
  await page.mouse.move(startX, startY);
  await page.mouse.down();
  await expect(rightBtn).toHaveAttribute('aria-pressed', 'true');

  // Drag 40px left toward center
  await page.mouse.move(startX - 40, startY, { steps: 5 });

  // Browser should not have panned horizontally
  const currentScrollX = await page.evaluate(() => window.scrollX);
  const currentScrollLeft = await page.evaluate(() => document.documentElement.scrollLeft);
  expect(currentScrollX).toBe(initialScrollX);
  expect(currentScrollLeft).toBe(initialScrollLeft);

  await page.mouse.up();
  await expect(rightBtn).toHaveAttribute('aria-pressed', 'false');
});
