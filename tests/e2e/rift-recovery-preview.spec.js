const {test,expect}=require('@playwright/test');

test('checkpoint recovery previews match confirmed continuation and exclude leaving',async({page})=>{
  await page.goto('/abyss/rift?scenario=checkpoint&condition=wounded');await page.locator('#rift-auto').uncheck();
  await page.locator('#rift-start').click();
  await expect(page.locator('#rift-recovery-preview')).toHaveText('Continue: +85.0 HP → 255.0/340.0 HP; mana refills to 100. Leaving gives no recovery.');
  const nextResponse=page.waitForResponse(response=>response.url().endsWith('/api/abyss/rift')&&response.request().postDataJSON()?.kind==='next');
  await page.locator('#rift-next').click();const next=(await(await nextResponse).json()).run;expect(next.player.hp).toBe(255);expect(next.player.mana).toBe(100);
  await page.keyboard.press('Escape');
});

test('full-health and final checkpoint previews reflect continuation mode',async({page})=>{
  await page.goto('/abyss/rift?scenario=checkpoint&room=final');await page.locator('#rift-auto').uncheck();await page.locator('#rift-start').click();
  await expect(page.locator('#rift-recovery-preview')).toContainText('No next-tier recovery.');
  await page.locator('#rift-campaign > summary').click();await page.locator('#rift-auto').check();
  await expect(page.locator('#rift-recovery-preview')).toContainText('Continue: +0.0 HP → 340.0/340.0 HP');
  await page.keyboard.press('Escape');
});
