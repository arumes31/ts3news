const { test, expect } = require('@playwright/test');

for (const viewport of [{ width: 1440, height: 1000 }, { width: 2560, height: 1440 }, { width: 390, height: 844 }]) {
  test(`completed floor keeps feedback and expiry readable at ${viewport.width}px`, async ({ page }) => {
    await page.setViewportSize(viewport);
    await page.emulateMedia({ reducedMotion: 'reduce' });
    await page.goto('/abyss?active=1');
    await page.evaluate(() => {
      window.activeBuffs = [{ id: 'iron_skin', icon: '🛡️', label: 'Iron Skin (+10 DEF)', fights: 1 }];
      renderBuffStrip();
      setBanner('Floor 12 cleared! +247g to your cache. Final overkill: 222,298 excess damage.', 'good');
      const biome = document.getElementById('biomeChip');
      biome.textContent = 'Cinder-Choked Mournhollow';
      biome.style.display = '';
      focusAbyssDescend();
    });
    const layout = await page.evaluate(() => {
      const box = id => document.getElementById(id).getBoundingClientRect().toJSON();
      return { stage: box('abyssStage'), banner: box('abyssBanner'), scene: box('stageScene'),
        status: box('abStatus'), actions: box('abyssPrimaryActions'), warning: box('abExpiryWarning'),
        overflow: document.documentElement.scrollWidth - innerWidth };
    });
    expect(layout.warning.width).toBeGreaterThan(layout.stage.width * 0.65);
    expect(layout.warning.height).toBeLessThan(90);
    expect(layout.banner.bottom).toBeLessThanOrEqual(layout.stage.top);
    expect(layout.status.bottom).toBeLessThanOrEqual(layout.actions.top);
    expect(layout.scene.bottom).toBeLessThanOrEqual(layout.actions.top);
    expect(layout.overflow).toBeLessThanOrEqual(1);
    if (viewport.width > 900) expect(layout.stage.height).toBeLessThan(520);
    const actions = viewport.width > 900 ? ['#btnDescend', '#btnBank'] : ['#abyssMobileActions [data-mobile-action=btnDescend]', '#abyssMobileActions [data-mobile-action=btnBank]'];
    for (const action of actions) await expect(page.locator(action)).toBeVisible();
  });
}
