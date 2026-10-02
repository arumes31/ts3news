const { test, expect } = require('@playwright/test');

test.describe('Countdown announcement throttling (Proposal 0158)', () => {
  test('rapid countdown elements suppress frame-by-frame live region mutations', async ({ page }) => {
    await page.goto('/abyss/rift');
    const transition = page.locator('#rift-transition');
    const bossAttack = page.locator('#rift-boss-attack');
    const announcer = page.locator('#rift-announcer');

    // Both countdown ticker nodes must have aria-live="off" and not role="status"
    await expect(transition).toHaveAttribute('aria-live', 'off');
    expect(await transition.getAttribute('role')).toBeNull();

    await expect(bossAttack).toHaveAttribute('aria-live', 'off');
    expect(await bossAttack.getAttribute('role')).toBeNull();

    // The dedicated polite announcer exists for milestone announcements
    await expect(announcer).toHaveAttribute('aria-live', 'polite');
    await expect(announcer).toHaveAttribute('role', 'status');
  });

  test('room transition countdown announces milestone at start and finish without frame spam', async ({ page }) => {
    await page.goto('/abyss/rift?scenario=checkpoint');
    await expect(page.locator('#rift-start')).toBeEnabled();

    // Configure 3s delay
    await page.locator('.rift-settings > summary').click();
    await page.locator('#rift-transition-delay').selectOption('3');
    await page.locator('.rift-settings > summary').click();

    const transition = page.locator('#rift-transition');

    await page.evaluate(() => {
      window._announcements = [];
      const obs = new MutationObserver(() => {
        const txt = document.getElementById('rift-announcer')?.textContent || '';
        if (txt) window._announcements.push(txt);
      });
      obs.observe(document.getElementById('rift-announcer'), { childList: true, characterData: true, subtree: true });
    });

    // Start countdown
    await page.locator('#rift-start').click();

    // Visual transition element ticks smoothly
    await expect(transition).toBeVisible();
    await expect(transition).toContainText('Next:');

    // Announcer gets initial start milestone announcement
    await expect.poll(async () => {
      return await page.evaluate(() => window._announcements.some(a => a.includes('Next tier:')));
    }).toBe(true);

    // Wait 1.5 seconds during the countdown
    await page.waitForTimeout(1500);

    // Verify announcer did NOT receive dozens of frame-by-frame updates
    const countDuringCountdown = await page.evaluate(() => window._announcements.length);
    // Over 1.5s (18 frames of 85ms), count should be very small (e.g. <= 3), NOT 18!
    expect(countDuringCountdown).toBeLessThanOrEqual(3);

    // Eventually completes and advances
    await expect.poll(async () => {
      return await page.evaluate(() => window._announcements.some(a => a.includes('Banking rewards and advancing')));
    }, { timeout: 5000 }).toBe(true);
  });

  test('boss attack windup displays smoothly with single onset announcement', async ({ page }) => {
    await page.goto('/abyss/rift?scenario=boss-windup');
    await expect(page.locator('#rift-start')).toBeEnabled();

    await page.evaluate(() => {
      window._bossAnnouncements = [];
      const obs = new MutationObserver(() => {
        const txt = document.getElementById('rift-announcer')?.textContent || '';
        if (txt) window._bossAnnouncements.push(txt);
      });
      obs.observe(document.getElementById('rift-announcer'), { childList: true, characterData: true, subtree: true });
    });

    // Resume boss fight
    await page.locator('#rift-start').click();

    const bossAttack = page.locator('#rift-boss-attack');
    await expect(bossAttack).toBeVisible();
    await expect(bossAttack).toContainText('⚡ Mossbound Slam');
    await expect(bossAttack).toContainText('windup');
    await expect(bossAttack).toHaveAttribute('aria-live', 'off');

    // Announcer should receive exactly one boss preparation announcement
    await expect.poll(async () => {
      return await page.evaluate(() => window._bossAnnouncements.filter(a => a.includes('Boss preparing')).length);
    }).toBe(1);

    // Wait 500ms while attack is still in windup
    await page.waitForTimeout(500);

    // Announcer should NOT have grown on every 85ms frame
    const windupAnnounceCount = await page.evaluate(() => window._bossAnnouncements.filter(a => a.includes('Boss preparing')).length);
    expect(windupAnnounceCount).toBe(1);
  });
});
