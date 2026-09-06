const { test, expect } = require('@playwright/test');

for (const width of [390, 768, 1440]) {
  test(`workspace destinations clear sticky navigation at ${width}px`, async ({ page }) => {
    await page.setViewportSize({ width, height: 900 });
    await page.emulateMedia({ reducedMotion: 'reduce' });
    await page.goto('/abyss?active=1&gear=1');
    await page.getByLabel('Workspace', { exact: true }).selectOption('forge');
    const panel = page.locator('#abyssForgePanel');
    await expect(panel).toBeInViewport();
    const gap = await page.evaluate(() => document.getElementById('abyssForgePanel').getBoundingClientRect().top - document.getElementById('abCommandCenter').getBoundingClientRect().bottom);
    expect(gap).toBeGreaterThanOrEqual(8);
    await page.getByRole('button', { name: 'Back to run', exact: true }).click();
    const runGap = await page.evaluate(() => document.getElementById('abyssStage').getBoundingClientRect().top - document.getElementById('abCommandCenter').getBoundingClientRect().bottom);
    // The run can remain at its natural document position with a smaller gap.
    expect(runGap).toBeGreaterThanOrEqual(0);
  });
}

test('mobile back-to-top clears the entire run dock', async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await page.emulateMedia({ reducedMotion: 'reduce' });
  await page.goto('/abyss?active=1');
  await page.getByLabel('Workspace', { exact: true }).selectOption('forge');
  const back = page.getByRole('button', { name: 'Back to top', exact: true });
  await expect(back).toBeVisible();
  const gap = await page.evaluate(() => document.getElementById('abyssMobileActions').getBoundingClientRect().top - document.getElementById('abBackTop').getBoundingClientRect().bottom);
  expect(gap).toBeGreaterThanOrEqual(8);
  await back.click();
  await expect.poll(() => page.evaluate(() => scrollY)).toBe(0);
});
