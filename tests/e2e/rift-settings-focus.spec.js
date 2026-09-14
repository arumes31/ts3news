const {test,expect}=require('@playwright/test');

test('returning from settings restores battlefield keys without changing the selected setting',async({page})=>{
  await page.goto('/abyss/rift');await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();await page.locator('.rift-settings > summary').click();await page.locator('#rift-jump-buffer').selectOption('200');await page.locator('#rift-jump-buffer').focus();
  await expect(page.locator('#rift-settings-return')).toBeVisible();await page.locator('#rift-settings-return').click();await expect(page.locator('#rift-canvas')).toBeFocused();await expect(page.locator('.rift-settings')).not.toHaveAttribute('open','');
  const read=async()=>(await(await page.request.get('/api/abyss/rift')).json()).run;const x=(await read()).player.x;await page.keyboard.down('KeyD');await expect.poll(async()=>(await read()).player.x).toBeGreaterThan(x);await page.keyboard.up('KeyD');await page.keyboard.press('Space');await expect.poll(async()=>(await read()).stats.jumps).toBe(1);await expect(page.locator('#rift-jump-buffer')).toHaveValue('200');await page.keyboard.press('Escape');
});

test('closing settings returns active focus but leaves paused navigation alone',async({page})=>{
  await page.goto('/abyss/rift');const summary=page.locator('.rift-settings > summary');await summary.click();await expect(page.locator('#rift-settings-return')).toBeDisabled();await summary.click();await expect(summary).toBeFocused();
  await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();await summary.click();await page.locator('#rift-effects-volume').focus();await summary.click();await expect(page.locator('#rift-canvas')).toBeFocused();
  await page.keyboard.press('Escape');await expect(page.locator('#rift-overlay')).toBeVisible();await summary.click();await expect(page.locator('#rift-settings-return')).toBeDisabled();await summary.click();await expect(summary).toBeFocused();
});
