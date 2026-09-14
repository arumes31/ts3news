const {test,expect}=require('@playwright/test');

test('guard reduction describes confirmed frontal protection and distinguishes inactive and paused guard',async({page})=>{
  await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();await page.locator('#rift-start').click();
  const indicator=page.locator('#rift-guard-reduction');await expect(indicator).toHaveText('Guard inactive');
  await page.keyboard.down('l');await expect(indicator).toHaveText('Guard: 82% frontal reduction after armor');
  expect((await(await page.request.get('/api/abyss/rift')).json()).run.player.guard).toBe(true);
  await expect(indicator).toHaveAttribute('title',/Attacks from behind bypass guard/);
  await page.keyboard.up('l');await expect(indicator).toHaveText('Guard inactive');
  await page.keyboard.down('l');await expect(indicator).toContainText('82%');await page.keyboard.press('Escape');await page.keyboard.up('l');
  await expect(page.locator('#rift-paused-badge')).toHaveText('Paused');await expect(indicator).not.toContainText('82%');
  await page.setViewportSize({width:390,height:844});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
});
