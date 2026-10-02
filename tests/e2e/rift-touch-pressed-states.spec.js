const { test, expect } = require('@playwright/test');

test.describe('Give touch controls descriptive pressed states (Proposal 0156)', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/abyss/rift');
    await expect(page.locator('#rift-start')).toBeEnabled();
  });

  test('touch movement buttons have accessible labels, tooltips, and default aria-pressed="false"', async ({ page }) => {
    const leftBtn = page.locator('.rift-touch button[data-move="left"]');
    await expect(leftBtn).toBeAttached();
    await expect(leftBtn).toHaveAttribute('aria-label', 'Move left');
    await expect(leftBtn).toHaveAttribute('title', 'Move left');
    await expect(leftBtn).toHaveAttribute('aria-pressed', 'false');

    const upBtn = page.locator('.rift-touch button[data-move="up"]');
    await expect(upBtn).toHaveAttribute('aria-label', 'Move up');
    await expect(upBtn).toHaveAttribute('title', 'Move up');

    const downBtn = page.locator('.rift-touch button[data-move="down"]');
    await expect(downBtn).toHaveAttribute('aria-label', 'Move down');
    await expect(downBtn).toHaveAttribute('title', 'Move down');

    const rightBtn = page.locator('.rift-touch button[data-move="right"]');
    await expect(rightBtn).toHaveAttribute('aria-label', 'Move right');
    await expect(rightBtn).toHaveAttribute('title', 'Move right');
  });

  test('pressing touch movement control activates descriptive aria-label, aria-pressed, and visual pressed state', async ({ page }) => {
    await page.setViewportSize({ width: 480, height: 800 });
    // Start game so touch input is active
    await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();
    await expect(page.locator('#rift-pause')).toBeEnabled();

    const leftBtn = page.locator('.rift-touch button[data-move="left"]');
    await expect(leftBtn).toBeVisible();

    await leftBtn.scrollIntoViewIfNeeded();
    const box = await leftBtn.boundingBox();
    expect(box).not.toBeNull();

    // Trigger pointerdown (touch press)
    await page.mouse.move(box.x + box.width / 2, box.y + box.height / 2);
    await page.mouse.down();

    // Verify descriptive pressed state while held down
    await expect(leftBtn).toHaveAttribute('aria-pressed', 'true');
    await expect(leftBtn).toHaveAttribute('data-pressed', 'true');
    await expect(leftBtn).toHaveClass(/rift-held/);
    await expect(leftBtn).toHaveAttribute('aria-label', 'Moving left (holding)');

    // Wait for the button's background transition to reach its pressed color.
    await expect(leftBtn).toHaveCSS('background-color', 'rgb(52, 82, 60)'); // #34523c
    await expect(leftBtn).toHaveCSS('border-color', 'rgb(247, 214, 145)'); // #f7d691

    // Release pointer (touch release)
    await page.mouse.up();

    // Verify clean restoration
    await expect(leftBtn).toHaveAttribute('aria-pressed', 'false');
    await expect(leftBtn).not.toHaveAttribute('data-pressed', 'true');
    await expect(leftBtn).not.toHaveClass(/rift-held/);
    await expect(leftBtn).toHaveAttribute('aria-label', 'Move left');
  });

  test('combat action buttons reflect aria-pressed and data-pressed while held down', async ({ page }) => {
    await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();
    await expect(page.locator('#rift-pause')).toBeEnabled();

    const attackBtn = page.locator('.rift-basics button[data-bind="attack"]');
    await expect(attackBtn).toBeVisible();
    await expect(attackBtn).toHaveAttribute('aria-pressed', 'false');

    await attackBtn.scrollIntoViewIfNeeded();
    const box = await attackBtn.boundingBox();
    await page.mouse.move(box.x + box.width / 2, box.y + box.height / 2);
    await page.mouse.down();

    await expect(attackBtn).toHaveAttribute('aria-pressed', 'true');
    await expect(attackBtn).toHaveAttribute('data-pressed', 'true');
    await expect(attackBtn).toHaveClass(/rift-held/);

    await page.mouse.up();
    await expect(attackBtn).toHaveAttribute('aria-pressed', 'false');
    await expect(attackBtn).not.toHaveAttribute('data-pressed', 'true');
    await expect(attackBtn).not.toHaveClass(/rift-held/);
  });

  test('toggle guard maintains pressed state until toggled off', async ({ page }) => {
    // Open controls dialog and enable toggle guard
    await page.locator('#rift-controls-open').click();
    const toggleCheckbox = page.locator('#rift-toggle-guard');
    await toggleCheckbox.check();
    await page.locator('#rift-controls-close').click();

    // Start expedition
    await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();
    await expect(page.locator('#rift-pause')).toBeEnabled();

    const guardBtn = page.locator('.rift-basics button[data-bind="guard"]');

    // Click guard to latch on
    await guardBtn.click();
    await expect(guardBtn).toHaveAttribute('aria-pressed', 'true');
    await expect(guardBtn).toHaveAttribute('data-pressed', 'true');
    await expect(guardBtn).toHaveClass(/rift-held/);

    // Click guard again to unlatch
    await guardBtn.click();
    await expect(guardBtn).toHaveAttribute('aria-pressed', 'false');
    await expect(guardBtn).not.toHaveAttribute('data-pressed', 'true');
    await expect(guardBtn).not.toHaveClass(/rift-held/);
  });

  test('touch controls adapt pressed states in forced-colors mode and respect reduced motion', async ({ page }) => {
    await page.setViewportSize({ width: 480, height: 800 });
    await page.emulateMedia({ forcedColors: 'active', reducedMotion: 'reduce' });
    await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();
    await expect(page.locator('#rift-pause')).toBeEnabled();

    const leftBtn = page.locator('.rift-touch button[data-move="left"]');
    await expect(leftBtn).toBeVisible();
    await leftBtn.scrollIntoViewIfNeeded();
    const box = await leftBtn.boundingBox();
    expect(box).not.toBeNull();
    await page.mouse.move(box.x + box.width / 2, box.y + box.height / 2);
    await page.mouse.down();

    await expect(leftBtn).toHaveAttribute('aria-pressed', 'true');

    // In reduced motion, transform should be none
    const transform = await leftBtn.evaluate((el) => window.getComputedStyle(el).transform);
    expect(transform).toBe('none');

    await page.mouse.up();
  });
});
