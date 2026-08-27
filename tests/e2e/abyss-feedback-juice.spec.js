const { test, expect } = require('@playwright/test');

for (const viewport of [
  { name: 'desktop', width: 1440, height: 1000 },
  { name: 'mobile', width: 390, height: 844 },
]) {
  test(`run feedback remains meaningful on ${viewport.name}`, async ({ page }) => {
    const pageErrors = [];
    page.on('pageerror', error => pageErrors.push(error.message));
    await page.setViewportSize(viewport);
    await page.goto('/abyss?active=1');

    await page.evaluate(() => {
      window.reduceMotion = false;
      const layer = document.getElementById('damageFloatLayer');
      layer.replaceChildren();
      window.combatRecorderState.playing = true;
      window.combatRecorderState.lastHP = 1000;
      window.combatRecorderState.lastEnemyHP = 1000;
      window.updateCombatRecorderFrame({ hp: 990, max_hp: 1000, enemy_hp: 990, enemy_max: 1000 });
    });
    await expect(page.locator('#damageFloatLayer .ab-damage-float')).toHaveCount(0);
    await page.evaluate(() => window.updateCombatRecorderFrame({ hp: 890, max_hp: 1000, enemy_hp: 890, enemy_max: 1000 }));
    await expect(page.locator('#damageFloatLayer .ab-damage-float')).toHaveCount(2);

    await page.evaluate(() => {
      window.__abyssFeedbackJuice.trackFrame({ hp: 1000, max_hp: 1000, enemy_hp: 1000, enemy_max: 1000 });
      window.__abyssFeedbackJuice.trackFrame({ hp: 850, max_hp: 1000, enemy_hp: 720, enemy_max: 1000 });
      window.inRun = true;
      window.setEscrow(54321);
      window.inRun = false;
      window.setEscrow(0);
      window.showRunSummary('💀 Run over', [['Deepest floor', '12'], ['Floors cleared', '11'], ['Loot escrowed (lost)', '4']], 'bad');
    });
    const deathSummary = page.locator('#sharedModalCard .ab-runsummary');
    await expect(deathSummary).toContainText('Damage dealt280');
    await expect(deathSummary).toContainText('Damage taken150');
    await expect(deathSummary).toContainText('Biggest hit280');
    await expect(deathSummary).toContainText('Cache lost54.3k');
    await expect(deathSummary.locator('.ab-run-lesson')).toContainText('Insure a valuable cache');
    await page.locator('#modalOkBtn').click();

    await page.evaluate(() => {
      window.reduceMotion = true;
      window.recordBurst(42);
      window.showBankSummary({ depth: 42, base: 1000, depth_bonus: 200, banked: 1200, escrow_loot: [{ id: 1 }] });
    });
    const bankSummary = page.locator('#sharedModalCard .ab-runsummary');
    await expect(bankSummary).toContainText('Damage dealt280');
    await expect(bankSummary).toContainText('Biggest hit280');
    await expect(bankSummary).toContainText('RecordNew record · floor 42');
    await expect(bankSummary).toHaveAttribute('aria-label', 'Banked run summary for floor 42');
    await page.locator('#modalOkBtn').click();

    await page.evaluate(() => {
      window.reduceMotion = false;
      window.inRun = true;
      window.setEscrow(1);
      window.setEscrow(100000);
    });
    await expect(page.locator('#coinPile')).toHaveClass(/growing/);
    await expect(page.locator('#coinPile')).toHaveAttribute('data-tier', '6');
    expect(pageErrors).toEqual([]);
  });
}
