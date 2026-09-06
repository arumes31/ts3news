const { test, expect } = require('@playwright/test');

for (const viewport of [{ width: 2560, height: 1440 }, { width: 1440, height: 900 }, { width: 390, height: 844 }]) {
  test(`run header leaves room for decisions at ${viewport.width}px`, async ({ page }) => {
    await page.setViewportSize(viewport);
    await page.emulateMedia({ reducedMotion: 'reduce' });
    await page.goto('/abyss?active=1&briefing=1');
    await expect(page.locator('#abyssWorkspaceSelect')).toBeVisible();
    await page.evaluate(() => { window.scrollTo(0, 0); });
    const layout = await page.evaluate(() => {
      const hero = document.querySelector('.abyss-hero').getBoundingClientRect();
      const center = document.getElementById('abCommandCenter').getBoundingClientRect();
      const controls = document.getElementById('abyssControls');
      return { header: center.bottom - hero.top,
        auto: document.getElementById('abyssAutoContinue').getBoundingClientRect().toJSON(),
        controls: controls.getBoundingClientRect().toJSON(),
        overflow: document.documentElement.scrollWidth - innerWidth };
    });
    expect(layout.overflow).toBeLessThanOrEqual(1);
    if (viewport.width > 900) {
      expect(layout.header).toBeLessThan(180);
      expect(layout.auto.bottom).toBeLessThan(viewport.height);
      expect(layout.auto.bottom).toBeLessThanOrEqual(layout.controls.bottom);
      if (viewport.height === 1440) expect(layout.controls.height).toBeGreaterThanOrEqual(700);
    }
    const briefing = page.locator('#abyssDailyBriefing');
    await expect(briefing).not.toHaveAttribute('open');
    await briefing.locator('summary').focus();
    await page.keyboard.press('Enter');
    await expect(briefing).toHaveAttribute('open');
    await expect(briefing.locator('.ab-daily-briefing-body')).toBeVisible();
    await expect(briefing.locator('#bountyCard')).toContainText('Defeat 5 Abyss bosses today');
    await expect(briefing.locator('#bountyProg')).toHaveText('4');
    expect(await page.evaluate(() => document.documentElement.scrollWidth - innerWidth)).toBeLessThanOrEqual(1);
    await page.keyboard.press('Enter');
    await expect(briefing).not.toHaveAttribute('open');
  });
}

for (const reducedMotion of ['no-preference', 'reduce']) {
  test(`scroll feedback follows the scrolled surface with motion ${reducedMotion}`, async ({ page }) => {
    await page.emulateMedia({ reducedMotion });
    await page.goto('/abyss?active=1');
    await expect(page.locator('#abyssWorkspaceSelect')).toBeVisible();
    const log = page.locator('#abyssLog');
    await log.evaluate(node => {
      node.textContent = 'Combat exchange\n'.repeat(200);
      node.scrollTop = 100;
    });
    await expect(log).toHaveClass(/ab-scroll-active/);
    await expect(log).toHaveCSS('animation-name', reducedMotion === 'reduce' ? 'none' : 'ab-scroll-reveal');
    await expect(log).not.toHaveClass(/ab-scroll-active/, { timeout: 3000 });
    await expect(log).toHaveCSS('scrollbar-color', 'rgb(172, 139, 80) rgb(16, 25, 35)');
    await page.evaluate(() => window.scrollTo(0, document.documentElement.scrollHeight));
    await expect(page.locator('html')).toHaveClass(/ab-scroll-active/);
  });
}
