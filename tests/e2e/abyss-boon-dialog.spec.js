const { test, expect } = require('@playwright/test');

for (const viewport of [{ width: 834, height: 542 }, { width: 1440, height: 900 }, { width: 320, height: 640 }]) {
  test(`boon choices wrap inside the dialog at ${viewport.width}px`, async ({ page }) => {
    await page.setViewportSize(viewport);
    await page.goto('/abyss?active=1&chronicle=draft');
    const dialog = page.getByRole('dialog', { name: /Choose a boon/ });
    await expect(dialog).toBeVisible();
    await page.evaluate(() => document.fonts.ready);
    const choices = dialog.locator('.ab-boon-draft button');
    await expect(choices).toHaveCount(3);
    await expect(choices.first().locator('b')).toHaveText("⚔ Giant's Favor");
    await expect(choices.first().locator('span')).toHaveText('+8% STR per stack');
    await choices.first().evaluate(button => {
      button.querySelector('b').textContent = 'Deep Well of the Unbroken Constellation';
      button.querySelector('span').textContent = '+10% maximum HP per stack. Your empowered guardian restores health after every encounter.';
    });
    const overflow = await dialog.evaluate(node => [node, ...node.querySelectorAll('.ab-boon-choice-dialog,.ab-boon-draft,button,b,span,p')].map(el => ({ text: el.textContent.slice(0, 40), overflow: el.scrollWidth - el.clientWidth })));
    for (const item of overflow) expect(item.overflow, item.text).toBeLessThanOrEqual(1);
    const box = await dialog.boundingBox();
    expect(box.x).toBeGreaterThanOrEqual(15);
    expect(box.x + box.width).toBeLessThanOrEqual(viewport.width - 15);
    await expect(choices.first()).toHaveCSS('background-color', 'rgb(23, 34, 48)');
    await expect(choices.first().locator('span')).toHaveCSS('color', 'rgb(197, 210, 225)');
    await choices.first().focus();
    await page.keyboard.press('Shift+Tab');
    const later = dialog.getByRole('button', { name: 'Decide later' });
    await expect(later).toBeFocused();
    await later.click();
    await expect(dialog).toBeHidden();
  });
}
