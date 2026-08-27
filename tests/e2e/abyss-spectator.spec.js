const { test, expect } = require('@playwright/test');

const session = '0123456789abcdef0123456789abcdef';
const viewports = [
  { name: 'desktop', width: 1440, height: 900 },
  { name: 'mobile', width: 390, height: 844 },
];

for (const viewport of viewports) {
  test(`read-only spectator is safe and contained on ${viewport.name}`, async ({ page }) => {
    const pageErrors = [];
    const failedAssets = [];
    page.on('pageerror', error => pageErrors.push(error.message));
    page.on('response', response => {
      if (/\/static\/abyss_spectate\.(css|js)/.test(response.url()) && !response.ok()) {
        failedAssets.push(`${response.status()} ${response.url()}`);
      }
    });
    await page.setViewportSize({ width: viewport.width, height: viewport.height });
    await page.emulateMedia({ reducedMotion: 'reduce', colorScheme: 'light' });
    await page.goto(`/abyss/spectate?session=${session}`);

    await expect(page.locator('#spectateConnection')).toHaveText('Live now');
    await expect(page.locator('#spectateRound')).toHaveText('Round 7');
    await expect(page.locator('#spectatePhase')).toHaveText('ACTIVE');
    await expect(page.locator('#spectateAllies [role="listitem"]')).toHaveCount(2);
    await expect(page.locator('#spectateEnemies [role="listitem"]')).toHaveCount(2);
    await expect(page.locator('[role="meter"]')).toHaveCount(4);
    await expect(page.locator('#spectateEnemies')).toContainText('<img src=x');
    await expect(page.locator('#spectateLog')).toContainText('<svg onload=');
    await expect(page.locator('#spectateEnemies img, #spectateLog svg')).toHaveCount(0);
    expect(await page.evaluate(() => ({
      nameInjected: window.spectatorInjected === true,
      logInjected: window.spectatorLogInjected === true,
    }))).toEqual({ nameInjected: false, logInjected: false });

    const layout = await page.evaluate(() => {
      const allies = document.querySelector('.ab-spectator-party').getBoundingClientRect();
      const enemies = document.querySelector('.ab-spectator-hostiles').getBoundingClientRect();
      return {
        allies: { left: allies.left, top: allies.top, width: allies.width },
        enemies: { left: enemies.left, top: enemies.top, width: enemies.width },
        overflow: document.documentElement.scrollWidth - document.documentElement.clientWidth,
        bodyBackground: getComputedStyle(document.body).backgroundColor,
        stageBackground: getComputedStyle(document.querySelector('.ab-spectator-stage')).backgroundColor,
      };
    });
    expect(layout.overflow).toBeLessThanOrEqual(1);
    expect(layout.bodyBackground).toBe('rgb(5, 8, 13)');
    expect(layout.stageBackground).toBe('rgb(10, 17, 27)');
    if (viewport.name === 'desktop') {
      expect(layout.allies.top).toBe(layout.enemies.top);
      expect(layout.allies.left + layout.allies.width).toBeLessThanOrEqual(layout.enemies.left + 1);
    } else {
      expect(layout.enemies.top).toBeGreaterThan(layout.allies.top);
      await expect(page.locator('.ab-spectator-footer a')).toBeVisible();
    }
    expect(pageErrors).toEqual([]);
    expect(failedAssets).toEqual([]);
  });
}

test('spectator closes cleanly when the live session has settled', async ({ page }) => {
  await page.route('**/api/abyss/spectate?**', route => route.fulfill({
    status: 200,
    contentType: 'application/json',
    body: JSON.stringify({ ok: false, error: 'combat is no longer live' }),
  }));
  await page.goto(`/abyss/spectate?session=${session}`);
  await expect(page.locator('#spectateConnection')).toHaveText('Feed closed');
  await expect(page.locator('#spectateConnection')).toHaveClass(/is-ended/);
  await expect(page.locator('#spectatePhase')).toHaveText('combat is no longer live');
});
