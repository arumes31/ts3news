const { test, expect } = require('@playwright/test');

test.use({ hasTouch: true, isMobile: true, viewport: { width: 390, height: 844 } });

test('offers abbreviated ability names with full accessible labels', async ({ page }) => {
  await page.goto('/abyss/rift?practice=skills&subclass=elementalist');
  await expect(page.locator('#rift-start')).toBeEnabled();

  // Open settings and enable abbreviated ability names
  const settings = page.locator('.rift-settings');
  if (!(await settings.evaluate(el => el.open))) {
    await page.locator('.rift-settings > summary').tap();
  }
  const toggle = page.locator('#rift-abbreviate-abilities');
  await expect(toggle).toBeVisible();
  await toggle.check();

  // Verify persistence
  const saved = await page.evaluate(() => localStorage.getItem('riftAbbreviateAbilities'));
  expect(saved).toBe('true');

  await page.locator('#rift-start').tap();
  await expect(page.locator('#rift-overlay')).toBeHidden();

  // Find the Frost Detonation button
  const frostBtn = page.locator('#rift-signatures button, #rift-skills button').filter({ hasText: /Frost/ });
  await expect(frostBtn).toHaveCount(1);

  // Full accessible label must be preserved (starts with full ability name)
  await expect(frostBtn).toHaveAttribute('aria-label', /^Frost Detonation/);

  // Visible text label must be abbreviated
  const label = frostBtn.locator('.rift-action-label');
  await expect(label).toHaveText('Frost Det.');

  // Unchecking returns to full name
  await page.locator('#rift-pause').tap();
  if (!(await settings.evaluate(el => el.open))) {
    await page.locator('.rift-settings > summary').tap();
  }
  await expect(toggle).toBeVisible();
  await toggle.uncheck();
  await expect(label).toHaveText('Frost Detonation');
  await expect(frostBtn).toHaveAttribute('aria-label', /^Frost Detonation/);
});

test('handles mobile browser toolbar resizing smoothly without overflow', async ({ page }) => {
  await page.goto('/abyss/rift?subclass=vanguard');
  await expect(page.locator('#rift-start')).toBeEnabled();

  // Baseline 390x844 (retracted toolbar)
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true);

  // Simulate mobile browser top/bottom toolbar expansion (e.g. height shrinks to 720px)
  await page.setViewportSize({ width: 390, height: 720 });
  await page.waitForTimeout(100);
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true);

  const visualHeight720 = await page.evaluate(() => {
    return getComputedStyle(document.getElementById('rift-app')).getPropertyValue('--rift-visual-height');
  });
  expect(visualHeight720).toBe('720px');

  const startBtn = page.locator('#rift-start');
  await expect(startBtn).toBeVisible();
  await startBtn.scrollIntoViewIfNeeded();
  const rect720 = await startBtn.evaluate(el => {
    const r = el.getBoundingClientRect();
    return { top: r.top, bottom: r.bottom };
  });
  expect(rect720.bottom).toBeLessThanOrEqual(720);

  // Further shrink (e.g. keyboard or small phone toolbar 600px)
  await page.setViewportSize({ width: 390, height: 600 });
  await page.waitForTimeout(100);
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true);

  const visualHeight600 = await page.evaluate(() => {
    return getComputedStyle(document.getElementById('rift-app')).getPropertyValue('--rift-visual-height');
  });
  expect(visualHeight600).toBe('600px');
});
