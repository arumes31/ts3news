const {test,expect}=require('@playwright/test');

test('facing indicator follows confirmed horizontal movement and persists through vertical movement and pause',async({page})=>{
  await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();await page.locator('#rift-start').click();
  // The preview already faces right; wait for combat to accept keyboard input.
  await expect(page.locator('#rift-overlay')).toBeHidden();
  await expect(page.locator('#rift-facing')).toHaveText('Facing right →');
  await page.keyboard.down('a');await expect(page.locator('#rift-facing')).toHaveText('← Facing left');await page.keyboard.up('a');
  expect((await(await page.request.get('/api/abyss/rift')).json()).run.player.facing).toBe(-1);
  await page.keyboard.down('w');await expect.poll(async()=>(await(await page.request.get('/api/abyss/rift')).json()).run.player.y).toBeLessThan(405);await page.keyboard.up('w');
  await expect(page.locator('#rift-facing')).toHaveText('← Facing left');
  await page.keyboard.press('Escape');await expect(page.locator('#rift-paused-badge')).toHaveText('Paused');await expect(page.locator('#rift-facing')).toHaveText('← Facing left');
  await page.reload();await expect(page.locator('#rift-facing')).toHaveText('← Facing left');await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();
  await page.keyboard.down('d');await expect(page.locator('#rift-facing')).toHaveText('Facing right →');await page.keyboard.up('d');
  expect((await(await page.request.get('/api/abyss/rift')).json()).run.player.facing).toBe(1);
  await page.keyboard.press('Escape');await page.setViewportSize({width:390,height:844});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
});
