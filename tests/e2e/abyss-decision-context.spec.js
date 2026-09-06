const { test, expect } = require('@playwright/test');

test('preparation exposes one step and keeps setup beside entry on mobile', async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto('/abyss');
  await expect(page.locator('#startPicker')).toBeHidden();
  await expect(page.locator('#buildPicker')).toBeHidden();
  await page.getByRole('tab', { name: '2 Route', exact: true }).click();
  await expect(page.locator('#startPicker')).toBeVisible();
  await expect(page.locator('#tierPicker')).toBeHidden();
  await page.keyboard.press('ArrowRight');
  await expect(page.locator('#buildPicker')).toBeVisible();
  await expect(page.locator('#startPicker')).toBeHidden();
  await page.locator('#buildKit').selectOption('arcanist');
  await expect(page.locator('#abyssEntrySummaryLine')).toContainText('Arcanist');
  const gap = await page.evaluate(() => document.getElementById('btnEnter').getBoundingClientRect().top - document.getElementById('abyssEntrySummary').getBoundingClientRect().bottom);
  expect(gap).toBeGreaterThanOrEqual(0);
  expect(gap).toBeLessThan(160);
  await page.evaluate(() => window.scrollTo(0, 0));
  await page.screenshot({ path: '.impeccable/critique/runtime/adapt-mobile-entry.png' });
});

test('mobile dock mirrors live stakes and banking lock without separate actions', async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto('/abyss?active=1');
  await page.evaluate(() => { setHP(750, 1000); setDepth(12); setEscrow(3456); renderState(); });
  await expect(page.locator('#abyssMobileContext')).toContainText('750');
  await expect(page.locator('#abyssMobileContext')).toContainText('12');
  await expect(page.locator('[data-mobile-action="btnBank"]')).toHaveAccessibleName('Bank & Leave');
  await page.evaluate(() => { setHP(600, 1000); setDepth(13); setEscrow(5000); bankLocked = 2; renderState(); });
  await expect(page.locator('#abyssMobileContext')).toContainText('600');
  await expect(page.locator('#abyssMobileContext')).toContainText('13');
  await expect(page.locator('[data-mobile-action="btnBank"]')).toBeDisabled();
  await expect(page.locator('[data-mobile-action="btnBank"]')).toContainText('Sealed');
  await page.evaluate(() => window.scrollTo(0, 1800));
  const boxes = await page.evaluate(() => {
    const box = id => document.getElementById(id).getBoundingClientRect().toJSON();
    return { context: box('abyssMobileContext'), dock: box('abyssMobileActions'), height: innerHeight, overflow: document.documentElement.scrollWidth - innerWidth };
  });
  expect(boxes.context.top).toBeGreaterThan(0);
  expect(boxes.context.bottom).toBeLessThan(boxes.height);
  expect(boxes.dock.bottom).toBeLessThanOrEqual(boxes.height);
  expect(boxes.overflow).toBeLessThanOrEqual(1);
  await page.evaluate(() => { bankLocked = 0; document.body.classList.add('ab-pact-blind'); renderState(); });
  await expect(page.locator('#abMobileThreat')).toHaveText('Hidden by pact');
  await page.evaluate(() => { document.body.classList.remove('ab-pact-blind'); renderState(); window.scrollTo(0, 0); });
  await page.screenshot({ path: '.impeccable/critique/runtime/adapt-mobile-run.png' });
  await page.evaluate(() => { inRun = false; renderState(); });
  await expect(page.locator('#abyssMobileActions')).toBeHidden();
});

for (const width of [390, 1440]) {
  test(`persistent workspace switcher reaches Forge and returns to run at ${width}px`, async ({ page }) => {
    await page.setViewportSize({ width, height: 900 });
    await page.emulateMedia({ reducedMotion: 'reduce' });
    await page.goto('/abyss?active=1');
    if (width === 1440) await page.screenshot({ path: '.impeccable/critique/runtime/adapt-desktop-run.png' });
    const switcher = page.getByLabel('Workspace', { exact: true });
    expect(await switcher.evaluate(node => node.getBoundingClientRect().width)).toBeGreaterThan(150);
    await switcher.selectOption('forge');
    await expect(page.locator('#abyss-tab-forge')).toHaveAttribute('aria-selected', 'true');
    await expect(page.locator('#abyssWorkshop')).toBeVisible();
    await expect(switcher).toBeInViewport();
    await page.getByRole('button', { name: 'Back to run', exact: true }).click();
    await expect(page.locator('#abyssStage')).toBeInViewport();
    await page.locator('#abyss-tab-shop').evaluate(node => node.click());
    await expect(switcher).toHaveValue('shop');
  });
}
